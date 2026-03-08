// Ported from: packages/core/src/workspace/types.test.ts
package workspace

import (
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

func TestWorkspaceStatus(t *testing.T) {
	t.Run("has correct status values", func(t *testing.T) {
		cases := []struct {
			status WorkspaceStatus
			want   string
		}{
			{WorkspaceStatusPending, "pending"},
			{WorkspaceStatusInitializing, "initializing"},
			{WorkspaceStatusReady, "ready"},
			{WorkspaceStatusPaused, "paused"},
			{WorkspaceStatusError, "error"},
			{WorkspaceStatusDestroying, "destroying"},
			{WorkspaceStatusDestroyed, "destroyed"},
		}
		for _, tc := range cases {
			if string(tc.status) != tc.want {
				t.Errorf("status = %q, want %q", tc.status, tc.want)
			}
		}
	})
}

func TestInstructionsOptionStatic(t *testing.T) {
	t.Run("resolves to the static string regardless of defaults", func(t *testing.T) {
		opt := InstructionsOptionStatic("my custom instructions")
		result := opt.resolveInstructions("default instructions", nil)
		if result != "my custom instructions" {
			t.Errorf("got %q, want %q", result, "my custom instructions")
		}
	})

	t.Run("ignores request context", func(t *testing.T) {
		opt := InstructionsOptionStatic("static")
		rc := requestcontext.NewRequestContext()
		result := opt.resolveInstructions("default", rc)
		if result != "static" {
			t.Errorf("got %q, want %q", result, "static")
		}
	})
}

func TestInstructionsOptionFunc(t *testing.T) {
	t.Run("receives default instructions and request context", func(t *testing.T) {
		var receivedDefault string
		var receivedRC *requestcontext.RequestContext

		fn := InstructionsOptionFunc(func(opts InstructionsOptionFuncArgs) string {
			receivedDefault = opts.DefaultInstructions
			receivedRC = opts.RequestContext
			return opts.DefaultInstructions + " + custom"
		})

		rc := requestcontext.NewRequestContext()
		result := fn.resolveInstructions("default", rc)

		if receivedDefault != "default" {
			t.Errorf("default = %q, want %q", receivedDefault, "default")
		}
		if receivedRC != rc {
			t.Error("request context should be passed through")
		}
		if result != "default + custom" {
			t.Errorf("result = %q, want %q", result, "default + custom")
		}
	})

	t.Run("can completely replace instructions", func(t *testing.T) {
		fn := InstructionsOptionFunc(func(_ InstructionsOptionFuncArgs) string {
			return "completely replaced"
		})
		result := fn.resolveInstructions("default", nil)
		if result != "completely replaced" {
			t.Errorf("got %q, want %q", result, "completely replaced")
		}
	})
}

func TestResolveInstructions(t *testing.T) {
	t.Run("returns default when override is nil", func(t *testing.T) {
		getDefault := func() string { return "default instructions" }
		result := ResolveInstructions(nil, getDefault, nil)
		if result != "default instructions" {
			t.Errorf("got %q, want %q", result, "default instructions")
		}
	})

	t.Run("resolves static override", func(t *testing.T) {
		getDefault := func() string { return "default" }
		override := InstructionsOptionStatic("override")
		result := ResolveInstructions(override, getDefault, nil)
		if result != "override" {
			t.Errorf("got %q, want %q", result, "override")
		}
	})

	t.Run("resolves function override", func(t *testing.T) {
		getDefault := func() string { return "default" }
		override := InstructionsOptionFunc(func(opts InstructionsOptionFuncArgs) string {
			return opts.DefaultInstructions + " extended"
		})
		result := ResolveInstructions(override, getDefault, nil)
		if result != "default extended" {
			t.Errorf("got %q, want %q", result, "default extended")
		}
	})
}
