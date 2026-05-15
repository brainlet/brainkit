package browser

import (
	"context"
	"os"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	browsercap "github.com/brainlet/brainkit/modulecap/browser"
	"github.com/stretchr/testify/require"
)

func TestFactoryDescriptorProvidesBrowserManager(t *testing.T) {
	desc := bkmodule.NormalizeDescriptor("browser", Factory{}.Describe())
	require.Equal(t, "browser", desc.Name)
	require.Equal(t, bkmodule.StatusWIP, desc.Status)
	require.Len(t, desc.Commands, 3)

	var foundManager, foundDebug bool
	for _, cap := range desc.Capabilities {
		switch cap.Name {
		case bkmodule.CapabilityBrowserManager:
			foundManager = true
			require.Equal(t, bkmodule.CapabilityProvided, cap.Direction)
			require.Contains(t, cap.Type, "Manager")
		case bkmodule.CapabilityLifecycleDebugRegistry:
			foundDebug = true
			require.Equal(t, bkmodule.CapabilityOptional, cap.Direction)
		}
	}
	require.True(t, foundManager)
	require.True(t, foundDebug)
}

func TestManagerRejectsInvalidScopeBeforeLaunching(t *testing.T) {
	manager := NewManager(Config{})
	_, err := manager.Launch(context.Background(), browsercap.LaunchRequest{Scope: browsercap.Scope("invalid")})
	require.Error(t, err)
	require.Contains(t, err.Error(), "scope")
	require.Equal(t, 0, manager.DebugSnapshot().Sessions)
}

func TestManagerCloseIsIdempotentForUnknownSession(t *testing.T) {
	manager := NewManager(Config{})
	require.NoError(t, manager.Close(context.Background(), "missing"))
	require.Empty(t, manager.List())
}

func TestManagerLaunchesRealBrowserWhenEnabled(t *testing.T) {
	if os.Getenv("BRAINKIT_TEST_BROWSER_LIVE") != "1" {
		t.Skip("set BRAINKIT_TEST_BROWSER_LIVE=1 to launch a real local browser")
	}
	headless := true
	manager := NewManager(Config{Headless: true, LaunchTimeout: 30 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	session, err := manager.Launch(ctx, browsercap.LaunchRequest{
		Provider: "test",
		ThreadID: "thread-1",
		Scope:    browsercap.ScopeThread,
		Headless: &headless,
	})
	require.NoError(t, err)
	require.NotEmpty(t, session.ID)
	require.NotEmpty(t, session.CDPURL)
	require.NotEmpty(t, session.WebSocketDebuggerURL)
	require.NotZero(t, session.PID)
	require.DirExists(t, session.ProfileDir)

	snapshot := manager.DebugSnapshot()
	require.Equal(t, 1, snapshot.Sessions)
	require.Equal(t, 1, snapshot.OwnedProcesses)
	require.Equal(t, 1, snapshot.OwnedProfiles)

	require.NoError(t, manager.Close(ctx, session.ID))
	require.Empty(t, manager.List())
	require.NoDirExists(t, session.ProfileDir)
	require.Equal(t, 0, manager.DebugSnapshot().Sessions)
	require.NoError(t, manager.CloseContext(ctx))
	require.True(t, manager.DebugSnapshot().Closed)
}
