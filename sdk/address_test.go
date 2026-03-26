package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveServiceTopic(t *testing.T) {
	tests := []struct {
		service  string
		topic    string
		expected string
	}{
		{"my-agent.ts", "ask", "ts.my-agent.ask"},
		{"my-agent", "ask", "ts.my-agent.ask"},
		{"nested/svc.ts", "rpc", "ts.nested.svc.rpc"},
		{"simple", "ping", "ts.simple.ping"},
	}
	for _, tt := range tests {
		t.Run(tt.service+"/"+tt.topic, func(t *testing.T) {
			assert.Equal(t, tt.expected, ResolveServiceTopic(tt.service, tt.topic))
		})
	}
}
