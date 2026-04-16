package data

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/action"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&CursorAcpDsl{})
}

// CursorAcpDsl 启动官方文档中的 agent acp（stdio、按行 JSON-RPC）。
// stdinLines 每条渲染后作为一行写入子进程 stdin；stdout 在进程结束后汇总（适合短请求/探测）。
type CursorAcpDsl struct {
	Config        CursorAcpConfiguration
	pathTpl       el.Template
	argsTpl       []el.Template
	stdinLinesTpl []el.Template
	apiKeyTpl     el.Template // 非空则注入 --api-key（与 CLI 一致）
	workspaceTpl  el.Template // 非空则注入 --workspace（代码仓根 / 上下文）
	workTpl       el.Template
	hasVar        bool
}

// CursorAcpConfiguration 与 flowgram DSL 对齐；兼容历史 cursorPath。
type CursorAcpConfiguration struct {
	AgentPath  string `json:"agentPath"`
	CursorPath string `json:"cursorPath"`
	// Args 须以 acp 为首项；勿手写 --api-key/--workspace，请用 ApiKey、WorkspacePath。
	Args       []string `json:"args"`
	// StdinLines 每行一条 JSON-RPC；用户要说的内容在对应方法的 JSON 内（如 session/prompt 的 params），非独立配置键。
	StdinLines []string `json:"stdinLines"`
	// ApiKey 非空时在 argv 最前插入 --api-key；配置为空时使用环境变量 CURSOR_API_KEY（参见身份验证文档）。
	ApiKey string `json:"apiKey"`
	// WorkspacePath 非空时插入 --workspace，指定仓库根目录（与「在命令行界面中使用 Agent」文档一致）。
	WorkspacePath string `json:"workspacePath"`
	Log           bool   `json:"log"`
	ReplaceData   bool   `json:"replaceData"`
	// WorkDir 子进程 cwd；留空则用 metadata.workDir。
	WorkDir   string `json:"workDir"`
	TimeoutMs int    `json:"timeoutMs"`
}

func pickAcpAgentPath(c *CursorAcpConfiguration) string {
	if s := strings.TrimSpace(c.AgentPath); s != "" {
		return s
	}
	if s := strings.TrimSpace(c.CursorPath); s != "" {
		return s
	}
	return "agent"
}

func (c *CursorAcpDsl) New() types.Node {
	return &CursorAcpDsl{Config: CursorAcpConfiguration{
		AgentPath:    "agent",
		Args:         []string{"acp"},
		StdinLines:   nil,
		Log:          false,
		ReplaceData:  true,
		WorkDir:      "",
		TimeoutMs:    120000,
	}}
}

func (c *CursorAcpDsl) Type() string {
	return "x/cursorAcp"
}

func (c *CursorAcpDsl) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "cursorAcp",
		Desc:  "启动 agent acp（stdio JSON-RPC），stdinLines 为逐行 JSON-RPC",
	}
}

func (c *CursorAcpDsl) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}
	if len(c.Config.Args) == 0 {
		c.Config.Args = []string{"acp"}
	}
	if !strings.EqualFold(strings.TrimSpace(c.Config.Args[0]), "acp") {
		return errors.New("cursorAcp: args 首项必须为 acp")
	}
	pathStr := pickAcpAgentPath(&c.Config)
	pt, err := el.NewTemplate(pathStr)
	if err != nil {
		return err
	}
	c.pathTpl = pt
	c.hasVar = pt.HasVar()
	for _, a := range c.Config.Args {
		t, err := el.NewTemplate(a)
		if err != nil {
			return err
		}
		c.argsTpl = append(c.argsTpl, t)
		if t.HasVar() {
			c.hasVar = true
		}
	}
	for _, line := range c.Config.StdinLines {
		t, err := el.NewTemplate(line)
		if err != nil {
			return err
		}
		c.stdinLinesTpl = append(c.stdinLinesTpl, t)
		if t.HasVar() {
			c.hasVar = true
		}
	}
	akTpl, err := el.NewTemplate(c.Config.ApiKey)
	if err != nil {
		return err
	}
	c.apiKeyTpl = akTpl
	if akTpl.HasVar() {
		c.hasVar = true
	}
	wsPathTpl, err := el.NewTemplate(c.Config.WorkspacePath)
	if err != nil {
		return err
	}
	c.workspaceTpl = wsPathTpl
	if wsPathTpl.HasVar() {
		c.hasVar = true
	}
	wt, err := el.NewTemplate(c.Config.WorkDir)
	if err != nil {
		return err
	}
	c.workTpl = wt
	if wt.HasVar() {
		c.hasVar = true
	}
	return nil
}

