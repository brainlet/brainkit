package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"google.golang.org/grpc"
)

// Serve starts the gRPC server and blocks until shutdown.
// Prints "LISTEN:{addr}\n" to stdout so the Kit can connect.
func Serve(plugin Plugin) error {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("sdk: listen: %w", err)
	}

	// Print LISTEN line — Kit reads this from stdout
	fmt.Fprintf(os.Stdout, "LISTEN:%s\n", lis.Addr().String())

	srv := grpc.NewServer()
	handler := &pluginServer{plugin: plugin}
	pluginv1.RegisterBrainkitPluginServiceServer(srv, handler)

	// Handle shutdown signals
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
	plugin Plugin
	client *brainkitClient
}

func (s *pluginServer) Handshake(_ context.Context, req *pluginv1.HandshakeRequest) (*pluginv1.HandshakeResponse, error) {
	manifest := s.plugin.Manifest()
	if req.Version != "v1" {
		return &pluginv1.HandshakeResponse{
			Name:            manifest.Name,
			Version:         "v1",
			Accepted:        false,
			RejectionReason: fmt.Sprintf("unsupported protocol version %q, expected v1", req.Version),
		}, nil
	}
	return &pluginv1.HandshakeResponse{
		Name:     manifest.Name,
		Version:  "v1",
		Accepted: true,
	}, nil
}

func (s *pluginServer) Manifest(_ context.Context, _ *pluginv1.ManifestRequest) (*pluginv1.PluginManifest, error) {
	m := s.plugin.Manifest()
	pm := &pluginv1.PluginManifest{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
	}
	for _, t := range m.Tools {
		pm.Tools = append(pm.Tools, &pluginv1.ToolDefinition{
			Name:         t.Name,
			Description:  t.Description,
			InputSchema:  t.InputSchema,
			OutputSchema: t.OutputSchema,
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
	for _, a := range m.Agents {
		pm.Agents = append(pm.Agents, &pluginv1.AgentDefinition{
			Name:         a.Name,
			Description:  a.Description,
			Model:        a.Model,
			Tools:        a.Tools,
			Instructions: a.Instructions,
		})
	}
	for _, f := range m.Files {
		pm.Files = append(pm.Files, &pluginv1.FileDefinition{
			Path:    f.Path,
			Type:    f.Type,
			Content: f.Content,
		})
	}
	return pm, nil
}

func (s *pluginServer) MessageStream(stream pluginv1.BrainkitPluginService_MessageStreamServer) error {
	// Thread-safe stream sender (gRPC Send is NOT thread-safe)
	sendMu := &sync.Mutex{}
	safeSend := func(msg *pluginv1.PluginMessage) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return stream.Send(msg)
	}

	client := newBrainkitClient(safeSend)
	s.client = client

	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		switch msg.Type {
		case "lifecycle.start":
			if err := s.plugin.OnStart(client); err != nil {
				return fmt.Errorf("plugin OnStart: %w", err)
			}

		case "lifecycle.stop":
			s.plugin.OnStop()
			return nil

		case "tool.call":
			go func(m *pluginv1.PluginMessage) {
				result, terr := s.plugin.HandleToolCall(
					stream.Context(), m.Topic, m.Payload,
				)
				reply := &pluginv1.PluginMessage{
					Id:      m.Id,
					Type:    "tool.result",
					ReplyTo: m.ReplyTo,
					TraceId: m.TraceId,
				}
				if terr != nil {
					errJSON, _ := json.Marshal(map[string]string{"error": terr.Error()})
					reply.Payload = errJSON
				} else {
					reply.Payload = result
				}
				safeSend(reply)
			}(msg)

		case "event":
			go func(m *pluginv1.PluginMessage) {
				s.plugin.HandleEvent(stream.Context(), Event{
					Topic:    m.Topic,
					Payload:  m.Payload,
					TraceID:  m.TraceId,
					CallerID: m.CallerId,
				})
			}(msg)

		case "intercept":
			go func(m *pluginv1.PluginMessage) {
				imsg := InterceptMessage{
					Topic:    m.Topic,
					CallerID: m.CallerId,
					Payload:  m.Payload,
					Metadata: m.Metadata,
				}
				result, ierr := s.plugin.HandleIntercept(stream.Context(), imsg)
				reply := &pluginv1.PluginMessage{
					Id:      m.Id,
					Type:    "intercept.result",
					ReplyTo: m.ReplyTo,
					TraceId: m.TraceId,
				}
				if ierr != nil {
					errJSON, _ := json.Marshal(map[string]string{"error": ierr.Error()})
					reply.Payload = errJSON
				} else if result != nil {
					reply.Payload = result.Payload
					reply.Metadata = result.Metadata
				}
				safeSend(reply)
			}(msg)

		case "bus.ask.reply":
			client.handleReply(msg)
		}
	}
}

func (s *pluginServer) Health(_ context.Context, _ *pluginv1.HealthRequest) (*pluginv1.HealthResponse, error) {
	return &pluginv1.HealthResponse{Healthy: true}, nil
}
