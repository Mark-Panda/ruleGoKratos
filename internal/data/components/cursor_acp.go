package data

import (
	"bufio"
	"context"
	"encoding/json"
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
// 简易模式：按 https://cursor.com/cn/docs/cli/acp 自动完成 initialize → authenticate(cursor_login) → session/new → session/prompt，
// 身份由 ApiKey/--api-key 与环境变量 CURSOR_API_KEY 提供。
// 高级模式：stdinLines 每条渲染后一次性写入 stdin（兼容历史规则链）。
type CursorAcpDsl struct {
	Config        CursorAcpConfiguration
	pathTpl       el.Template
	argsTpl       []el.Template
	stdinLinesTpl []el.Template
	workspaceTpl  el.Template // 非空则注入 --workspace（代码仓根 / 上下文）
	workTpl       el.Template
	taskTpl       el.Template // 简易模式 session/prompt 文案
	hasVar        bool
}

// CursorAcpConfiguration 与 flowgram DSL 对齐；兼容历史 cursorPath。
type CursorAcpConfiguration struct {
	AgentPath  string `json:"agentPath"`
	CursorPath string `json:"cursorPath"`
	// Args 须以 acp 为首项；勿手写 --workspace，请用 WorkspacePath。
	Args       []string `json:"args"`
	// StdinLines 每行一条 JSON-RPC；用户要说的内容在对应方法的 JSON 内（如 session/prompt 的 params），非独立配置键。
	StdinLines []string `json:"stdinLines"`
	// WorkspacePath 非空时插入 --workspace，指定仓库根目录（与「在命令行界面中使用 Agent」文档一致）。
	WorkspacePath string `json:"workspacePath"`
	// Worktree 为 true 时注入 --worktree（无参数值），让 Agent 在新的 Git worktree 中运行而非直接编辑当前 checkout。
	Worktree bool `json:"worktree"`
	// Force 为 true 时注入 -f/--force（除非用户已在 Args 显式传入 force/yolo 相关参数）。
	Force bool `json:"force"`
	Log      bool `json:"log"`
	ReplaceData   bool   `json:"replaceData"`
	// WorkDir 子进程 cwd；留空则用 metadata.workDir。
	WorkDir   string `json:"workDir"`
	TimeoutMs int    `json:"timeoutMs"`
	// AcpSimpleMode 为 true 时使用交互式 JSON-RPC（忽略 StdinLines）。
	AcpSimpleMode bool `json:"acpSimpleMode"`
	// AcpTask 简易模式下传给 session/prompt 的自然语言任务，支持 ${msg.*}/${metadata.*}。
	AcpTask string `json:"acpTask"`
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
		Force:        true,
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
		Desc:  "agent acp：简易模式自动生成 ACP JSON-RPC；高级模式逐行 stdinLines",
	}
}

