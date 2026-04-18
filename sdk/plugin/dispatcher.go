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
	sem      chan struct{} // semaphore for max concurrent tool calls
	wg       sync.WaitGroup
	cancelMu sync.Mutex
	cancels  map[string]context.CancelFunc // toolCallID → cancel
}

// newToolDispatcher creates a dispatcher with the given max concurrency.
// maxConcurrency <= 0 defaults to 10.
func newToolDispatcher(maxConcurrency int) *toolDispatcher {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	return &toolDispatcher{
		sem:     make(chan struct{}, maxConcurrency),
		cancels: make(map[string]context.CancelFunc),
	}
}

// cancel invokes and removes the CancelFunc registered for toolCallID.
// No-op when the call has already finished or was never dispatched —
// matches the at-least-once nature of the WS cancel frame.
func (d *toolDispatcher) cancel(toolCallID string) {
	d.cancelMu.Lock()
	cancel := d.cancels[toolCallID]
	delete(d.cancels, toolCallID)
	d.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// dispatch runs a tool handler in a bounded goroutine. A per-call ctx
// is stored in the cancel map so host-side TypeCancel frames can abort
// it via dispatcher.cancel.
func (d *toolDispatcher) dispatch(
	parentCtx context.Context,
	conn *websocket.Conn,
	msgID string,
	rt *wsClient,
	call pluginws.ToolCall,
	handler func(context.Context, Client, json.RawMessage) (json.RawMessage, error),
) {
	d.wg.Add(1)
	d.sem <- struct{}{} // acquire slot

	callCtx, callCancel := context.WithCancel(parentCtx)
	d.cancelMu.Lock()
	d.cancels[msgID] = callCancel
	d.cancelMu.Unlock()

	go func() {
		defer d.wg.Done()
		defer func() { <-d.sem }() // release slot
		defer func() {
			d.cancelMu.Lock()
			delete(d.cancels, msgID)
			d.cancelMu.Unlock()
			callCancel()
		}()

		result, err := handler(callCtx, rt, call.Input)

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
		wsjson.Write(parentCtx, conn, pluginws.Message{
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