func (c *CursorAcpDsl) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if c.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	if evn == nil {
		evn = map[string]interface{}{}
	}
	bin := strings.TrimSpace(c.pathTpl.ExecuteAsString(evn))
	if bin == "" || strings.Contains(bin, "..") || !allowCliExecutable(bin) {
		ctx.TellFailure(msg, errors.New("cursorAcp: 非法的可执行路径，仅允许 basename 为 agent 或 cursor"))
		return
	}
	userArgs := make([]string, 0, len(c.argsTpl))
	for _, t := range c.argsTpl {
		userArgs = append(userArgs, t.ExecuteAsString(evn))
	}
	if len(userArgs) == 0 || !strings.EqualFold(strings.TrimSpace(userArgs[0]), "acp") {
		ctx.TellFailure(msg, errors.New("cursorAcp: args 首项须为 acp"))
		return
	}
	args := buildAgentArgv(evn, c.apiKeyTpl, c.workspaceTpl, userArgs)
	stdinLines := make([]string, 0, len(c.stdinLinesTpl))
	for _, t := range c.stdinLinesTpl {
		stdinLines = append(stdinLines, t.ExecuteAsString(evn))
	}
	if len(stdinLines) == 0 {
		ctx.TellFailure(msg, errors.New("cursorAcp: stdinLines 不能为空（每行一条 JSON-RPC）"))
		return
	}
	workDir := strings.TrimSpace(c.workTpl.ExecuteAsString(evn))
	if workDir == "" {
		workDir = msg.Metadata.GetValue(action.KeyWorkDir)
	}

	timeout := c.Config.TimeoutMs
	if timeout <= 0 {
		timeout = 120000
	}
	runCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(runCtx, bin, args...)
	cmd.Dir = workDir

	inPipe, err := cmd.StdinPipe()
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}

	var stdoutBuf, stderrBuf strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if c.Config.Log {
			chainID := ""
			if ctx.RuleChain() != nil {
				chainID = ctx.RuleChain().GetNodeId().Id
			}
			msgCopy := msg.Copy()
			_, _ = io.Copy(
				io.MultiWriter(&stdoutBuf, &agentCliDebugWriter{ctx: ctx, msg: msgCopy, relationType: "info", chainID: chainID}),
				stdoutPipe,
			)
		} else if c.Config.ReplaceData {
			_, _ = io.Copy(&stdoutBuf, stdoutPipe)
		} else {
			_, _ = io.Copy(io.Discard, stdoutPipe)
		}
	}()
	go func() {
		defer wg.Done()
		if c.Config.Log {
			chainID := ""
			if ctx.RuleChain() != nil {
				chainID = ctx.RuleChain().GetNodeId().Id
			}
			msgCopy := msg.Copy()
			_, _ = io.Copy(
				io.MultiWriter(&stderrBuf, &agentCliDebugWriter{ctx: ctx, msg: msgCopy, relationType: "error", chainID: chainID}),
				stderrPipe,
			)
		} else {
			_, _ = io.Copy(&stderrBuf, stderrPipe)
		}
	}()

	go func() {
		defer inPipe.Close()
		for _, line := range stdinLines {
			if _, err := io.WriteString(inPipe, line+"\n"); err != nil {
				return
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	waitErr := cmd.Wait()
	wg.Wait()
	if waitErr != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			ctx.TellFailure(msg, fmt.Errorf("cursorAcp: 超时 (%d ms): %w", timeout, waitErr))
			return
		}
		ctx.TellFailure(msg, waitErr)
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

func (c *CursorAcpDsl) Destroy() {}
