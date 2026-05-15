package browser_test

import (
	"context"
	"testing"
	"time"

	brainkit "github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/browser"
	"github.com/brainlet/brainkit/modules/browser/browsermsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/require"
)

func TestModuleMountProvidesBrowserCommandsAndUnmountCleansCapability(t *testing.T) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Modules: []bkmodule.Module{
			browser.NewModule(browser.Config{}),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := sdk.Call[browsermsg.BrowserSessionListMsg, browsermsg.BrowserSessionListResp](k, ctx, browsermsg.BrowserSessionListMsg{})
	require.NoError(t, err)
	require.Empty(t, resp.Sessions)

	var desc bkmodule.Descriptor
	for _, mounted := range k.MountedModules() {
		if mounted.Name == "browser" {
			desc = mounted
			break
		}
	}
	require.Equal(t, "browser", desc.Name)
	require.NotEmpty(t, desc.CapabilityGroups)
	require.Len(t, desc.Commands, 3)

	require.NoError(t, k.Unmount(ctx, "browser"))
	_, err = sdk.Call[browsermsg.BrowserSessionListMsg, browsermsg.BrowserSessionListResp](k, ctx, browsermsg.BrowserSessionListMsg{}, sdk.WithCallTimeout(200*time.Millisecond))
	require.Error(t, err)
}
