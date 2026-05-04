// Package storebackends registers all built-in configfile KitStore backends.
//
// Import backend-specific subpackages when a binary should link only one store
// backend.
package storebackends

import (
	_ "github.com/brainlet/brainkit/server/configfile/storebackends/sqlite"
)
