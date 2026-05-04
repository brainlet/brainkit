package module

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeDescriptorDerivesCapabilityGroups(t *testing.T) {
	desc := NormalizeDescriptor("manifest-test", Descriptor{
		Capabilities: []CapabilityDescriptor{
			OptionalCapabilityOf[string]("test.optional"),
			ProvidedCapabilityOf[int]("test.provided"),
			RequiredCapabilityOf[bool]("test.required"),
			RequiredCapabilityOf[bool]("test.required"),
		},
	})

	require.Len(t, desc.Capabilities, 3)
	require.NotNil(t, desc.CapabilityGroups)
	require.Equal(t, []CapabilityDescriptor{
		RequiredCapabilityOf[bool]("test.required"),
	}, desc.CapabilityGroups.Required)
	require.Equal(t, []CapabilityDescriptor{
		OptionalCapabilityOf[string]("test.optional"),
	}, desc.CapabilityGroups.Optional)
	require.Equal(t, []CapabilityDescriptor{
		ProvidedCapabilityOf[int]("test.provided"),
	}, desc.CapabilityGroups.Provided)
}

func TestNormalizeDescriptorClearsStaleCapabilityGroups(t *testing.T) {
	desc := NormalizeDescriptor("manifest-test", Descriptor{
		CapabilityGroups: &CapabilityGroups{
			Required: []CapabilityDescriptor{RequiredCapabilityOf[string]("stale")},
		},
	})

	require.Nil(t, desc.CapabilityGroups)
}
