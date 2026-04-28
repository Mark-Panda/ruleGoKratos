package data

import "testing"

func TestParseFeishuCliAuthStatus(t *testing.T) {
	raw := `{"tokenStatus":"valid","userName":"demo"}`
	parsed, err := parseFeishuCliAuthStatus(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.TokenStatus != "valid" {
		t.Fatalf("unexpected tokenStatus: %q", parsed.TokenStatus)
	}
	if parsed.UserName != "demo" {
		t.Fatalf("unexpected userName: %q", parsed.UserName)
	}
}

func TestParseFeishuCliAuthStatus_AllowWrappedLogs(t *testing.T) {
	raw := "some logs...\n" + `{"tokenStatus":"invalid","ok":false}` + "\nmore logs"
	parsed, err := parseFeishuCliAuthStatus(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if parsed.TokenStatus != "invalid" {
		t.Fatalf("unexpected tokenStatus: %q", parsed.TokenStatus)
	}
	if parsed.OK == nil || *parsed.OK {
		t.Fatalf("unexpected ok field: %#v", parsed.OK)
	}
}

func TestFeishuCliStatusLooksAuthed(t *testing.T) {
	authed, parsed, err := feishuCliStatusLooksAuthed(`{"tokenStatus":"valid"}`, "")
	if err != nil {
		t.Fatalf("status parse failed: %v", err)
	}
	if !authed {
		t.Fatalf("expected authed=true")
	}
	if parsed == nil || parsed.TokenStatus != "valid" {
		t.Fatalf("unexpected parsed payload: %#v", parsed)
	}

	authed, parsed, err = feishuCliStatusLooksAuthed(`{"tokenStatus":"invalid","ok":false}`, "")
	if err != nil {
		t.Fatalf("status parse failed: %v", err)
	}
	if authed {
		t.Fatalf("expected authed=false")
	}
	if parsed == nil || parsed.TokenStatus != "invalid" {
		t.Fatalf("unexpected parsed payload: %#v", parsed)
	}
}
