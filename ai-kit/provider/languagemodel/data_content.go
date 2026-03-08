// Ported from: packages/provider/src/language-model/v3/language-model-v3-data-content.ts
package languagemodel

// DataContent represents data content. Can be a byte slice (Uint8Array),
// base64 encoded data as a string, or a URL string.
//
// In the TS version this is: Uint8Array | string | URL.
// In Go we use a sealed interface.
type DataContent interface {
	dataContent()
}

// DataContentBytes represents binary data (Uint8Array in TS).
type DataContentBytes struct {
	Data []byte
}

func (DataContentBytes) dataContent() {}

// DataContentString represents a base64 encoded string or URL.
type DataContentString struct {
	Value string
}

func (DataContentString) dataContent() {}
