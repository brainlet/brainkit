// Package transportbackends registers all built-in configfile transport backends.
//
// Import backend-specific subpackages when a binary should link only one
// transport backend.
package transportbackends

import (
	_ "github.com/brainlet/brainkit/server/configfile/transportbackends/amqp"
	_ "github.com/brainlet/brainkit/server/configfile/transportbackends/embeddednats"
	_ "github.com/brainlet/brainkit/server/configfile/transportbackends/nats"
	_ "github.com/brainlet/brainkit/server/configfile/transportbackends/redis"
)
