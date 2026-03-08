// Ported from: packages/core/src/workspace/sandbox/local-sandbox.ts
package sandbox

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/workspace/sandbox/nativesandbox"
)

// =============================================================================
// Mount Path Validation
// =============================================================================

// MarkerDir is the directory for mount marker files used to detect config changes across restarts.
var MarkerDir = filepath.Join(os.TempDir(), ".mastra-mounts")

// safeMountPathRegex is the allowlist pattern for mount paths.
var safeMountPathRegex = regexp.MustCompile(`^/[a-zA-Z0-9_.\-/]+$`)

func validateMountPath(mountPath string) error {
	if !safeMountPathRegex.MatchString(mountPath) {
		return fmt.Errorf(
			"invalid mount path: %s. Must be an absolute path with alphanumeric, dash, dot, underscore, or slash characters only",
			mountPath,
		)
	}
	segments := strings.Split(mountPath, "/")
	var filtered []string
	for _, s := range segments {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return fmt.Errorf("invalid mount path: %s. Root path \"/\" is not allowed", mountPath)
	}
	for _, seg := range filtered {
		if seg == "." || seg == ".." {
			return fmt.Errorf("invalid mount path: %s. Path segments cannot be \".\" or \"..\"", mountPath)
		}
	}
	return nil
}

