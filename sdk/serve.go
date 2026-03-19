package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/brainlet/brainkit/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"google.golang.org/grpc"
)

// Run starts the gRPC server and blocks until shutdown.
// This is the entry point for plugin main().
func (p *Plugin) Run() error {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("sdk: listen: %w", err)
	}

	fmt.Fprintf(os.Stdout, "LISTEN:%s\n", lis.Addr().String())

	srv := grpc.NewServer()
	handler := &pluginServer{plugin: p}
	pluginv1.RegisterBrainkitPluginServiceServer(srv, handler)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		srv.GracefulStop()
	}()

	return srv.Serve(lis)
}

// pluginServer implements BrainkitPluginServiceServer.
type pluginServer struct {
	pluginv1.UnimplementedBrainkitPluginServiceServer
	plugin *Plugin
	client *grpcClient
}

func (s *pluginServer) Handshake(_ context.Context, req *pluginv1.HandshakeRequest) (*pluginv1.HandshakeResponse, error) {
	if req.Version != "v1" {
		return &pluginv1.HandshakeResponse{
			Name:            s.plugin.name,
			Version:         "v1",
			Accepted:        false,
			RejectionReason: fmt.Sprintf("unsupported protocol version %q, expected v1", req.Version),
		}, nil
	}
	return &pluginv1.HandshakeResponse{
		Name:     s.plugin.name,
		Version:  "v1",
		Accepted: true,
	}, nil
}

func (s *pluginServer) Manifest(_ context.Context, _ *pluginv1.ManifestRequest) (*pluginv1.PluginManifest, error) {
	m := s.plugin.buildManifest()

	pm := &pluginv1.PluginManifest{
		Owner:       m.Owner,
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
	}

	for _, t := range m.Tools {
		pm.Tools = append(pm.Tools, &pluginv1.ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	for _, sub := range m.Subscriptions {
		pm.Subscriptions = append(pm.Subscriptions, &pluginv1.SubscriptionDefinition{
			Topic: sub.Topic,
		})
	}

	for _, e := range m.Events {
		pm.Events = append(pm.Events, &pluginv1.EventDefinition{
			Name:   e.Name,
			Schema: e.Schema,
		})
	}

	for _, i := range m.Interceptors {
		pm.Interceptors = append(pm.Interceptors, &pluginv1.InterceptorDefinition{
			Name:        i.Name,
			Priority:    int32(i.Priority),
			TopicFilter: i.TopicFilter,
		})
	}

	return pm, nil
}

func (s *pluginServer) MessageStream(stream pluginv1.BrainkitPluginService_MessageStreamServer) error {
	sendMu := &sync.Mutex{}
	safeSend := func(msg *pluginv1.PluginMessage) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return stream.Send(msg)
	}

	client := newGRPCClient(safeSend)
	s.client = client

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		switch msg.Type {
		case "lifecycle.start":
			if s.plugin.onStartFn != nil {
				// Run OnStart in a goroutine so it doesn't block the stream
				// receive loop — OnStart may call Client.Ask which needs the
				// loop to deliver ask.reply messages.
				go func() {
					if err := s.plugin.onStartFn(client); err != nil {
						log.Printf("plugin OnStart error: %v", err)
					}
				}()
			}

		case "lifecycle.stop":
			if s.plugin.onStopFn != nil {
				s.plugin.onStopFn()
			}
			return nil

		case "tool.call":
			go s.dispatchToolCall(stream.Context(), msg, safeSend)

		case "event":
			go s.dispatchEvent(stream.Context(), msg, client)

		case "intercept":
			go s.dispatchIntercept(stream.Context(), msg, safeSend)

		case "ask.reply":
			client.handleReply(msg)
		}
	}
}

func (s *pluginServer) dispatchToolCall(ctx context.Context, msg *pluginv1.PluginMessage, safeSend func(*pluginv1.PluginMessage) error) {
	toolName := msg.Topic

	s.plugin.mu.Lock()
	var handler func(context.Context, Client, json.RawMessage) (json.RawMessage, error)
	for _, t := range s.plugin.tools {
		if t.name == toolName {
			handler = t.handler
			break
		}
	}
	s.plugin.mu.Unlock()

	reply := &pluginv1.PluginMessage{
		Id:      msg.Id,
		Type:    "reply",
		ReplyTo: msg.ReplyTo,
		TraceId: msg.TraceId,
	}

	if handler == nil {
		errJSON, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("no handler for tool %q", toolName)})
		reply.Payload = errJSON
		safeSend(reply)
		return
	}

	result, err := handler(ctx, s.client, msg.Payload)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		reply.Payload = errJSON
	} else {
		reply.Payload = result
	}
	safeSend(reply)
}

func (s *pluginServer) dispatchEvent(ctx context.Context, msg *pluginv1.PluginMessage, client Client) {
	s.plugin.mu.Lock()
	var handlers []subscriptionRegistration
	for _, sub := range s.plugin.subscriptions {
		if bus.TopicMatches(sub.topic, msg.Topic) {
			handlers = append(handlers, sub)
		}
	}
	s.plugin.mu.Unlock()

	for _, sub := range handlers {
		sub.handler(ctx, msg.Payload, client, nil)
	}
}

func (s *pluginServer) dispatchIntercept(ctx context.Context, msg *pluginv1.PluginMessage, safeSend func(*pluginv1.PluginMessage) error) {
	imsg := InterceptMessage{
		Topic:    msg.Topic,
		CallerID: msg.CallerId,
		Payload:  msg.Payload,
		Metadata: msg.Metadata,
	}

	s.plugin.mu.Lock()
	var handler func(context.Context, InterceptMessage) (*InterceptMessage, error)
	for _, i := range s.plugin.interceptors {
		if bus.TopicMatches(i.topicFilter, msg.Topic) {
			handler = i.handler
			break
		}
	}
	s.plugin.mu.Unlock()

	reply := &pluginv1.PluginMessage{
		Id:      msg.Id,
		Type:    "intercept.result",
		ReplyTo: msg.ReplyTo,
		TraceId: msg.TraceId,
	}

	if handler == nil {
		reply.Payload = msg.Payload
		reply.Metadata = msg.Metadata
		safeSend(reply)
		return
	}

	result, err := handler(ctx, imsg)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		reply.Payload = errJSON
	} else if result != nil {
		reply.Payload = result.Payload
		reply.Metadata = result.Metadata
	}
	safeSend(reply)
}

func (s *pluginServer) Health(_ context.Context, _ *pluginv1.HealthRequest) (*pluginv1.HealthResponse, error) {
	return &pluginv1.HealthResponse{Healthy: true}, nil
}
