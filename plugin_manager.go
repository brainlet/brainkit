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

	"github.com/brainlet/brainkit/sdk"
)

// RunningPlugin describes a running plugin process.
type RunningPlugin struct {
	Name     string
	PID      int
	Uptime   time.Duration
	Status   string // "running"
	Restarts int
	Config   PluginConfig
}

// pluginManager manages plugin subprocesses for a Node.
type pluginManager struct {
	node         *Node
	plugins      map[string]*pluginConn
	mu           sync.Mutex
	startCounter int32
}

// pluginConn tracks one connected plugin subprocess.
type pluginConn struct {
	config    PluginConfig
	identity  string
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	done      chan struct{} // closed when process exits AND no restart will happen
	startedAt time.Time

	mu       sync.Mutex
	restarts int  // number of times restarted after crash
	stopping bool // true when stopPlugin was called — prevents auto-restart
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
		if err := pm.startPlugin(cfg, 0); err != nil {
			InvokeErrorHandler(pm.node.Kernel.config.ErrorHandler, fmt.Errorf("plugin %s: %w", cfg.Name, err), ErrorContext{
				Operation: "StartPlugin", Component: "plugin", Source: cfg.Name,
			})
		}
	}
}

// startPlugin launches the plugin subprocess. restartCount tracks how many
// times this plugin has been restarted (0 for initial start).
func (pm *pluginManager) startPlugin(cfg PluginConfig, restartCount int) error {
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
	// Resolve $secret: references in env
	if pm.node.Kernel.secretStore != nil {
		for i, e := range env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 && strings.HasPrefix(parts[1], "$secret:") {
				secretName := strings.TrimPrefix(parts[1], "$secret:")
				val, err := pm.node.Kernel.secretStore.Get(context.Background(), secretName)
				if err == nil && val != "" {
					env[i] = parts[0] + "=" + val
				}
			}
		}
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
		return &sdk.TimeoutError{Operation: "plugin READY"}
	}

	pc := &pluginConn{
		config:    cfg,
		identity:  cfg.Name,
		cmd:       cmd,
		cancel:    cancel,
		done:      make(chan struct{}),
		restarts:  restartCount,
		startedAt: time.Now(),
	}

	pm.mu.Lock()
	pm.plugins[cfg.Name] = pc
	pm.mu.Unlock()

	// Watch for process exit — log reason and auto-restart if configured
	go pm.watchProcess(pc)

	if restartCount > 0 {
		log.Printf("[plugin:%s] restarted (pid=%d, restart #%d)", cfg.Name, cmd.Process.Pid, restartCount)
	} else {
		log.Printf("[plugin:%s] started (pid=%d)", cfg.Name, cmd.Process.Pid)
	}
	return nil
}

// watchProcess waits for the plugin to exit, logs the reason, and auto-restarts
// if configured. Runs in its own goroutine.
func (pm *pluginManager) watchProcess(pc *pluginConn) {
	err := pc.cmd.Wait()

	// Log exit reason
	exitCode := -1
	exitSignal := ""
	if pc.cmd.ProcessState != nil {
		exitCode = pc.cmd.ProcessState.ExitCode()
		if ws, ok := pc.cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() {
				exitSignal = ws.Signal().String()
			}
		}
	}

	pc.mu.Lock()
	stopping := pc.stopping
	pc.mu.Unlock()

	if stopping {
		// Intentional shutdown — don't restart
		close(pc.done)
		return
	}

	// Unexpected exit — log it
	if exitSignal != "" {
		log.Printf("[plugin:%s] crashed: signal %s (exit code %d)", pc.config.Name, exitSignal, exitCode)
	} else if err != nil {
		log.Printf("[plugin:%s] crashed: %v (exit code %d)", pc.config.Name, err, exitCode)
	} else {
		log.Printf("[plugin:%s] exited: code %d", pc.config.Name, exitCode)
	}

	// Auto-restart if configured
	if !pc.config.AutoRestart {
		log.Printf("[plugin:%s] auto-restart disabled, not restarting", pc.config.Name)
		close(pc.done)
		return
	}

	nextRestart := pc.restarts + 1
	if nextRestart > pc.config.MaxRestarts {
		log.Printf("[plugin:%s] max restarts reached (%d/%d), giving up", pc.config.Name, pc.restarts, pc.config.MaxRestarts)
		close(pc.done)
		return
	}

	// Exponential backoff: 1s, 2s, 4s, 8s, 16s, capped at 30s
	backoff := time.Duration(1<<uint(pc.restarts)) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	log.Printf("[plugin:%s] restarting in %s (%d/%d)", pc.config.Name, backoff, nextRestart, pc.config.MaxRestarts)

	// Wait for backoff OR shutdown — whichever comes first.
	// Previous: time.Sleep(backoff) blocked the goroutine for up to 30s during shutdown.
	select {
	case <-time.After(backoff):
		// Backoff elapsed — proceed with restart
	case <-pm.node.Kernel.bridge.GoContext().Done():
		// Bridge shutting down — don't restart, exit immediately
		close(pc.done)
		return
	}

	// Check again — stopPlugin may have been called during backoff
	pc.mu.Lock()
	stopping = pc.stopping
	pc.mu.Unlock()
	if stopping {
		close(pc.done)
		return
	}

	// Clean up old cancel context
	pc.cancel()

	if restartErr := pm.startPlugin(pc.config, nextRestart); restartErr != nil {
		InvokeErrorHandler(pm.node.Kernel.config.ErrorHandler, fmt.Errorf("plugin %s: %w", pc.config.Name, restartErr), ErrorContext{
			Operation: "RestartPlugin", Component: "plugin", Source: pc.config.Name,
		})
		close(pc.done)
	}
	// If restart succeeded, startPlugin registered a new pluginConn with a new done channel.
	// The old pc.done is never closed — that's fine, nobody is waiting on it except stopPlugin
	// which already set stopping=true before this path.
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
	// Mark as stopping — prevents auto-restart
	pc.mu.Lock()
	pc.stopping = true
	pc.mu.Unlock()

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
		// Wait for watchProcess to finish
		<-pc.done
	}

	pc.cancel()

	pm.mu.Lock()
	delete(pm.plugins, name)
	pm.mu.Unlock()

	log.Printf("[plugin:%s] stopped", name)
}

func (pm *pluginManager) listPlugins() []RunningPlugin {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	result := make([]RunningPlugin, 0, len(pm.plugins))
	for name, pc := range pm.plugins {
		pid := 0
		if pc.cmd != nil && pc.cmd.Process != nil {
			pid = pc.cmd.Process.Pid
		}
		result = append(result, RunningPlugin{
			Name:     name,
			PID:      pid,
			Uptime:   time.Since(pc.startedAt),
			Status:   "running",
			Restarts: pc.restarts,
			Config:   pc.config,
		})
	}
	return result
}

func (pm *pluginManager) nextStartOrder() int {
	pm.mu.Lock()
	pm.startCounter++
	order := pm.startCounter
	pm.mu.Unlock()
	return int(order)
}

type logWriter struct {
	prefix string
}

func (w *logWriter) Write(p []byte) (int, error) {
	log.Printf("%s%s", w.prefix, strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
