package data

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/action"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&CursorCliDsl{})
}

// CursorCliDsl 调用官方 Cursor CLI（默认可执行文件 agent），参数支持 ${msg.*}/${metadata.*} 模板。
// 与文档一致：安装验证为 agent --version；无头模式为 agent -p / --print 等。
type CursorCliDsl struct {
	Config        CursorCliConfiguration
	pathTpl       el.Template
	argsTpl       []el.Template
	promptTpl        el.Template // PrintMode 时接在 -p 后的任务文案
	outputFormatTpl  el.Template // PrintMode 时追加 --output-format（仅与 -p 同时生效）；空则视为 text
	modelTpl         el.Template // 非空则追加 --model <值>
	workspaceTpl  el.Template // 渲染后非空则注入 argv：--workspace <路径>（指定代码仓根，作为 Agent 上下文）
	workTpl       el.Template
	hasVar        bool
}

// CursorCliConfiguration 与 flowgram DSL configuration 对齐；兼容历史 cursorPath JSON 键。
type CursorCliConfiguration struct {
	// AgentPath 可执行文件路径（一般为 agent）；空则回退 CursorPath，再默认 agent。
	AgentPath  string `json:"agentPath"`
	CursorPath string `json:"cursorPath"` // 历史字段：仅当 AgentPath 为空时使用
	// Args 额外 argv 片段，拼在「直接配置」之后；与 printMode/prompt/model 可同时使用。
	Args []string `json:"args"`
	// PrintMode 为 true 时注入 -p（无头打印模式）；任务文案用 Prompt 字段配置（等价 agent -p "..." 中带引号的那段）。
	PrintMode bool `json:"printMode"`
	// Prompt 无头模式下的说明/任务文本；在 PrintMode 为 true 时紧接在 -p 之后传入。
	Prompt string `json:"prompt"`
	// Model 非空且不为 auto（忽略大小写）时在 argv 中追加 --model <Model>；与官方全局参数一致。
	Model string `json:"model"`
	// OutputFormat 在 PrintMode 时追加 --output-format（text / json / stream-json）；空则按 text 处理（与 CLI 文档默认一致）。
	OutputFormat string `json:"outputFormat"`
	// WorkspacePath 非空时插入 --workspace <路径>，指定「代码仓库根」供 CLI 解析规则与代码上下文（区别于 WorkDir 的进程 cwd）。
	WorkspacePath string `json:"workspacePath"`
	// Worktree 为 true 时注入 --worktree（无参数值），让 Agent 在新的 Git worktree 中运行而非直接编辑当前 checkout；
	// 配合 --workspace 使用可显式指定仓库根（见官方文档）。
	Worktree bool `json:"worktree"`
	Log      bool `json:"log"`
	ReplaceData   bool   `json:"replaceData"`
	// WorkDir 子进程操作系统工作目录（cmd.Dir）；留空则用 metadata.workDir。
	WorkDir   string `json:"workDir"`
	TimeoutMs int    `json:"timeoutMs"`
}

func pickCliExecutablePath(c *CursorCliConfiguration) string {
	if s := strings.TrimSpace(c.AgentPath); s != "" {
		return s
	}
	if s := strings.TrimSpace(c.CursorPath); s != "" {
		return s
	}
	return "agent"
}

func (c *CursorCliDsl) New() types.Node {
	return &CursorCliDsl{Config: CursorCliConfiguration{
		AgentPath:   "agent",
		Args:        nil,
		Log:         false,
		ReplaceData: true,
		WorkDir:     "",
		TimeoutMs:   0,
	}}
}

func (c *CursorCliDsl) Type() string {
	return "x/cursorCli"
}

func (c *CursorCliDsl) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "cursorCli",
		Desc:  "调用 Cursor CLI（agent，与官方文档一致；支持 -p/--print 等无头参数）",
	}
}

func (c *CursorCliDsl) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}
	pathStr := pickCliExecutablePath(&c.Config)
	pathTpl, err := el.NewTemplate(pathStr)
	if err != nil {
		return err
	}
	c.pathTpl = pathTpl
	c.hasVar = pathTpl.HasVar()
	var argsTpl []el.Template
	for _, a := range c.Config.Args {
		at, err := el.NewTemplate(a)
		if err != nil {
			return err
		}
		argsTpl = append(argsTpl, at)
		if at.HasVar() {
			c.hasVar = true
		}
	}
	c.argsTpl = argsTpl
	wsPathTpl, err := el.NewTemplate(c.Config.WorkspacePath)
	if err != nil {
		return err
	}
	c.workspaceTpl = wsPathTpl
	if wsPathTpl.HasVar() {
		c.hasVar = true
	}
	promptTpl, err := el.NewTemplate(c.Config.Prompt)
	if err != nil {
		return err
	}
	c.promptTpl = promptTpl
	if promptTpl.HasVar() {
		c.hasVar = true
	}
	modelTpl, err := el.NewTemplate(c.Config.Model)
	if err != nil {
		return err
	}
	c.modelTpl = modelTpl
	if modelTpl.HasVar() {
		c.hasVar = true
	}
	outFmtTpl, err := el.NewTemplate(c.Config.OutputFormat)
	if err != nil {
		return err
	}
	c.outputFormatTpl = outFmtTpl
	if outFmtTpl.HasVar() {
		c.hasVar = true
	}
	wdTpl, err := el.NewTemplate(c.Config.WorkDir)
	if err != nil {
		return err
	}
	c.workTpl = wdTpl
	if wdTpl.HasVar() {
		c.hasVar = true
	}
	return nil
}

