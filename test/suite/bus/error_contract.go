package bus

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBusErrorResponseCarriesCode(t *testing.T, env *suite.TestEnv) {
	payload, ok := publishAndWaitPayload(t, env.Kit, toolmsg.ToolCallMsg{Name: "nonexistent-tool"}, 5*time.Second)
	require.True(t, ok)
	assert.True(t, suite.ResponseHasError(payload))
	assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload))
	assert.NotEmpty(t, suite.ResponseErrorMessage(payload))
	if d := suite.ResponseErrorDetails(payload); d != nil {
		assert.Equal(t, "nonexistent-tool", d["name"])
	}
}
