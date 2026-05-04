package bus

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/internal/protocoltest"
)

func publishAndWaitMessage(t *testing.T, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (protocoltest.PublishResult, sdk.Message, bool) {
	t.Helper()
	return protocoltest.PublishAndWaitMessage(t, rt, msg, timeout)
}

func publishAndWaitPayload(t *testing.T, rt sdk.Runtime, msg sdk.BrainkitMessage, timeout time.Duration) (json.RawMessage, bool) {
	t.Helper()
	return protocoltest.PublishAndWaitPayload(t, rt, msg, timeout)
}
