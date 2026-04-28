package biz

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

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
	UserID      string         `json:"user_id"`
	LastUpdated time.Time      `json:"last_updated"`
	Preferences MemorySection  `json:"preferences"`  // 用户偏好
	Feedback    MemorySection  `json:"feedback"`     // 用户反馈/修正
}

// ProjectMemory 项目级记忆
type ProjectMemory struct {
	ProjectPath string        `json:"project_path"`
	LastUpdated time.Time     `json:"last_updated"`
	Facts       MemorySection `json:"facts"`        // 项目事实（结构、技术栈）
	Decisions   MemorySection `json:"decisions"`    // 重要决策
	Summaries   MemorySection `json:"summaries"`    // 会话摘要
}

// FileMemoryStore 基于文件的记忆存储实现
type FileMemoryStore struct {
	root   string
	mu     sync.RWMutex
	userCache map[string]*UserMemory
	projectCache map[string]*ProjectMemory
}

// NewFileMemoryStore 创建文件记忆存储
func NewFileMemoryStore(root string) (*FileMemoryStore, error) {
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
		root:        root,
		userCache:   make(map[string]*UserMemory),
		projectCache: make(map[string]*ProjectMemory),
	}, nil
}

func (s *FileMemoryStore) userFilePath(userID string) string {
	return filepath.Join(s.root, "users", fmt.Sprintf("%s.json", userID))
}

func (s *FileMemoryStore) projectFilePath(projectPath string) string {
	// 使用安全的文件名，避免路径遍历
	hash := fmt.Sprintf("%x", hashString(projectPath))
	return filepath.Join(s.root, "projects", fmt.Sprintf("%s.json", hash))
}

func hashString(s string) uint64 {
	var h uint64
	for _, c := range s {
		h = h*31 + uint64(c)
	}
	return h
}

func (s *FileMemoryStore) GetUserMemory(ctx interface{}, userID string) (*UserMemory, error) {
	s.mu.RLock()
	if mem, ok := s.userCache[userID]; ok {
		s.mu.RUnlock()
		return mem, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查
	if mem, ok := s.userCache[userID]; ok {
		return mem, nil
	}

	path := s.userFilePath(userID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 返回空记忆
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
	return &mem, nil
}

func (s *FileMemoryStore) SaveUserMemory(ctx interface{}, userID string, mem *UserMemory) error {
	mem.LastUpdated = time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *FileMemoryStore) GetProjectMemory(ctx interface{}, projectPath string) (*ProjectMemory, error) {
	projectPath = filepath.Clean(projectPath)

	s.mu.RLock()
	if mem, ok := s.projectCache[projectPath]; ok {
		s.mu.RUnlock()
		return mem, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查
	if mem, ok := s.projectCache[projectPath]; ok {
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
	return &mem, nil
}

func (s *FileMemoryStore) SaveProjectMemory(ctx interface{}, projectPath string, mem *ProjectMemory) error {
	projectPath = filepath.Clean(projectPath)
	mem.LastUpdated = time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

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

// AddUserPreference 添加用户偏好记忆
func (s *FileMemoryStore) AddUserPreference(ctx interface{}, userID, content, source string) error {
	mem, err := s.GetUserMemory(ctx, userID)
	if err != nil {
		return err
	}
	addMemoryEntry(&mem.Preferences, content, source)
	return s.SaveUserMemory(ctx, userID, mem)
}

// AddProjectFact 添加项目事实记忆
func (s *FileMemoryStore) AddProjectFact(ctx interface{}, projectPath, content, source string) error {
	mem, err := s.GetProjectMemory(ctx, projectPath)
	if err != nil {
		return err
	}
	addMemoryEntry(&mem.Facts, content, source)
	return s.SaveProjectMemory(ctx, projectPath, mem)
}

// AddDecision 添加项目决策记忆
func (s *FileMemoryStore) AddDecision(ctx interface{}, projectPath, content string) error {
	mem, err := s.GetProjectMemory(ctx, projectPath)
	if err != nil {
		return err
	}
	addMemoryEntry(&mem.Decisions, content, "decision")
	return s.SaveProjectMemory(ctx, projectPath, mem)
}

// AddSessionSummary 添加会话摘要记忆
func (s *FileMemoryStore) AddSessionSummary(ctx interface{}, projectPath, summary string) error {
	mem, err := s.GetProjectMemory(ctx, projectPath)
	if err != nil {
		return err
	}
	addMemoryEntry(&mem.Summaries, summary, "session_summary")
	// 限制摘要数量，避免无限增长
	const maxSummaries = 20
	if len(mem.Summaries.Entries) > maxSummaries {
		mem.Summaries.Entries = mem.Summaries.Entries[len(mem.Summaries.Entries)-maxSummaries:]
	}
	return s.SaveProjectMemory(ctx, projectPath, mem)
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
	if len(parts) == 0 {
		return ""
	}
	return "\n" + strings.Join(parts, "\n")
}

// Ensure this file compiles
var _ MemoryStore = (*FileMemoryStore)(nil)
