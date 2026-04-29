package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/engine"
	toolreg "github.com/brainlet/brainkit/internal/tools"
	xport "github.com/brainlet/brainkit/internal/transport"
	transportbackends "github.com/brainlet/brainkit/internal/transport/backends"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/require"
)

// TestPluginToolCallViaBusEmbedded verifies that tools.call for a plugin-style
// tool works on Embedded transport with the pass-through replyTo protocol.
// Uses internal/engine directly because it tests low-level plugin protocol
// that requires direct transport and tool registry access.
func TestPluginToolCallViaBusEmbedded(t *testing.T) {
	transport, err := transportbackends.NewTransportSet(xport.TransportConfig{
		Type: "embedded",
	})
	require.NoError(t, err)
	defer transport.Close()

	kernel, err := engine.NewKernel(types.KernelConfig{
		Namespace: "test-plugin-bus",
		Transport: transport,
	})
	require.NoError(t, err)
	defer kernel.Close()

	mountToolCallCommand(t, kernel)

	// Simulate plugin side: subscribe to tool topic, respond.
	fakeTopic := "fake.plugin.tool.echo"
	fakeResultTopic := fakeTopic + ".result"

	_, err = kernel.SubscribeRaw(context.Background(), fakeTopic, func(msg sdk.Message) {
		correlationID := msg.Metadata["correlationId"]
		result := json.RawMessage(`{"echoed":"ok"}`)
		resp, _ := json.Marshal(toolmsg.ToolCallResp{Result: result})

		if replyTo := msg.Metadata["replyTo"]; replyTo != "" {
			_ = kernel.ReplyRaw(context.Background(), replyTo, correlationID, resp, true)
			return
		}
		ctx := xport.ContextWithCorrelationID(context.Background(), correlationID)
		kernel.PublishRaw(ctx, fakeResultTopic, resp)
	})
	require.NoError(t, err)

	kernel.Tools.Register(toolreg.RegisteredTool{
		Name:      "test/echo@0.1.0/echo",
		ShortName: "echo",
		Executor: &toolreg.GoFuncExecutor{
			Fn: func(callCtx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				callerReplyTo := xport.ReplyToFromContext(callCtx)
				if callerReplyTo != "" {
					if _, err := kernel.Remote().PublishRawWithMeta(callCtx, fakeTopic, input, map[string]string{
						"replyTo": callerReplyTo,
					}); err != nil {
						return nil, err
					}
					return nil, nil
				}

				correlationID := fmt.Sprintf("%d", time.Now().UnixNano())
				waitCtx, cancel := context.WithCancel(callCtx)
				defer cancel()

				resultCh := make(chan sdk.Message, 1)
				stop, subErr := kernel.SubscribeRaw(waitCtx, fakeResultTopic, func(msg sdk.Message) {
					if msg.Metadata["correlationId"] == correlationID {
						resultCh <- msg
						cancel()
					}
				})
				if subErr != nil {
					return nil, subErr
				}
				defer stop()

				ctx := xport.ContextWithCorrelationID(callCtx, correlationID)
				if _, pubErr := kernel.PublishRaw(ctx, fakeTopic, input); pubErr != nil {
					return nil, pubErr
				}

				select {
				case <-callCtx.Done():
					return nil, callCtx.Err()
				case msg := <-resultCh:
					var resp toolmsg.ToolCallResp
					json.Unmarshal(msg.Payload, &resp)
					return resp.Result, nil
				}
			},
		},
	})

	t.Run("direct_executor", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tool, err := kernel.Tools.Resolve("echo")
		require.NoError(t, err)
		result, err := tool.Executor.Call(ctx, "test", []byte(`"hello"`))
		require.NoError(t, err)
		t.Logf("direct: %s", string(result))
	})

	t.Run("via_bus_command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		replyTo := fmt.Sprintf("tools.call.reply.%d", time.Now().UnixNano())
		replyCh := make(chan sdk.Message, 1)
		unsub, err := kernel.SubscribeRaw(ctx, replyTo, func(msg sdk.Message) {
			replyCh <- msg
		})
		require.NoError(t, err)
		defer unsub()

		start := time.Now()
		_, err = sdk.Publish(kernel, ctx, toolmsg.ToolCallMsg{
			Name:  "echo",
			Input: "test input",
		}, sdk.WithReplyTo(replyTo))
		require.NoError(t, err)

		select {
		case msg := <-replyCh:
			elapsed := time.Since(start)
			data := suite.ResponseDataFromMsg(msg)
			t.Logf("bus response in %s: %s", elapsed.Round(time.Millisecond), string(data))
			require.Less(t, elapsed, 3*time.Second)
			require.Empty(t, suite.ResponseErrorMessage(msg.Payload))
			var resp toolmsg.ToolCallResp
			require.NoError(t, json.Unmarshal(data, &resp))
			require.True(t, len(resp.Result) > 0 && string(resp.Result) != "null")
		case <-ctx.Done():
			t.Fatal("REGRESSION: tools.call via bus for plugin-style tool times out on Embedded transport")
		}
	})
}

func mountToolCallCommand(t *testing.T, kernel *engine.Kernel) {
	t.Helper()
	var zero toolmsg.ToolCallMsg
	topic := zero.BusTopic()
	_, err := kernel.MountCommand(context.Background(), bkmodule.CommandSpec{
		Name:  topic,
		Topic: topic,
		Handle: func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
			var req toolmsg.ToolCallMsg
			if len(payload) > 0 {
				if err := json.Unmarshal(payload, &req); err != nil {
					return nil, err
				}
			}
			resp, err := kernel.CallTool(ctx, bkmodule.ToolCallRequest{Name: req.Name, Input: req.Input})
			if err != nil {
				return nil, err
			}
			if resp == nil {
				return nil, nil
			}
			return json.Marshal(toolmsg.ToolCallResp{Result: resp.Result})
		},
	})
	require.NoError(t, err)
}
