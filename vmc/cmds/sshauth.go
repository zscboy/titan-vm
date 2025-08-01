package cmds

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"titan-vm/vms/api/ws/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"
)

const (
	vmapiMultipass           = "multipass"
	vmapiLibvirt             = "libvirt"
	linuxSSHAuthFile         = "/root/.ssh/authorized_keys"
	linuxMultipassAuthFile   = "/var/snap/multipass/common/data/multipassd/authenticated-certs/multipass_client_certs.pem"
	windowsMultipassAuthFile = "C://ProgramData//Multipass//data//authenticated-certs/multipass_client_certs.pem"
)

type SSHAuth struct {
}

func NewSSHAuth(vmapi string) *SSHAuth {
	return &SSHAuth{}
}

func (auth *SSHAuth) Auth(req []byte) *pb.CmdSSHAuthResponse {
	logx.Debug("Auth")
	return auth.auth(req)
}

func (auth *SSHAuth) auth(req []byte) *pb.CmdSSHAuthResponse {
	authRequest := &pb.CmdSSHAuthRequest{}
	err := proto.Unmarshal(req, authRequest)
	if err != nil {
		return &pb.CmdSSHAuthResponse{ErrMsg: err.Error()}
	}

	if runtime.GOOS == "linux" {
		if err := auth.addSSHPubKey(authRequest.SshPubKey); err != nil {
			return &pb.CmdSSHAuthResponse{ErrMsg: err.Error()}
		}

		port, err := auth.getSSHPort()
		if err != nil {
			return &pb.CmdSSHAuthResponse{ErrMsg: err.Error()}
		}

		return &pb.CmdSSHAuthResponse{SshPort: port, Success: true}

	}

	return &pb.CmdSSHAuthResponse{ErrMsg: fmt.Sprintf("ssh auth unsupport os %s", runtime.GOOS)}
}

func (auth *SSHAuth) addSSHPubKey(pubKey []byte) error {
	if err := auth.createFileIfNotExist(linuxSSHAuthFile); err != nil {
		return err
	}

	authFile, err := os.ReadFile(linuxSSHAuthFile)
	if err != nil {
		return err
	}

	authFileString := string(authFile)
	if strings.Contains(authFileString, string(pubKey)) {
		return nil
	}

	file, err := os.OpenFile(linuxSSHAuthFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = file.Write(pubKey); err != nil {
		return err
	}
	return nil
}

func (auth *SSHAuth) getSSHPort() (int32, error) {
	defaultPort := int32(22)
	file, err := os.Open("/etc/ssh/sshd_config")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(strings.ToLower(line), "port ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				portStr := parts[1]
				port, err := strconv.Atoi(portStr)
				if err != nil {
					return 0, err
				}
				return int32(port), nil
			}
		}
	}
	return defaultPort, nil
}

func (auth *SSHAuth) createFileIfNotExist(filePath string) error {
	_, err := os.Stat(filePath)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}
