package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	einoskill "github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/components/tool"
	"github.com/go-kratos/kratos/v2/log"
)

// recoverableSkillError 是工具调用中可恢复的错误（如技能名拼写错误）。
// Harness 会将此错误作为 ToolMessage 返回给 LLM，让其有机会自我纠正。
type recoverableSkillError struct {
	msg        string
	suggestion string // 最接近的技能名建议
}

func (e *recoverableSkillError) Error() string { return e.msg }
func (e *recoverableSkillError) IsRecoverable() bool { return true }

// IsRecoverableToolError 检查错误是否为可恢复的工具错误。
func IsRecoverableToolError(err error) bool {
	if err == nil {
		return false
	}
	var re *recoverableSkillError
	if errors.As(err, &re) {
		return true
	}
	// 兼容 errors.Is 方式包装的错误
	return errors.Is(err, &recoverableSkillError{})
}

// suggestClosestSkillName 找到最接近的技能名。
// 优先使用前缀/后缀/包含匹配，如果匹配则直接返回；否则使用编辑距离。
func suggestClosestSkillName(available []einoskill.FrontMatter, requested string) string {
	if len(available) == 0 || strings.TrimSpace(requested) == "" {
		return ""
	}
	reqLower := strings.ToLower(strings.TrimSpace(requested))

	// 第一轮：精确/前缀/后缀/包含匹配 - 直接返回最佳匹配
	var bestExact, bestPrefix, bestSuffix, bestContains string

	for _, item := range available {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		nameLower := strings.ToLower(name)

		if nameLower == reqLower {
			bestExact = name
			break // 精确匹配最好，直接返回
		}
		if strings.HasPrefix(nameLower, reqLower) && bestPrefix == "" {
			bestPrefix = name
		}
		if strings.HasSuffix(nameLower, reqLower) && bestSuffix == "" {
			bestSuffix = name
		}
		if strings.Contains(nameLower, reqLower) && bestContains == "" {
			bestContains = name
		}
	}

	// 按优先级返回
	if bestExact != "" {
		return bestExact
	}
	if bestPrefix != "" {
		return bestPrefix
	}
	if bestSuffix != "" {
		return bestSuffix
	}
	if bestContains != "" {
		return bestContains
	}

	// 第二轮：编辑距离匹配
	best := ""
	bestDist := -1
	threshold := len(requested)/2 + 2

	for _, item := range available {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		dist := levenshteinDistance(requested, name)
		if bestDist < 0 || dist < bestDist {
			best = name
			bestDist = dist
		}
	}

	if bestDist >= 0 && bestDist <= threshold {
		return best
	}
	return ""
}

// levenshteinDistance 计算两个字符串的编辑距离。
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	// 小写化以支持大小写不敏感匹配
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	// 使用优化：只保留两行
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min(
				prev[j]+1,   // 删除
				curr[j-1]+1, // 插入
				prev[j-1]+cost, // 替换
			)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

// min returns the minimum of three integers.
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

type localSkillFilesystemBackend struct{}

func (localSkillFilesystemBackend) LsInfo(ctx context.Context, req *filesystem.LsInfoRequest) ([]filesystem.FileInfo, error) {
	return nil, errors.New("LsInfo is not supported by skill runtime")
}

func (localSkillFilesystemBackend) Read(ctx context.Context, req *filesystem.ReadRequest) (*filesystem.FileContent, error) {
	if req == nil || strings.TrimSpace(req.FilePath) == "" {
		return nil, errors.New("file path is required")
	}
	data, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, err
	}
	return &filesystem.FileContent{Content: string(data)}, nil
}

func (localSkillFilesystemBackend) GrepRaw(ctx context.Context, req *filesystem.GrepRequest) ([]filesystem.GrepMatch, error) {
	return nil, errors.New("GrepRaw is not supported by skill runtime")
}

