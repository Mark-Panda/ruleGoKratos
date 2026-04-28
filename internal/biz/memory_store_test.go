package biz

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUserFilePathUsesStableHash(t *testing.T) {
	store, err := NewFileMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileMemoryStore failed: %v", err)
	}
	p1 := store.userFilePath("../../evil")
	p2 := store.userFilePath("../../evil")
	if p1 != p2 {
		t.Fatalf("expected stable path for same user id, got %q vs %q", p1, p2)
	}
	if strings.Contains(p1, "..") {
		t.Fatalf("user memory path should not contain traversal segments: %q", p1)
	}
	usersDir := filepath.Join(store.root, "users")
	if !strings.HasPrefix(p1, usersDir+string(os.PathSeparator)) {
		t.Fatalf("user memory path should stay under users dir, got %q", p1)
	}
}

func TestSaveUserMemoryWithTraversalLikeUserIDStaysUnderRoot(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileMemoryStore(root)
	if err != nil {
		t.Fatalf("NewFileMemoryStore failed: %v", err)
	}
	userID := "../../escape"
	mem := &UserMemory{UserID: userID}
	if err := store.SaveUserMemory(context.Background(), userID, mem); err != nil {
		t.Fatalf("SaveUserMemory failed: %v", err)
	}
	usersDir := filepath.Join(root, "users")
	entries, err := os.ReadDir(usersDir)
	if err != nil {
		t.Fatalf("ReadDir users failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one user memory file, got %d", len(entries))
	}
	filePath := filepath.Join(usersDir, entries[0].Name())
	if !strings.HasPrefix(filePath, usersDir+string(os.PathSeparator)) {
		t.Fatalf("unexpected file path out of users dir: %q", filePath)
	}
}

func TestProjectContextIncludesRecentSummaries(t *testing.T) {
	mem := &ProjectMemory{
		Summaries: MemorySection{
			Entries: []MemoryEntry{
				{Content: "s1"},
				{Content: "s2"},
				{Content: "s3"},
				{Content: "s4"},
			},
		},
	}
	ctx := mem.BuildContext()
	if !strings.Contains(ctx, "近期会话摘要:") {
		t.Fatalf("expected summaries section in project context, got %q", ctx)
	}
	if strings.Contains(ctx, "s1") {
		t.Fatalf("expected only recent summaries, got %q", ctx)
	}
	for _, s := range []string{"s2", "s3", "s4"} {
		if !strings.Contains(ctx, s) {
			t.Fatalf("expected summary %q in project context, got %q", s, ctx)
		}
	}
}

func TestGetUserMemoryReturnsClone(t *testing.T) {
	store, err := NewFileMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileMemoryStore failed: %v", err)
	}
	userID := "u1"
	origin := &UserMemory{UserID: userID}
	addMemoryEntry(&origin.Preferences, "A", "test")
	if err := store.SaveUserMemory(context.Background(), userID, origin); err != nil {
		t.Fatalf("SaveUserMemory failed: %v", err)
	}
	got, err := store.GetUserMemory(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserMemory failed: %v", err)
	}
	addMemoryEntry(&got.Preferences, "B", "test")
	got2, err := store.GetUserMemory(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserMemory(2) failed: %v", err)
	}
	if len(got2.Preferences.Entries) != 1 {
		t.Fatalf("expected cached memory unchanged by caller mutation, got %d entries", len(got2.Preferences.Entries))
	}
}

func TestAddSessionSummaryTrimToMaxEntries(t *testing.T) {
	store, err := NewFileMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileMemoryStore failed: %v", err)
	}
	projectPath := "/workspace/demo"
	for i := 1; i <= defaultMaxProjectSummaryEntries+5; i++ {
		if err := store.AddSessionSummary(context.Background(), projectPath, fmt.Sprintf("summary-%d", i)); err != nil {
			t.Fatalf("AddSessionSummary failed at %d: %v", i, err)
		}
	}
	mem, err := store.GetProjectMemory(context.Background(), projectPath)
	if err != nil {
		t.Fatalf("GetProjectMemory failed: %v", err)
	}
	if len(mem.Summaries.Entries) != defaultMaxProjectSummaryEntries {
		t.Fatalf("expected summaries trimmed to %d, got %d", defaultMaxProjectSummaryEntries, len(mem.Summaries.Entries))
	}
	first := mem.Summaries.Entries[0].Content
	last := mem.Summaries.Entries[len(mem.Summaries.Entries)-1].Content
	if first != "summary-6" || last != "summary-25" {
		t.Fatalf("expected to keep latest summaries, got first=%q last=%q", first, last)
	}
}

