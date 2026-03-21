package kit

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/brainlet/brainkit/internal/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// peerConn tracks one connected remote Kit (outbound — we connected to them).
type peerConn struct {
	name     string
	addr     string
	conn     *grpc.ClientConn
	stream   pluginv1.BrainkitHostService_MessageStreamClient
	sendMu   sync.Mutex
	done     chan struct{}
	doneOnce sync.Once
	subs     []bus.SubscriptionID
}

func (pc *peerConn) closeDone() {
	pc.doneOnce.Do(func() { close(pc.done) })
}

func (pc *peerConn) safeSend(msg *pluginv1.PluginMessage) error {
	pc.sendMu.Lock()
	defer pc.sendMu.Unlock()
	if pc.stream == nil {
		return fmt.Errorf("peer %s: stream closed", pc.name)
	}
	return pc.stream.Send(msg)
}

// hostServer implements BrainkitHostService for accepting incoming peer connections.
type hostServer struct {
	pluginv1.UnimplementedBrainkitHostServiceServer
	kit        *Kit
	grpcServer *grpc.Server
	listener   net.Listener
	mu         sync.Mutex
	inbound    map[string]*inboundPeer
}

// inboundPeer tracks a remote Kit that connected TO us.
type inboundPeer struct {
	name   string
	stream pluginv1.BrainkitHostService_MessageStreamServer
	sendMu sync.Mutex
	cancel context.CancelFunc
	subs   []bus.SubscriptionID
}

func (p *inboundPeer) safeSend(msg *pluginv1.PluginMessage) error {
	p.sendMu.Lock()
	defer p.sendMu.Unlock()
	return p.stream.Send(msg)
}

func newHostServer(kit *Kit) *hostServer {
	return &hostServer{
		kit:     kit,
		inbound: make(map[string]*inboundPeer),
	}
}

func (s *hostServer) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("network: listen %s: %w", addr, err)
	}
	s.listener = lis
	s.grpcServer = grpc.NewServer()
	pluginv1.RegisterBrainkitHostServiceServer(s.grpcServer, s)

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Printf("[network] server stopped: %v", err)
		}
	}()

	log.Printf("[network] listening on %s", lis.Addr().String())
	return nil
}

func (s *hostServer) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

func (s *hostServer) Stop() {
	s.mu.Lock()
	for _, peer := range s.inbound {
		for _, subID := range peer.subs {
			s.kit.Bus.Off(subID)
		}
		peer.cancel()
	}
	s.inbound = make(map[string]*inboundPeer)
	s.mu.Unlock()

	if s.grpcServer != nil {
		s.grpcServer.Stop() // Force stop — GracefulStop blocks forever on open bidirectional streams
	}
}

func (s *hostServer) Handshake(_ context.Context, req *pluginv1.HandshakeRequest) (*pluginv1.HandshakeResponse, error) {
	if req.Version != "v1" {
		return &pluginv1.HandshakeResponse{
			Name:            s.kit.config.Name,
			Version:         "v1",
			Accepted:        false,
			RejectionReason: fmt.Sprintf("unsupported protocol version: %q", req.Version),
		}, nil
	}
	if req.Type != "kit" {
		return &pluginv1.HandshakeResponse{
			Name:            s.kit.config.Name,
			Version:         "v1",
			Accepted:        false,
			RejectionReason: fmt.Sprintf("expected type 'kit', got %q", req.Type),
		}, nil
	}

	log.Printf("[network] accepted handshake from %q", req.Name)
	return &pluginv1.HandshakeResponse{
		Name:     s.kit.config.Name,
		Version:  "v1",
		Accepted: true,
	}, nil
}

func (s *hostServer) MessageStream(stream pluginv1.BrainkitHostService_MessageStreamServer) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// First message identifies the peer
	firstMsg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("network: receive peer identity: %w", err)
	}

	peerName := firstMsg.CallerId
	if peerName == "" {
		peerName = "unknown-" + uuid.NewString()[:8]
	}

	peer := &inboundPeer{
		name:   peerName,
		stream: stream,
		cancel: cancel,
	}

	s.mu.Lock()
	s.inbound[peerName] = peer
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		for _, subID := range peer.subs {
			s.kit.Bus.Off(subID)
		}
		delete(s.inbound, peerName)
		s.mu.Unlock()
		log.Printf("[network] peer %q disconnected", peerName)
	}()

	log.Printf("[network] peer %q connected via inbound stream", peerName)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("network: peer %s recv: %w", peerName, err)
		}

		s.handleInboundMessage(peerName, peer, msg)
	}
}

func (s *hostServer) handleInboundMessage(peerName string, peer *inboundPeer, msg *pluginv1.PluginMessage) {
	switch msg.Type {
	case "send":
		s.kit.Bus.Send(bus.Message{
			Topic:    msg.Topic,
			CallerID: "peer/" + peerName,
			TraceID:  msg.TraceId,
			Payload:  msg.Payload,
			Metadata: msg.Metadata,
		})

	case "ask":
		replyTo := msg.ReplyTo
		s.kit.Bus.Ask(bus.Message{
			Topic:    msg.Topic,
			CallerID: "peer/" + peerName,
			TraceID:  msg.TraceId,
			Payload:  msg.Payload,
			Metadata: msg.Metadata,
		}, func(reply bus.Message) {
			peer.safeSend(&pluginv1.PluginMessage{
				Id:       uuid.NewString(),
				Type:     "ask.reply",
				ReplyTo:  replyTo,
				Payload:  reply.Payload,
				Metadata: reply.Metadata,
			})
		})

	case "subscribe":
		pattern := msg.Topic
		subID := s.kit.Bus.On(pattern, func(busMsg bus.Message, _ bus.ReplyFunc) {
			peer.safeSend(&pluginv1.PluginMessage{
				Id:       uuid.NewString(),
				Type:     "event",
				Topic:    busMsg.Topic,
				CallerId: busMsg.CallerID,
				TraceId:  busMsg.TraceID,
				Payload:  busMsg.Payload,
				Metadata: busMsg.Metadata,
			})
		})
		peer.subs = append(peer.subs, subID)

	default:
		log.Printf("[network] peer %s: unknown message type %q", peerName, msg.Type)
	}
}