func (localSkillFilesystemBackend) GlobInfo(ctx context.Context, req *filesystem.GlobInfoRequest) ([]filesystem.FileInfo, error) {
	if req == nil {
		return nil, errors.New("glob request is required")
	}
	base := strings.TrimSpace(req.Path)
	if base == "" {
		return nil, errors.New("glob base path is required")
	}
	matches, err := filepath.Glob(filepath.Join(base, req.Pattern))
	if err != nil {
		return nil, err
	}
	out := make([]filesystem.FileInfo, 0, len(matches))
	for _, p := range matches {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		out = append(out, filesystem.FileInfo{
			Path:       p,
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (localSkillFilesystemBackend) Write(ctx context.Context, req *filesystem.WriteRequest) error {
	return errors.New("Write is not supported by skill runtime")
}

func (localSkillFilesystemBackend) Edit(ctx context.Context, req *filesystem.EditRequest) error {
	return errors.New("Edit is not supported by skill runtime")
}

type multiSkillBackend struct {
	backends []einoskill.Backend
}

func (b multiSkillBackend) List(ctx context.Context) ([]einoskill.FrontMatter, error) {
	seen := make(map[string]struct{})
	out := make([]einoskill.FrontMatter, 0)
	for _, backend := range b.backends {
		items, err := backend.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, item)
		}
	}
	return out, nil
}

func (b multiSkillBackend) Get(ctx context.Context, name string) (einoskill.Skill, error) {
	name = strings.TrimSpace(name)
	for _, backend := range b.backends {
		s, err := backend.Get(ctx, name)
		if err == nil {
			return s, nil
		}
	}
	// 技能不存在时，尝试提供最接近的建议
	available, listErr := b.List(ctx)
	if listErr != nil {
		return einoskill.Skill{}, &recoverableSkillError{msg: fmt.Sprintf("skill not found: %s", name)}
	}
	if suggestion := suggestClosestSkillName(available, name); suggestion != "" {
		return einoskill.Skill{}, &recoverableSkillError{
			msg: fmt.Sprintf("skill not found: %s. Did you mean: %s?", name, suggestion),
			suggestion: suggestion,
		}
	}
	names := make([]string, 0, len(available))
	for _, item := range available {
		names = append(names, item.Name)
	}
	return einoskill.Skill{}, &recoverableSkillError{
		msg: fmt.Sprintf("skill not found: %s. Available skills: %s", name, strings.Join(names, ", ")),
	}
}

type filteredSkillBackend struct {
	base  einoskill.Backend
	allow map[string]struct{}
}

func (b filteredSkillBackend) List(ctx context.Context) ([]einoskill.FrontMatter, error) {
	items, err := b.base.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(b.allow) == 0 {
		return items, nil
	}
	out := make([]einoskill.FrontMatter, 0, len(items))
	for _, item := range items {
		if _, ok := b.allow[item.Name]; ok {
			out = append(out, item)
		}
	}
	return out, nil
}

func (b filteredSkillBackend) Get(ctx context.Context, name string) (einoskill.Skill, error) {
	// 先检查 base backend 中 skill 是否存在，以区分"不存在"和"不允许调用"
	_, baseErr := b.base.Get(ctx, name)
	if baseErr != nil {
		// skill 在 base 中不存在，说明是 LLM 幻觉或名称错误，提供建议
		available, listErr := b.List(ctx)
		if listErr != nil {
			return einoskill.Skill{}, &recoverableSkillError{msg: fmt.Sprintf("skill not found: %s", name)}
		}
		if suggestion := suggestClosestSkillName(available, name); suggestion != "" {
			return einoskill.Skill{}, &recoverableSkillError{
				msg: fmt.Sprintf("skill not found: %s. Did you mean: %s?", name, suggestion),
				suggestion: suggestion,
			}
		}
		names := make([]string, 0, len(available))
		for _, item := range available {
			names = append(names, item.Name)
		}
		return einoskill.Skill{}, &recoverableSkillError{
			msg: fmt.Sprintf("skill not found: %s. Available skills: %s", name, strings.Join(names, ", ")),
		}
	}
	if len(b.allow) > 0 {
		if _, ok := b.allow[name]; !ok {
			// skill 存在但不在白名单中，说明是权限问题
			return einoskill.Skill{}, &recoverableSkillError{msg: fmt.Sprintf("skill not allowed: %s", name)}
		}
	}
	return b.base.Get(ctx, name)
}

func skillAllowMap(names []string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			out[name] = struct{}{}
		}
	}
	return out
}

// --- Custom Extension Points for Eino Skill Middleware ---

// extractTriggerSkipValue 从 "TRIGGER" 或 "SKIP" 行中提取值。
// 匹配格式: "TRIGGER when: value", "TRIGGER: value", "SKIP when: value", "SKIP: value"
func extractTriggerSkipValue(line, keyword string) string {
	upper := strings.ToUpper(line)
	upperKeyword := strings.ToUpper(keyword)

	if !strings.HasPrefix(upper, upperKeyword) {
		return ""
	}

	// 去掉 "TRIGGER" 或 "SKIP" 关键字
	val := strings.TrimPrefix(line, keyword)
	val = strings.TrimPrefix(val, strings.ToUpper(keyword))

	// 去掉可选的 ":" 和 "when" 和空格
	for {
		old := val
		val = strings.TrimPrefix(val, ":")
		val = strings.TrimPrefix(val, " ")
		val = strings.TrimPrefix(val, "when")
		val = strings.TrimPrefix(val, "When")
		val = strings.TrimPrefix(val, "WHEN")
		val = strings.TrimSpace(val)
		if val == old {
			break
		}
	}

	return strings.TrimSpace(val)
}

