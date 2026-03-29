package infra_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type streamMsg struct {
	Type  string          `json:"type"`
	Event string          `json:"event,omitempty"`
	Data  json.RawMessage `json:"data"`
}

func deployAndCollect(t *testing.T, k *testutil.TestKernel, source, code, topic string, expectCount int) []streamMsg {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(k, ctx, messages.KitDeployMsg{Source: source, Code: code})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	collected := make(chan streamMsg, 20)
	sendPR, _ := sdk.SendToService(k, ctx, source, topic, map[string]bool{"go": true})
	replyUnsub, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var sm streamMsg
		json.Unmarshal(msg.Payload, &sm)
		collected <- sm
	})
	defer replyUnsub()

	var msgs []streamMsg
	for i := 0; i < expectCount; i++ {
		select {
		case m := <-collected:
			msgs = append(msgs, m)
		case <-ctx.Done():
			t.Fatalf("timeout waiting for stream message %d/%d", i+1, expectCount)
		}
	}
	return msgs
}

func TestStream_TextChunks(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	msgs := deployAndCollect(t, k, "streamer.ts", `bus.on("stream", async (msg) => {
		msg.stream.text("hello ");
		msg.stream.text("world");
		msg.stream.end({ total: 2 });
	});`, "stream", 3)

	require.Len(t, msgs, 3)
	assert.Equal(t, "text", msgs[0].Type)
	assert.Equal(t, `"hello "`, string(msgs[0].Data))
	assert.Equal(t, "text", msgs[1].Type)
	assert.Equal(t, `"world"`, string(msgs[1].Data))
	assert.Equal(t, "end", msgs[2].Type)
}

func TestStream_Progress(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	msgs := deployAndCollect(t, k, "progress.ts", `bus.on("work", async (msg) => {
		msg.stream.progress(0.0, "starting");
		msg.stream.progress(0.5, "halfway");
		msg.stream.progress(1.0, "done");
		msg.stream.end({ items: 42 });
	});`, "work", 4)

	assert.Equal(t, "progress", msgs[0].Type)
	assert.Equal(t, "progress", msgs[1].Type)
	assert.Equal(t, "progress", msgs[2].Type)
	assert.Equal(t, "end", msgs[3].Type)

	var p struct{ Value float64; Message string }
	json.Unmarshal(msgs[1].Data, &p)
	assert.Equal(t, 0.5, p.Value)
	assert.Equal(t, "halfway", p.Message)
}

func TestStream_ErrorMidStream(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	msgs := deployAndCollect(t, k, "errstream.ts", `bus.on("fail", async (msg) => {
		msg.stream.text("partial");
		msg.stream.error("something broke");
	});`, "fail", 2)

	assert.Equal(t, "text", msgs[0].Type)
	assert.Equal(t, "error", msgs[1].Type)

	var errData struct{ Message string }
	json.Unmarshal(msgs[1].Data, &errData)
	assert.Equal(t, "something broke", errData.Message)
}

func TestStream_EventSequence(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	msgs := deployAndCollect(t, k, "events.ts", `bus.on("run", async (msg) => {
		msg.stream.event("tool_start", { name: "search" });
		msg.stream.event("tool_end", { name: "search", found: 3 });
		msg.stream.end({ ok: true });
	});`, "run", 3)

	assert.Equal(t, "event", msgs[0].Type)
	assert.Equal(t, "tool_start", msgs[0].Event)
	assert.Equal(t, "event", msgs[1].Type)
	assert.Equal(t, "tool_end", msgs[1].Event)
	assert.Equal(t, "end", msgs[2].Type)
}

func TestStream_RawSendStillWorks(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "rawsend.ts",
		Code:   `bus.on("raw", async (msg) => { msg.send({ chunk: "hello" }); msg.reply({ done: true }); });`,
	})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()
	time.Sleep(100 * time.Millisecond)

	collected := make(chan json.RawMessage, 10)
	sendPR, _ := sdk.SendToService(k, ctx, "rawsend.ts", "raw", map[string]bool{"go": true})
	replyUnsub, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		collected <- json.RawMessage(msg.Payload)
	})
	defer replyUnsub()

	var msgs []json.RawMessage
	for i := 0; i < 2; i++ {
		select {
		case m := <-collected:
			msgs = append(msgs, m)
		case <-ctx.Done():
			t.Fatalf("timeout at %d", i)
		}
	}

	assert.Contains(t, string(msgs[0]), "chunk")
	assert.NotContains(t, string(msgs[0]), `"type"`)
	assert.Contains(t, string(msgs[1]), "done")
}
