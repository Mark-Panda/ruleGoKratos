package service

import "testing"

func TestResolveUserIDWithSessionFallback(t *testing.T) {
	t.Run("prefer user id", func(t *testing.T) {
		userID, projectPath, sessionID := resolveIdentityWithSessionFallback("u-1", "p-1", "s-1")
		if userID != "u-1" || sessionID != "s-1" || projectPath != "p-1" {
			t.Fatalf("unexpected result: userID=%q projectPath=%q sessionID=%q", userID, projectPath, sessionID)
		}
	})

	t.Run("fallback to session id", func(t *testing.T) {
		userID, projectPath, sessionID := resolveIdentityWithSessionFallback("", "", "s-2")
		if userID != "s-2" || sessionID != "s-2" || projectPath == "" {
			t.Fatalf("expected session fallback, got userID=%q projectPath=%q sessionID=%q", userID, projectPath, sessionID)
		}
	})

	t.Run("generate session when missing both", func(t *testing.T) {
		userID, projectPath, sessionID := resolveIdentityWithSessionFallback("", "", "")
		if userID == "" || sessionID == "" || projectPath == "" {
			t.Fatalf("expected generated ids, got userID=%q projectPath=%q sessionID=%q", userID, projectPath, sessionID)
		}
		if userID != sessionID {
			t.Fatalf("expected generated userID=sessionID, got userID=%q sessionID=%q", userID, sessionID)
		}
	})
}
