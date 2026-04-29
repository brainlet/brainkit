package engine

import (
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modules/plugins/pluginmsg"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/systemmsg"
)

type eventSpec struct {
	topic    string
	validate func(json.RawMessage) error
}

type knownEventRegistry struct {
	byTopic  map[string]eventSpec
	commands *commandRegistry // for "is this a command topic?" check
}

func (r *knownEventRegistry) Validate(topic string, payload json.RawMessage) error {
	if r.commands.HasCommand(topic) {
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

func buildEventCatalog(catalog *commandRegistry) *knownEventRegistry {
	specs := []eventSpec{
		eventOf[systemmsg.KitDeployedEvent](),
		eventOf[systemmsg.KitTeardownedEvent](),
		eventOf[pluginmsg.PluginRegisteredEvent](),
		eventOf[systemmsg.HandlerFailedEvent](),
		eventOf[systemmsg.HandlerExhaustedEvent](),
		eventOf[pluginmsg.PluginStartedEvent](),
		eventOf[pluginmsg.PluginStoppedEvent](),
		eventOf[secretmsg.SecretsAccessedEvent](),
		eventOf[secretmsg.SecretsStoredEvent](),
		eventOf[secretmsg.SecretsRotatedEvent](),
		eventOf[secretmsg.SecretsDeletedEvent](),
	}
	byTopic := make(map[string]eventSpec, len(specs))
	for _, spec := range specs {
		if _, exists := byTopic[spec.topic]; exists {
			panic(fmt.Sprintf("duplicate event topic registered: %s", spec.topic))
		}
		if catalog.HasCommand(spec.topic) {
			panic(fmt.Sprintf("event topic collides with command topic: %s", spec.topic))
		}
		byTopic[spec.topic] = spec
	}
	return &knownEventRegistry{byTopic: byTopic, commands: catalog}
}