func (c *CursorAcpDsl) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}
	if _, ok := configuration["force"]; !ok {
		c.Config.Force = true
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
	if c.Config.AcpSimpleMode {
		taskStr := strings.TrimSpace(c.Config.AcpTask)
		taskTpl, err := el.NewTemplate(taskStr)
		if err != nil {
			return err
		}
		c.taskTpl = taskTpl
		if taskTpl.HasVar() {
			c.hasVar = true
		}
	} else {
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

// cursorAcpPreflightAgentLoggedIn 执行 agent status；未登录或 CLI 异常时 TellFailure 并返回 false。
func (c *CursorAcpDsl) cursorAcpPreflightAgentLoggedIn(ctx types.RuleContext, msg types.RuleMsg, bin string, evn map[string]interface{}, workDir string) bool {
	statusArgv := buildAgentArgv(evn, c.workspaceTpl, c.Config.Worktree, c.Config.Force, []string{"status"})
	sOut, sErr, stErr := runAgentStatusCheck(context.Background(), bin, statusArgv, workDir)
	combined := strings.TrimSpace(strings.TrimSpace(sOut) + "\n" + strings.TrimSpace(sErr))
	if stErr != nil {
		ctx.TellFailure(msg, fmt.Errorf("cursorAcp: agent status 预检失败（无法判断 Cursor CLI 是否已登录，常见于未安装 agent、PATH 错误或 CLI 报错）: %w; stdout=%q; stderr=%q",
			stErr, cursorAgentTruncateForErr(sOut, 2000), cursorAgentTruncateForErr(sErr, 2000)))
		return false
	}
	if !cursorAgentStatusLooksAuthed(combined) {
		ctx.TellFailure(msg, fmt.Errorf("cursorAcp: Cursor CLI 未登录或 agent status 未显示已登录。执行 agent acp 前请先执行 agent login（或配置 CURSOR_API_KEY 等）。当前 agent status 输出=%q",
			cursorAgentTruncateForErr(combined, 2000)))
		return false
	}
	return true
}

func (c *CursorAcpDsl) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if c.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	if evn == nil {
		evn = map[string]interface{}{}
	}
	if c.Config.AcpSimpleMode {
		c.onMsgSimpleMode(ctx, msg, evn)
		return
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
	args := buildAgentArgv(evn, c.workspaceTpl, c.Config.Worktree, c.Config.Force, userArgs)
	if ws := resolveWorkspacePath(c.workspaceTpl.ExecuteAsString(evn)); ws == "" {
		ctx.TellFailure(msg, errors.New("cursorAcp: workspacePath 未配置且无法解析 home 目录，请填写代码仓库根目录（--workspace）"))
		return
	}
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
	if !c.cursorAcpPreflightAgentLoggedIn(ctx, msg, bin, evn, workDir) {
		return
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
			ctx.TellFailure(msg, fmt.Errorf("cursorAcp: 超时 (%d ms): %w | stderr=%q", timeout, waitErr,
				cursorAgentTruncateForErr(stderrBuf.String(), 4000)))
			return
		}
		ctx.TellFailure(msg, cursorAgentWrapExecErr("cursorAcp", waitErr, stdoutBuf.String(), stderrBuf.String()))
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

func readAcpLine(br *bufio.Reader) (string, error) {
	line, err := br.ReadString('\n')
	return strings.TrimSpace(line), err
}

// readJSONRPCResult 读取 stdout 直至出现带 id 的响应（跳过无 id 的通知等行）。
func readJSONRPCResult(br *bufio.Reader, wantID int64) (json.RawMessage, error) {
	for {
		line, err := readAcpLine(br)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if line == "" {
			if err == io.EOF {
				return nil, errors.New("stdout 在未收到响应前结束")
			}
			continue
		}
		var env struct {
			ID     *float64        `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
			Err    json.RawMessage `json:"error"`
		}
		if jsonErr := json.Unmarshal([]byte(line), &env); jsonErr != nil {
			if err == io.EOF {
				return nil, errors.New("stdout 在未收到响应前结束")
			}
			continue
		}
		if env.Method != "" && env.ID == nil {
			continue
		}
		if env.ID != nil && int64(*env.ID) == wantID {
			if len(env.Err) > 0 {
				return nil, fmt.Errorf("rpc error: %s", strings.TrimSpace(string(env.Err)))
			}
			return env.Result, nil
		}
		if err == io.EOF {
			return nil, errors.New("stdout 在未收到响应前结束")
		}
	}
}

func (c *CursorAcpDsl) onMsgSimpleMode(ctx types.RuleContext, msg types.RuleMsg, evn map[string]interface{}) {
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
	args := buildAgentArgv(evn, c.workspaceTpl, c.Config.Worktree, c.Config.Force, userArgs)
	if ws := resolveWorkspacePath(c.workspaceTpl.ExecuteAsString(evn)); ws == "" {
		ctx.TellFailure(msg, errors.New("cursorAcp: workspacePath 未配置且无法解析 home 目录，请填写代码仓库根目录（--workspace）"))
		return
	}
	task := strings.TrimSpace(c.taskTpl.ExecuteAsString(evn))
	if task == "" {
		ctx.TellFailure(msg, errors.New("cursorAcp: 简易模式下请填写「任务说明」（或关闭简易模式自行配置 JSON-RPC 行）"))
		return
	}
	workDir := strings.TrimSpace(c.workTpl.ExecuteAsString(evn))
	if workDir == "" {
		workDir = msg.Metadata.GetValue(action.KeyWorkDir)
	}
	if !c.cursorAcpPreflightAgentLoggedIn(ctx, msg, bin, evn, workDir) {
		return
	}
	timeout := c.Config.TimeoutMs
	if timeout <= 0 {
		timeout = 120000
	}
	runCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(runCtx, bin, args...)
	cmd.Dir = workDir
	stdinPipe, err := cmd.StdinPipe()
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
	var stderrBuf strings.Builder
	var stderrWg sync.WaitGroup
	stderrWg.Add(1)
	go func() {
		defer stderrWg.Done()
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
	br := bufio.NewReaderSize(stdoutPipe, 1024*1024)
	writeRPC := func(id int64, method string, params interface{}) error {
		payload, err := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
			"params":  params,
		})
		if err != nil {
			return err
		}
		_, err = stdinPipe.Write(append(payload, '\n'))
		return err
	}
	initParams := map[string]interface{}{
		"protocolVersion": 1,
		"clientCapabilities": map[string]interface{}{
			"fs":       map[string]bool{"readTextFile": false, "writeTextFile": false},
			"terminal": false,
		},
		"clientInfo": map[string]string{"name": "rulego-cursorAcp", "version": "1.0"},
	}
	cwd := strings.TrimSpace(c.workspaceTpl.ExecuteAsString(evn))
	if cwd == "" {
		cwd = "."
	}
	if err := cmd.Start(); err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	if err := writeRPC(1, "initialize", initParams); err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	if _, err := readJSONRPCResult(br, 1); err != nil {
		ctx.TellFailure(msg, fmt.Errorf("cursorAcp: initialize: %w", err))
		return
	}
	if err := writeRPC(2, "authenticate", map[string]interface{}{"methodId": "cursor_login"}); err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	if _, err := readJSONRPCResult(br, 2); err != nil {
		ctx.TellFailure(msg, fmt.Errorf("cursorAcp: authenticate: %w", err))
		return
	}
	if err := writeRPC(3, "session/new", map[string]interface{}{
		"cwd":        cwd,
		"mcpServers": []interface{}{},
	}); err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	res3, err := readJSONRPCResult(br, 3)
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("cursorAcp: session/new: %w", err))
		return
	}
	var sn struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(res3, &sn); err != nil {
		ctx.TellFailure(msg, fmt.Errorf("cursorAcp: 解析 session/new 结果: %w", err))
		return
	}
	sid := strings.TrimSpace(sn.SessionID)
	if sid == "" {
		var loose map[string]interface{}
		if json.Unmarshal(res3, &loose) == nil {
			if v, ok := loose["sessionId"].(string); ok {
				sid = strings.TrimSpace(v)
			}
		}
	}
	if sid == "" {
		ctx.TellFailure(msg, errors.New("cursorAcp: session/new 未返回 sessionId"))
		return
	}
	promptParams := map[string]interface{}{
		"sessionId": sid,
		"prompt": []map[string]string{
			{"type": "text", "text": task},
		},
	}
	if err := writeRPC(4, "session/prompt", promptParams); err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	res4, err := readJSONRPCResult(br, 4)
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("cursorAcp: session/prompt: %w", err))
		return
	}
	_ = stdinPipe.Close()
	rest, _ := io.ReadAll(br)
	out := string(res4)
	if len(rest) > 0 {
		out += "\n" + string(rest)
	}
	waitErr := cmd.Wait()
	stderrWg.Wait()
	if waitErr != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			ctx.TellFailure(msg, fmt.Errorf("cursorAcp: 超时 (%d ms): %w | stderr=%q", timeout, waitErr,
				cursorAgentTruncateForErr(stderrBuf.String(), 4000)))
			return
		}
		ctx.TellFailure(msg, cursorAgentWrapExecErr("cursorAcp", waitErr, "", stderrBuf.String()))
		return
	}
	if c.Config.ReplaceData {
		if out != "" {
			msg.SetData(out)
		} else if stderrBuf.Len() > 0 {
			msg.SetData(stderrBuf.String())
		}
	}
	ctx.TellSuccess(msg)
}

func (c *CursorAcpDsl) Destroy() {}
