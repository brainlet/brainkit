// Ported from: packages/provider-utils/src/strip-file-extension.ts
package providerutils

import "strings"

// StripFileExtension strips file extension segments from a filename.
//
// Examples:
//   - "report.pdf" -> "report"
//   - "archive.tar.gz" -> "archive"
//   - "filename" -> "filename"
func StripFileExtension(filename string) string {
	idx := strings.Index(filename, ".")
	if idx == -1 {
		return filename
	}
	return filename[:idx]
}
