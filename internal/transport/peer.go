package transport

import (
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"google.golang.org/grpc"
)

// PeerConn tracks one connected remote Kit (outbound — we connected to them).
type PeerConn struct {
	Name     string
	Addr     string
	Conn     *grpc.ClientConn
	Stream   pluginv1.BrainkitHostService_MessageStreamClient
	SendMu   sync.Mutex
	Done     chan struct{}
	DoneOnce sync.Once
	Subs     []bus.SubscriptionID
}

func (pc *PeerConn) CloseDone() {
	pc.DoneOnce.Do(func() { close(pc.Done) })
}

func (pc *PeerConn) SafeSend(msg *pluginv1.PluginMessage) error {
	pc.SendMu.Lock()
	defer pc.SendMu.Unlock()
	if pc.Stream == nil {
		return fmt.Errorf("peer %s: stream closed", pc.Name)
	}
	return pc.Stream.Send(msg)
}
