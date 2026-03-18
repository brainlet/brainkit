package brainkit

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/brainlet/brainkit/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// pluginManager manages all plugin subprocesses for a Kit.
type pluginManager struct {
	kit     *Kit
	plugins map[string]*pluginConn
	mu      sync.Mutex
}

// pluginConn tracks one connected plugin.
type pluginConn struct {
	config   PluginConfig
	cmd      *exec.Cmd
	conn     *grpc.ClientConn
	client   pluginv1.BrainkitPluginServiceClient
	stream   pluginv1.BrainkitPluginService_MessageStreamClient
	manifest *pluginv1.PluginManifest
	subs     []bus.SubscriptionID
	sendMu   sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
}

// safeSend sends a message on the plugin stream with mutex protection.
func (pc *pluginConn) safeSend(msg *pluginv1.PluginMessage) error {
	pc.sendMu.Lock()
	defer pc.sendMu.Unlock()
	return pc.stream.Send(msg)
}

func newPluginManager(kit *Kit) *pluginManager {
	return &pluginManager{
		kit:     kit,
		plugins: make(map[string]*pluginConn),
	}
}

func (pm *pluginManager) startAll(configs []PluginConfig) {
	for i := range configs {
		cfg := configs[i]
		pluginDefaults(&cfg)
		if err := pm.startPlugin(cfg); err != nil {
			log.Printf("[plugin:%s] failed to start: %v", cfg.Name, err)
		}
	}
}

func (pm *pluginManager) startPlugin(cfg PluginConfig) error {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, cfg.Binary, cfg.Args...)

	// Set environment
	var env []string
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	if cfg.Config != nil {
		env = append(env, fmt.Sprintf("PLUGIN_CONFIG=%s", string(cfg.Config)))
	}
	cmd.Env = append(cmd.Environ(), env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = &logWriter{prefix: fmt.Sprintf("[plugin:%s] ", cfg.Name)}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start: %w", err)
	}

	// Read LISTEN line with timeout
	addrCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "LISTEN:") {
				addrCh <- strings.TrimPrefix(line, "LISTEN:")
			}
		}
		io.Copy(io.Discard, stdout)
	}()

	var addr string
	select {
	case addr = <-addrCh:
	case <-time.After(cfg.StartTimeout):
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("timeout waiting for LISTEN line")
	}

	// gRPC connect
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("grpc connect: %w", err)
	}

	client := pluginv1.NewBrainkitPluginServiceClient(conn)

	// Handshake
	hsCtx, hsCancel := context.WithTimeout(ctx, 10*time.Second)
	hsResp, err := client.Handshake(hsCtx, &pluginv1.HandshakeRequest{
		Name:    pm.kit.config.Name,
		Version: "v1",
		Type:    "kit",
	})
	hsCancel()
	if err != nil {
		conn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("handshake: %w", err)
	}
	if !hsResp.Accepted {
		conn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("handshake rejected: %s", hsResp.RejectionReason)
	}
	if hsResp.Version != "v1" {
		conn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("unsupported plugin protocol version %q", hsResp.Version)
	}

	// Get manifest
	manifest, err := client.Manifest(ctx, &pluginv1.ManifestRequest{})
	if err != nil {
		conn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("manifest: %w", err)
	}

	// Open bidirectional stream BEFORE processing manifest
	stream, err := client.MessageStream(ctx)
	if err != nil {
		conn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("open stream: %w", err)
	}

	pc := &pluginConn{
		config:   cfg,
		cmd:      cmd,
		conn:     conn,
		client:   client,
		stream:   stream,
		manifest: manifest,
		cancel:   cancel,
		done:     make(chan struct{}),
	}

	// Process manifest
	pm.processManifest(cfg.Name, pc)

	// Start stream reader and health loop
	go pm.readStream(cfg.Name, pc)
	go pm.healthLoop(cfg.Name, pc)

	// Send lifecycle.start
	pc.safeSend(&pluginv1.PluginMessage{
		Id:   uuid.NewString(),
		Type: "lifecycle.start",
	})

	pm.mu.Lock()
	pm.plugins[cfg.Name] = pc
	pm.mu.Unlock()

	log.Printf("[plugin:%s] started at %s (%d tools, %d subscriptions)",
		cfg.Name, addr, len(manifest.Tools), len(manifest.Subscriptions))

	return nil
}

