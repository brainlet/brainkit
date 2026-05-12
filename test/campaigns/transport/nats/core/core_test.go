package core_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns/transport/internal/transportcampaign"
)

func TestTransport_NATS_Core(t *testing.T) {
	transportcampaign.RunBackendShard(t, "nats", transportcampaign.CoreShard)
}
