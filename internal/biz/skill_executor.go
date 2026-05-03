package biz

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type FileSkillExecutor struct {
	mu          sync.RWMutex
	dirs        []string
	skills      map[string]string
	fingerprint string

	namespace    string
	allowAll     bool
	allowExact   map[string]struct{}
	allowPrefix  []string
	hotReload    bool
	scanInterval time.Duration
	lastScan     time.Time
}

type FileSkillExecutorOptions struct {
	Namespace         string
	AllowList         string
	HotReload         bool
	HotReloadSet      bool
	ScanIntervalMS    int
	ScanIntervalMSSet bool
}

// defaultHomeToolSkillRoots 容器/根用户下常见工具链技能目录（排在服务三类目录之后，同名以服务内为准）。
var defaultHomeToolSkillRoots = []string{
	"/root/.agents/skills",
	"/root/.claude/skills",
	"/root/.cursor/skills",
}

// defaultSkillDirs 组装服务内允许的技能目录，顺序即同名技能优先级：
// 系统 > Agent > 工作流 > ~/.agents/skills > ~/.claude/skills > ~/.cursor/skills（后三项默认路径见 defaultHomeToolSkillRoots）。
func defaultSkillDirs(dir string, dirsCSV string) []string {
	dirs := make([]string, 0, 3+len(defaultHomeToolSkillRoots))
	appendIfNeeded := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		for _, existing := range dirs {
			if existing == raw {
				return
			}
		}
		dirs = append(dirs, raw)
	}

	appDir := strings.TrimSpace(os.Getenv("APP_SKILL_DIR"))
	if appDir == "" {
		appDir = "/app/skills"
	}
	agentDir := strings.TrimSpace(dir)
	if agentDir == "" {
		agentDir = strings.TrimSpace(os.Getenv("AGENT_SKILL_DIR"))
	}
	if agentDir == "" {
		agentDir = "/agent/skills"
	}
	workflowDir := strings.TrimSpace(os.Getenv("WORKFLOW_SKILL_DIR"))
	if workflowDir == "" {
		workflowDir = strings.TrimSpace(os.Getenv("RULE_CHAIN_SKILL_DIR"))
	}
	if workflowDir == "" {
		workflowDir = "/workflow/skills"
	}
	_ = dirsCSV // 历史额外目录不再参与服务 SKILL 扫描；默认路径见本函数与前缀常量。
	appendIfNeeded(appDir)
	appendIfNeeded(agentDir)
	appendIfNeeded(workflowDir)
	for _, root := range defaultHomeToolSkillRoots {
		appendIfNeeded(root)
	}
	return dirs
}

// Dirs 返回当前已加载的技能目录列表（绝对路径化后的副本）。
func (e *FileSkillExecutor) Dirs() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]string, len(e.dirs))
	copy(result, e.dirs)
	return result
}

var _ = isSkillFile

// isSkillFile 判断文件是否属于可加载的技能文件类型。
func isSkillFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".txt" || ext == ".yaml" || ext == ".yml" || ext == ".json"
}

// normalizeSkillName 统一技能名格式，兼容不同路径分隔符。
func normalizeSkillName(name string) string {
	return strings.Trim(strings.ReplaceAll(name, "\\", "/"), "/")
}

// normalizeByNamespace 将技能名映射到命名空间下，避免跨域同名冲突。
func normalizeByNamespace(namespace, skillName string) string {
	skillName = normalizeSkillName(skillName)
	if namespace == "" || skillName == "" {
		return skillName
	}
	if strings.HasPrefix(skillName, namespace+"/") {
		return skillName
	}
	return namespace + "/" + skillName
}

func stripSkillYAMLScalarQuotes(v string) string {
	v = strings.TrimSpace(v)
	if len(v) < 2 {
		return v
	}
	if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
		return strings.TrimSpace(v[1 : len(v)-1])
	}
	return v
}

// SkillNameFromFrontMatter 从 SKILL.md 的 YAML frontmatter 中解析 name。
func SkillNameFromFrontMatter(content string) string {
	content = strings.TrimPrefix(content, "\ufeff")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "---" || line == "..." {
			return ""
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "name" {
			continue
		}
		return normalizeSkillName(stripSkillYAMLScalarQuotes(val))
	}
	return ""
}

// loadSkills 按 Eino 官方 Skill 包约定扫描：每个一级子目录中的 SKILL.md 是唯一入口。
func loadSkills(dirs []string, namespace string) (map[string]string, string, error) {
	skills := make(map[string]string)
	fingerprintParts := make([]string, 0, 64)
	registerSkill := func(name string, info os.FileInfo, content string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		fingerprintParts = append(fingerprintParts, fmt.Sprintf("%s:%d:%d", name, info.ModTime().UnixNano(), info.Size()))
		if _, exists := skills[name]; exists {
			return
		}
		skills[name] = content
	}
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		stat, err := os.Stat(dir)
		if err != nil || !stat.IsDir() {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, "", fmt.Errorf("扫描skill目录失败: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			packageName := normalizeSkillName(entry.Name())
			if packageName == "" {
				continue
			}
			skillPath := filepath.Join(dir, entry.Name(), "SKILL.md")
			data, err := os.ReadFile(skillPath)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, "", fmt.Errorf("读取skill文件失败: %w", err)
			}
			info, err := os.Stat(skillPath)
			if err != nil {
				return nil, "", fmt.Errorf("读取skill文件信息失败: %w", err)
			}
			content := string(data)
			name := SkillNameFromFrontMatter(content)
			if name == "" {
				name = packageName
			}
			name = normalizeByNamespace(namespace, name)
			registerSkill(name, info, content)
		}
	}
	sort.Strings(fingerprintParts)
	return skills, strings.Join(fingerprintParts, "|"), nil
}

