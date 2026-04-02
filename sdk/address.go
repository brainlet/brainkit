package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/sdk/messages"
)

// SendToService publishes to a deployed .ts service's mailbox topic.
// Resolves the naming convention: "my-agent.ts" + "ask" → "ts.my-agent.ask"
// Returns PublishResult with replyTo for request/response patterns.
func SendToService(rt Runtime, ctx context.Context, service, topic string, payload any, opts ...PublishOption) (PublishResult, error) {
	resolved := ResolveServiceTopic(service, topic)
	data, err := json.Marshal(payload)
	if err != nil {
		return PublishResult{}, fmt.Errorf("sdk: marshal payload: %w", err)
	}
	return Publish(rt, ctx, messages.CustomMsg{Topic: resolved, Payload: data}, opts...)
}

// ResolveServiceTopic converts a service name + local topic to the bus topic.
// Convention: ts.<name-without-ext>.<topic>
//
//	"my-agent.ts" + "ask" → "ts.my-agent.ask"
//	"my-agent"    + "ask" → "ts.my-agent.ask"
//	"nested/svc"  + "rpc" → "ts.nested.svc.rpc"
func ResolveServiceTopic(service, topic string) string {
	name := strings.TrimSuffix(service, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}
