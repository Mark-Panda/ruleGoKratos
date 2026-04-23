package biz

import (
	"encoding/base64"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestResolveAttachmentMIME_octetStreamPNG(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	b64 := base64.StdEncoding.EncodeToString(png)
	got := resolveAttachmentMIME("x.bin", "application/octet-stream", b64)
	if got != "image/png" {
		t.Fatalf("got %q want image/png", got)
	}
}

func TestResolveAttachmentMIME_declaredImage(t *testing.T) {
	got := resolveAttachmentMIME("x", "image/jpeg", "AAAA")
	if got != "image/jpeg" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAttachmentMIME_extFallback(t *testing.T) {
	got := resolveAttachmentMIME("shot.png", "", base64.StdEncoding.EncodeToString([]byte("not-image")))
	if got != "image/png" {
		t.Fatalf("got %q want image/png", got)
	}
}

func TestBuildHarnessInputPartsShouldKeepGenericFileAsFilePart(t *testing.T) {
	parts := buildHarnessInputParts("请阅读附件", []HarnessAttachment{{
		Filename:      "spec.pdf",
		MimeType:      "application/pdf",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("%PDF-1.7 sample")),
	}})

	if len(parts) != 2 {
		t.Fatalf("got %d parts want 2", len(parts))
	}
	if parts[1].Type != schema.ChatMessagePartTypeFileURL || parts[1].File == nil {
		t.Fatalf("expected file part, got %#v", parts[1])
	}
	if parts[1].File.Name != "spec.pdf" {
		t.Fatalf("unexpected file name: %#v", parts[1].File)
	}
}

func TestBuildHarnessInputPartsShouldFallbackFileToLegacyTextWhenRequested(t *testing.T) {
	parts := buildHarnessInputPartsWithOptions("读文件", []HarnessAttachment{{
		Filename:      "notes.pdf",
		MimeType:      "application/pdf",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("pdf")),
	}}, HarnessMultimodalOptions{DisableGenericFilePart: true})

	if len(parts) < 2 {
		t.Fatalf("expected text fallback parts, got %#v", parts)
	}
	last := parts[len(parts)-1]
	if last.Type != schema.ChatMessagePartTypeText {
		t.Fatalf("expected legacy text fallback, got %#v", last)
	}
	if last.Text == "" {
		t.Fatalf("expected fallback text content, got %#v", last)
	}
}
