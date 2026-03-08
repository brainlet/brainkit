// Ported from: packages/provider-utils/src/convert-to-form-data.ts
package providerutils

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
)

// FormDataEntry represents a single entry in form data.
type FormDataEntry struct {
	// Key is the field name.
	Key string
	// Value is the field value (string or []byte).
	Value interface{}
	// Filename is the optional filename for file uploads.
	Filename string
}

// ConvertToFormDataOptions are the options for ConvertToFormData.
type ConvertToFormDataOptions struct {
	// UseArrayBrackets controls whether to add [] suffix for multi-element arrays.
	// Defaults to true.
	UseArrayBrackets *bool
}

// ConvertToFormDataResult contains the multipart form data content and content type.
type ConvertToFormDataResult struct {
	// Body is the multipart form data body.
	Body io.Reader
	// ContentType is the Content-Type header including the boundary.
	ContentType string
	// Values is a map of the original values for error reporting.
	Values map[string]interface{}
}

// ConvertToFormData converts a map of values to multipart form data.
// Handles: nil values (skipped), arrays (single element: no brackets, multi: brackets by default),
// and all other values are appended directly.
func ConvertToFormData(input map[string]interface{}, opts *ConvertToFormDataOptions) (*ConvertToFormDataResult, error) {
	useArrayBrackets := true
	if opts != nil && opts.UseArrayBrackets != nil {
		useArrayBrackets = *opts.UseArrayBrackets
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, value := range input {
		if value == nil {
			continue
		}

		if arr, ok := value.([]interface{}); ok {
			if len(arr) == 1 {
				if err := writeFormField(writer, key, arr[0]); err != nil {
					return nil, err
				}
				continue
			}

			arrayKey := key
			if useArrayBrackets {
				arrayKey = key + "[]"
			}
			for _, item := range arr {
				if err := writeFormField(writer, arrayKey, item); err != nil {
					return nil, err
				}
			}
			continue
		}

		if err := writeFormField(writer, key, value); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &ConvertToFormDataResult{
		Body:        &buf,
		ContentType: writer.FormDataContentType(),
		Values:      input,
	}, nil
}

func writeFormField(writer *multipart.Writer, key string, value interface{}) error {
	switch v := value.(type) {
	case string:
		return writer.WriteField(key, v)
	case []byte:
		part, err := writer.CreateFormFile(key, key)
		if err != nil {
			return err
		}
		_, err = part.Write(v)
		return err
	default:
		return writer.WriteField(key, fmt.Sprintf("%v", v))
	}
}