// readStream processes messages from plugin → Kit.
func (pm *pluginManager) readStream(name string, pc *pluginConn) {
	defer close(pc.done)

	for {
		msg, err := pc.stream.Recv()
		if err != nil {
			log.Printf("[plugin:%s] stream closed: %v", name, err)
			return
		}

		switch msg.Type {
		case "tool.result":
			if msg.ReplyTo != "" {
				pm.kit.Bus.Send(bus.Message{
					Topic:    msg.ReplyTo,
					CallerID: "plugin." + name,
					Payload:  msg.Payload,
					TraceID:  msg.TraceId,
				})
			}

		case "bus.send":
			pm.kit.Bus.Send(bus.Message{
				Topic:    msg.Topic,
				CallerID: "plugin." + name,
				Payload:  msg.Payload,
				TraceID:  msg.TraceId,
				Metadata: msg.Metadata,
			})

		case "bus.ask":
			go func(m *pluginv1.PluginMessage) {
				pm.kit.Bus.Ask(bus.Message{
					Topic:    m.Topic,
					CallerID: "plugin." + name,
					Payload:  m.Payload,
					TraceID:  m.TraceId,
				}, func(reply bus.Message) {
					pc.safeSend(&pluginv1.PluginMessage{
						Id:      uuid.NewString(),
						Type:    "bus.ask.reply",
						ReplyTo: m.ReplyTo,
						TraceId: reply.TraceID,
						Payload: reply.Payload,
					})
				})
			}(msg)

		case "intercept.result":
			if msg.ReplyTo != "" {
				pm.kit.Bus.Send(bus.Message{
					Topic:    msg.ReplyTo,
					CallerID: "plugin." + name,
					Payload:  msg.Payload,
					TraceID:  msg.TraceId,
				})
			}
		}
	}
}

// callPluginTool sends a tool.call to the plugin and waits for tool.result.
func (pm *pluginManager) callPluginTool(pluginName, toolName string, input []byte) ([]byte, error) {
	pm.mu.Lock()
	pc, ok := pm.plugins[pluginName]
	pm.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("plugin %q not connected", pluginName)
	}

	replyTo := "_plugin_tool." + uuid.NewString()

	ch := make(chan []byte, 1)
	subID := pm.kit.Bus.On(replyTo, func(msg bus.Message, _ bus.ReplyFunc) {
		ch <- msg.Payload
	})
	defer pm.kit.Bus.Off(subID)

	pc.safeSend(&pluginv1.PluginMessage{
		Id:      uuid.NewString(),
		Type:    "tool.call",
		Topic:   toolName,
		ReplyTo: replyTo,
		Payload: input,
	})

	select {
	case result := <-ch:
		return result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("plugin %q tool %q: timeout", pluginName, toolName)
	}
}

// healthLoop periodically checks plugin health.
func (pm *pluginManager) healthLoop(name string, pc *pluginConn) {
	ticker := time.NewTicker(pc.config.HealthInterval)
	defer ticker.Stop()

	failures := 0
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := pc.client.Health(ctx, &pluginv1.HealthRequest{})
			cancel()

			if err != nil || !resp.Healthy {
				failures++
				log.Printf("[plugin:%s] health check failed (%d/%d): %v",
					name, failures, pc.config.MaxRestarts, err)
				if failures >= pc.config.MaxRestarts {
					log.Printf("[plugin:%s] max health failures reached, stopping", name)
					pm.stopPlugin(name, pc)
					return
				}
			} else {
				failures = 0
			}
		case <-pc.done:
			return
		}
	}
}

func (pm *pluginManager) stopAll() {
	pm.mu.Lock()
	plugins := make(map[string]*pluginConn, len(pm.plugins))
	for k, v := range pm.plugins {
		plugins[k] = v
	}
	pm.mu.Unlock()

	for name, pc := range plugins {
		pm.stopPlugin(name, pc)
	}
}

func (pm *pluginManager) stopPlugin(name string, pc *pluginConn) {
	for _, subID := range pc.subs {
		pm.kit.Bus.Off(subID)
	}

	pc.safeSend(&pluginv1.PluginMessage{
		Id:   uuid.NewString(),
		Type: "lifecycle.stop",
	})

	select {
	case <-pc.done:
	case <-time.After(pc.config.ShutdownTimeout):
		log.Printf("[plugin:%s] shutdown timeout, sending SIGTERM", name)
		if pc.cmd.Process != nil {
			pc.cmd.Process.Signal(syscall.SIGTERM)
		}
		select {
		case <-pc.done:
		case <-time.After(3 * time.Second):
			log.Printf("[plugin:%s] SIGTERM timeout, sending SIGKILL", name)
			if pc.cmd.Process != nil {
				pc.cmd.Process.Kill()
			}
		}
	}

	pc.conn.Close()
	pc.cancel()
	pc.cmd.Wait()

	pm.mu.Lock()
	delete(pm.plugins, name)
	pm.mu.Unlock()

	log.Printf("[plugin:%s] stopped", name)
}

type logWriter struct {
	prefix string
}

func (w *logWriter) Write(p []byte) (int, error) {
	log.Printf("%s%s", w.prefix, strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
