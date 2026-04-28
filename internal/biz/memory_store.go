package biz

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultMaxUserPreferenceEntries  = 100
	defaultMaxUserFeedbackEntries    = 100
	defaultMaxProjectFactEntries     = 120
	defaultMaxProjectDecisionEntries = 120
	defaultMaxProjectSummaryEntries  = 20

	maxUserPreferenceEntriesLimit  = 1000
	maxUserFeedbackEntriesLimit    = 1000
	maxProjectFactEntriesLimit     = 2000
	maxProjectDecisionEntriesLimit = 2000
	maxProjectSummaryEntriesLimit  = 500
	defaultMemoryEntryTTLDays      = 90
)

type MemoryLimits struct {
	UserPreferencesLimit  int
	UserFeedbackLimit     int
	ProjectFactsLimit     int
	ProjectDecisionsLimit int
	ProjectSummariesLimit int
}

type MemoryLimitCaps struct {
	MaxUserPreferencesLimit  int
	MaxUserFeedbackLimit     int
	MaxProjectFactsLimit     int
	MaxProjectDecisionsLimit int
	MaxProjectSummariesLimit int
}

func DefaultMemoryLimits() MemoryLimits {
	return MemoryLimits{
		UserPreferencesLimit:  defaultMaxUserPreferenceEntries,
		UserFeedbackLimit:     defaultMaxUserFeedbackEntries,
		ProjectFactsLimit:     defaultMaxProjectFactEntries,
		ProjectDecisionsLimit: defaultMaxProjectDecisionEntries,
		ProjectSummariesLimit: defaultMaxProjectSummaryEntries,
	}
}

func DefaultMemoryLimitCaps() MemoryLimitCaps {
	return MemoryLimitCaps{
		MaxUserPreferencesLimit:  maxUserPreferenceEntriesLimit,
		MaxUserFeedbackLimit:     maxUserFeedbackEntriesLimit,
		MaxProjectFactsLimit:     maxProjectFactEntriesLimit,
		MaxProjectDecisionsLimit: maxProjectDecisionEntriesLimit,
		MaxProjectSummariesLimit: maxProjectSummaryEntriesLimit,
	}
}

func normalizeMemoryLimitCaps(in MemoryLimitCaps) MemoryLimitCaps {
	out := in
	def := DefaultMemoryLimitCaps()
	if out.MaxUserPreferencesLimit <= 0 {
		out.MaxUserPreferencesLimit = def.MaxUserPreferencesLimit
	}
	if out.MaxUserFeedbackLimit <= 0 {
		out.MaxUserFeedbackLimit = def.MaxUserFeedbackLimit
	}
	if out.MaxProjectFactsLimit <= 0 {
		out.MaxProjectFactsLimit = def.MaxProjectFactsLimit
	}
	if out.MaxProjectDecisionsLimit <= 0 {
		out.MaxProjectDecisionsLimit = def.MaxProjectDecisionsLimit
	}
	if out.MaxProjectSummariesLimit <= 0 {
		out.MaxProjectSummariesLimit = def.MaxProjectSummariesLimit
	}
	return out
}

func normalizeMemoryLimits(in MemoryLimits, caps MemoryLimitCaps) MemoryLimits {
	out := in
	def := DefaultMemoryLimits()
	normalizedCaps := normalizeMemoryLimitCaps(caps)
	out.UserPreferencesLimit = clampMemoryLimit(out.UserPreferencesLimit, def.UserPreferencesLimit, normalizedCaps.MaxUserPreferencesLimit)
	out.UserFeedbackLimit = clampMemoryLimit(out.UserFeedbackLimit, def.UserFeedbackLimit, normalizedCaps.MaxUserFeedbackLimit)
	out.ProjectFactsLimit = clampMemoryLimit(out.ProjectFactsLimit, def.ProjectFactsLimit, normalizedCaps.MaxProjectFactsLimit)
	out.ProjectDecisionsLimit = clampMemoryLimit(out.ProjectDecisionsLimit, def.ProjectDecisionsLimit, normalizedCaps.MaxProjectDecisionsLimit)
	out.ProjectSummariesLimit = clampMemoryLimit(out.ProjectSummariesLimit, def.ProjectSummariesLimit, normalizedCaps.MaxProjectSummariesLimit)
	return out
}

