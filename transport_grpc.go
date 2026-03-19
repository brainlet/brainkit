package brainkit

import (
	"strings"
	"sync"

	"github.com/brainlet/brainkit/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/google/uuid"
)

// GRPCTransport implements bus.Transport by combining local delivery
// with gRPC-based forwarding to remote Kits.
type GRPCTransport struct {
	local       *bus.InProcessTransport
	mu          sync.RWMutex
	peers       map[string]*peerConn
	discovery   Discovery                    // optional — resolve unknown peers
	ConnectFunc func(name, addr string) error // callback to connect (avoids Kit back-reference)
}

func NewGRPCTransport() *GRPCTransport {
	return &GRPCTransport{
		local: bus.NewInProcessTransport(),
		peers: make(map[string]*peerConn),
	}
}

func (t *GRPCTransport) Publish(msg bus.Message) error {
	return t.local.Publish(msg)
}

func (t *GRPCTransport) Forward(msg bus.Message, target string) error {
	peerName := extractPeerName(target)

	t.mu.RLock()
	peer, ok := t.peers[peerName]
	t.mu.RUnlock()

	if !ok {
		// Try discovery before giving up
		if t.discovery != nil && t.ConnectFunc != nil {
			addr, err := t.discovery.Resolve(peerName)
			if err == nil {
				if connectErr := t.ConnectFunc(peerName, addr); connectErr == nil {
					t.mu.RLock()
					peer, ok = t.peers[peerName]
					t.mu.RUnlock()
				}
			}
		}
		if !ok {
			return bus.ErrNoRoute
		}
	}

	msgType := "send"
	if msg.ReplyTo != "" {
		msgType = "ask"
	}

	return peer.safeSend(&pluginv1.PluginMessage{
		Id:       uuid.NewString(),
		Type:     msgType,
		Topic:    msg.Topic,
		CallerId: msg.CallerID,
		TraceId:  msg.TraceID,
		ReplyTo:  msg.ReplyTo,
		Payload:  msg.Payload,
		Metadata: msg.Metadata,
		Address:  msg.Address,
	})
}

func (t *GRPCTransport) Subscribe(info bus.SubscriberInfo) error {
	return t.local.Subscribe(info)
}

func (t *GRPCTransport) Unsubscribe(id bus.SubscriptionID) error {
	return t.local.Unsubscribe(id)
}

func (t *GRPCTransport) Metrics() bus.TransportMetrics {
	return t.local.Metrics()
}

func (t *GRPCTransport) SubscriberCount() int {
	return t.local.SubscriberCount()
}

func (t *GRPCTransport) Close() error {
	t.mu.Lock()
	for name, peer := range t.peers {
		peer.closeDone()
		if peer.conn != nil {
			peer.conn.Close()
		}
		delete(t.peers, name)
	}
	t.mu.Unlock()
	return t.local.Close()
}

func (t *GRPCTransport) addPeer(pc *peerConn) {
	t.mu.Lock()
	t.peers[pc.name] = pc
	t.mu.Unlock()
}

func (t *GRPCTransport) removePeer(name string) {
	t.mu.Lock()
	delete(t.peers, name)
	t.mu.Unlock()
}

func extractPeerName(addr string) string {
	if strings.HasPrefix(addr, "kit:") {
		return strings.TrimPrefix(addr, "kit:")
	}
	if strings.HasPrefix(addr, "host:") {
		return strings.TrimPrefix(addr, "host:")
	}
	return addr
}
