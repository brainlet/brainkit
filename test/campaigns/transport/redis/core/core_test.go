package core_test

import (
	"testing"

	"github.com/brainlet/brainkit/test/campaigns/transport/internal/transportcampaign"
)

func TestTransport_Redis_Core(t *testing.T) {
	transportcampaign.RunBackendShard(t, "redis", transportcampaign.CoreShard)
}
