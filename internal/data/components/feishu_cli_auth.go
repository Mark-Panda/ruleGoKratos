package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/action"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&FeishuCliAuthDsl{})
}

// FeishuCliAuthDsl 检测飞书 CLI（lark-cli auth status）是否已授权。
type FeishuCliAuthDsl struct {
	Config  FeishuCliAuthConfiguration
	pathTpl el.Template
	argsTpl []el.Template
	workTpl el.Template
	hasVar  bool
}

type FeishuCliAuthConfiguration struct {
	CliPath     string   `json:"cliPath"`
	LarkCliPath string   `json:"larkCliPath"`
	Args        []string `json:"args"`
	WorkDir     string   `json:"workDir"`
	TimeoutMs   int      `json:"timeoutMs"`
	ReplaceData bool     `json:"replaceData"`
}

func pickFeishuCliPath(c *FeishuCliAuthConfiguration) string {
	if s := strings.TrimSpace(c.CliPath); s != "" {
		return s
	}
	if s := strings.TrimSpace(c.LarkCliPath); s != "" {
		return s
	}
	return "lark-cli"
}

func allowFeishuCliExecutable(path string) bool {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(path), ".exe"))
	return base == "lark-cli" || base == "lark"
}

func (c *FeishuCliAuthDsl) New() types.Node {
	return &FeishuCliAuthDsl{Config: FeishuCliAuthConfiguration{
		CliPath:     "lark-cli",
		Args:        []string{"auth", "status"},
		TimeoutMs:   15000,
		ReplaceData: true,
	}}
}

func (c *FeishuCliAuthDsl) Type() string {
	return "x/feishuCliAuth"
}

func (c *FeishuCliAuthDsl) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "feishuCliAuth",
		Desc:  "检查飞书 CLI（lark-cli auth status）是否已授权；已授权走 Success，未授权走 Failure",
	}
}

func (c *FeishuCliAuthDsl) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}
	if len(c.Config.Args) == 0 {
		c.Config.Args = []string{"auth", "status"}
	}
	if _, ok := configuration["replaceData"]; !ok {
		c.Config.ReplaceData = true
	}
	pathTpl, err := el.NewTemplate(pickFeishuCliPath(&c.Config))
	if err != nil {
		return err
	}
	c.pathTpl = pathTpl
	c.hasVar = pathTpl.HasVar()
	c.argsTpl = nil
	for _, raw := range c.Config.Args {
		tpl, tErr := el.NewTemplate(raw)
		if tErr != nil {
			return tErr
		}
		c.argsTpl = append(c.argsTpl, tpl)
		if tpl.HasVar() {
			c.hasVar = true
		}
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

func (c *FeishuCliAuthDsl) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	var evn map[string]interface{}
	if c.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	if evn == nil {
		evn = map[string]interface{}{}
	}
	bin := strings.TrimSpace(c.pathTpl.ExecuteAsString(evn))
	if bin == "" || strings.Contains(bin, "..") || !allowFeishuCliExecutable(bin) {
		ctx.TellFailure(msg, errors.New("feishuCliAuth: 非法的可执行路径，仅允许 basename 为 lark-cli 或 lark"))
		return
	}
	args := make([]string, 0, len(c.argsTpl))
	for _, tpl := range c.argsTpl {
		args = append(args, tpl.ExecuteAsString(evn))
	}
	if len(args) == 0 {
		args = []string{"auth", "status"}
	}
	workDir := strings.TrimSpace(c.workTpl.ExecuteAsString(evn))
	if workDir == "" {
		workDir = msg.Metadata.GetValue(action.KeyWorkDir)
	}
	stdout, stderr, err := runCommandWithTimeout(bin, args, workDir, c.Config.TimeoutMs)
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("feishuCliAuth: 执行 status 失败: %w; stdout=%q; stderr=%q",
			err, cursorAgentTruncateForErr(stdout, 2000), cursorAgentTruncateForErr(stderr, 2000)))
		return
	}
	authed, parsed, parseErr := feishuCliStatusLooksAuthed(stdout, stderr)
	if parseErr != nil {
		ctx.TellFailure(msg, fmt.Errorf("feishuCliAuth: 无法解析 lark-cli auth status 输出: %w; stdout=%q; stderr=%q",
			parseErr, cursorAgentTruncateForErr(stdout, 2000), cursorAgentTruncateForErr(stderr, 2000)))
		return
	}
	if !authed {
		tokenStatus := ""
		if parsed != nil {
			tokenStatus = strings.TrimSpace(parsed.TokenStatus)
		}
		ctx.TellFailure(msg, fmt.Errorf("feishuCliAuth: 飞书 CLI 未授权（tokenStatus=%q），请先执行 lark-cli auth login --recommend",
			tokenStatus))
		return
	}
	if c.Config.ReplaceData {
		payload := map[string]interface{}{
			"authed": true,
			"stdout": stdout,
			"stderr": stderr,
		}
		if parsed != nil {
			payload["tokenStatus"] = parsed.TokenStatus
			payload["userName"] = parsed.UserName
			payload["identity"] = parsed.Identity
			payload["grantedAt"] = parsed.GrantedAt
			payload["appId"] = parsed.AppID
			payload["brand"] = parsed.Brand
		}
		if raw, marshalErr := json.Marshal(payload); marshalErr == nil {
			msg.SetData(string(raw))
		} else {
			msg.SetData(stdout)
		}
	}
	ctx.TellSuccess(msg)
}

func (c *FeishuCliAuthDsl) Destroy() {}
