package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/action"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&CursorCliAuthDsl{})
}

// CursorCliAuthDsl 检测 Cursor CLI（agent）是否已登录。
type CursorCliAuthDsl struct {
	Config       CursorCliAuthConfiguration
	pathTpl      el.Template
	workspaceTpl el.Template
	workTpl      el.Template
	hasVar       bool
}

type CursorCliAuthConfiguration struct {
	AgentPath     string `json:"agentPath"`
	CursorPath    string `json:"cursorPath"`
	WorkspacePath string `json:"workspacePath"`
	Worktree      bool   `json:"worktree"`
	Force         bool   `json:"force"`
	WorkDir       string `json:"workDir"`
	TimeoutMs     int    `json:"timeoutMs"`
	ReplaceData   bool   `json:"replaceData"`
}

func (c *CursorCliAuthDsl) New() types.Node {
	return &CursorCliAuthDsl{Config: CursorCliAuthConfiguration{
		AgentPath:     "agent",
		WorkspacePath: "$HOME",
		Force:         true,
		TimeoutMs:     15000,
		ReplaceData:   true,
	}}
}

func (c *CursorCliAuthDsl) Type() string {
	return "x/cursorCliAuth"
}

func (c *CursorCliAuthDsl) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "cursorCliAuth",
		Desc:  "检查 Cursor CLI（agent status）是否已登录；已登录走 Success，未登录走 Failure",
	}
}

func (c *CursorCliAuthDsl) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}
	if _, ok := configuration["force"]; !ok {
		c.Config.Force = true
	}
	if _, ok := configuration["replaceData"]; !ok {
		c.Config.ReplaceData = true
	}
	pathTpl, err := el.NewTemplate(pickCliExecutablePath(&CursorCliConfiguration{
		AgentPath:  c.Config.AgentPath,
		CursorPath: c.Config.CursorPath,
	}))
	if err != nil {
		return err
	}
	c.pathTpl = pathTpl
	c.hasVar = pathTpl.HasVar()
	wsTpl, err := el.NewTemplate(c.Config.WorkspacePath)
	if err != nil {
		return err
	}
	c.workspaceTpl = wsTpl
	if wsTpl.HasVar() {
		c.hasVar = true
	}
	workTpl, err := el.NewTemplate(c.Config.WorkDir)
	if err != nil {
		return err
	}
	c.workTpl = workTpl
	if workTpl.HasVar() {
		c.hasVar = true
	}
	return nil
}

func (c *CursorCliAuthDsl) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if c.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	if evn == nil {
		evn = map[string]interface{}{}
	}
	bin := strings.TrimSpace(c.pathTpl.ExecuteAsString(evn))
	if bin == "" || strings.Contains(bin, "..") || !allowCliExecutable(bin) {
		ctx.TellFailure(msg, errors.New("cursorCliAuth: 非法的可执行路径，仅允许 basename 为 agent 或 cursor"))
		return
	}
	if ws := resolveWorkspacePath(c.workspaceTpl.ExecuteAsString(evn)); ws == "" {
		ctx.TellFailure(msg, errors.New("cursorCliAuth: workspacePath 未配置且无法解析 home 目录，请填写代码仓库根目录（--workspace）"))
		return
	}
	workDir := strings.TrimSpace(c.workTpl.ExecuteAsString(evn))
	if workDir == "" {
		workDir = msg.Metadata.GetValue(action.KeyWorkDir)
	}
	args := buildAgentArgv(evn, c.workspaceTpl, c.Config.Worktree, c.Config.Force, []string{"status"})
	stdout, stderr, err := runCommandWithTimeout(bin, args, workDir, c.Config.TimeoutMs)
	combined := strings.TrimSpace(strings.TrimSpace(stdout) + "\n" + strings.TrimSpace(stderr))
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("cursorCliAuth: 执行 agent status 失败: %w; stdout=%q; stderr=%q",
			err, cursorAgentTruncateForErr(stdout, 2000), cursorAgentTruncateForErr(stderr, 2000)))
		return
	}
	if !cursorAgentStatusLooksAuthed(combined) {
		ctx.TellFailure(msg, fmt.Errorf("cursorCliAuth: Cursor CLI 未登录。请先执行 agent login（或配置 CURSOR_API_KEY）。status 输出=%q",
			cursorAgentTruncateForErr(combined, 2000)))
		return
	}
	if c.Config.ReplaceData {
		payload := map[string]interface{}{
			"authed":       true,
			"statusOutput": combined,
			"stdout":       stdout,
			"stderr":       stderr,
		}
		if raw, marshalErr := json.Marshal(payload); marshalErr == nil {
			msg.SetData(string(raw))
		} else {
			msg.SetData(combined)
		}
	}
	ctx.TellSuccess(msg)
}

func (c *CursorCliAuthDsl) Destroy() {}
