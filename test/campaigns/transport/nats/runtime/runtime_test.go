package runtime_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns/transport/internal/transportcampaign"
)

func TestTransport_NATS_Runtime(t *testing.T) {
	transportcampaign.RunBackendShard(t, "nats", transportcampaign.RuntimeShard)
}
