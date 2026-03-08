// Ported from: packages/core/src/stream/aisdk/v5/file.ts
package v5

import (
	"encoding/base64"
)

// ---------------------------------------------------------------------------
// GeneratedFile interface
// ---------------------------------------------------------------------------

// GeneratedFile represents a generated file with lazy conversion between
// base64 and byte representations.
type GeneratedFile interface {
	// Base64 returns the file as a base64 encoded string.
	Base64() string
	// Bytes returns the file as a byte slice (Uint8Array equivalent).
	Bytes() []byte
	// MediaType returns the IANA media type of the file.
	MediaType() string
}

// ---------------------------------------------------------------------------
// DefaultGeneratedFile
// ---------------------------------------------------------------------------

// DefaultGeneratedFile implements GeneratedFile with lazy conversion between
// base64 and byte representations. Mirrors the TS DefaultGeneratedFile class.
type DefaultGeneratedFile struct {
	base64Data *string
	bytesData  []byte
	mediaType  string
}

// DefaultGeneratedFileOptions are the constructor parameters for DefaultGeneratedFile.
type DefaultGeneratedFileOptions struct {
	// Data is either a base64 string or raw bytes.
	Data any // string | []byte
	// MediaType is the IANA media type.
	MediaType string
}

// NewDefaultGeneratedFile creates a new DefaultGeneratedFile.
func NewDefaultGeneratedFile(opts DefaultGeneratedFileOptions) *DefaultGeneratedFile {
	f := &DefaultGeneratedFile{
		mediaType: opts.MediaType,
	}
	switch data := opts.Data.(type) {
	case []byte:
		f.bytesData = data
	case string:
		f.base64Data = &data
	}
	return f
}

// Base64 returns the file as a base64 encoded string.
// Lazy conversion with caching to avoid unnecessary conversion overhead.
func (f *DefaultGeneratedFile) Base64() string {
	if f.base64Data == nil {
		encoded := base64.StdEncoding.EncodeToString(f.bytesData)
		f.base64Data = &encoded
	}
	return *f.base64Data
}

// Bytes returns the file as a byte slice.
// Lazy conversion with caching to avoid unnecessary conversion overhead.
func (f *DefaultGeneratedFile) Bytes() []byte {
	if f.bytesData == nil && f.base64Data != nil {
		decoded, err := base64.StdEncoding.DecodeString(*f.base64Data)
		if err == nil {
			f.bytesData = decoded
		}
	}
	return f.bytesData
}

// MediaType returns the IANA media type of the file.
func (f *DefaultGeneratedFile) MediaType() string {
	return f.mediaType
}

// ---------------------------------------------------------------------------
// DefaultGeneratedFileWithType
// ---------------------------------------------------------------------------

// DefaultGeneratedFileWithType extends DefaultGeneratedFile with a Type field
// set to "file". Mirrors the TS DefaultGeneratedFileWithType class.
type DefaultGeneratedFileWithType struct {
	*DefaultGeneratedFile
	// Type is always "file".
	Type string
}

// NewDefaultGeneratedFileWithType creates a new DefaultGeneratedFileWithType.
func NewDefaultGeneratedFileWithType(opts DefaultGeneratedFileOptions) *DefaultGeneratedFileWithType {
	return &DefaultGeneratedFileWithType{
		DefaultGeneratedFile: NewDefaultGeneratedFile(opts),
		Type:                 "file",
	}
}
