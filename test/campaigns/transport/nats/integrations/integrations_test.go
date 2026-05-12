package integrations_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns/transport/internal/transportcampaign"
)

func TestTransport_NATS_Integrations(t *testing.T) {
	transportcampaign.RunBackendShard(t, "nats", transportcampaign.IntegrationsShard)
}