func TestAddUserPreferenceTrimToMaxEntries(t *testing.T) {
	store, err := NewFileMemoryStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileMemoryStore failed: %v", err)
	}
	userID := "user-a"
	for i := 1; i <= defaultMaxUserPreferenceEntries+7; i++ {
		if err := store.AddUserPreference(context.Background(), userID, fmt.Sprintf("pref-%d", i), "test"); err != nil {
			t.Fatalf("AddUserPreference failed at %d: %v", i, err)
		}
	}
	mem, err := store.GetUserMemory(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserMemory failed: %v", err)
	}
	if len(mem.Preferences.Entries) != defaultMaxUserPreferenceEntries {
		t.Fatalf("expected preferences trimmed to %d, got %d", defaultMaxUserPreferenceEntries, len(mem.Preferences.Entries))
	}
	first := mem.Preferences.Entries[0].Content
	last := mem.Preferences.Entries[len(mem.Preferences.Entries)-1].Content
	if first != "pref-8" || last != "pref-107" {
		t.Fatalf("expected to keep latest preferences, got first=%q last=%q", first, last)
	}
}

func TestNewFileMemoryStoreWithLimitsHonorsCustomCaps(t *testing.T) {
	store, err := NewFileMemoryStoreWithLimits(t.TempDir(), MemoryLimits{
		UserPreferencesLimit:  2,
		UserFeedbackLimit:     2,
		ProjectFactsLimit:     2,
		ProjectDecisionsLimit: 2,
		ProjectSummariesLimit: 2,
	})
	if err != nil {
		t.Fatalf("NewFileMemoryStoreWithLimits failed: %v", err)
	}
	projectPath := "/workspace/custom"
	for i := 1; i <= 5; i++ {
		if err := store.AddSessionSummary(context.Background(), projectPath, fmt.Sprintf("s-%d", i)); err != nil {
			t.Fatalf("AddSessionSummary failed: %v", err)
		}
	}
	mem, err := store.GetProjectMemory(context.Background(), projectPath)
	if err != nil {
		t.Fatalf("GetProjectMemory failed: %v", err)
	}
	if len(mem.Summaries.Entries) != 2 {
		t.Fatalf("expected custom summary cap=2, got %d", len(mem.Summaries.Entries))
	}
	if mem.Summaries.Entries[0].Content != "s-4" || mem.Summaries.Entries[1].Content != "s-5" {
		t.Fatalf("expected to keep latest 2 summaries, got %#v", mem.Summaries.Entries)
	}
}

func TestNormalizeMemoryLimitsFallsBackWhenNonPositive(t *testing.T) {
	got := normalizeMemoryLimits(MemoryLimits{
		UserPreferencesLimit:  0,
		UserFeedbackLimit:     -1,
		ProjectFactsLimit:     0,
		ProjectDecisionsLimit: -1,
		ProjectSummariesLimit: 0,
	}, DefaultMemoryLimitCaps())
	def := DefaultMemoryLimits()
	if got != def {
		t.Fatalf("expected fallback to defaults, got %#v want %#v", got, def)
	}
}

func TestNormalizeMemoryLimitsClampsTooLargeValues(t *testing.T) {
	got := normalizeMemoryLimits(MemoryLimits{
		UserPreferencesLimit:  maxUserPreferenceEntriesLimit + 1,
		UserFeedbackLimit:     maxUserFeedbackEntriesLimit + 1,
		ProjectFactsLimit:     maxProjectFactEntriesLimit + 1,
		ProjectDecisionsLimit: maxProjectDecisionEntriesLimit + 1,
		ProjectSummariesLimit: maxProjectSummaryEntriesLimit + 1,
	}, DefaultMemoryLimitCaps())
	if got.UserPreferencesLimit != maxUserPreferenceEntriesLimit {
		t.Fatalf("unexpected UserPreferencesLimit: %d", got.UserPreferencesLimit)
	}
	if got.UserFeedbackLimit != maxUserFeedbackEntriesLimit {
		t.Fatalf("unexpected UserFeedbackLimit: %d", got.UserFeedbackLimit)
	}
	if got.ProjectFactsLimit != maxProjectFactEntriesLimit {
		t.Fatalf("unexpected ProjectFactsLimit: %d", got.ProjectFactsLimit)
	}
	if got.ProjectDecisionsLimit != maxProjectDecisionEntriesLimit {
		t.Fatalf("unexpected ProjectDecisionsLimit: %d", got.ProjectDecisionsLimit)
	}
	if got.ProjectSummariesLimit != maxProjectSummaryEntriesLimit {
		t.Fatalf("unexpected ProjectSummariesLimit: %d", got.ProjectSummariesLimit)
	}
}

