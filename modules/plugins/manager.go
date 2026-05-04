package plugins

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
)

// pluginManager manages plugin subprocesses for a Module.
type pluginManager struct {
	mod          *Module
	wsServer     *pluginWSServer
	plugins      map[string]*pluginConn
	mu           syncx.Mutex
	startCounter int32
	stopping     bool
}

// pluginConn tracks one connected plugin subprocess.
type pluginConn struct {
	config    PluginConfig
	identity  string
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	exited    chan struct{} // closed as soon as the subprocess Wait returns
	done      chan struct{} // closed when process exits AND no restart will happen
	startedAt time.Time

	mu       syncx.Mutex
	restarts int  // number of times restarted after crash
	stopping bool // true when stopPlugin was called — prevents auto-restart
	stopOnce sync.Once
	stopCh   chan struct{}
}

func newPluginManager(mod *Module) *pluginManager {
	return &pluginManager{
		mod:     mod,
		plugins: make(map[string]*pluginConn),
	}
}

func (pm *pluginManager) log() *slog.Logger {
	return pm.mod.kit.Logger()
}

func (pm *pluginManager) startAll(configs []PluginConfig) {
	for i := range configs {
		cfg := configs[i]
		pluginDefaults(&cfg)
		if err := pm.startPlugin(cfg, 0); err != nil {
			pm.mod.kit.ReportError(fmt.Errorf("plugin %s: %w", cfg.Name, err), types.ErrorContext{
				Operation: "StartPlugin", Component: "plugin", Source: cfg.Name,
			})
		}
	}
}

// startPlugin launches the plugin subprocess. restartCount tracks how many
// times this plugin has been restarted (0 for initial start).
func (pm *pluginManager) startPlugin(cfg PluginConfig, restartCount int) error {
	ctx, cancel := context.WithCancel(context.Background())
	var createdWSServer *pluginWSServer
	cleanupStartupFailure := func() {
		cancel()
		if createdWSServer != nil {
			pm.closeCreatedWSServer(createdWSServer)
		}
	}

	cmd := exec.CommandContext(ctx, cfg.Binary, cfg.Args...)

	// Start WS server on first plugin (lazy init)
	ws, wsURL, err := pm.ensureWSServer()
	if err != nil {
		cancel()
		return err
	}
	createdWSServer = ws

	// Pass WS URL to plugin — no transport env vars needed
	var env []string
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, fmt.Sprintf("BRAINKIT_PLUGIN_WS_URL=%s", wsURL))
	env = append(env, fmt.Sprintf("BRAINKIT_NAMESPACE=%s", pm.mod.kit.Namespace()))
	env = append(env, fmt.Sprintf("BRAINKIT_NODE_ID=%s", pm.mod.kit.CallerID()))
	if len(cfg.Config) > 0 {
		env = append(env, fmt.Sprintf("BRAINKIT_PLUGIN_CONFIG=%s", string(cfg.Config)))
	}
	// Resolve $secret: references in env
	if secretStore := pm.mod.kit.SecretStore(); secretStore != nil {
		for i, e := range env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 && strings.HasPrefix(parts[1], "$secret:") {
				secretName := strings.TrimPrefix(parts[1], "$secret:")
				val, err := secretStore.Get(context.Background(), secretName)
				if err == nil && val != "" {
					env[i] = parts[0] + "=" + val
				}
			}
		}
	}

	cmd.Env = append(cmd.Environ(), env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cleanupStartupFailure()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = &logWriter{logger: pm.log(), plugin: cfg.Name}

	if err := cmd.Start(); err != nil {
		cleanupStartupFailure()
		return fmt.Errorf("start: %w", err)
	}

	// Read READY line with timeout
	readyCh := make(chan string, 1)
	stdoutDone := make(chan struct{})
	go func() {
		defer close(stdoutDone)
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
	case <-stdoutDone:
		select {
		case identity := <-readyCh:
			pm.log().Info("plugin ready", slog.String("plugin", cfg.Name), slog.String("identity", identity))
		default:
			cleanupStartupFailure()
			if waitErr := cmd.Wait(); waitErr != nil {
				return fmt.Errorf("plugin exited before READY: %w", waitErr)
			}
			return fmt.Errorf("plugin exited before READY")
		}
	case <-time.After(cfg.StartTimeout):
		cleanupStartupFailure()
		var err error = &sdk.TimeoutError{Operation: "plugin READY"}
		if cmd.Process != nil {
			if killErr := cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
				err = errors.Join(err, fmt.Errorf("plugin %q kill after READY timeout: %w", cfg.Name, killErr))
			}
		}
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
		err = errors.Join(err, waitCommand(waitCtx, cmd))
		waitCancel()
		return err
	}

	pc := &pluginConn{
		config:    cfg,
		identity:  cfg.Name,
		cmd:       cmd,
		cancel:    cancel,
		exited:    make(chan struct{}),
		done:      make(chan struct{}),
		stopCh:    make(chan struct{}),
		restarts:  restartCount,
		startedAt: time.Now(),
	}

	pm.mu.Lock()
	if pm.stopping {
		pm.mu.Unlock()
		if signalErr := cmd.Process.Signal(syscall.SIGTERM); signalErr != nil && !errors.Is(signalErr, os.ErrProcessDone) {
			pm.log().Warn("plugin manager stopping: signal started plugin failed",
				slog.String("plugin", cfg.Name),
				slog.String("error", signalErr.Error()))
		}
		waitCtx, waitCancel := context.WithTimeout(context.Background(), pc.config.ShutdownTimeout)
		if pc.config.ShutdownTimeout <= 0 {
			waitCancel()
			waitCtx, waitCancel = context.WithTimeout(context.Background(), 5*time.Second)
		}
		if waitErr := waitCommand(waitCtx, cmd); waitErr != nil {
			if killErr := cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
				pm.log().Warn("plugin manager stopping: kill started plugin failed",
					slog.String("plugin", cfg.Name),
					slog.String("error", killErr.Error()))
			}
			killCtx, killCancel := context.WithTimeout(context.Background(), time.Second)
			_ = waitCommand(killCtx, cmd)
			killCancel()
		}
		waitCancel()
		cleanupStartupFailure()
		return errPluginManagerStopping
	}
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

