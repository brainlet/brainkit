package tools_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/require"
)

type echoInput struct {
	Message string `json:"message"`
}

func TestGoToolModuleMountsScopedTool(t *testing.T) {
	const toolName = "test.go.echo"
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Modules: []bkmodule.Module{
			toolsmod.New(),
			toolsmod.GoTool(toolName, toolsmod.TypedTool[echoInput]{
				Description: "echoes from a typed Go tool module",
				Execute: func(_ context.Context, input echoInput) (any, error) {
					return map[string]string{"echoed": input.Message}, nil
				},
			}),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	resp, err := toolmsg.CallToolCall(k, context.Background(), toolmsg.ToolCallMsg{
		Name:  toolName,
		Input: map[string]any{"message": "hello"},
	}, sdk.WithCallTimeout(2*time.Second))
	require.NoError(t, err)

	var decoded map[string]string
	require.NoError(t, json.Unmarshal(resp.Result, &decoded))
	require.Equal(t, "hello", decoded["echoed"])

	var toolDesc bkmodule.Descriptor
	for _, desc := range k.MountedModules() {
		if desc.Name == "go-tool:"+toolName {
			toolDesc = desc
			break
		}
	}
	require.Equal(t, "go-tool:"+toolName, toolDesc.Name)
	require.Len(t, toolDesc.Resources, 1)
	require.Equal(t, bkmodule.ResourceKindTool, toolDesc.Resources[0].Kind)
	require.Equal(t, toolName, toolDesc.Resources[0].Name)

	require.NoError(t, k.Unmount(context.Background(), "go-tool:"+toolName))
	_, err = toolmsg.CallToolCall(k, context.Background(), toolmsg.ToolCallMsg{
		Name:  toolName,
		Input: map[string]any{"message": "after"},
	}, sdk.WithCallTimeout(2*time.Second))
	require.Error(t, err)
}
