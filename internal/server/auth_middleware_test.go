package server

import (
	"context"
	"net/http"
	"testing"
)

type headerCarrier struct {
	header http.Header
}

func (h headerCarrier) Header() http.Header {
	return h.header
}

func TestResolveIdentityWithFallback(t *testing.T) {
	t.Run("prefer explicit user and project", func(t *testing.T) {
		userID, projectPath, sessionID := resolveIdentityWithFallback("u-1", "p-1", "s-1")
		if userID != "u-1" || projectPath != "p-1" || sessionID != "s-1" {
			t.Fatalf("unexpected identity: userID=%q projectPath=%q sessionID=%q", userID, projectPath, sessionID)
		}
	})

	t.Run("generate when all empty", func(t *testing.T) {
		userID, projectPath, sessionID := resolveIdentityWithFallback("", "", "")
		if userID == "" || projectPath == "" || sessionID == "" {
			t.Fatalf("identity should be generated, got userID=%q projectPath=%q sessionID=%q", userID, projectPath, sessionID)
		}
		if userID != sessionID {
			t.Fatalf("userID should equal sessionID when generated, got userID=%q sessionID=%q", userID, sessionID)
		}
	})
}

func TestAuthMiddlewareInjectsFallbackIdentity(t *testing.T) {
	m := AuthMiddleware()
	handler := m(func(ctx context.Context, in interface{}) (interface{}, error) {
		userID := UserIDFromContext(ctx)
		projectPath := ProjectPathFromContext(ctx)
		if userID == "" {
			t.Fatalf("expected fallback userID injected")
		}
		if projectPath == "" {
			t.Fatalf("expected fallback projectPath injected")
		}
		return nil, nil
	})

	_, err := handler(context.Background(), headerCarrier{header: make(http.Header)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
