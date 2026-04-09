package plugin

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/brainlet/brainkit/sdk/pluginws"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// toolDispatcher runs tool handlers concurrently with bounded concurrency.
// This prevents the WS read loop from blocking when a tool handler needs
// to make bus round-trips (publish + wait for event response over WS).
type toolDispatcher struct {
	sem chan struct{} // semaphore for max concurrent tool calls
	wg  sync.WaitGroup
}

// newToolDispatcher creates a dispatcher with the given max concurrency.
// maxConcurrency <= 0 defaults to 10.
func newToolDispatcher(maxConcurrency int) *toolDispatcher {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	return &toolDispatcher{
		sem: make(chan struct{}, maxConcurrency),
	}
}

// dispatch runs a tool handler in a bounded goroutine.
func (d *toolDispatcher) dispatch(
	ctx context.Context,
	conn *websocket.Conn,
	msgID string,
	rt *wsClient,
	call pluginws.ToolCall,
	handler func(context.Context, Client, json.RawMessage) (json.RawMessage, error),
) {
	d.wg.Add(1)
	d.sem <- struct{}{} // acquire slot

	go func() {
		defer d.wg.Done()
		defer func() { <-d.sem }() // release slot

		result, err := handler(ctx, rt, call.Input)

		var errStr string
		if err != nil {
			errStr = err.Error()
		}

		resultData, _ := json.Marshal(pluginws.ToolResult{
			Result: result,
			Error:  errStr,
		})

		rt.mu.Lock()
		defer rt.mu.Unlock()
		wsjson.Write(ctx, conn, pluginws.Message{
			Type: pluginws.TypeToolResult,
			ID:   msgID,
			Data: resultData,
		})
	}()
}

// wait blocks until all dispatched tool handlers complete.
func (d *toolDispatcher) wait() {
	d.wg.Wait()
}
