package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
	"titan-vm/vmc/downloader"
	"titan-vm/vmc/wallet"
	"titan-vm/vms/api/ws/pb"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type Tunnel struct {
	uuid      string
	conn      *websocket.Conn
	writeLock sync.Mutex

	url      string
	waitping int

	proxySessions sync.Map
	isDestroy     bool
	command       *Command
	vmapi         string
	wallet        *wallet.Wallet
}

func NewTunnel(serverUrl, uuid, vmapi string, wallet *wallet.Wallet) (*Tunnel, error) {
	tun := &Tunnel{
		uuid:      uuid,
		writeLock: sync.Mutex{},
		url:       serverUrl,
		isDestroy: false,
		vmapi:     vmapi,
		wallet:    wallet,
	}

	tun.command = &Command{tunnel: tun, downloadManager: downloader.NewManager()}

	return tun, nil
}

func (t *Tunnel) Connect() error {
	pubKey, err := t.wallet.GetPubKey(wallet.DefaultKeyName)
	if err != nil {
		return err
	}

	signBytes, err := t.wallet.Sign(wallet.DefaultKeyName, []byte(t.uuid))
	if err != nil {
		return err
	}

	header := http.Header{}
	header.Add("pubkey", hex.EncodeToString(pubKey.Bytes()))
	header.Add("sign", hex.EncodeToString(signBytes))

	url := fmt.Sprintf("%s?id=%s&os=%s&vmapi=%s", t.url, t.uuid, runtime.GOOS, t.vmapi)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, url, header)
	if err != nil {
		var data []byte
		if resp != nil {
			data, _ = io.ReadAll(resp.Body)
		}
		return fmt.Errorf("dial %s failed %s, msg:%s", url, err.Error(), string(data))
	}

	conn.SetPingHandler(func(data string) error {
		t.writePong([]byte(data))
		return nil
	})

	conn.SetPongHandler(func(data string) error {
		t.onPong([]byte(data))
		return nil
	})

	t.conn = conn

	log.Debugf("new tun %s", url)
	return nil
}

func (t *Tunnel) Destroy() error {
	if t.conn != nil {
		t.isDestroy = true
		return t.conn.Close()
	}

	return nil
}

func (t *Tunnel) IsDestroy() bool {
	return t.isDestroy
}

func (t *Tunnel) Serve() error {
	conn := t.conn
	defer conn.Close()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Error("Error reading message:", err)
			break
		}

		if messageType != websocket.BinaryMessage {
			log.Errorf("unsupport message type %d", messageType)
			continue
		}

		if err = t.onTunnelMsg(p); err != nil {
			log.Errorf("onTunnelMsg: %s", err.Error())
		}
	}

	log.Debugf("tunnel %s close", t.uuid)

	return nil
}

func (t *Tunnel) onTunnelMsg(message []byte) error {
	msg := &pb.Message{}
	err := proto.Unmarshal(message, msg)
	if err != nil {
		return fmt.Errorf("DecodeMessage failed:%s", err.Error())
	}

	log.Debugf("Tunnel.onTunnelMsg, msgType:%s, session id:%s", msg.Type.String(), msg.GetSessionId())

	switch msg.Type {
	case pb.MessageType_COMMAND:
		return t.command.onCommandMessage(msg)
	case pb.MessageType_PROXY_SESSION_CREATE:
		return t.onProxySessionCreate(msg)
	case pb.MessageType_PROXY_SESSION_DATA:
		return t.onProxySessionData(msg)
	case pb.MessageType_PROXY_SESSION_CLOSE:
		return t.onProxySessionClose(msg)
	default:
		log.Errorf("onTunnelMsg unsupoort message type %d", msg.Type)
	}

	return nil
}

func (t *Tunnel) onProxySessionCreate(msg *pb.Message) error {
	session, ok := t.proxySessions.Load(msg.GetSessionId())
	if ok {
		ps := session.(ProxySession)
		ps.close()
		// TODO: wait session delete
	}

	destAddr := &pb.DestAddr{}
	err := proto.Unmarshal(msg.GetPayload(), destAddr)
	if err != nil {
		return fmt.Errorf("decode message failed:%s", err.Error())
	}

	if destAddr.Network != "unix" && destAddr.Network != "tcp" {
		return fmt.Errorf("dest addr unsupport netowrk type %s", destAddr.Network)
	}

	conn, err := net.DialTimeout(destAddr.GetNetwork(), destAddr.GetAddr(), 3*time.Second)
	if err != nil {
		t.onProxyConnClose(msg.GetSessionId())
		return fmt.Errorf("dial %s, network: %s, failed %s", destAddr.Addr, destAddr.Network, err.Error())
	}

	proxySession := ProxySession{id: msg.GetSessionId(), conn: conn}
	t.proxySessions.Store(msg.GetSessionId(), proxySession)

	go proxySession.proxyConn(t)

	return nil
}

func (t *Tunnel) onProxySessionData(msg *pb.Message) error {
	session, ok := t.proxySessions.Load(msg.GetSessionId())
	if !ok {
		return fmt.Errorf("onProxySessionData session %s not found", msg.GetSessionId())
	}

	ps := session.(ProxySession)
	return ps.write(msg.Payload)
}

func (t *Tunnel) onProxySessionClose(msg *pb.Message) error {
	session, ok := t.proxySessions.Load(msg.GetSessionId())
	if !ok {
		return fmt.Errorf("onProxySessionData session %s not found", msg.GetSessionId())
	}

	ps := session.(ProxySession)
	ps.close()
	return nil
}

func (t *Tunnel) onProxyConnClose(sessionID string) {
	log.Debugf("Tunnel.onProxyConnClose session id:%s", sessionID)
	msg := &pb.Message{}
	msg.Type = pb.MessageType_PROXY_SESSION_CLOSE
	msg.SessionId = sessionID
	msg.Payload = nil

	buf, err := proto.Marshal(msg)
	if err != nil {
		log.Errorf("onProxyData encode message failed:%s", err.Error())
		return
	}

	if err = t.write(buf); err != nil {
		log.Errorf("write message to tunnel failed:%s", err.Error())
	}
}

func (t *Tunnel) onProxyData(sessionID string, data []byte) {
	log.Debugf("Tunnel.onProxyData session id:%s", sessionID)
	msg := &pb.Message{}
	msg.Type = pb.MessageType_PROXY_SESSION_DATA
	msg.SessionId = sessionID
	msg.Payload = data

	buf, err := proto.Marshal(msg)
	if err != nil {
		log.Errorf("onProxyData encode message failed:%s", err.Error())
		return
	}

	if err = t.write(buf); err != nil {
		log.Errorf("write message to tunnel failed:%s", err.Error())
	}
}

func (t *Tunnel) writePong(msg []byte) error {
	t.writeLock.Lock()
	defer t.writeLock.Unlock()
	return t.conn.WriteMessage(websocket.PongMessage, msg)
}

func (t *Tunnel) onPong(_ []byte) {
	t.waitping = 0
}

func (t *Tunnel) write(msg []byte) error {
	t.writeLock.Lock()
	defer t.writeLock.Unlock()
	return t.conn.WriteMessage(websocket.BinaryMessage, msg)
}
