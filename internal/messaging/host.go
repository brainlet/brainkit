package messaging

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RawCommandBinding binds a logical command topic to a raw JSON handler.
// Response is published to the replyTo topic from inbound message metadata.
type RawCommandBinding struct {
	Name   string
	Topic  string
	Handle func(context.Context, json.RawMessage) (json.RawMessage, error)
}

// Host binds raw command handlers onto a router and transport.
type Host struct {
	namespace      string
	router         *message.Router
	sub            message.Subscriber
	pub            message.Publisher
	topicSanitizer func(string) string
}

func NewHost(namespace string, router *message.Router, sub message.Subscriber, pub message.Publisher) *Host {
	return &Host{
		namespace: namespace,
		router:    router,
		sub:       sub,
		pub:       pub,
	}
}

// NewHostWithTransport creates a Host that uses the transport's topic sanitizer.
func NewHostWithTransport(namespace string, router *message.Router, transport *Transport) *Host {
	return &Host{
		namespace:      namespace,
		router:         router,
		sub:            transport.Subscriber,
		pub:            transport.Publisher,
		topicSanitizer: transport.TopicSanitizer,
	}
}

func (h *Host) resolvedTopic(logicalTopic string) string {
	topic := NamespacedTopic(h.namespace, logicalTopic)
	if h.topicSanitizer != nil {
		topic = h.topicSanitizer(topic)
	}
	return topic
}

// RegisterCommands installs all command bindings onto the router.
// Handlers read replyTo from inbound message metadata and publish responses there.
func (h *Host) RegisterCommands(bindings []RawCommandBinding) {
	for _, binding := range bindings {
		binding := binding
		commandTopic := h.resolvedTopic(binding.Topic)
		handlerName := rawHandlerName(binding.Name, binding.Topic)

		h.router.AddConsumerHandler(
			handlerName,
			commandTopic,
			h.sub,
			func(wmsg *message.Message) error {
				cmdCtx := withInboundMetadata(wmsg.Context(), wmsg, binding.Topic)
				payload, err := binding.Handle(cmdCtx, json.RawMessage(wmsg.Payload))

				replyTo := wmsg.Metadata.Get("replyTo")
				if replyTo == "" {
					if err != nil {
						log.Printf("[host] command %s failed with no replyTo: %v", binding.Topic, err)
					}
					return nil
				}

				// Build response payload — on error, wrap in structured error response
				var responsePayload []byte
				if err != nil {
					if IsDecodeFailure(err) {
						return err
					}
					responsePayload = SerializeBrainkitError(err)
				} else if payload != nil {
					responsePayload = payload
				} else {
					return nil
				}

				result := message.NewMessage(watermill.NewUUID(), responsePayload)
				correlationID := wmsg.Metadata.Get("correlationId")
				if correlationID != "" {
					result.Metadata.Set("correlationId", correlationID)
				}

				// replyTo is already namespaced+sanitized by the publisher
				return h.pub.Publish(replyTo, result)
			},
		)
	}
}

// SerializeBrainkitError converts an error to a JSON response with code and details.
// If the error implements BrainkitError (Code() + Details()), those are included.
// Otherwise, falls back to INTERNAL_ERROR with the error message.
func SerializeBrainkitError(err error) []byte {
	type brainkitError interface {
		Code() string
		Details() map[string]any
	}
	if bk, ok := err.(brainkitError); ok {
		payload, _ := json.Marshal(map[string]any{
			"error":   err.Error(),
			"code":    bk.Code(),
			"details": bk.Details(),
		})
		return payload
	}
	payload, _ := json.Marshal(map[string]any{
		"error": err.Error(),
		"code":  "INTERNAL_ERROR",
	})
	return payload
}

func rawHandlerName(name, topic string) string {
	if strings.TrimSpace(name) != "" {
		return sanitizeDurable(name)
	}
	return "command_" + sanitizeDurable(topic)
}
