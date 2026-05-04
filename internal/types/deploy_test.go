package types

import "testing"

func TestDeployConfigEffectiveArtifactKindDefaultsToSource(t *testing.T) {
	var cfg DeployConfig
	if cfg.EffectiveArtifactKind() != DeployArtifactSource {
		t.Fatalf("default deploy artifact kind = %q, want source", cfg.EffectiveArtifactKind())
	}

	WithNormalizedJS()(&cfg)
	if cfg.EffectiveArtifactKind() != DeployArtifactNormalizedJS {
		t.Fatalf("normalized deploy artifact kind = %q, want normalized_js", cfg.EffectiveArtifactKind())
	}
}

func TestPersistedDeploymentEffectiveArtifactKind(t *testing.T) {
	if got := (PersistedDeployment{}).EffectiveArtifactKind(); got != DeployArtifactSource {
		t.Fatalf("empty persisted artifact kind = %q, want source", got)
	}

	legacyPackage := PersistedDeployment{PackageName: "pkg"}
	if got := legacyPackage.EffectiveArtifactKind(); got != DeployArtifactNormalizedJS {
		t.Fatalf("legacy package artifact kind = %q, want normalized_js", got)
	}

	explicitSourcePackage := PersistedDeployment{PackageName: "pkg", ArtifactKind: DeployArtifactSource}
	if got := explicitSourcePackage.EffectiveArtifactKind(); got != DeployArtifactSource {
		t.Fatalf("explicit artifact kind = %q, want source", got)
	}
}
