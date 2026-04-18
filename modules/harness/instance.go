package harness

import "encoding/json"

// EventType is the frozen set of event kinds every Instance subscriber
// can rely on. Internal harness code emits many more event types —
// those are explicitly NOT frozen and subject to churn.
type EventType string

const (
	EvAgentStart    EventType = "agent_start"
	EvAgentEnd      EventType = "agent_end"
	EvMessageDelta  EventType = "message_update"
	EvToolStart     EventType = "tool_start"
	EvToolEnd       EventType = "tool_end"
	EvError         EventType = "error"
)

// Event is the opaque envelope delivered to Instance.Subscribe
// callbacks. Type is one of the frozen EventType constants above;
// Payload carries the rest of the event as raw JSON so internal
// schema drift doesn't leak into consumers.
type Event struct {
	Type    EventType
	Payload json.RawMessage
}

// Instance is the frozen minimum surface every harness consumer can
// rely on across releases. Everything else on *Harness is WIP — use
// this interface from production code.
type Instance interface {
	SendMessage(content string, opts ...SendOption) error
	Abort() error
	Steer(content string, opts ...SendOption) error
	FollowUp(content string, opts ...SendOption) error
	Subscribe(fn func(Event)) func()
	CurrentThread() string
	CurrentMode() string
	Close() error
}

// instanceAdapter bridges the WIP *Harness surface to the frozen
// Instance contract. A distinct type keeps the blast radius small
// when Harness methods move around — only the adapter needs to
// track them.
type instanceAdapter Harness

func (a *instanceAdapter) harness() *Harness { return (*Harness)(a) }

func (a *instanceAdapter) SendMessage(content string, opts ...SendOption) error {
	return a.harness().SendMessage(content, opts...)
}

func (a *instanceAdapter) Abort() error { return a.harness().Abort() }

func (a *instanceAdapter) Steer(content string, opts ...SendOption) error {
	return a.harness().Steer(content, opts...)
}

func (a *instanceAdapter) FollowUp(content string, opts ...SendOption) error {
	return a.harness().FollowUp(content, opts...)
}

func (a *instanceAdapter) Subscribe(fn func(Event)) func() {
	return a.harness().Subscribe(func(e HarnessEvent) {
		fn(narrowEvent(e))
	})
}

func (a *instanceAdapter) CurrentThread() string { return a.harness().GetCurrentThreadID() }
func (a *instanceAdapter) CurrentMode() string   { return a.harness().GetCurrentModeID() }
func (a *instanceAdapter) Close() error          { return a.harness().Close() }

// narrowEvent maps the internal HarnessEvent onto the frozen Event
// shape. Events outside the frozen set pass through with Type set to
// the raw internal string — consumers can switch on known values and
// drop the rest.
func narrowEvent(e HarnessEvent) Event {
	payload, _ := json.Marshal(e)
	return Event{Type: EventType(e.Type), Payload: payload}
}
