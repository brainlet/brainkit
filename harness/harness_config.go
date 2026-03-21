package harness

import (
	"fmt"
	"sync"
)

// ---------------------------------------------------------------------------
// HarnessConfig
// ---------------------------------------------------------------------------

// HarnessConfig configures a Harness instance.
type HarnessConfig struct {
	// ID identifies this Harness instance.
	ID string

	// ResourceID scopes threads to a specific resource.
	ResourceID string

	// Modes — at least one required. One must have Default=true.
	Modes []ModeConfig

	// StateSchema as JSON Schema object (validated by Zod in JS). Optional.
	// Use StateSchemaOf[T]() to generate from a Go struct:
	//
	//   type MyState struct {
	//       ProjectName string `json:"projectName" default:""`
	//       Yolo        bool   `json:"yolo" default:"true"`
	//   }
	//   StateSchema: brainkit.StateSchemaOf[MyState](),
	StateSchema map[string]any

	// InitialState values for the state schema. Optional.
	// Accepts a struct (matching StateSchema) or map[string]any.
	InitialState any

	// Subagents — constrained subagent definitions. Optional.
	Subagents []HarnessSubagentConfig

	// Tools — extra tool names to include. Optional.
	Tools []string

	// Workspace — static workspace config. Optional.
	Workspace *WorkspaceHarnessConfig

	// OMConfig — observational memory settings. Optional.
	OMConfig *HarnessOMConfig

	// HeartbeatHandlers — Go-side periodic tasks. Optional.
	HeartbeatHandlers []HeartbeatHandler

	// ThreadLock — Go-level thread locking. Optional.
	// If nil, a default in-process mutex-based lock is used.
	ThreadLock *ThreadLock

	// Permissions — per-category permission policies. Optional.
	// Use DefaultPermissions() for the standard set (read=allow, rest=ask).
	Permissions map[ToolCategory]PermissionPolicy

	// ToolCategories — static map of tool name → category. Optional.
	// Used for permission resolution: tool → category → policy.
	//
	//   ToolCategories: map[string]brainkit.ToolCategory{
	//       "view":            brainkit.CategoryRead,
	//       "write_file":      brainkit.CategoryEdit,
	//       "execute_command":  brainkit.CategoryExecute,
	//   }
	ToolCategories map[string]ToolCategory

	// AlwaysAllowTools — additional tools that never need approval. Optional.
	// Built-in always-allowed: ask_user, task_write, task_check, submit_plan.
	AlwaysAllowTools []string

	// ModelAuthChecker checks if a provider's API key is available. Optional.
	ModelAuthChecker func(provider string) bool

	// CustomModels — additional models beyond the provider registry. Optional.
	CustomModels []AvailableModel
}

// ModeConfig defines a Harness mode.
type ModeConfig struct {
	ID             string // unique mode ID (e.g., "build", "plan", "fast")
	Name           string // display name
	Default        bool   // true for the default mode
	DefaultModelID string // default model ID for this mode
	Color          string // color hint for TUI rendering
	AgentName      string // name of the agent in globalThis.__agents
}

// HarnessSubagentConfig defines a constrained subagent.
type HarnessSubagentConfig struct {
	ID             string   // subagent type ID (e.g., "explore", "execute")
	AllowedTools   []string // tool names this subagent can use
	DefaultModelID string   // default model ID
	Instructions   string   // system instructions
}

// HarnessOMConfig configures observational memory for the Harness.
type HarnessOMConfig struct {
	DefaultObserverModel  string
	DefaultReflectorModel string
	ObservationThreshold  int // messages between observations (default: 5)
	ReflectionThreshold   int // observations between reflections (default: 3)
}

// WorkspaceHarnessConfig configures a static workspace.
type WorkspaceHarnessConfig struct {
	RootDir string // workspace root directory
}

// HeartbeatHandler defines a Go-side periodic task.
type HeartbeatHandler struct {
	ID         string       // unique handler ID
	IntervalMs int          // interval in milliseconds
	Handler    func() error // the periodic function
	Immediate  bool         // run immediately on start
	Shutdown   func() error // cleanup on stop (optional)
}

// ThreadLock provides thread-level locking.
type ThreadLock struct {
	Acquire func(threadID string) error
	Release func(threadID string) error
}

// validateHarnessConfig checks the config for required fields.
func validateHarnessConfig(cfg HarnessConfig) error {
	if cfg.ID == "" {
		return fmt.Errorf("harness: ID is required")
	}
	if len(cfg.Modes) == 0 {
		return fmt.Errorf("harness: at least one mode is required")
	}
	hasDefault := false
	for _, m := range cfg.Modes {
		if m.ID == "" {
			return fmt.Errorf("harness: mode ID is required")
		}
		if m.AgentName == "" {
			return fmt.Errorf("harness: mode %q: AgentName is required", m.ID)
		}
		if m.Default {
			if hasDefault {
				return fmt.Errorf("harness: multiple default modes")
			}
			hasDefault = true
		}
	}
	if !hasDefault {
		return fmt.Errorf("harness: one mode must have Default=true")
	}
	for _, s := range cfg.Subagents {
		if s.ID == "" {
			return fmt.Errorf("harness: subagent ID is required")
		}
		if len(s.AllowedTools) == 0 {
			return fmt.Errorf("harness: subagent %q: AllowedTools is required", s.ID)
		}
	}
	return nil
}

// defaultThreadLock creates a simple in-process mutex-based thread lock.
func defaultThreadLock() *ThreadLock {
	var mu sync.Mutex
	locks := make(map[string]bool)
	return &ThreadLock{
		Acquire: func(threadID string) error {
			mu.Lock()
			defer mu.Unlock()
			if locks[threadID] {
				return fmt.Errorf("thread %s already locked", threadID)
			}
			locks[threadID] = true
			return nil
		},
		Release: func(threadID string) error {
			mu.Lock()
			defer mu.Unlock()
			delete(locks, threadID)
			return nil
		},
	}
}
