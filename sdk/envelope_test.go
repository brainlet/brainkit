package sdk

import (
	"errors"
	"testing"

	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

func TestEnvelopeRoundTripPackageResolverError(t *testing.T) {
	err := &sdkerrors.PackageResolverError{
		Specifier:          "uuid",
		Importer:           "index.ts",
		Source:             "fixture",
		Profile:            "source-relative",
		AllowedBareImports: []string{"kit", "ai", "agent", "compiler"},
		SuggestedOwner:     "future opt-in npm ecosystem resolver profile",
	}

	roundTrip := FromEnvelope(ToEnvelope(nil, err))

	var resolverErr *sdkerrors.PackageResolverError
	if !errors.As(roundTrip, &resolverErr) {
		t.Fatalf("round trip error = %T, want PackageResolverError", roundTrip)
	}
	if resolverErr.Specifier != "uuid" ||
		resolverErr.Importer != "index.ts" ||
		resolverErr.Source != "fixture" ||
		resolverErr.Profile != "source-relative" ||
		resolverErr.SuggestedOwner != "future opt-in npm ecosystem resolver profile" {
		t.Fatalf("round trip resolver error = %#v", resolverErr)
	}
	if got, want := resolverErr.AllowedBareImports, []string{"kit", "ai", "agent", "compiler"}; len(got) != len(want) {
		t.Fatalf("allowed imports length = %d, want %d", len(got), len(want))
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("allowed import %d = %q, want %q", i, got[i], want[i])
			}
		}
	}
}
