//go:build !darwin

package local

import (
	"context"
	"errors"
)

// probeDevice is a no-op on non-darwin hosts today. The
// CheckResult carries -1 / empty-string + a warning so the
// caller knows the probe didn't run rather than getting a
// false "everything looks fine" reading.
func probeDevice(_ context.Context) (string, int, bool, error) {
	return "", -1, false, errors.New("device probe not implemented for this platform")
}
