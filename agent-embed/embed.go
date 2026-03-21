package agentembed

import (
	internalagent "github.com/brainlet/brainkit/internal/embed/agent"
	"github.com/brainlet/brainkit/jsbridge"
)

func LoadBundle(b *jsbridge.Bridge) error {
	return internalagent.LoadBundle(b)
}
