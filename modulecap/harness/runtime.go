// Package harnesscap defines module-facing harness runtime capability
// contracts.
package harnesscap

import "context"

// Runtime is the minimal JS runtime bridge surface required by
// modules/harness. It intentionally lives outside modules/harness so
// modules/jsruntime can provide a typed capability without importing the
// harness module implementation.
type Runtime interface {
	EvalJS(ctx context.Context, filename, code string) (string, error)
	RuntimeContext() context.Context
	RegisterEventBridge(func(string)) error
	RegisterLockBridge(acquire func(string) error, release func(string) error) error
	EvalControl(ctx context.Context, filename, code string) (string, error)
}