// parseTriggerSkip 解析 description 中的 TRIGGER when: 和 SKIP when: 行。
// 返回清理后的描述、trigger 条件和 skip 条件。
func parseTriggerSkip(desc string) (cleanDesc, trigger, skip string) {
	if desc == "" {
		return "", "", ""
	}
	lines := strings.Split(desc, "\n")
	var cleanLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)

		// 匹配 TRIGGER when: 或 TRIGGER:
		if strings.HasPrefix(upper, "TRIGGER") {
			if val := extractTriggerSkipValue(trimmed, "TRIGGER"); val != "" {
				trigger = val
			}
			continue
		}

		// 匹配 SKIP when: 或 SKIP:
		if strings.HasPrefix(upper, "SKIP") {
			if val := extractTriggerSkipValue(trimmed, "SKIP"); val != "" {
				skip = val
			}
			continue
		}

		cleanLines = append(cleanLines, line)
	}

	cleanDesc = strings.TrimSpace(strings.Join(cleanLines, "\n"))
	return cleanDesc, trigger, skip
}

// customSkillToolDescription 使用 CustomToolDescription 扩展点渲染技能列表，
// 支持 TRIGGER/SKIP 条件。
func customSkillToolDescription(ctx context.Context, skills []einoskill.FrontMatter) string {
	if len(skills) == 0 {
		return `<available_skills>
(No skills available)
</available_skills>`
	}

	var b strings.Builder
	b.WriteString("<available_skills>\n")

	for _, skill := range skills {
		cleanDesc, trigger, skip := parseTriggerSkip(skill.Description)
		b.WriteString("<skill>\n")
		b.WriteString("<name>\n")
		b.WriteString(skill.Name)
		b.WriteString("\n</name>\n")
		b.WriteString("<description>\n")
		b.WriteString(strings.TrimSpace(cleanDesc))
		b.WriteString("\n</description>\n")
		if trigger != "" {
			b.WriteString("<trigger>\n")
			b.WriteString(trigger)
			b.WriteString("\n</trigger>\n")
		}
		if skip != "" {
			b.WriteString("<skip>\n")
			b.WriteString(skip)
			b.WriteString("\n</skip>\n")
		}
		b.WriteString("</skill>\n")
	}

	b.WriteString("</available_skills>\n")
	return b.String()
}

// customSkillSystemPrompt 使用 CustomSystemPrompt 扩展点提供防护栏指令。
func customSkillSystemPrompt(ctx context.Context, toolName string) string {
	return `# Skill 系统

**如何使用 Skill（技能）（渐进式展示）：**

Skill 遵循**渐进式展示**模式 - 你可以在上方看到 Skill 的名称和描述，但只在需要时才阅读完整说明：

1. **识别 Skill 适用场景**：检查用户的任务是否匹配某个 Skill 的 <trigger> 条件
2. **检查排除条件**：确认用户请求不匹配任何 Skill 的 <skip> 条件
3. **调用 Skill 工具**：使用 '` + toolName + `' 工具加载 Skill
4. **遵循 Skill 说明操作**：工具结果包含逐步工作流程、最佳实践和示例

**严格规则（必须遵守）：**

- **只调用列出的 Skill**：只使用 <available_skills> 中 <name> 标签内的精确 Skill 名称，不要猜测或发明 Skill 名称
- **精确匹配名称**：Skill 名称必须完全匹配，包括大小写和连字符
- **触发条件检查**：调用前，确认用户请求符合 Skill 的 <trigger> 条件
- **跳过条件检查**：如果请求符合 <skip> 条件，不要调用该 Skill
- **错误自我纠正**：如果工具返回 "Did you mean:" 建议，下一轮必须使用建议的 Skill 名称
- **严格执行**：Skill 返回的指令必须严格遵守，不得 paraphrase、跳过步骤或用自己的知识替代

**合规要求：**

收到 Skill 指令后，你必须：
- 严格按顺序执行每个步骤，不得跳过或重排序
- 不在未完成步骤时声称成功
- 任何步骤失败时，清晰报告失败而非猜测或继续

**执行 Skill 脚本：**
Skill 可能包含 Python 脚本或其他可执行文件。始终使用绝对路径。
`
}

// customBuildContent 使用 BuildContent 扩展点包装技能内容，添加合规强制标记。
func customBuildContent(ctx context.Context, skill einoskill.Skill, rawArgs string) (string, error) {
	// 基础格式与 Eino 默认一致
	result := fmt.Sprintf("正在启动 Skill：%s\n此 Skill 的目录：%s\n\n%s",
		skill.Name, skill.BaseDirectory, skill.Content)

	// 包装为强制执行格式
	wrapped := fmt.Sprintf(`<skill_instruction name="%s" enforcement="mandatory">
%s
</skill_instruction>

<compliance_directive>
你必须严格遵守上述 Skill 指令。这不是建议，而是必须执行的操作。

**禁止行为：**
- Paraphrase 或总结而不实际执行
- 跳过任何步骤或重新排序工作流程
- 用自己的知识替代技能的领域指导
- 在步骤失败或输出异常时声称成功
- 省略脚本执行而直接给出"应该是这样"的结果

**执行要求：**
- 按顺序执行每个步骤
- 使用技能提供的脚本和工具，不要自己编写替代方案
- 任何步骤失败时，清晰报告失败原因和阻塞点
- 只有在完成所有步骤后才能输出最终结论

如果无法完成某个步骤，明确报告障碍并停止，等待用户指示。
</compliance_directive>`, skill.Name, result)

	return wrapped, nil
}

