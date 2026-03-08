// Ported from: packages/core/src/agent/message-list/prompt/attachments-to-parts.ts
package prompt

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// Attachment is a stub for AI SDK V5 Attachment type.
// TODO: In TS this comes from @ai-sdk/ui-utils-v5 Attachment.
type Attachment struct {
	URL         string `json:"url"`
	ContentType string `json:"contentType,omitempty"`
}

// ContentPart represents a TextPart, ImagePart, or FilePart for AIV4.
// We reuse MastraMessagePart since it covers all these cases.
type ContentPart = state.MastraMessagePart

// AttachmentsToParts converts a list of attachments to a list of content parts
// for consumption by ai/core functions. Currently supports images and text attachments.
func AttachmentsToParts(attachments []Attachment) ([]ContentPart, error) {
	var parts []ContentPart

	for _, attachment := range attachments {
		categorized := CategorizeFileData(attachment.URL, attachment.ContentType)

		// If it's raw data (base64), convert it to a data URI
		urlString := attachment.URL
		if categorized.Type == "raw" {
			ct := attachment.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			urlString = CreateDataUri(attachment.URL, ct)
		}

		parsed, err := url.Parse(urlString)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %s", attachment.URL)
		}

		scheme := strings.ToLower(parsed.Scheme)

		switch scheme {
		case "http", "https", "gs", "s3":
			if strings.HasPrefix(attachment.ContentType, "image/") {
				parts = append(parts, ContentPart{
					Type:     "image",
					Data:     parsed.String(),
					MimeType: attachment.ContentType,
				})
			} else {
				if attachment.ContentType == "" {
					return nil, fmt.Errorf("if the attachment is not an image, it must specify a content type")
				}
				parts = append(parts, ContentPart{
					Type:     "file",
					Data:     parsed.String(),
					MimeType: attachment.ContentType,
				})
			}

		case "data":
			if strings.HasPrefix(attachment.ContentType, "image/") {
				parts = append(parts, ContentPart{
					Type:     "image",
					Data:     urlString,
					MimeType: attachment.ContentType,
				})
			} else if strings.HasPrefix(attachment.ContentType, "text/") {
				parts = append(parts, ContentPart{
					Type:     "file",
					Data:     urlString,
					MimeType: attachment.ContentType,
				})
			} else {
				if attachment.ContentType == "" {
					return nil, fmt.Errorf("if the attachment is not an image or text, it must specify a content type")
				}
				parts = append(parts, ContentPart{
					Type:     "file",
					Data:     urlString,
					MimeType: attachment.ContentType,
				})
			}

		default:
			return nil, fmt.Errorf("unsupported URL protocol: %s", scheme)
		}
	}

	return parts, nil
}
