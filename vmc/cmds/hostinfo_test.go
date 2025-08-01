package cmds

import (
	"testing"
	"time"
)

func TestProto(t *testing.T) {
	now := time.Now()
	hostInfo := NewHostInfo()
	resp := hostInfo.Get()
	diff := time.Since(now)
	t.Logf("diff:%f, %v", float32(diff)/float32(time.Second), *resp)

}
