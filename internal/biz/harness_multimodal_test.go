package biz

import (
	"encoding/base64"
	"testing"
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
