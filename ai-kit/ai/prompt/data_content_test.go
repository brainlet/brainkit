// Ported from: packages/ai/src/prompt/data-content.test.ts
package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertDataContentToBase64String(t *testing.T) {
	t.Run("should convert byte slice to base64", func(t *testing.T) {
		data := []byte("Hello, world!")
		result := ConvertDataContentToBase64String(data)
		assert.Equal(t, "SGVsbG8sIHdvcmxkIQ==", result)
	})

	t.Run("should return string as-is", func(t *testing.T) {
		result := ConvertDataContentToBase64String("SGVsbG8sIHdvcmxkIQ==")
		assert.Equal(t, "SGVsbG8sIHdvcmxkIQ==", result)
	})
}

func TestConvertDataContentToUint8Array(t *testing.T) {
	t.Run("should return byte slice as-is", func(t *testing.T) {
		data := []byte("Hello, world!")
		result, err := ConvertDataContentToUint8Array(data)
		assert.NoError(t, err)
		assert.Equal(t, data, result)
	})

	t.Run("should decode base64 string", func(t *testing.T) {
		result, err := ConvertDataContentToUint8Array("SGVsbG8sIHdvcmxkIQ==")
		assert.NoError(t, err)
		assert.Equal(t, []byte("Hello, world!"), result)
	})

	t.Run("should return error for invalid base64 string", func(t *testing.T) {
		_, err := ConvertDataContentToUint8Array("not-valid-base64!!!")
		assert.Error(t, err)
		assert.True(t, IsInvalidDataContentError(err))
	})

	t.Run("should return error for unsupported type", func(t *testing.T) {
		_, err := ConvertDataContentToUint8Array(42)
		assert.Error(t, err)
		assert.True(t, IsInvalidDataContentError(err))
	})
}

func TestSplitDataURL(t *testing.T) {
	t.Run("should split a data URL", func(t *testing.T) {
		result := SplitDataURL("data:image/png;base64,iVBOR///")
		assert.NotNil(t, result.MediaType)
		assert.Equal(t, "image/png", *result.MediaType)
		assert.NotNil(t, result.Base64Content)
		assert.Equal(t, "iVBOR///", *result.Base64Content)
	})

	t.Run("should return nil for invalid data URL", func(t *testing.T) {
		result := SplitDataURL("not-a-data-url")
		assert.Nil(t, result.MediaType)
		assert.Nil(t, result.Base64Content)
	})
}
