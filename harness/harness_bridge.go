package harness

import (
	"encoding/json"
	"log"
	"sync"

	iharness "github.com/brainlet/brainkit/internal/harness"
)

// harnessSubscriber wraps a subscriber callback with an ID for unsubscription.
type harnessSubscriber struct {
	id int
	fn func(HarnessEvent)
}

// initRuntime creates the internal harness Runtime and registers all bridges.
func (h *Harness) initRuntime() {
	h.rt = iharness.New(h.bridge, h.evalTS)

	h.rt.RegisterEventBridge(h.handleEvent)
	h.rt.RegisterLockBridges(iharness.LockFuncs{
		Acquire: h.threadLock.Acquire,
		Release: h.threadLock.Release,
	})
}

// handleEvent processes a JSON event string from JS.
func (h *Harness) handleEvent(jsonStr string) {
	var event HarnessEvent
	if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
		log.Printf("harness: failed to parse event: %v (json: %.100s)", err, jsonStr)
		return
	}
	event.Raw = json.RawMessage(jsonStr)

	// Update display state cache
	h.dsMu.Lock()
	updateDisplayState(h.displayState, event)
	h.dsMu.Unlock()

	// Update token usage cache
	if event.Type == EventUsageUpdate && event.Usage != nil {
		h.tuMu.Lock()
		h.tokenUsage.PromptTokens += event.Usage.PromptTokens
		h.tokenUsage.CompletionTokens += event.Usage.CompletionTokens
		h.tokenUsage.TotalTokens += event.Usage.TotalTokens
		h.tuMu.Unlock()
	}

	// Dispatch to subscribers (copy slice for safe iteration)
	h.subMu.RLock()
	subs := make([]harnessSubscriber, len(h.subscribers))
	copy(subs, h.subscribers)
	h.subMu.RUnlock()

	for _, sub := range subs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("harness: subscriber panic: %v", r)
				}
			}()
			sub.fn(event)
		}()
	}
}

// callJS delegates to the internal runtime.
func (h *Harness) callJS(method string, argsJSON string) (string, error) {
	return h.rt.CallJS(method, argsJSON)
}

// callJSVoid delegates to the internal runtime.
func (h *Harness) callJSVoid(method string, argsJSON string) error {
	return h.rt.CallJSVoid(method, argsJSON)
}

// callJSSimple delegates to the internal runtime.
func (h *Harness) callJSSimple(method string) (string, error) {
	return h.rt.CallJSSimple(method)
}

// callJSDirect delegates to the internal runtime.
func (h *Harness) callJSDirect(code string) (string, error) {
	return h.rt.EvalDirect("harness-direct.ts", code)
}

// quoteJSString delegates to the internal runtime utility.
func quoteJSString(s string) string {
	return iharness.QuoteJSString(s)
}

// Subscribe registers a listener for all Harness events.
// Returns an unsubscribe function.
func (h *Harness) Subscribe(fn func(HarnessEvent)) func() {
	h.subMu.Lock()
	id := h.nextSubID
	h.nextSubID++
	h.subscribers = append(h.subscribers, harnessSubscriber{id: id, fn: fn})
	h.subMu.Unlock()

	return func() {
		h.subMu.Lock()
		defer h.subMu.Unlock()
		for i, s := range h.subscribers {
			if s.id == id {
				h.subscribers = append(h.subscribers[:i], h.subscribers[i+1:]...)
				break
			}
		}
	}
}

// subscriberMu helpers to avoid exposing sync primitives
var _ = sync.RWMutex{}
