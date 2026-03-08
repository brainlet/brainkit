// Ported from: packages/provider-utils/src/strip-file-extension.test.ts
package providerutils

import "testing"

func TestStripFileExtension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"report.pdf", "report"},
		{"report", "report"},
		{"archive.tar.gz", "archive"},
		{"report.", "report"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := StripFileExtension(tt.input)
			if result != tt.expected {
				t.Errorf("StripFileExtension(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