func clampMemoryLimit(value int, fallback int, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

// MemoryStore 持久化记忆存储接口
type MemoryStore interface {
	// GetUserMemory 获取用户记忆
	GetUserMemory(ctx interface{}, userID string) (*UserMemory, error)
	// SaveUserMemory 保存用户记忆
	SaveUserMemory(ctx interface{}, userID string, mem *UserMemory) error
	// GetProjectMemory 获取项目记忆
	GetProjectMemory(ctx interface{}, projectPath string) (*ProjectMemory, error)
	// SaveProjectMemory 保存项目记忆
	SaveProjectMemory(ctx interface{}, projectPath string, mem *ProjectMemory) error
	// AddUserPreference 添加用户偏好记忆
	AddUserPreference(ctx interface{}, userID, content, source string) error
	// AddUserFeedback 添加用户反馈记忆
	AddUserFeedback(ctx interface{}, userID, content, source string) error
	// AddProjectFact 添加项目事实记忆
	AddProjectFact(ctx interface{}, projectPath, content, source string) error
	// AddDecision 添加项目决策记忆
	AddDecision(ctx interface{}, projectPath, content string) error
	// AddSessionSummary 添加会话摘要记忆
	AddSessionSummary(ctx interface{}, projectPath, summary string) error
}

// MemoryEntry 一条记忆条目
type MemoryEntry struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"` // 来源标识，如 "session_summary", "user_preference", "project_fact"
}

// MemorySection 记忆分区
type MemorySection struct {
	Entries []MemoryEntry `json:"entries"`
}

// UserMemory 用户级记忆（跨项目共享）
type UserMemory struct {
	UserID      string        `json:"user_id"`
	LastUpdated time.Time     `json:"last_updated"`
	Preferences MemorySection `json:"preferences"` // 用户偏好
	Feedback    MemorySection `json:"feedback"`    // 用户反馈/修正
}

// ProjectMemory 项目级记忆
type ProjectMemory struct {
	ProjectPath string        `json:"project_path"`
	LastUpdated time.Time     `json:"last_updated"`
	Facts       MemorySection `json:"facts"`     // 项目事实（结构、技术栈）
	Decisions   MemorySection `json:"decisions"` // 重要决策
	Summaries   MemorySection `json:"summaries"` // 会话摘要
}

// FileMemoryStore 基于文件的记忆存储实现
type FileMemoryStore struct {
	root         string
	mu           sync.RWMutex
	userCache    map[string]*UserMemory
	projectCache map[string]*ProjectMemory
	limits       MemoryLimits
	entryTTLDays int
}

// NewFileMemoryStore 创建文件记忆存储
func NewFileMemoryStore(root string) (*FileMemoryStore, error) {
	return NewFileMemoryStoreWithLimitsAndCapsAndTTL(root, DefaultMemoryLimits(), DefaultMemoryLimitCaps(), defaultMemoryEntryTTLDays)
}

func NewFileMemoryStoreWithLimits(root string, limits MemoryLimits) (*FileMemoryStore, error) {
	return NewFileMemoryStoreWithLimitsAndCapsAndTTL(root, limits, DefaultMemoryLimitCaps(), defaultMemoryEntryTTLDays)
}

func NewFileMemoryStoreWithLimitsAndCaps(root string, limits MemoryLimits, caps MemoryLimitCaps) (*FileMemoryStore, error) {
	return NewFileMemoryStoreWithLimitsAndCapsAndTTL(root, limits, caps, defaultMemoryEntryTTLDays)
}

func NewFileMemoryStoreWithLimitsAndCapsAndTTL(root string, limits MemoryLimits, caps MemoryLimitCaps, entryTTLDays int) (*FileMemoryStore, error) {
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		root = filepath.Join(home, ".claude", "memory")
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create memory root failed: %w", err)
	}
	return &FileMemoryStore{
		root:         root,
		userCache:    make(map[string]*UserMemory),
		projectCache: make(map[string]*ProjectMemory),
		limits:       normalizeMemoryLimits(limits, caps),
		entryTTLDays: entryTTLDays,
	}, nil
}

func (s *FileMemoryStore) ttlCutoff(now time.Time) (time.Time, bool) {
	if s.entryTTLDays <= 0 {
		return time.Time{}, false
	}
	return now.AddDate(0, 0, -s.entryTTLDays), true
}

func (s *FileMemoryStore) userFilePath(userID string) string {
	return filepath.Join(s.root, "users", fmt.Sprintf("%s.json", stableHashKey(userID)))
}

