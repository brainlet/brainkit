package brainkit

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	quickjs "github.com/buke/quickjs-go"
)

// harnessSubscriber wraps a subscriber callback with an ID for unsubscription.
type harnessSubscriber struct {
	id int
	fn func(HarnessEvent)
}

// registerEventBridge registers the __go_harness_event sync bridge function.
// JS calls this for every Harness event: __go_harness_event(jsonString)
func (h *Harness) registerEventBridge() {
	qctx := h.kit.bridge.Context()
	qctx.Globals().Set("__go_harness_event",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.NewUndefined()
			}
			jsonStr := args[0].String()
			h.handleEvent(jsonStr)
			return qctx.NewUndefined()
		}))
}

// registerLockBridges registers thread lock bridge functions.
func (h *Harness) registerLockBridges() {
	qctx := h.kit.bridge.Context()
	lock := h.threadLock

	qctx.Globals().Set("__go_harness_lock_acquire",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("harness lock acquire: expected threadId"))
			}
			threadID := args[0].String()
			if err := lock.Acquire(threadID); err != nil {
				return qctx.NewString(err.Error())
			}
			return qctx.NewNull()
		}))

	qctx.Globals().Set("__go_harness_lock_release",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("harness lock release: expected threadId"))
			}
			threadID := args[0].String()
			if err := lock.Release(threadID); err != nil {
				return qctx.NewString(err.Error())
			}
			return qctx.NewNull()
		}))
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

// callJS calls a method on the JS __brainkit_harness object and returns the result.
// The method is awaited (most Harness methods are async).
func (h *Harness) callJS(method string, argsJSON string) (string, error) {
	var code string
	if argsJSON == "" {
		code = fmt.Sprintf(`return JSON.stringify(await __brainkit_harness.%s())`, method)
	} else {
		code = fmt.Sprintf(`return JSON.stringify(await __brainkit_harness.%s(JSON.parse(%s)))`, method, quoteJSString(argsJSON))
	}
	return h.kit.EvalTS(h.kit.bridge.GoContext(), "harness-call.ts", code)
}

// callJSVoid calls a JS method and discards the result.
func (h *Harness) callJSVoid(method string, argsJSON string) error {
	_, err := h.callJS(method, argsJSON)
	return err
}

// callJSDirect calls with a pre-built expression (no arg parsing).
func (h *Harness) callJSDirect(code string) (string, error) {
	return h.kit.EvalTS(h.kit.bridge.GoContext(), "harness-direct.ts", code)
}

// callJSSimple calls a method that returns a primitive (string, bool, number).
// Wraps result in JSON.stringify for safe Go parsing.
func (h *Harness) callJSSimple(method string) (string, error) {
	code := fmt.Sprintf(`return JSON.stringify(await __brainkit_harness.%s())`, method)
	return h.kit.EvalTS(h.kit.bridge.GoContext(), "harness-simple.ts", code)
}

// quoteJSString returns a JSON-encoded string (which is a valid JS string literal).
func quoteJSString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
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
