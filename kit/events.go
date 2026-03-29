package kit

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/sdk/messages"
)

type eventSpec struct {
	topic    string
	validate func(json.RawMessage) error
}

type knownEventRegistry struct {
	byTopic map[string]eventSpec
}

func (r *knownEventRegistry) Validate(topic string, payload json.RawMessage) error {
	if commandCatalog().HasCommand(topic) {
		return fmt.Errorf("%w: %s", ErrCommandTopic, topic)
	}
	spec, ok := r.byTopic[topic]
	if !ok {
		return nil
	}
	return spec.validate(payload)
}

func eventOf[T messages.BrainkitMessage]() eventSpec {
	var zero T
	return eventSpec{
		topic: zero.BusTopic(),
		validate: func(payload json.RawMessage) error {
			var decoded T
			return json.Unmarshal(payload, &decoded)
		},
	}
}

var (
	eventCatalogOnce sync.Once
	eventCatalogInst *knownEventRegistry
)

func eventCatalog() *knownEventRegistry {
	eventCatalogOnce.Do(func() {
		specs := []eventSpec{
			eventOf[messages.KitDeployedEvent](),
			eventOf[messages.KitTeardownedEvent](),
			eventOf[messages.PluginRegisteredEvent](),
			eventOf[messages.HandlerFailedEvent](),
			eventOf[messages.HandlerExhaustedEvent](),
			eventOf[messages.PluginStartedEvent](),
			eventOf[messages.PluginStoppedEvent](),
			eventOf[messages.SecretsAccessedEvent](),
			eventOf[messages.SecretsStoredEvent](),
			eventOf[messages.SecretsRotatedEvent](),
			eventOf[messages.SecretsDeletedEvent](),
			eventOf[messages.PermissionDeniedEvent](),
		}
		byTopic := make(map[string]eventSpec, len(specs))
		for _, spec := range specs {
			if _, exists := byTopic[spec.topic]; exists {
				panic(fmt.Sprintf("duplicate event topic registered: %s", spec.topic))
			}
			if commandCatalog().HasCommand(spec.topic) {
				panic(fmt.Sprintf("event topic collides with command topic: %s", spec.topic))
			}
			byTopic[spec.topic] = spec
		}
		eventCatalogInst = &knownEventRegistry{byTopic: byTopic}
	})
	return eventCatalogInst
}
