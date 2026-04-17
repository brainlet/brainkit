package transport

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
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
						slog.Error("command failed with no replyTo", slog.String("topic", binding.Topic), slog.String("error", err.Error()))
					}
					return nil
				}

				// All command replies go out as wire envelopes. Success
				// path wraps the handler's raw JSON as ok=true data;
				// error path serializes the BrainkitError into ok=false.
				if err != nil && IsDecodeFailure(err) {
					return err
				}
				var envelope sdk.Envelope
				if err != nil {
					envelope = sdkErrorToEnvelope(err)
				} else {
					envelope = sdk.Envelope{Ok: true, Data: json.RawMessage(payload)}
					if len(envelope.Data) == 0 {
						envelope.Data = json.RawMessage("null")
					}
				}
				responsePayload, _ := sdk.EncodeEnvelope(envelope)

				result := message.NewMessage(watermill.NewUUID(), responsePayload)
				correlationID := wmsg.Metadata.Get("correlationId")
				if correlationID != "" {
					result.Metadata.Set("correlationId", correlationID)
				}
				// Command replies are always terminal — mark done=true so
				// the shared-inbox Caller finalizes immediately instead of
				// treating the payload as a stream chunk. envelope=true
				// signals the Caller to unwrap payload via sdk.FromEnvelope.
				result.Metadata.Set("done", "true")
				result.Metadata.Set("envelope", "true")

				// replyTo is already namespaced+sanitized by the publisher
				return h.pub.Publish(replyTo, result)
			},
		)
	}
}

// SerializeBrainkitError converts an error to a wire envelope. Typed
// brainkit errors keep their Code/Details; plain errors collapse to
// INTERNAL_ERROR. Error messages are sanitized to strip absolute paths
// and credential patterns before leaving the process.
func SerializeBrainkitError(err error) []byte {
	envelope := sdkErrorToEnvelope(err)
	payload, _ := sdk.EncodeEnvelope(envelope)
	return payload
}

// sdkErrorToEnvelope builds the ok=false envelope for err.
func sdkErrorToEnvelope(err error) sdk.Envelope {
	msg := SanitizeErrorMessage(err.Error())
	var bk sdkerrors.BrainkitError
	if errors.As(err, &bk) {
		return sdk.EnvelopeErr(bk.Code(), msg, bk.Details())
	}
	return sdk.EnvelopeErr("INTERNAL_ERROR", msg, nil)
}

// absolutePathRe matches absolute filesystem paths (/foo/bar/baz or C:\foo\bar).
var absolutePathRe = regexp.MustCompile(`(?:/[a-zA-Z0-9_.~-]+){3,}`)

// connectionStringRe matches connection strings with credentials (postgres://user:pass@host, amqp://...).
var connectionStringRe = regexp.MustCompile(`\w+://[^\s]+:[^\s]+@[^\s]+`)

// SanitizeErrorMessage removes sensitive information from error messages
// before they're returned to callers via bus or HTTP.
// Strips: absolute filesystem paths, connection strings with credentials.
func SanitizeErrorMessage(msg string) string {
	// Replace absolute paths (3+ segments) with just the last component
	msg = absolutePathRe.ReplaceAllStringFunc(msg, func(path string) string {
		parts := strings.Split(path, "/")
		return "<path>/" + parts[len(parts)-1]
	})
	// Redact connection strings (postgres://user:pass@host → postgres://****@****)
	msg = connectionStringRe.ReplaceAllStringFunc(msg, func(cs string) string {
		idx := strings.Index(cs, "://")
		if idx < 0 {
			return "****"
		}
		return cs[:idx] + "://****"
	})
	return msg
}

func rawHandlerName(name, topic string) string {
	if strings.TrimSpace(name) != "" {
		return sanitizeDurable(name)
	}
	return "command_" + sanitizeDurable(topic)
}
