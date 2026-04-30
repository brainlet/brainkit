package engine

import (
	"context"

	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// RestartActiveWorkflows is the public Go API for manually triggering workflow recovery.
func (k *Kernel) RestartActiveWorkflows(ctx context.Context) error {
	if k.jsRuntime == nil {
		return &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	if k.runtimeHost != nil {
		k.runtimeHost.RestartActiveWorkflows(k.config)
	}
	return nil
}