// parseAllowList 解析技能白名单配置，支持精确匹配与前缀通配。
func parseAllowList(namespace string, raw string) (bool, map[string]struct{}, []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil, nil
	}
	if raw == "*" {
		return true, nil, nil
	}
	exact := make(map[string]struct{})
	prefix := make([]string, 0, 8)
	items := strings.Split(raw, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if item == "*" {
			return true, nil, nil
		}
		item = normalizeByNamespace(namespace, item)
		if strings.HasSuffix(item, "*") {
			prefix = append(prefix, strings.TrimSuffix(item, "*"))
			continue
		}
		exact[item] = struct{}{}
	}
	return false, exact, prefix
}

// NewFileSkillExecutor 创建基于目录的技能执行器。
func NewFileSkillExecutor(dirs []string, opts FileSkillExecutorOptions) (*FileSkillExecutor, error) {
	const (
		defaultHotReload      = true
		defaultScanIntervalMS = 1000
	)
	namespace := normalizeSkillName(strings.TrimSpace(opts.Namespace))
	allowAll, allowExact, allowPrefix := parseAllowList(namespace, opts.AllowList)
	hotReload := defaultHotReload
	if opts.HotReloadSet {
		hotReload = opts.HotReload
	}
	scanIntervalMS := defaultScanIntervalMS
	if opts.ScanIntervalMSSet && opts.ScanIntervalMS >= 0 {
		scanIntervalMS = opts.ScanIntervalMS
	}
	interval := time.Duration(scanIntervalMS) * time.Millisecond
	skills, fingerprint, err := loadSkills(dirs, namespace)
	if err != nil {
		return nil, err
	}
	return &FileSkillExecutor{
		dirs:         dirs,
		skills:       skills,
		fingerprint:  fingerprint,
		namespace:    namespace,
		allowAll:     allowAll,
		allowExact:   allowExact,
		allowPrefix:  allowPrefix,
		hotReload:    hotReload,
		scanInterval: interval,
		lastScan:     time.Now(),
	}, nil
}

// isAllowed 判断指定技能是否通过白名单策略。
func (e *FileSkillExecutor) isAllowed(skillName string) bool {
	if e.allowAll {
		return true
	}
	if len(e.allowExact) == 0 && len(e.allowPrefix) == 0 {
		return true
	}
	if _, ok := e.allowExact[skillName]; ok {
		return true
	}
	for _, p := range e.allowPrefix {
		if strings.HasPrefix(skillName, p) {
			return true
		}
	}
	return false
}

// tryReload 按间隔触发目录重扫，支持技能热加载。
func (e *FileSkillExecutor) tryReload() {
	if !e.hotReload {
		return
	}
	now := time.Now()
	e.mu.RLock()
	needScan := e.scanInterval == 0 || now.Sub(e.lastScan) >= e.scanInterval
	e.mu.RUnlock()
	if !needScan {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.scanInterval != 0 && now.Sub(e.lastScan) < e.scanInterval {
		return
	}
	e.lastScan = now
	skills, fingerprint, err := loadSkills(e.dirs, e.namespace)
	if err != nil || fingerprint == e.fingerprint {
		return
	}
	e.skills = skills
	e.fingerprint = fingerprint
}

// ListAvailableSkillNames 返回当前目录中已加载且通过白名单的官方 Skill name，已排序。
func (e *FileSkillExecutor) ListAvailableSkillNames() []string {
	e.tryReload()
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, 0, len(e.skills))
	for name := range e.skills {
		if e.isAllowed(name) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// Execute 执行技能：先做热更新与权限校验，再读取内容并注入 payload。
func (e *FileSkillExecutor) Execute(ctx context.Context, skillName string, payload string) (string, error) {
	e.tryReload()
	normalizedName := normalizeByNamespace(e.namespace, skillName)
	log.Info("skill_execute", "namespace", e.namespace, "skill", skillName, "normalized", normalizedName, "payload_len", len(payload))

	if !e.isAllowed(normalizedName) {
		log.Warn("skill_execute_rejected", "skill", skillName, "reason", "not_in_allowlist", "normalized", normalizedName)
		return "", fmt.Errorf("skill无权限调用: %s", normalizedName)
	}

	e.mu.RLock()
	content, ok := e.skills[normalizedName]
	names := make([]string, 0, len(e.skills))
	for k := range e.skills {
		names = append(names, k)
	}
	e.mu.RUnlock()

	if !ok {
		sort.Strings(names)
		log.Warn("skill_execute_not_found", "skill", skillName, "normalized", normalizedName, "available", strings.Join(names, ","))
		if len(names) == 0 {
			return "", fmt.Errorf("skill目录中暂无可用技能，请检查目录: %v", e.dirs)
		}
		return "", fmt.Errorf("skill不存在: %s，可用skills: %s", normalizedName, strings.Join(names, ","))
	}
	content = strings.TrimSpace(content)
	if payload == "" {
		return content, nil
	}
	if strings.Contains(content, "{{payload}}") {
		return strings.ReplaceAll(content, "{{payload}}", payload), nil
	}
	return content + "\n\npayload:\n" + payload, nil
}