func (s *FileMemoryStore) projectFilePath(projectPath string) string {
	// 使用安全的文件名，避免路径遍历
	hash := fmt.Sprintf("%x", hashString(projectPath))
	return filepath.Join(s.root, "projects", fmt.Sprintf("%s.json", hash))
}

func hashString(s string) uint64 {
	return hashStringFNV64A(s)
}

func hashStringFNV64A(s string) uint64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(s))
	return hasher.Sum64()
}

func stableHashKey(s string) string {
	return fmt.Sprintf("%x", hashStringFNV64A(s))
}

func cloneUserMemory(mem *UserMemory) *UserMemory {
	if mem == nil {
		return nil
	}
	b, err := json.Marshal(mem)
	if err != nil {
		cp := *mem
		return &cp
	}
	var out UserMemory
	if err := json.Unmarshal(b, &out); err != nil {
		cp := *mem
		return &cp
	}
	return &out
}

func cloneProjectMemory(mem *ProjectMemory) *ProjectMemory {
	if mem == nil {
		return nil
	}
	b, err := json.Marshal(mem)
	if err != nil {
		cp := *mem
		return &cp
	}
	var out ProjectMemory
	if err := json.Unmarshal(b, &out); err != nil {
		cp := *mem
		return &cp
	}
	return &out
}

func pruneMemorySectionExpired(section *MemorySection, cutoff time.Time) bool {
	if section == nil || len(section.Entries) == 0 {
		return false
	}
	kept := make([]MemoryEntry, 0, len(section.Entries))
	for _, entry := range section.Entries {
		// 兼容旧数据：CreatedAt 为空时按未过期处理，避免一次性误删历史数据。
		if entry.CreatedAt.IsZero() || !entry.CreatedAt.Before(cutoff) {
			kept = append(kept, entry)
		}
	}
	if len(kept) == len(section.Entries) {
		return false
	}
	section.Entries = kept
	return true
}

func (s *FileMemoryStore) applyUserMemoryPolicyLocked(userID string, mem *UserMemory, forcePersist bool) error {
	if mem == nil {
		return nil
	}
	changed := false
	if cutoff, ok := s.ttlCutoff(time.Now()); ok {
		if pruneMemorySectionExpired(&mem.Preferences, cutoff) {
			changed = true
		}
		if pruneMemorySectionExpired(&mem.Feedback, cutoff) {
			changed = true
		}
	}
	beforePreferences := len(mem.Preferences.Entries)
	beforeFeedback := len(mem.Feedback.Entries)
	trimMemorySection(&mem.Preferences, s.limits.UserPreferencesLimit)
	trimMemorySection(&mem.Feedback, s.limits.UserFeedbackLimit)
	if len(mem.Preferences.Entries) != beforePreferences || len(mem.Feedback.Entries) != beforeFeedback {
		changed = true
	}
	if !changed && !forcePersist {
		return nil
	}
	mem.LastUpdated = time.Now()
	return s.saveUserMemoryLocked(userID, mem)
}

func (s *FileMemoryStore) applyProjectMemoryPolicyLocked(projectPath string, mem *ProjectMemory, forcePersist bool) error {
	if mem == nil {
		return nil
	}
	changed := false
	if cutoff, ok := s.ttlCutoff(time.Now()); ok {
		if pruneMemorySectionExpired(&mem.Facts, cutoff) {
			changed = true
		}
		if pruneMemorySectionExpired(&mem.Decisions, cutoff) {
			changed = true
		}
		if pruneMemorySectionExpired(&mem.Summaries, cutoff) {
			changed = true
		}
	}
	beforeFacts := len(mem.Facts.Entries)
	beforeDecisions := len(mem.Decisions.Entries)
	beforeSummaries := len(mem.Summaries.Entries)
	trimMemorySection(&mem.Facts, s.limits.ProjectFactsLimit)
	trimMemorySection(&mem.Decisions, s.limits.ProjectDecisionsLimit)
	trimMemorySection(&mem.Summaries, s.limits.ProjectSummariesLimit)
	if len(mem.Facts.Entries) != beforeFacts || len(mem.Decisions.Entries) != beforeDecisions || len(mem.Summaries.Entries) != beforeSummaries {
		changed = true
	}
	if !changed && !forcePersist {
		return nil
	}
	mem.LastUpdated = time.Now()
	return s.saveProjectMemoryLocked(projectPath, mem)
}

