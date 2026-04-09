package engine

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
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
		return fmt.Errorf("%w: %s", types.ErrCommandTopic, topic)
	}
	spec, ok := r.byTopic[topic]
	if !ok {
		return nil
	}
	return spec.validate(payload)
}

func eventOf[T sdk.BrainkitMessage]() eventSpec {
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
			eventOf[sdk.KitDeployedEvent](),
			eventOf[sdk.KitTeardownedEvent](),
			eventOf[sdk.PluginRegisteredEvent](),
			eventOf[sdk.HandlerFailedEvent](),
			eventOf[sdk.HandlerExhaustedEvent](),
			eventOf[sdk.PluginStartedEvent](),
			eventOf[sdk.PluginStoppedEvent](),
			eventOf[sdk.SecretsAccessedEvent](),
			eventOf[sdk.SecretsStoredEvent](),
			eventOf[sdk.SecretsRotatedEvent](),
			eventOf[sdk.SecretsDeletedEvent](),
			eventOf[sdk.PermissionDeniedEvent](),
			eventOf[sdk.ReplyDeniedEvent](),
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