func waitCommand(ctx context.Context, cmd *exec.Cmd) error {
	if ctx == nil {
		ctx = context.Background()
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

var errPluginManagerStopping = errors.New("plugin manager is stopping")

var pluginKillWaitTimeout = time.Second

func (pc *pluginConn) requestStop() {
	if pc == nil || pc.stopCh == nil {
		return
	}
	pc.stopOnce.Do(func() { close(pc.stopCh) })
}

func (pc *pluginConn) stopRequested() <-chan struct{} {
	if pc == nil {
		return nil
	}
	return pc.stopCh
}

func (pm *pluginManager) ensureWSServer() (*pluginWSServer, string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.stopping {
		return nil, "", errPluginManagerStopping
	}
	if pm.wsServer != nil {
		return nil, pm.wsServer.URL(), nil
	}
	ws, err := newPluginWSServer(pm.mod)
	if err != nil {
		return nil, "", fmt.Errorf("plugin ws server: %w", err)
	}
	pm.wsServer = ws
	return ws, ws.URL(), nil
}

func (pm *pluginManager) closeCreatedWSServer(ws *pluginWSServer) {
	if ws == nil {
		return
	}
	pm.mu.Lock()
	if pm.wsServer != ws {
		pm.mu.Unlock()
		return
	}
	pm.wsServer = nil
	pm.mu.Unlock()
	ws.Close()
}

func (pm *pluginManager) closeWSServer(ctx context.Context) error {
	if pm == nil {
		return nil
	}
	pm.mu.Lock()
	ws := pm.wsServer
	pm.mu.Unlock()
	if ws == nil {
		return nil
	}
	if err := ws.CloseContext(ctx); err != nil {
		return err
	}
	pm.mu.Lock()
	if pm.wsServer == ws {
		pm.wsServer = nil
	}
	pm.mu.Unlock()
	return nil
}

func (pm *pluginManager) isStopping() bool {
	pm.mu.Lock()
	stopping := pm.stopping
	pm.mu.Unlock()
	return stopping
}

// watchProcess waits for the plugin to exit, logs the reason, and auto-restarts
// if configured. Runs in its own goroutine.
func (pm *pluginManager) watchProcess(pc *pluginConn) {
	err := pc.cmd.Wait()
	if pc.exited != nil {
		close(pc.exited)
	}

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
	if pm.isStopping() {
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
	case <-pc.stopRequested():
		close(pc.done)
		return
	case <-pm.mod.kit.ShutdownSignal():
		// Kit shutting down — don't restart, exit immediately
		close(pc.done)
		return
	}

	// Check again — stopPlugin may have been called during backoff
	pc.mu.Lock()
	stopping = pc.stopping
	pc.mu.Unlock()
	if stopping || pm.isStopping() {
		close(pc.done)
		return
	}

	// Clean up old cancel context
	pc.cancel()

	if restartErr := pm.startPlugin(pc.config, nextRestart); restartErr != nil {
		if errors.Is(restartErr, errPluginManagerStopping) {
			close(pc.done)
			return
		}
		pm.mod.kit.ReportError(fmt.Errorf("plugin %s: %w", pc.config.Name, restartErr), types.ErrorContext{
			Operation: "RestartPlugin", Component: "plugin", Source: pc.config.Name,
		})
		close(pc.done)
		return
	}
	pc.mu.Lock()
	stopping = pc.stopping
	pc.mu.Unlock()
	if stopping || pm.isStopping() {
		pm.mu.Lock()
		next := pm.plugins[pc.config.Name]
		pm.mu.Unlock()
		if next != nil && next != pc {
			if err := pm.stopPlugin(context.Background(), pc.config.Name, next); err != nil {
				pm.mod.kit.ReportError(fmt.Errorf("plugin %s: stop restarted process: %w", pc.config.Name, err), types.ErrorContext{
					Operation: "StopRestartedPlugin", Component: "plugin", Source: pc.config.Name,
				})
			}
		}
	}
	close(pc.done)
}

func (pm *pluginManager) stopAll(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	pm.mu.Lock()
	pm.stopping = true
	plugins := make(map[string]*pluginConn, len(pm.plugins))
	for k, v := range pm.plugins {
		plugins[k] = v
	}
	pm.mu.Unlock()

	var err error
	for name, pc := range plugins {
		err = errors.Join(err, pm.stopPlugin(ctx, name, pc))
	}
	return err
}

func (pm *pluginManager) stopPlugin(ctx context.Context, name string, pc *pluginConn) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if pc == nil {
		return nil
	}
	// Mark as stopping — prevents auto-restart
	pc.mu.Lock()
	pc.stopping = true
	pc.mu.Unlock()
	pc.requestStop()

	var err error

	// Send SIGTERM
	if pc.cmd != nil && pc.cmd.Process != nil {
		if signalErr := pc.cmd.Process.Signal(syscall.SIGTERM); signalErr != nil && !errors.Is(signalErr, os.ErrProcessDone) {
			err = errors.Join(err, fmt.Errorf("plugin %q signal: %w", name, signalErr))
		}
	}
	if pc.done == nil {
		if pc.cancel != nil {
			pc.cancel()
		}
		pm.removePluginIfSame(name, pc)
		return err
	}

	timeout := pc.config.ShutdownTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-pc.done:
	case <-timer.C:
		pm.log().Warn("plugin shutdown timeout, killing", slog.String("plugin", name))
		err = errors.Join(err, fmt.Errorf("plugin %q shutdown timeout", name))
		killCtx, cancel := context.WithTimeout(context.Background(), pluginKillWaitTimeout)
		err = errors.Join(err, pm.killAndWait(killCtx, name, pc))
		cancel()
	case <-ctx.Done():
		pm.log().Warn("plugin shutdown context canceled, killing", slog.String("plugin", name), slog.String("error", ctx.Err().Error()))
		err = errors.Join(err, ctx.Err())
		killCtx, cancel := context.WithTimeout(context.Background(), pluginKillWaitTimeout)
		err = errors.Join(err, pm.killAndWait(killCtx, name, pc))
		cancel()
	}

	if pc.cancel != nil {
		pc.cancel()
	}

	if pluginConnDone(pc) {
		pm.removePluginIfSame(name, pc)
	}

	pm.log().Info("plugin stopped", slog.String("plugin", name))
	return err
}

func pluginConnDone(pc *pluginConn) bool {
	if pc == nil || pc.done == nil {
		return true
	}
	select {
	case <-pc.done:
		return true
	default:
		return false
	}
}

func (pm *pluginManager) removePluginIfSame(name string, pc *pluginConn) {
	pm.mu.Lock()
	if pm.plugins[name] == pc {
		delete(pm.plugins, name)
	}
	pm.mu.Unlock()
}

func (pm *pluginManager) killAndWait(ctx context.Context, name string, pc *pluginConn) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	if pc.cmd != nil && pc.cmd.Process != nil {
		if killErr := pc.cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			err = errors.Join(err, fmt.Errorf("plugin %q kill: %w", name, killErr))
		}
	}
	if pc.done == nil {
		return err
	}
	select {
	case <-pc.done:
	case <-ctx.Done():
		err = errors.Join(err, ctx.Err())
	}
	return err
}

func (pm *pluginManager) listPlugins() []types.RunningPlugin {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	result := make([]types.RunningPlugin, 0, len(pm.plugins))
	for name, pc := range pm.plugins {
		pid := 0
		if pc.cmd != nil && pc.cmd.Process != nil {
			pid = pc.cmd.Process.Pid
		}
		result = append(result, types.RunningPlugin{
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
