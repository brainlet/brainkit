package plugin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"syscall"
	"time"

	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (pm *Manager) startPlugin(cfg Config) error {
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

	grpcConn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("grpc connect: %w", err)
	}

	client := pluginv1.NewBrainkitPluginServiceClient(grpcConn)

	hsCtx, hsCancel := context.WithTimeout(ctx, 10*time.Second)
	hsResp, err := client.Handshake(hsCtx, &pluginv1.HandshakeRequest{
		Name:    pm.bridge.KitName(),
		Version: "v1",
		Type:    "kit",
	})
	hsCancel()
	if err != nil {
		grpcConn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("handshake: %w", err)
	}
	if !hsResp.Accepted {
		grpcConn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("handshake rejected: %s", hsResp.RejectionReason)
	}
	if hsResp.Version != "v1" {
		grpcConn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("unsupported plugin protocol version %q", hsResp.Version)
	}

	manifest, err := client.Manifest(ctx, &pluginv1.ManifestRequest{})
	if err != nil {
		grpcConn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("manifest: %w", err)
	}

	stream, err := client.MessageStream(ctx)
	if err != nil {
		grpcConn.Close()
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("open stream: %w", err)
	}

	pc := &conn{
		config:           cfg,
		cmd:              cmd,
		grpcConn:         grpcConn,
		client:           client,
		stream:           stream,
		manifest:         manifest,
		cancel:           cancel,
		done:             make(chan struct{}),
		stopping:         make(chan struct{}),
		eventSem:         make(chan struct{}, cfg.MaxPending),
		interceptReplies: make(map[string]chan interceptResult),
	}

	pm.processManifest(cfg.Name, pc)

	go pm.readStream(cfg.Name, pc)
	go pm.healthLoop(cfg.Name, pc)

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

func (pm *Manager) healthLoop(name string, pc *conn) {
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
			return

		case <-pc.done:
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

			backoff := time.Duration(pc.restarts) * 500 * time.Millisecond
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
			time.Sleep(backoff)

			cfg := pc.config
			if err := pm.startPlugin(cfg); err != nil {
				log.Printf("[plugin:%s] restart failed: %v", name, err)
			} else {
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

func (pm *Manager) cleanupPlugin(name string, pc *conn) {
	for _, subID := range pc.subs {
		pm.bridge.Bus().Off(subID)
	}

	if pc.grpcConn != nil {
		pc.grpcConn.Close()
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

func (pm *Manager) StopAll() {
	pm.mu.Lock()
	plugins := make(map[string]*conn, len(pm.plugins))
	for k, v := range pm.plugins {
		plugins[k] = v
	}
	pm.mu.Unlock()

	for name, pc := range plugins {
		pm.stopPlugin(name, pc)
	}
}

func (pm *Manager) stopPlugin(name string, pc *conn) {
	close(pc.stopping)

	for _, subID := range pc.subs {
		pm.bridge.Bus().Off(subID)
	}

	if pc.manifest != nil {
		for _, a := range pc.manifest.Agents {
			source := fmt.Sprintf("__plugin_%s_agent_%s.ts", name, a.Name)
			pm.bridge.Teardown(context.Background(), source)
		}
		for _, f := range pc.manifest.Files {
			source := fmt.Sprintf("__plugin_%s_%s", name, f.Path)
			pm.bridge.Teardown(context.Background(), source)
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

	pc.grpcConn.Close()
	pc.cancel()
	pc.cmd.Wait()

	pm.mu.Lock()
	delete(pm.plugins, name)
	pm.mu.Unlock()

	log.Printf("[plugin:%s] stopped", name)
}