// normalizeMountPath canonicalizes mount path so /data, /data/, //data all resolve to /data.
func normalizeMountPath(mountPath string) string {
	segments := strings.Split(mountPath, "/")
	var filtered []string
	for _, s := range segments {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	return "/" + strings.Join(filtered, "/")
}

// =============================================================================
// Local Sandbox Options
// =============================================================================

// LocalSandboxOptions holds configuration for a local sandbox.
type LocalSandboxOptions struct {
	// ID is a unique identifier for this sandbox instance.
	ID string
	// WorkingDirectory is the working directory for command execution.
	WorkingDirectory string
	// Env holds environment variables to set for command execution.
	Env map[string]string
	// Timeout is the default timeout for operations in ms (default: 30000).
	Timeout int64
	// Isolation is the isolation backend for sandboxed execution.
	// Default: "none"
	Isolation nativesandbox.IsolationBackend
	// NativeSandbox holds configuration for native sandboxing.
	NativeSandbox *nativesandbox.NativeSandboxConfig

	// Lifecycle hooks
	OnStart  SandboxLifecycleHook
	OnStop   SandboxLifecycleHook
	OnDestroy SandboxLifecycleHook
}

// =============================================================================
// Local Sandbox
// =============================================================================

// LocalSandbox executes commands directly on the host machine.
// This is the recommended sandbox for development and trusted local execution.
//
// Supports optional native OS sandboxing:
//   - macOS: Uses seatbelt (sandbox-exec) for filesystem and network isolation
//   - Linux: Uses bubblewrap (bwrap) for namespace isolation
type LocalSandbox struct {
	MastraSandboxBase

	id       string
	name     string
	provider string

	WorkingDir string
	Isolation  nativesandbox.IsolationBackend

	env                   map[string]string
	nativeSandboxConfig   nativesandbox.NativeSandboxConfig
	seatbeltProfile       string
	seatbeltProfilePath   string
	sandboxFolderPath     string
	userProvidedProfile   bool
	createdAt             time.Time
	activeMountPaths      map[string]bool

	executeCommandFn func(command string, args []string, opts *ExecuteCommandOptions) (*CommandResult, error)
}

// NewLocalSandbox creates a new LocalSandbox.
func NewLocalSandbox(options *LocalSandboxOptions) (*LocalSandbox, error) {
	if options == nil {
		options = &LocalSandboxOptions{}
	}

	isolation := options.Isolation
	if isolation == "" {
		isolation = nativesandbox.IsolationNone
	}

	// Validate isolation backend before construction (fail fast)
	if isolation != nativesandbox.IsolationNone && !nativesandbox.IsIsolationAvailable(isolation) {
		detection := nativesandbox.DetectIsolation()
		return nil, NewIsolationUnavailableError(string(isolation), detection.Message)
	}

	workingDir := options.WorkingDirectory
	if workingDir == "" {
		cwd, _ := os.Getwd()
		workingDir = filepath.Join(cwd, ".sandbox")
	}
	workingDir = expandTilde(workingDir)

	env := options.Env
	if env == nil {
		env = make(map[string]string)
	}

	nsc := nativesandbox.NativeSandboxConfig{}
	if options.NativeSandbox != nil {
		nsc = *options.NativeSandbox
		// Copy slices to avoid shared references
		nsc.ReadWritePaths = make([]string, len(options.NativeSandbox.ReadWritePaths))
		copy(nsc.ReadWritePaths, options.NativeSandbox.ReadWritePaths)
		nsc.ReadOnlyPaths = make([]string, len(options.NativeSandbox.ReadOnlyPaths))
		copy(nsc.ReadOnlyPaths, options.NativeSandbox.ReadOnlyPaths)
	}

	ls := &LocalSandbox{
		id:                  options.ID,
		name:                "LocalSandbox",
		provider:            "local",
		WorkingDir:          workingDir,
		Isolation:           isolation,
		env:                 env,
		nativeSandboxConfig: nsc,
		createdAt:           time.Now(),
		activeMountPaths:    make(map[string]bool),
	}

	if ls.id == "" {
		ls.id = ls.generateID()
	}

	// Create process manager
	pm := NewLocalProcessManager(env)
	pm.SetLocalSandbox(ls)

	ls.InitMastraSandboxBase(MastraSandboxOptions{
		OnStart:   options.OnStart,
		OnStop:    options.OnStop,
		OnDestroy: options.OnDestroy,
		Processes: pm.BaseProcessManager,
	})

	// Set mount implementation
	ls.MountImpl = ls.mountImpl

	// Initialize mount manager and process manager references
	ls.InitMountManager(ls)

	// Set up default executeCommand
	ls.executeCommandFn = ls.SetupExecuteCommand("LocalSandbox")

	// Set start/stop/destroy implementations
	ls.StartImpl = ls.startImpl
	ls.StopImpl = ls.stopImpl
	ls.DestroyImpl = ls.destroyImpl

	return ls, nil
}

// =============================================================================
// WorkspaceSandbox Interface
// =============================================================================

func (ls *LocalSandbox) ID() string       { return ls.id }
func (ls *LocalSandbox) Name() string     { return ls.name }
func (ls *LocalSandbox) Provider() string  { return ls.provider }
func (ls *LocalSandbox) Status() ProviderStatus { return ls.StatusValue }

func (ls *LocalSandbox) Start() error   { return ls.WrappedStart() }
func (ls *LocalSandbox) Stop() error    { return ls.WrappedStop() }
func (ls *LocalSandbox) Destroy() error { return ls.WrappedDestroy() }

func (ls *LocalSandbox) IsReady() bool {
	return ls.StatusValue == ProviderStatusRunning
}

func (ls *LocalSandbox) GetInfo() (*SandboxInfo, error) {
	totalMem := float64(0) // os-level memory info would require platform-specific calls
	cpuCores := float64(runtime.NumCPU())
	return &SandboxInfo{
		ID:        ls.id,
		Name:      ls.name,
		Provider:  ls.provider,
		Status:    ls.StatusValue,
		CreatedAt: ls.createdAt,
		Resources: &SandboxResources{
			MemoryMB: &totalMem,
			CPUCores: &cpuCores,
		},
		Metadata: map[string]interface{}{
			"workingDirectory": ls.WorkingDir,
			"platform":         runtime.GOOS,
			"goVersion":        runtime.Version(),
			"isolation":        string(ls.Isolation),
		},
	}, nil
}

func (ls *LocalSandbox) GetInstructions() string {
	return fmt.Sprintf("Local command execution. Working directory: %q.", ls.WorkingDir)
}

func (ls *LocalSandbox) ExecuteCommand(command string, args []string, options *ExecuteCommandOptions) (*CommandResult, error) {
	if ls.executeCommandFn == nil {
		return nil, fmt.Errorf("execute command not available")
	}
	return ls.executeCommandFn(command, args, options)
}

func (ls *LocalSandbox) Processes() ProcessManager {
	if ls.ProcessManager == nil {
		return nil
	}
	return ls.ProcessManager
}

func (ls *LocalSandbox) Mounts() *MountManager {
	return ls.MountMgr
}

func (ls *LocalSandbox) Mount(filesystem WorkspaceFilesystemRef, mountPath string) (*MountResult, error) {
	return ls.mountImpl(filesystem, mountPath)
}

func (ls *LocalSandbox) Unmount(mountPath string) error {
	return ls.unmountImpl(mountPath)
}

// =============================================================================
// Lifecycle Implementations
// =============================================================================

func (ls *LocalSandbox) startImpl() error {
	ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Starting sandbox: workingDirectory=%s isolation=%s", ls.WorkingDir, ls.Isolation))

	if err := os.MkdirAll(ls.WorkingDir, 0o755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	// Set up seatbelt profile for macOS sandboxing
	if ls.Isolation == nativesandbox.IsolationSeatbelt {
		userProvidedPath := ls.nativeSandboxConfig.SeatbeltProfilePath

		if userProvidedPath != "" {
			ls.seatbeltProfilePath = userProvidedPath
			ls.userProvidedProfile = true

			// Check if file exists at user's path
			content, err := os.ReadFile(userProvidedPath)
			if err != nil {
				if !os.IsNotExist(err) {
					return err
				}
				// File doesn't exist, generate default and write
				profile, genErr := nativesandbox.GenerateSeatbeltProfile(ls.WorkingDir, ls.nativeSandboxConfig)
				if genErr != nil {
					return genErr
				}
				ls.seatbeltProfile = profile
				if err := os.MkdirAll(filepath.Dir(userProvidedPath), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(userProvidedPath, []byte(profile), 0o644); err != nil {
					return err
				}
			} else {
				ls.seatbeltProfile = string(content)
			}
		} else {
			// No custom path, use default location
			profile, err := nativesandbox.GenerateSeatbeltProfile(ls.WorkingDir, ls.nativeSandboxConfig)
			if err != nil {
				return err
			}
			ls.seatbeltProfile = profile

			// Generate deterministic hash from workspace path and config
			h := sha256.New()
			h.Write([]byte(ls.WorkingDir))
			configJSON, _ := fmt.Sprintf("%+v", ls.nativeSandboxConfig), error(nil)
			h.Write([]byte(configJSON))
			configHash := fmt.Sprintf("%x", h.Sum(nil))[:8]

			cwd, _ := os.Getwd()
			ls.sandboxFolderPath = filepath.Join(cwd, ".sandbox-profiles")
			if err := os.MkdirAll(ls.sandboxFolderPath, 0o755); err != nil {
				return err
			}
			ls.seatbeltProfilePath = filepath.Join(ls.sandboxFolderPath, fmt.Sprintf("seatbelt-%s.sb", configHash))
			if err := os.WriteFile(ls.seatbeltProfilePath, []byte(profile), 0o644); err != nil {
				return err
			}
		}
	}

	ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Sandbox started: workingDirectory=%s", ls.WorkingDir))
	return nil
}

func (ls *LocalSandbox) stopImpl() error {
	ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Stopping sandbox: workingDirectory=%s", ls.WorkingDir))

	// Unmount all active mounts (best-effort)
	for mountPath := range ls.activeMountPaths {
		_ = ls.unmountImpl(mountPath)
	}
	return nil
}

func (ls *LocalSandbox) destroyImpl() error {
	ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Destroying sandbox: workingDirectory=%s", ls.WorkingDir))

	// Kill all background processes
	if ls.ProcessManager != nil {
		procs, _ := ls.ProcessManager.List()
		for _, p := range procs {
			_, _ = ls.ProcessManager.Kill(p.PID)
		}
	}

	// Unmount all active mounts
	for mountPath := range ls.activeMountPaths {
		_ = ls.unmountImpl(mountPath)
	}
	ls.activeMountPaths = make(map[string]bool)
	if ls.MountMgr != nil {
		ls.MountMgr.Clear()
	}

	// Clean up seatbelt profile (only if auto-generated)
	if ls.seatbeltProfilePath != "" && !ls.userProvidedProfile {
		_ = os.Remove(ls.seatbeltProfilePath)
	}
	ls.seatbeltProfilePath = ""
	ls.seatbeltProfile = ""
	ls.userProvidedProfile = false

	// Try to remove .sandbox-profiles folder if empty
	if ls.sandboxFolderPath != "" {
		_ = os.Remove(ls.sandboxFolderPath) // only succeeds if empty
		ls.sandboxFolderPath = ""
	}

	return nil
}

// =============================================================================
// Internal Utils
// =============================================================================

func (ls *LocalSandbox) generateID() string {
	return fmt.Sprintf("local-sandbox-%s-%s",
		fmt.Sprintf("%x", time.Now().UnixMilli()),
		randomAlpha(6),
	)
}

func randomAlpha(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// BuildEnv builds the environment object for execution.
// Always includes PATH by default (needed for finding executables).
func (ls *LocalSandbox) BuildEnv(additionalEnv map[string]string) map[string]string {
	result := map[string]string{
		"PATH": os.Getenv("PATH"),
	}
	for k, v := range ls.env {
		result[k] = v
	}
	for k, v := range additionalEnv {
		result[k] = v
	}
	return result
}

// WrapCommandForIsolation wraps a command with the configured isolation backend.
func (ls *LocalSandbox) WrapCommandForIsolation(command string) struct {
	Command string
	Args    []string
} {
	if ls.Isolation == nativesandbox.IsolationNone {
		return struct {
			Command string
			Args    []string
		}{Command: command}
	}

	wrapped, err := nativesandbox.WrapCommand(command, nativesandbox.WrapCommandOptions{
		Backend:         ls.Isolation,
		WorkspacePath:   ls.WorkingDir,
		SeatbeltProfile: ls.seatbeltProfile,
		Config:          ls.nativeSandboxConfig,
	})
	if err != nil {
		// If wrapping fails, fall back to no isolation
		ls.Logger.Error(fmt.Sprintf("[LocalSandbox] Failed to wrap command for isolation: %v", err))
		return struct {
			Command string
			Args    []string
		}{Command: command}
	}

	return struct {
		Command string
		Args    []string
	}{Command: wrapped.Command, Args: wrapped.Args}
}

// DetectIsolation detects the best available isolation backend for this platform.
func DetectIsolation() nativesandbox.SandboxDetectionResult {
	return nativesandbox.DetectIsolation()
}

// =============================================================================
// Mount Support
// =============================================================================

func (ls *LocalSandbox) mountImpl(filesystem WorkspaceFilesystemRef, mountPath string) (*MountResult, error) {
	if err := validateMountPath(mountPath); err != nil {
		return nil, err
	}
	mountPath = normalizeMountPath(mountPath)

	hostPath := ls.resolveHostPath(mountPath)
	ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Mounting %q -> %q...", mountPath, hostPath))

	// Get mount config
	config := filesystem.GetMountConfig()
	if config == nil {
		errMsg := fmt.Sprintf("Filesystem %q does not provide a mount config", filesystem.ID())
		ls.Logger.Error(fmt.Sprintf("[LocalSandbox] %s", errMsg))
		if ls.MountMgr != nil {
			ls.MountMgr.Set(mountPath, MountEntryUpdate{
				Filesystem: filesystem,
				State:      MountStateError,
				Error:      errMsg,
			})
		}
		return &MountResult{Success: false, MountPath: mountPath, Error: errMsg}, nil
	}

	// Reject unsupported types early
	if config.Type != "local" {
		errMsg := fmt.Sprintf("Unsupported mount type: %s", config.Type)
		if ls.MountMgr != nil {
			ls.MountMgr.Set(mountPath, MountEntryUpdate{
				Filesystem: filesystem,
				State:      MountStateUnsupported,
				Config:     config,
				Error:      errMsg,
			})
		}
		return &MountResult{Success: false, MountPath: mountPath, Error: errMsg}, nil
	}

	if ls.MountMgr != nil {
		ls.MountMgr.Set(mountPath, MountEntryUpdate{
			Filesystem: filesystem,
			State:      MountStateMounting,
			Config:     config,
		})
	}

	// Create symlink: ensure parent directory exists, then link
	if err := os.MkdirAll(filepath.Dir(hostPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create mount parent directory: %w", err)
	}

	// Check if host path exists and would conflict
	info, err := os.Lstat(hostPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			// Existing symlink — check if it's ours
			// For simplicity, remove and re-create
			_ = os.Remove(hostPath)
		} else if info.IsDir() {
			entries, _ := os.ReadDir(hostPath)
			if len(entries) > 0 {
				errMsg := fmt.Sprintf("Cannot mount at %s: directory exists and is not empty", hostPath)
				ls.Logger.Error(fmt.Sprintf("[LocalSandbox] %s", errMsg))
				if ls.MountMgr != nil {
					ls.MountMgr.Set(mountPath, MountEntryUpdate{
						Filesystem: filesystem,
						State:      MountStateError,
						Config:     config,
						Error:      errMsg,
					})
				}
				return &MountResult{Success: false, MountPath: mountPath, Error: errMsg}, nil
			}
			// Empty directory — remove it
			_ = os.Remove(hostPath)
		} else {
			errMsg := fmt.Sprintf("Cannot mount at %s: path is a regular file", hostPath)
			ls.Logger.Error(fmt.Sprintf("[LocalSandbox] %s", errMsg))
			if ls.MountMgr != nil {
				ls.MountMgr.Set(mountPath, MountEntryUpdate{
					Filesystem: filesystem,
					State:      MountStateError,
					Config:     config,
					Error:      errMsg,
				})
			}
			return &MountResult{Success: false, MountPath: mountPath, Error: errMsg}, nil
		}
	}

	if err := os.Symlink(config.BasePath, hostPath); err != nil {
		ls.Logger.Error(fmt.Sprintf("[LocalSandbox] Error mounting at %q: %v", hostPath, err))
		if ls.MountMgr != nil {
			ls.MountMgr.Set(mountPath, MountEntryUpdate{
				Filesystem: filesystem,
				State:      MountStateError,
				Config:     config,
				Error:      err.Error(),
			})
		}
		return &MountResult{Success: false, MountPath: mountPath, Error: err.Error()}, nil
	}

	// Mark as mounted
	if ls.MountMgr != nil {
		ls.MountMgr.Set(mountPath, MountEntryUpdate{
			Filesystem: filesystem,
			State:      MountStateMounted,
			Config:     config,
		})
	}
	ls.activeMountPaths[mountPath] = true

	// Write marker file
	ls.writeMarkerFile(mountPath, hostPath)

	// Dynamically add host path to isolation allowlist
	ls.addMountPathToIsolation(hostPath)

	ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Mounted %s -> %s", mountPath, hostPath))
	return &MountResult{Success: true, MountPath: mountPath}, nil
}

func (ls *LocalSandbox) unmountImpl(mountPath string) error {
	if err := validateMountPath(mountPath); err != nil {
		return err
	}
	mountPath = normalizeMountPath(mountPath)

	hostPath := ls.resolveHostPath(mountPath)
	ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Unmounting %s (%s)...", mountPath, hostPath))

	// Check if it's a symlink
	info, err := os.Lstat(hostPath)
	isSymlink := err == nil && info.Mode()&os.ModeSymlink != 0

	if ls.MountMgr != nil {
		ls.MountMgr.Delete(mountPath)
	}
	delete(ls.activeMountPaths, mountPath)

	// Clean up marker file
	if ls.MountMgr != nil {
		filename := ls.MountMgr.MarkerFilename(hostPath)
		markerPath := filepath.Join(MarkerDir, filename)
		_ = os.Remove(markerPath)
	}

	// Remove symlink
	if isSymlink {
		if err := os.Remove(hostPath); err != nil {
			ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Could not remove symlink %s: %v", hostPath, err))
		} else {
			ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Unmounted and removed symlink %s", hostPath))
		}
	}

	return nil
}

// =============================================================================
// Mount Helpers
// =============================================================================

func (ls *LocalSandbox) writeMarkerFile(mountPath, hostPath string) {
	if ls.MountMgr == nil {
		return
	}
	entry := ls.MountMgr.Get(mountPath)
	if entry == nil || entry.ConfigHash == "" {
		return
	}

	filename := ls.MountMgr.MarkerFilename(hostPath)
	markerContent := fmt.Sprintf("%s|%s", hostPath, entry.ConfigHash)
	markerFilePath := filepath.Join(MarkerDir, filename)

	if err := os.MkdirAll(MarkerDir, 0o755); err != nil {
		ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Warning: Could not create marker directory at %s", MarkerDir))
		return
	}
	if err := os.WriteFile(markerFilePath, []byte(markerContent), 0o644); err != nil {
		ls.Logger.Debug(fmt.Sprintf("[LocalSandbox] Warning: Could not write marker file at %s", markerFilePath))
	}
}

func (ls *LocalSandbox) addMountPathToIsolation(mountPath string) {
	if ls.Isolation == nativesandbox.IsolationNone {
		return
	}

	// Add to readWritePaths
	found := false
	for _, p := range ls.nativeSandboxConfig.ReadWritePaths {
		if p == mountPath {
			found = true
			break
		}
	}
	if !found {
		ls.nativeSandboxConfig.ReadWritePaths = append(ls.nativeSandboxConfig.ReadWritePaths, mountPath)
	}

	// Seatbelt: regenerate the inline profile
	if ls.Isolation == nativesandbox.IsolationSeatbelt {
		profile, err := nativesandbox.GenerateSeatbeltProfile(ls.WorkingDir, ls.nativeSandboxConfig)
		if err == nil {
			ls.seatbeltProfile = profile
		}
	}
	// Bwrap: reads config.ReadWritePaths each call, so no extra work needed
}

func (ls *LocalSandbox) resolveHostPath(mountPath string) string {
	return filepath.Join(ls.WorkingDir, strings.TrimLeft(mountPath, "/"))
}

// expandTilde expands ~ to the user's home directory.
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
