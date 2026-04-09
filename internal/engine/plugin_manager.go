package engine

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"github.com/brainlet/brainkit/internal/syncx"
	"syscall"
	"time"

	"github.com/brainlet/brainkit/sdk"
)


// pluginManager manages plugin subprocesses for a Node.
type pluginManager struct {
	node         *Node
	wsServer     *pluginWSServer
	plugins      map[string]*pluginConn
	mu           syncx.Mutex
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

	mu       syncx.Mutex
	restarts int  // number of times restarted after crash
	stopping bool // true when stopPlugin was called — prevents auto-restart
}

func newPluginManager(node *Node) *pluginManager {
	return &pluginManager{
		node:    node,
		plugins: make(map[string]*pluginConn),
	}
}

func (pm *pluginManager) log() *slog.Logger {
	return pm.node.Kernel.logger
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

	// Start WS server on first plugin (lazy init)
	if pm.wsServer == nil {
		ws, err := newPluginWSServer(pm.node)
		if err != nil {
			cancel()
			return fmt.Errorf("plugin ws server: %w", err)
		}
		pm.wsServer = ws
	}

	// Pass WS URL to plugin — no transport env vars needed
	var env []string
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, fmt.Sprintf("BRAINKIT_PLUGIN_WS_URL=%s", pm.wsServer.URL()))
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
	cmd.Stderr = &logWriter{logger: pm.log(), plugin: cfg.Name}

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
		pm.log().Info("plugin ready", slog.String("plugin", cfg.Name), slog.String("identity", identity))
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
		pm.log().Info("plugin restarted", slog.String("plugin", cfg.Name), slog.Int("pid", cmd.Process.Pid), slog.Int("restart", restartCount))
	} else {
		pm.log().Info("plugin started", slog.String("plugin", cfg.Name), slog.Int("pid", cmd.Process.Pid))
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
		pm.log().Error("plugin crashed", slog.String("plugin", pc.config.Name), slog.String("signal", exitSignal), slog.Int("exit_code", exitCode))
	} else if err != nil {
		pm.log().Error("plugin crashed", slog.String("plugin", pc.config.Name), slog.String("error", err.Error()), slog.Int("exit_code", exitCode))
	} else {
		pm.log().Info("plugin exited", slog.String("plugin", pc.config.Name), slog.Int("exit_code", exitCode))
	}

	// Auto-restart if configured
	if !pc.config.AutoRestart {
		pm.log().Info("plugin auto-restart disabled", slog.String("plugin", pc.config.Name))
		close(pc.done)
		return
	}

	nextRestart := pc.restarts + 1
	if nextRestart > pc.config.MaxRestarts {
		pm.log().Error("plugin max restarts reached", slog.String("plugin", pc.config.Name), slog.Int("restarts", pc.restarts), slog.Int("max", pc.config.MaxRestarts))
		close(pc.done)
		return
	}

	// Exponential backoff: 1s, 2s, 4s, 8s, 16s, capped at 30s
	backoff := time.Duration(1<<uint(pc.restarts)) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	pm.log().Info("plugin restarting", slog.String("plugin", pc.config.Name), slog.Duration("backoff", backoff), slog.Int("retry", nextRestart), slog.Int("max", pc.config.MaxRestarts))

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
		pm.log().Warn("plugin shutdown timeout, killing", slog.String("plugin", name))
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

	pm.log().Info("plugin stopped", slog.String("plugin", name))
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
	logger *slog.Logger
	plugin string
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.logger.Info(strings.TrimRight(string(p), "\n"), slog.String("plugin", w.plugin), slog.String("stream", "stderr"))
	return len(p), nil
}

func pluginDefaults(cfg *PluginConfig) {
	if cfg.MaxRestarts == 0 {
		cfg.MaxRestarts = 5
	}
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 10 * time.Second
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 5 * time.Second
	}
}
