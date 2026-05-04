// Package automation registers workflow and scheduling YAML module names.
//
// Schedules use the standard SQLite-backed factory when modules.schedules.path
// is set. Workflow registration itself stays light but needs jsruntime mounted
// at runtime to execute deployed workflows.
package automation

import (
	_ "github.com/brainlet/brainkit/modules/schedules/standard"
	_ "github.com/brainlet/brainkit/modules/workflow"
)
