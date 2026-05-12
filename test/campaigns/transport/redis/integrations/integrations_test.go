package integrations_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns/transport/internal/transportcampaign"
)

func TestTransport_Redis_Integrations(t *testing.T) {
	transportcampaign.RunBackendShard(t, "redis", transportcampaign.IntegrationsShard)
}