func TestNormalizeMemoryLimitsUsesCustomCaps(t *testing.T) {
	caps := normalizeMemoryLimitCaps(MemoryLimitCaps{
		MaxUserPreferencesLimit:  11,
		MaxUserFeedbackLimit:     12,
		MaxProjectFactsLimit:     13,
		MaxProjectDecisionsLimit: 14,
		MaxProjectSummariesLimit: 15,
	})
	got := normalizeMemoryLimits(MemoryLimits{
		UserPreferencesLimit:  999,
		UserFeedbackLimit:     999,
		ProjectFactsLimit:     999,
		ProjectDecisionsLimit: 999,
		ProjectSummariesLimit: 999,
	}, caps)
	if got.UserPreferencesLimit != 11 || got.UserFeedbackLimit != 12 || got.ProjectFactsLimit != 13 || got.ProjectDecisionsLimit != 14 || got.ProjectSummariesLimit != 15 {
		t.Fatalf("expected custom caps to apply, got %#v", got)
	}
}

func TestProjectMemoryTTLPrunesExpiredEntriesOnRead(t *testing.T) {
	store, err := NewFileMemoryStoreWithLimitsAndCapsAndTTL(t.TempDir(), DefaultMemoryLimits(), DefaultMemoryLimitCaps(), 30)
	if err != nil {
		t.Fatalf("NewFileMemoryStoreWithLimitsAndCapsAndTTL failed: %v", err)
	}
	projectPath := "/workspace/ttl-project"
	mem := &ProjectMemory{
		ProjectPath: projectPath,
		Summaries: MemorySection{
			Entries: []MemoryEntry{
				{ID: "old", Content: "old-summary", CreatedAt: time.Now().AddDate(0, 0, -31), Source: "session_summary"},
				{ID: "new", Content: "new-summary", CreatedAt: time.Now().AddDate(0, 0, -1), Source: "session_summary"},
			},
		},
	}
	if err := store.SaveProjectMemory(context.Background(), projectPath, mem); err != nil {
		t.Fatalf("SaveProjectMemory failed: %v", err)
	}
	got, err := store.GetProjectMemory(context.Background(), projectPath)
	if err != nil {
		t.Fatalf("GetProjectMemory failed: %v", err)
	}
	if len(got.Summaries.Entries) != 1 || got.Summaries.Entries[0].Content != "new-summary" {
		t.Fatalf("expected expired summary pruned, got %#v", got.Summaries.Entries)
	}
}

func TestUserMemoryTTLPrunesExpiredEntriesOnRead(t *testing.T) {
	store, err := NewFileMemoryStoreWithLimitsAndCapsAndTTL(t.TempDir(), DefaultMemoryLimits(), DefaultMemoryLimitCaps(), 30)
	if err != nil {
		t.Fatalf("NewFileMemoryStoreWithLimitsAndCapsAndTTL failed: %v", err)
	}
	userID := "ttl-user"
	mem := &UserMemory{
		UserID: userID,
		Feedback: MemorySection{
			Entries: []MemoryEntry{
				{ID: "old", Content: "old-feedback", CreatedAt: time.Now().AddDate(0, 0, -35), Source: "session_summary"},
				{ID: "new", Content: "new-feedback", CreatedAt: time.Now().AddDate(0, 0, -2), Source: "session_summary"},
			},
		},
	}
	if err := store.SaveUserMemory(context.Background(), userID, mem); err != nil {
		t.Fatalf("SaveUserMemory failed: %v", err)
	}
	got, err := store.GetUserMemory(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserMemory failed: %v", err)
	}
	if len(got.Feedback.Entries) != 1 || got.Feedback.Entries[0].Content != "new-feedback" {
		t.Fatalf("expected expired feedback pruned, got %#v", got.Feedback.Entries)
	}
}