func (uc *AgentUsecase) officialSkillBackend(ctx context.Context, allowlist []string) (einoskill.Backend, error) {
	fe, ok := uc.skillExecutor.(*FileSkillExecutor)
	if !ok {
		return nil, nil
	}
	dirs := fe.Dirs()
	backends := make([]einoskill.Backend, 0, len(dirs))
	fs := localSkillFilesystemBackend{}
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		backend, err := einoskill.NewBackendFromFilesystem(ctx, &einoskill.BackendFromFilesystemConfig{
			Backend: fs,
			BaseDir: dir,
		})
		if err != nil {
			return nil, err
		}
		backends = append(backends, backend)
	}
	if len(backends) == 0 {
		return nil, nil
	}
	var backend einoskill.Backend = multiSkillBackend{backends: backends}
	allow := skillAllowMap(allowlist)
	if len(allow) == 0 {
		allow = skillAllowMap(fe.ListAvailableSkillNames())
	}
	if len(allow) > 0 {
		backend = filteredSkillBackend{base: backend, allow: allow}
	}
	return backend, nil
}

func (uc *AgentUsecase) officialSkillMiddleware(ctx context.Context, allowlist []string) (adk.ChatModelAgentMiddleware, error) {
	backend, err := uc.officialSkillBackend(ctx, allowlist)
	if err != nil || backend == nil {
		return nil, err
	}
	return einoskill.NewMiddleware(ctx, &einoskill.Config{
		Backend:               backend,
		UseChinese:            true, // 保留用于 defaultToolParams 提示
		CustomSystemPrompt:     customSkillSystemPrompt,
		CustomToolDescription: customSkillToolDescription,
		BuildContent:           customBuildContent,
	})
}

func (uc *AgentUsecase) buildOfficialSkillTools(ctx context.Context, allowlist []string) ([]*HarnessTool, string, error) {
	middleware, err := uc.officialSkillMiddleware(ctx, allowlist)
	if err != nil || middleware == nil {
		return nil, "", err
	}
	runCtx := &adk.ChatModelAgentContext{}
	_, runCtx, err = middleware.BeforeAgent(ctx, runCtx)
	if err != nil {
		return nil, "", err
	}
	out := make([]*HarnessTool, 0, len(runCtx.Tools))
	for _, t := range runCtx.Tools {
		ht, err := harnessToolFromEinoTool(ctx, t)
		if err != nil {
			return nil, "", err
		}
		out = append(out, ht)
	}
	return out, strings.TrimSpace(runCtx.Instruction), nil
}

func harnessToolFromEinoTool(ctx context.Context, t tool.BaseTool) (*HarnessTool, error) {
	info, err := t.Info(ctx)
	if err != nil {
		return nil, err
	}
	if info == nil || strings.TrimSpace(info.Name) == "" {
		return nil, errors.New("tool info is empty")
	}
	invokable, ok := t.(tool.InvokableTool)
	if !ok {
		return nil, fmt.Errorf("tool %s is not invokable", info.Name)
	}
	return &HarnessTool{
		Info: info,
		Invoke: func(ctx context.Context, rawArgs string) (string, error) {
			// 解析 skill name 用于日志记录
			var skillName string
			if len(rawArgs) > 0 && rawArgs[0] == '{' {
				var args struct {
					Skill string `json:"skill"`
				}
				if err := json.Unmarshal([]byte(rawArgs), &args); err == nil {
					skillName = args.Skill
				}
			}
			log.Info("skill_invoke", "tool", info.Name, "skill", skillName, "args_len", len(rawArgs))
			result, err := invokable.InvokableRun(ctx, rawArgs)
			if err != nil {
				log.Info("skill_invoke_error", "tool", info.Name, "skill", skillName, "error", err)
			}
			return result, err
		},
	}, nil
}

func (uc *AgentUsecase) officialSkillInstruction(ctx context.Context, allowlist []string) string {
	_, instruction, err := uc.buildOfficialSkillTools(ctx, allowlist)
	if err != nil {
		return ""
	}
	return instruction
}

var _ filesystem.Backend = localSkillFilesystemBackend{}
var _ einoskill.Backend = multiSkillBackend{}
var _ einoskill.Backend = filteredSkillBackend{}