func (s *FileMemoryStore) getUserMemoryLocked(userID string) (*UserMemory, error) {
	if mem, ok := s.userCache[userID]; ok {
		if err := s.applyUserMemoryPolicyLocked(userID, mem, false); err != nil {
			return nil, err
		}
		return mem, nil
	}
	path := s.userFilePath(userID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			mem := &UserMemory{UserID: userID, LastUpdated: time.Now()}
			s.userCache[userID] = mem
			return mem, nil
		}
		return nil, fmt.Errorf("read user memory failed: %w", err)
	}
	var mem UserMemory
	if err := json.Unmarshal(data, &mem); err != nil {
		return nil, fmt.Errorf("parse user memory failed: %w", err)
	}
	s.userCache[userID] = &mem
	if err := s.applyUserMemoryPolicyLocked(userID, &mem, false); err != nil {
		return nil, err
	}
	return &mem, nil
}

func (s *FileMemoryStore) getProjectMemoryLocked(projectPath string) (*ProjectMemory, error) {
	if mem, ok := s.projectCache[projectPath]; ok {
		if err := s.applyProjectMemoryPolicyLocked(projectPath, mem, false); err != nil {
			return nil, err
		}
		return mem, nil
	}
	path := s.projectFilePath(projectPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			mem := &ProjectMemory{ProjectPath: projectPath, LastUpdated: time.Now()}
			s.projectCache[projectPath] = mem
			return mem, nil
		}
		return nil, fmt.Errorf("read project memory failed: %w", err)
	}
	var mem ProjectMemory
	if err := json.Unmarshal(data, &mem); err != nil {
		return nil, fmt.Errorf("parse project memory failed: %w", err)
	}
	s.projectCache[projectPath] = &mem
	if err := s.applyProjectMemoryPolicyLocked(projectPath, &mem, false); err != nil {
		return nil, err
	}
	return &mem, nil
}

func (s *FileMemoryStore) saveUserMemoryLocked(userID string, mem *UserMemory) error {
	path := s.userFilePath(userID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create user memory dir failed: %w", err)
	}
	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal user memory failed: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write user memory failed: %w", err)
	}
	s.userCache[userID] = mem
	return nil
}

func (s *FileMemoryStore) saveProjectMemoryLocked(projectPath string, mem *ProjectMemory) error {
	path := s.projectFilePath(projectPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create project memory dir failed: %w", err)
	}
	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal project memory failed: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write project memory failed: %w", err)
	}
	s.projectCache[projectPath] = mem
	return nil
}

func (s *FileMemoryStore) GetUserMemory(ctx interface{}, userID string) (*UserMemory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mem, err := s.getUserMemoryLocked(userID)
	if err != nil {
		return nil, err
	}
	return cloneUserMemory(mem), nil
}

func (s *FileMemoryStore) SaveUserMemory(ctx interface{}, userID string, mem *UserMemory) error {
	if mem == nil {
		mem = &UserMemory{UserID: userID}
	}
	cloned := cloneUserMemory(mem)
	cloned.UserID = userID
	cloned.LastUpdated = time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUserMemoryLocked(userID, cloned)
}

func (s *FileMemoryStore) GetProjectMemory(ctx interface{}, projectPath string) (*ProjectMemory, error) {
	projectPath = filepath.Clean(projectPath)

	s.mu.Lock()
	defer s.mu.Unlock()
	mem, err := s.getProjectMemoryLocked(projectPath)
	if err != nil {
		return nil, err
	}
	return cloneProjectMemory(mem), nil
}

func (s *FileMemoryStore) SaveProjectMemory(ctx interface{}, projectPath string, mem *ProjectMemory) error {
	projectPath = filepath.Clean(projectPath)
	if mem == nil {
		mem = &ProjectMemory{ProjectPath: projectPath}
	}
	cloned := cloneProjectMemory(mem)
	cloned.ProjectPath = projectPath
	cloned.LastUpdated = time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveProjectMemoryLocked(projectPath, cloned)
}

// addMemoryEntry 向记忆添加新条目，自动去重（内部函数）
func addMemoryEntry(section *MemorySection, content, source string) {
	// 简单去重：检查最近 N 条是否有相同内容
	const dedupWindow = 5
	start := len(section.Entries) - dedupWindow
	if start < 0 {
		start = 0
	}
	for i := start; i < len(section.Entries); i++ {
		if section.Entries[i].Content == content {
			return // 重复，跳过
		}
	}
	section.Entries = append(section.Entries, MemoryEntry{
		ID:        uuid.NewString(),
		Content:   content,
		CreatedAt: time.Now(),
		Source:    source,
	})
}

