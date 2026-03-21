package plugin

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/brainlet/brainkit/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"google.golang.org/grpc"
)

// interceptResult holds proto fields from the plugin's intercept.result message.
type interceptResult struct {
	Payload  json.RawMessage
	Metadata map[string]string
}

// conn tracks one connected plugin.
type conn struct {
	config   Config
	cmd      *exec.Cmd
	grpcConn *grpc.ClientConn
	client   pluginv1.BrainkitPluginServiceClient
	stream   pluginv1.BrainkitPluginService_MessageStreamClient
	manifest *pluginv1.PluginManifest
	subs     []bus.SubscriptionID
	sendMu   sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{} // closed when readStream exits
	stopping chan struct{} // closed when stopPlugin begins graceful shutdown

	// Backpressure: buffered channel as event semaphore
	eventSem chan struct{}

	// Restart tracking
	restarts int

	// Intercept reply tracking
	interceptMu      sync.Mutex
	interceptReplies map[string]chan interceptResult
}

// safeSend sends a message on the plugin stream with mutex protection.
func (pc *conn) safeSend(msg *pluginv1.PluginMessage) error {
	pc.sendMu.Lock()
	defer pc.sendMu.Unlock()
	return pc.stream.Send(msg)
}

// safeSendEvent sends an event message with backpressure control.
func (pc *conn) safeSendEvent(msg *pluginv1.PluginMessage) bool {
	if msg.Type != "event" {
		pc.safeSend(msg)
		return true
	}

	select {
	case pc.eventSem <- struct{}{}:
		if err := pc.safeSend(msg); err != nil {
			<-pc.eventSem
			return false
		}
		return true
	default:
		log.Printf("[plugin:%s] backpressure: dropping event (max=%d, topic=%s)",
			pc.config.Name, pc.config.MaxPending, msg.Topic)
		return false
	}
}

// TestBackpressure exposes backpressure internals for root-package tests.
// Fills the semaphore, calls the test function, then drains.
func (pc *conn) TestBackpressure(maxPending int, fn func(sendEvent func(*pluginv1.PluginMessage) bool)) {
	for i := 0; i < maxPending; i++ {
		pc.eventSem <- struct{}{}
	}
	fn(pc.safeSendEvent)
	for i := 0; i < maxPending; i++ {
		<-pc.eventSem
	}
}

// TestForceCloseStream force-closes the gRPC stream for recovery testing.
func (pc *conn) TestForceCloseStream() {
	pc.sendMu.Lock()
	pc.stream.CloseSend()
	pc.sendMu.Unlock()
}

type logWriter struct {
	prefix string
}

func (w *logWriter) Write(p []byte) (int, error) {
	log.Printf("%s%s", w.prefix, strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
