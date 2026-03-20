package brainkit

import (
	"bufio"
	"context"
	"encoding/json"
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
	cancel    context.CancelFunc
	done      chan struct{} // closed when readStream exits
	stopping  chan struct{} // closed when stopPlugin begins graceful shutdown

	// Backpressure: buffered channel as event semaphore
	eventSem chan struct{}

	// Restart tracking
	restarts int

	// Intercept reply tracking
	interceptMu      sync.Mutex
	interceptReplies map[string]chan interceptResult
}

// interceptResult holds proto fields from the plugin's intercept.result message.
type interceptResult struct {
	Payload  json.RawMessage
	Metadata map[string]string
}

// safeSend sends a message on the plugin stream with mutex protection.
func (pc *pluginConn) safeSend(msg *pluginv1.PluginMessage) error {
	pc.sendMu.Lock()
	defer pc.sendMu.Unlock()
	return pc.stream.Send(msg)
}

// safeSendEvent sends an event message with backpressure control.
// Returns true if sent, false if dropped. Critical messages bypass backpressure.
// Uses a buffered channel as semaphore — blocks until slot available or drops if full.
func (pc *pluginConn) safeSendEvent(msg *pluginv1.PluginMessage) bool {
	if msg.Type != "event" {
		pc.safeSend(msg)
		return true
	}

	// Try to acquire a slot (non-blocking)
	select {
	case pc.eventSem <- struct{}{}:
		// Got a slot — send the event
		if err := pc.safeSend(msg); err != nil {
			<-pc.eventSem // release slot on error
			return false
		}
		return true
	default:
		// Semaphore full — drop the event
		log.Printf("[plugin:%s] backpressure: dropping event (max=%d, topic=%s)",
			pc.config.Name, pc.config.MaxPending, msg.Topic)
		return false
	}
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

	// Open bidirectional stream
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
		stopping: make(chan struct{}),
		eventSem:         make(chan struct{}, cfg.MaxPending),
		interceptReplies: make(map[string]chan interceptResult),
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

	log.Printf("[plugin:%s] started at %s (owner=%s, version=%s, %d tools, %d subs)",
		cfg.Name, addr, manifest.Owner, manifest.Version,
		len(manifest.Tools), len(manifest.Subscriptions))

	return nil
}

// readStream processes messages from plugin → Kit.
// On stream error, attempts recovery if process is alive.
func (pm *pluginManager) readStream(name string, pc *pluginConn) {
	defer close(pc.done)

	for {
		msg, err := pc.stream.Recv()
		if err != nil {
			log.Printf("[plugin:%s] stream error: %v", name, err)

			// Check if process is still alive (use channels, not cmd fields — avoids race with cmd.Wait)
			select {
			case <-pc.done:
				// Process is dead, exit
				return
			case <-pc.stopping:
				// Graceful shutdown in progress, exit
				return
			default:
				// Process might still be alive — try stream recovery
				if pm.recoverStream(name, pc) {
					continue
				}
			}

			return
		}

		// Release backpressure slot when plugin sends any response
		select {
		case <-pc.eventSem:
		default:
		}

		switch msg.Type {
		case "tool.result", "reply":
			if msg.ReplyTo != "" {
				pm.kit.Bus.Send(bus.Message{
					Topic:    msg.ReplyTo,
					CallerID: "plugin/" + name,
					Payload:  msg.Payload,
					TraceID:  msg.TraceId,
				})
			}

		case "bus.send", "send":
			pm.kit.Bus.Send(bus.Message{
				Topic:    msg.Topic,
				CallerID: "plugin/" + name,
				Payload:  msg.Payload,
				TraceID:  msg.TraceId,
				Metadata: msg.Metadata,
			})

		case "bus.ask", "ask":
			go func(m *pluginv1.PluginMessage) {
				pm.kit.Bus.Ask(bus.Message{
					Topic:    m.Topic,
					CallerID: "plugin/" + name,
					Payload:  m.Payload,
					TraceID:  m.TraceId,
				}, func(reply bus.Message) {
					pc.safeSend(&pluginv1.PluginMessage{
						Id:      uuid.NewString(),
						Type:    "ask.reply",
						ReplyTo: m.ReplyTo,
						TraceId: reply.TraceID,
						Payload: reply.Payload,
					})
				})
			}(msg)

		case "intercept.result":
			replyTo := msg.ReplyTo
			if replyTo == "" {
				log.Printf("[plugin:%s] intercept.result missing reply_to", name)
				continue
			}
			result := interceptResult{
				Payload:  msg.Payload,
				Metadata: msg.Metadata,
			}
			pc.interceptMu.Lock()
			ch, ok := pc.interceptReplies[replyTo]
			if ok {
				delete(pc.interceptReplies, replyTo)
			}
			pc.interceptMu.Unlock()
			if ok {
				ch <- result
			} else {
				log.Printf("[plugin:%s] intercept.result for unknown reply %q", name, replyTo)
			}
		}
	}
}

