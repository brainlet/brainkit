package kit

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
)

// pluginManager manages plugin subprocesses for a Node.
type pluginManager struct {
	node    *Node
	plugins map[string]*pluginConn
	mu      sync.Mutex
}

// pluginConn tracks one connected plugin subprocess.
type pluginConn struct {
	config  PluginConfig
	identity string
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	done    chan struct{}
}

func newPluginManager(node *Node) *pluginManager {
	return &pluginManager{
		node:    node,
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

	// Pass transport config via environment
	var env []string
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, fmt.Sprintf("BRAINKIT_TRANSPORT=%s", pm.node.config.Messaging.Transport))
	env = append(env, fmt.Sprintf("BRAINKIT_NATS_URL=%s", pm.node.config.Messaging.NATSURL))
	env = append(env, fmt.Sprintf("BRAINKIT_NATS_NAME=%s", pm.node.config.Messaging.NATSName))
	env = append(env, fmt.Sprintf("BRAINKIT_NAMESPACE=%s", pm.node.Kernel.Namespace()))
	env = append(env, fmt.Sprintf("BRAINKIT_NODE_ID=%s", pm.node.nodeID))
	if len(cfg.Config) > 0 {
		env = append(env, fmt.Sprintf("BRAINKIT_PLUGIN_CONFIG=%s", string(cfg.Config)))
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

	// Read READY line with timeout
	readyCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "READY:") {
				readyCh <- strings.TrimPrefix(line, "READY:")
			}
		}
		io.Copy(io.Discard, stdout)
	}()

	select {
	case identity := <-readyCh:
		log.Printf("[plugin:%s] ready: %s", cfg.Name, identity)
	case <-time.After(cfg.StartTimeout):
		cancel()
		cmd.Process.Kill()
		return fmt.Errorf("timeout waiting for READY line")
	}

	pc := &pluginConn{
		config:   cfg,
		identity: cfg.Name,
		cmd:      cmd,
		cancel:   cancel,
		done:     make(chan struct{}),
	}

	// Watch for process exit
	go func() {
		cmd.Wait()
		close(pc.done)
	}()

	pm.mu.Lock()
	pm.plugins[cfg.Name] = pc
	pm.mu.Unlock()

	log.Printf("[plugin:%s] started (pid=%d)", cfg.Name, cmd.Process.Pid)
	return nil
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
	// Send SIGTERM
	if pc.cmd.Process != nil {
		pc.cmd.Process.Signal(syscall.SIGTERM)
	}

	select {
	case <-pc.done:
	case <-time.After(pc.config.ShutdownTimeout):
		log.Printf("[plugin:%s] shutdown timeout, killing", name)
		if pc.cmd.Process != nil {
			pc.cmd.Process.Kill()
		}
	}

	pc.cancel()

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
