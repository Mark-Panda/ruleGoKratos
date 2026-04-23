package data

import (
	"reflect"
	"testing"

	"ruleGoKratos/internal/biz"
)

func TestExtractHarnessAttachmentsShouldReadArrayFromMessageData(t *testing.T) {
	raw := `{"attachments":[{"filename":"demo.png","mimeType":"image/png","contentBase64":"YWJj"}]}`

	got := extractHarnessAttachments(raw, nil)
	want := []biz.HarnessAttachment{{
		Filename:      "demo.png",
		MimeType:      "image/png",
		ContentBase64: "YWJj",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestExtractHarnessAttachmentsShouldFallbackToMetadata(t *testing.T) {
	env := map[string]interface{}{
		"metadata": map[string]interface{}{
			"attachments": []interface{}{
				map[string]interface{}{
					"filename": "clip.mp4",
					"mimeType": "video/mp4",
				},
			},
		},
	}

	got := extractHarnessAttachments("", env)
	want := []biz.HarnessAttachment{{
		Filename: "clip.mp4",
		MimeType: "video/mp4",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