// buildAgentArgv 在业务 Args 之前插入官方全局选项：--workspace、--worktree（见 Cursor CLI 参数文档）。
func buildAgentArgv(evn map[string]interface{}, workspaceTpl el.Template, worktree bool, userArgs []string) []string {
	out := make([]string, 0, len(userArgs)+4)
	if ws := strings.TrimSpace(workspaceTpl.ExecuteAsString(evn)); ws != "" {
		out = append(out, "--workspace", ws)
	}
	if worktree {
		out = append(out, "--worktree")
	}
	out = append(out, userArgs...)
	return out
}

// buildCliMidArgs 组装 -p、Prompt、--output-format、--model 及用户 Args（与官方无头参数顺序一致）。
func (c *CursorCliDsl) buildCliMidArgs(evn map[string]interface{}) []string {
	var mid []string
	if c.Config.PrintMode {
		mid = append(mid, "-p")
		if p := strings.TrimSpace(c.promptTpl.ExecuteAsString(evn)); p != "" {
			mid = append(mid, p)
		}
		of := strings.ToLower(strings.TrimSpace(c.outputFormatTpl.ExecuteAsString(evn)))
		if of == "" {
			of = "text"
		}
		switch of {
		case "text", "json", "stream-json":
		default:
			of = "text"
		}
		mid = append(mid, "--output-format", of)
	}
	if m := strings.TrimSpace(c.modelTpl.ExecuteAsString(evn)); m != "" && strings.ToLower(m) != "auto" {
		mid = append(mid, "--model", m)
	}
	for _, t := range c.argsTpl {
		mid = append(mid, t.ExecuteAsString(evn))
	}
	return mid
}

func allowCliExecutable(resolved string) bool {
	base := filepath.Base(resolved)
	base = strings.TrimSuffix(base, ".exe")
	switch strings.ToLower(base) {
	case "agent", "cursor":
		return true
	default:
		return false
	}
}

func (c *CursorCliDsl) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if c.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	if evn == nil {
		evn = map[string]interface{}{}
	}
	bin := strings.TrimSpace(c.pathTpl.ExecuteAsString(evn))
	if bin == "" || strings.Contains(bin, "..") || !allowCliExecutable(bin) {
		ctx.TellFailure(msg, errors.New("cursorCli: 非法的可执行路径，仅允许 basename 为 agent 或 cursor"))
		return
	}
	midArgs := c.buildCliMidArgs(evn)
	args := buildAgentArgv(evn, c.workspaceTpl, c.Config.Worktree, midArgs)
	workDir := strings.TrimSpace(c.workTpl.ExecuteAsString(evn))
	if workDir == "" {
		workDir = msg.Metadata.GetValue(action.KeyWorkDir)
	}

	runCtx := context.Background()
	if c.Config.PrintMode {
		statusArgv := buildAgentArgv(evn, c.workspaceTpl, c.Config.Worktree, []string{"status"})
		sOut, sErr, stErr := runAgentStatusCheck(runCtx, bin, statusArgv, workDir)
		combined := strings.TrimSpace(strings.TrimSpace(sOut) + "\n" + strings.TrimSpace(sErr))
		if stErr != nil {
			ctx.TellFailure(msg, fmt.Errorf("cursorCli: agent status 预检失败（无法判断 Cursor CLI 是否已登录，常见于未安装 agent、PATH 错误或 CLI 报错）: %w; stdout=%q; stderr=%q",
				stErr, cursorAgentTruncateForErr(sOut, 2000), cursorAgentTruncateForErr(sErr, 2000)))
			return
		}
		if !cursorAgentStatusLooksAuthed(combined) {
			ctx.TellFailure(msg, fmt.Errorf("cursorCli: Cursor CLI 未登录或 agent status 未显示已登录。无头 -p 任务需要先认证：请在运行环境执行 agent login（或配置 CURSOR_API_KEY 等官方支持的方式）。当前 agent status 输出=%q",
				cursorAgentTruncateForErr(combined, 2000)))
			return
		}
	}
	var cancel context.CancelFunc
	if c.Config.TimeoutMs > 0 {
		runCtx, cancel = context.WithTimeout(runCtx, time.Duration(c.Config.TimeoutMs)*time.Millisecond)
		defer cancel()
	}
	cmd := exec.CommandContext(runCtx, bin, args...)
	cmd.Dir = workDir

	var stdoutBuf, stderrBuf bytes.Buffer
	if c.Config.Log {
		chainID := ""
		if ctx.RuleChain() != nil {
			chainID = ctx.RuleChain().GetNodeId().Id
		}
		msgCopy := msg.Copy()
		cmd.Stdout = io.MultiWriter(&stdoutBuf, &agentCliDebugWriter{ctx: ctx, msg: msgCopy, relationType: "info", chainID: chainID})
		cmd.Stderr = io.MultiWriter(&stderrBuf, &agentCliDebugWriter{ctx: ctx, msg: msgCopy, relationType: "error", chainID: chainID})
	} else if c.Config.ReplaceData {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if err := cmd.Start(); err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	if err := cmd.Wait(); err != nil {
		ctx.TellFailure(msg, cursorAgentWrapExecErr("cursorCli", err, stdoutBuf.String(), stderrBuf.String()))
		return
	}
	if c.Config.ReplaceData {
		out := stdoutBuf.String()
		if out != "" {
			msg.SetData(out)
		} else {
			msg.SetData(stderrBuf.String())
		}
	}
	ctx.TellSuccess(msg)
}

func (c *CursorCliDsl) Destroy() {}

type agentCliDebugWriter struct {
	ctx          types.RuleContext
	msg          types.RuleMsg
	relationType string
	chainID      string
}

func (w *agentCliDebugWriter) Write(p []byte) (n int, err error) {
	w.msg.SetData(string(p))
	w.ctx.OnDebug(w.chainID, types.Log, w.ctx.GetSelfId(), w.msg, w.relationType, nil)
	return len(p), nil
}
