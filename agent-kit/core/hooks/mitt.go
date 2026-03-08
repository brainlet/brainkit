// Ported from: packages/core/src/hooks/mitt.ts

package hooks

import (
	"reflect"
	"sync"
)

// Handler is an event handler that receives an event payload.
// Corresponds to TS: type Handler<T> = (event: T) => void
type Handler func(event any)

// WildcardHandler is a handler for wildcard '*' events that receives both the
// event type and the event payload.
// Corresponds to TS: type WildcardHandler<T> = (type: keyof T, event: T[keyof T]) => void
type WildcardHandler func(eventType string, event any)

// Emitter is a tiny functional event emitter / pubsub.
// Corresponds to TS: interface Emitter<Events>
type Emitter struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	wildcard []WildcardHandler
}

// New creates a new Emitter instance.
// Corresponds to TS: function mitt(all?: EventHandlerMap): Emitter
func New() *Emitter {
	return &Emitter{
		handlers: make(map[string][]Handler),
	}
}

// All returns a copy of the handler map (regular handlers only, not wildcard).
// Corresponds to TS: all property (Map)
func (e *Emitter) All() map[string][]Handler {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string][]Handler, len(e.handlers))
	for k, v := range e.handlers {
		cp := make([]Handler, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}

// On registers an event handler for the given type.
// Corresponds to TS: on<Key>(type: Key, handler: GenericEventHandler)
func (e *Emitter) On(eventType string, handler Handler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.handlers[eventType] = append(e.handlers[eventType], handler)
}

// OnWildcard registers a wildcard handler that receives all events.
// Corresponds to TS: on(type: '*', handler: WildcardHandler<Events>)
func (e *Emitter) OnWildcard(handler WildcardHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.wildcard = append(e.wildcard, handler)
}

// Off removes an event handler for the given type.
// If no handler is provided, all handlers for that type are removed.
// If a handler is provided but not found, existing handlers are preserved.
// Corresponds to TS: off<Key>(type: Key, handler?: GenericEventHandler)
//
// Note on the TS splice behavior:
//
//	handlers.splice(handlers.indexOf(handler) >>> 0, 1)
//
// If indexOf returns -1, >>> 0 converts it to 4294967295, so splice at that
// huge index is a no-op. We replicate this: if handler not found, do nothing.
func (e *Emitter) Off(eventType string, handler ...Handler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	handlers, ok := e.handlers[eventType]
	if !ok {
		return
	}

	if len(handler) == 0 {
		// No handler provided: remove all handlers for this event type.
		// TS: all!.set(type, [])
		e.handlers[eventType] = []Handler{}
		return
	}

	// Find the handler by reference (pointer comparison) and remove it.
	// TS: handlers.splice(handlers.indexOf(handler) >>> 0, 1)
	// Go functions are not comparable with ==, so we use reflect.ValueOf().Pointer()
	// to compare function identity, matching TS reference equality (===).
	target := reflect.ValueOf(handler[0]).Pointer()
	for i, existing := range handlers {
		if reflect.ValueOf(existing).Pointer() == target {
			e.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			return
		}
	}
	// Handler not found: no-op (matches TS >>> 0 behavior).
}

// OffWildcard removes a wildcard handler. If no handler is provided, all
// wildcard handlers are removed.
// Corresponds to TS: off(type: '*', handler: WildcardHandler<Events>)
func (e *Emitter) OffWildcard(handler ...WildcardHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(handler) == 0 {
		e.wildcard = []WildcardHandler{}
		return
	}

	target := reflect.ValueOf(handler[0]).Pointer()
	for i, existing := range e.wildcard {
		if reflect.ValueOf(existing).Pointer() == target {
			e.wildcard = append(e.wildcard[:i], e.wildcard[i+1:]...)
			return
		}
	}
}

// Emit invokes all handlers for the given type.
// If present, '*' handlers are invoked after type-matched handlers.
// Corresponds to TS: emit<Key>(type: Key, evt?: Events[Key])
func (e *Emitter) Emit(eventType string, event any) {
	e.mu.RLock()

	// Copy the handler slice before iterating (safe iteration during modification).
	// TS: (handlers as EventHandlerList).slice().map(handler => handler(evt))
	var regularCopy []Handler
	if handlers, ok := e.handlers[eventType]; ok && len(handlers) > 0 {
		regularCopy = make([]Handler, len(handlers))
		copy(regularCopy, handlers)
	}

	// Wildcard handlers receive both the event type and the event.
	// TS: (handlers as WildCardEventHandlerList).slice().map(handler => handler(type, evt))
	var wildcardCopy []WildcardHandler
	if len(e.wildcard) > 0 {
		wildcardCopy = make([]WildcardHandler, len(e.wildcard))
		copy(wildcardCopy, e.wildcard)
	}

	e.mu.RUnlock()

	for _, h := range regularCopy {
		h(event)
	}
	for _, h := range wildcardCopy {
		h(eventType, event)
	}
}