func trimMemorySection(section *MemorySection, maxEntries int) {
	if section == nil || maxEntries <= 0 {
		return
	}
	if len(section.Entries) <= maxEntries {
		return
	}
	section.Entries = section.Entries[len(section.Entries)-maxEntries:]
}

// AddUserPreference 添加用户偏好记忆
func (s *FileMemoryStore) AddUserPreference(ctx interface{}, userID, content, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mem, err := s.getUserMemoryLocked(userID)
	if err != nil {
		return err
	}
	addMemoryEntry(&mem.Preferences, content, source)
	return s.applyUserMemoryPolicyLocked(userID, mem, true)
}

// AddUserFeedback 添加用户反馈记忆
func (s *FileMemoryStore) AddUserFeedback(ctx interface{}, userID, content, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mem, err := s.getUserMemoryLocked(userID)
	if err != nil {
		return err
	}
	addMemoryEntry(&mem.Feedback, content, source)
	return s.applyUserMemoryPolicyLocked(userID, mem, true)
}

// AddProjectFact 添加项目事实记忆
func (s *FileMemoryStore) AddProjectFact(ctx interface{}, projectPath, content, source string) error {
	projectPath = filepath.Clean(projectPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	mem, err := s.getProjectMemoryLocked(projectPath)
	if err != nil {
		return err
	}
	addMemoryEntry(&mem.Facts, content, source)
	return s.applyProjectMemoryPolicyLocked(projectPath, mem, true)
}

// AddDecision 添加项目决策记忆
func (s *FileMemoryStore) AddDecision(ctx interface{}, projectPath, content string) error {
	projectPath = filepath.Clean(projectPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	mem, err := s.getProjectMemoryLocked(projectPath)
	if err != nil {
		return err
	}
	addMemoryEntry(&mem.Decisions, content, "decision")
	return s.applyProjectMemoryPolicyLocked(projectPath, mem, true)
}

// AddSessionSummary 添加会话摘要记忆
func (s *FileMemoryStore) AddSessionSummary(ctx interface{}, projectPath, summary string) error {
	projectPath = filepath.Clean(projectPath)
	s.mu.Lock()
	defer s.mu.Unlock()
	mem, err := s.getProjectMemoryLocked(projectPath)
	if err != nil {
		return err
	}
	addMemoryEntry(&mem.Summaries, summary, "session_summary")
	return s.applyProjectMemoryPolicyLocked(projectPath, mem, true)
}

// BuildUserContext 从用户记忆中构建上下文字符串
func (m *UserMemory) BuildContext() string {
	if m == nil {
		return ""
	}
	var parts []string
	if len(m.Preferences.Entries) > 0 {
		parts = append(parts, "用户偏好:")
		for _, e := range m.Preferences.Entries {
			parts = append(parts, fmt.Sprintf("  - %s", e.Content))
		}
	}
	if len(m.Feedback.Entries) > 0 {
		parts = append(parts, "用户反馈:")
		for _, e := range m.Feedback.Entries {
			parts = append(parts, fmt.Sprintf("  - %s", e.Content))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n" + strings.Join(parts, "\n")
}

// BuildProjectContext 从项目记忆中构建上下文字符串
func (m *ProjectMemory) BuildContext() string {
	if m == nil {
		return ""
	}
	var parts []string
	if len(m.Facts.Entries) > 0 {
		parts = append(parts, "项目事实:")
		for _, e := range m.Facts.Entries {
			parts = append(parts, fmt.Sprintf("  - %s", e.Content))
		}
	}
	if len(m.Decisions.Entries) > 0 {
		parts = append(parts, "重要决策:")
		for _, e := range m.Decisions.Entries {
			parts = append(parts, fmt.Sprintf("  - %s", e.Content))
		}
	}
	if len(m.Summaries.Entries) > 0 {
		parts = append(parts, "近期会话摘要:")
		start := len(m.Summaries.Entries) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(m.Summaries.Entries); i++ {
			parts = append(parts, fmt.Sprintf("  - %s", m.Summaries.Entries[i].Content))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n" + strings.Join(parts, "\n")
}

// Ensure this file compiles
var _ MemoryStore = (*FileMemoryStore)(nil)