// recoverStream attempts to re-open the MessageStream on an existing gRPC connection.
func (pm *pluginManager) recoverStream(name string, pc *pluginConn) bool {
	log.Printf("[plugin:%s] attempting stream recovery", name)

	// Use a long-lived context for the stream (not a timeout context)
	stream, err := pc.client.MessageStream(context.Background())
	if err != nil {
		log.Printf("[plugin:%s] stream recovery failed: %v", name, err)
		return false
	}

	pc.sendMu.Lock()
	pc.stream = stream
	pc.sendMu.Unlock()

	// Re-send lifecycle.start
	pc.safeSend(&pluginv1.PluginMessage{
		Id:   uuid.NewString(),
		Type: "lifecycle.start",
	})

	// Re-wire subscriptions to new stream
	pm.reprocessSubscriptions(name, pc)

	log.Printf("[plugin:%s] stream recovered", name)
	return true
}

// reprocessSubscriptions re-wires subscription forwarding to the current stream.
func (pm *pluginManager) reprocessSubscriptions(name string, pc *pluginConn) {
	for _, subID := range pc.subs {
		pm.kit.Bus.Off(subID)
	}
	pc.subs = pc.subs[:0]

	m := pc.manifest
	for _, sub := range m.Subscriptions {
		topic := sub.Topic
		subID := pm.kit.Bus.On(topic, func(msg bus.Message, _ bus.ReplyFunc) {
			pc.safeSendEvent(&pluginv1.PluginMessage{
				Id:       uuid.NewString(),
				Type:     "event",
				Topic:    msg.Topic,
				CallerId: msg.CallerID,
				TraceId:  msg.TraceID,
				Payload:  msg.Payload,
				Metadata: msg.Metadata,
			})
		})
		pc.subs = append(pc.subs, subID)
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

// healthLoop periodically checks plugin health and auto-restarts on crash.
func (pm *pluginManager) healthLoop(name string, pc *pluginConn) {
	ticker := time.NewTicker(pc.config.HealthInterval)
	defer ticker.Stop()

	healthFailures := 0
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := pc.client.Health(ctx, &pluginv1.HealthRequest{})
			cancel()

			if err != nil || !resp.Healthy {
				healthFailures++
				log.Printf("[plugin:%s] health check failed (%d): %v",
					name, healthFailures, err)
				if healthFailures >= 3 {
					log.Printf("[plugin:%s] too many health failures, stopping", name)
					pm.stopPlugin(name, pc)
					return
				}
			} else {
				healthFailures = 0
			}

		case <-pc.stopping:
			// Graceful shutdown initiated by stopPlugin — exit without restart
			return

		case <-pc.done:
			// Process died or stream closed — attempt restart
			if !pc.config.AutoRestart {
				log.Printf("[plugin:%s] died, auto-restart disabled", name)
				pm.cleanupPlugin(name, pc)
				return
			}
			if pc.restarts >= pc.config.MaxRestarts {
				log.Printf("[plugin:%s] died, max restarts reached (%d/%d)",
					name, pc.restarts, pc.config.MaxRestarts)
				pm.cleanupPlugin(name, pc)
				return
			}

			pc.restarts++
			log.Printf("[plugin:%s] crashed, restarting (%d/%d)",
				name, pc.restarts, pc.config.MaxRestarts)

			pm.cleanupPlugin(name, pc)

			// Backoff: 500ms * restart count (max 5s)
			backoff := time.Duration(pc.restarts) * 500 * time.Millisecond
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
			time.Sleep(backoff)

			// Start fresh
			cfg := pc.config
			if err := pm.startPlugin(cfg); err != nil {
				log.Printf("[plugin:%s] restart failed: %v", name, err)
			} else {
				// Transfer restart count to new pluginConn
				pm.mu.Lock()
				if newPC, ok := pm.plugins[name]; ok {
					newPC.restarts = pc.restarts
				}
				pm.mu.Unlock()
			}
			return
		}
	}
}

// cleanupPlugin cleans up a dead plugin's resources without attempting graceful stop.
func (pm *pluginManager) cleanupPlugin(name string, pc *pluginConn) {
	for _, subID := range pc.subs {
		pm.kit.Bus.Off(subID)
	}

	if pc.conn != nil {
		pc.conn.Close()
	}

	pc.cancel()

	if pc.cmd != nil && pc.cmd.Process != nil {
		pc.cmd.Wait()
	}

	pm.mu.Lock()
	delete(pm.plugins, name)
	pm.mu.Unlock()

	log.Printf("[plugin:%s] cleaned up", name)
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
	// Signal healthLoop to exit without triggering restart
	close(pc.stopping)

	for _, subID := range pc.subs {
		pm.kit.Bus.Off(subID)
	}

	// Teardown deployed .ts files (agents and files from manifest)
	if pc.manifest != nil {
		for _, a := range pc.manifest.Agents {
			source := fmt.Sprintf("__plugin_%s_agent_%s.ts", name, a.Name)
			pm.kit.Teardown(context.Background(), source)
		}
		for _, f := range pc.manifest.Files {
			source := fmt.Sprintf("__plugin_%s_%s", name, f.Path)
			pm.kit.Teardown(context.Background(), source)
		}
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
		case <-time.After(pc.config.SIGTERMTimeout):
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
