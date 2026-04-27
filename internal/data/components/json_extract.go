package data

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&JsonExtractComponent{})
}

type JsonExtractComponent struct {
	Config JsonExtractConfiguration

	sourceTpl        el.Template
	extractPatternTpl el.Template
	hasVar           bool
}

type JsonExtractConfiguration struct {
	Source        string `json:"source"`
	ExtractPattern string `json:"extractPattern"`
}

func (c *JsonExtractComponent) New() types.Node {
	return &JsonExtractComponent{}
}

func (c *JsonExtractComponent) Type() string {
	return "x/jsonExtract"
}

func (c *JsonExtractComponent) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "JSON提取与纠错",
		Desc:  "从文本中提取 JSON 并做格式纠错与补全",
	}
}

func (c *JsonExtractComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}

	tpls := map[*el.Template]string{
		&c.sourceTpl:         c.Config.Source,
		&c.extractPatternTpl: c.Config.ExtractPattern,
	}

	for tpl, s := range tpls {
		t, err := el.NewTemplate(s)
		if err != nil {
			return err
		}
		*tpl = t
		if t.HasVar() {
			c.hasVar = true
		}
	}

	return nil
}

func (c *JsonExtractComponent) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	evn := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	if evn == nil {
		evn = map[string]interface{}{}
	}

	// 优先使用 source 配置字段（支持模板变量），为空时回退到 msg.Data
	inputText := strings.TrimSpace(c.sourceTpl.ExecuteAsString(evn))
	if inputText == "" {
		inputText = strings.TrimSpace(msg.GetData())
	}

	if inputText == "" {
		ctx.TellFailure(msg, errors.New("输入文本不能为空"))
		return
	}

	mode := strings.TrimSpace(c.extractPatternTpl.ExecuteAsString(evn))
	if mode == "" {
		mode = "auto"
	}

	result := parseJsonWithFixes(inputText, mode)
	if result.Success {
		if result.ExtractedJson != nil {
			jsonBytes, _ := json.Marshal(result.ExtractedJson)
			msg.SetData(string(jsonBytes))
		} else {
			msg.SetData(result.Result)
		}
		ctx.TellSuccess(msg)
	} else {
		msg.SetData(result.Error)
		ctx.TellFailure(msg, errors.New(result.Error))
	}
}

func (c *JsonExtractComponent) Destroy() {}
func (c *JsonExtractComponent) Close() error {
	return nil
}

type Result struct {
	Success       bool
	Result        string
	ExtractedJson interface{}
	Error         string
}

func parseJsonWithFixes(text, mode string) Result {
	trimText := strings.TrimSpace(text)
	if trimText == "" {
		return Result{Success: false, Error: "输入文本不能为空"}
	}

	// 1. 尝试直接解析
	if isValidJSON(trimText) {
		var parsed interface{}
		if err := json.Unmarshal([]byte(trimText), &parsed); err == nil {
			return Result{Success: true, Result: formatJSON(parsed), ExtractedJson: parsed}
		}
	}

	// 2. 根据模式提取 JSON
	jsonStr := trimText
	if mode == "md" || mode == "auto" {
		extracted := extractJsonFromMarkdown(trimText)
		if extracted != "" {
			jsonStr = extracted
		}
	}

	// 3. 尝试修复格式后解析
	strategies := []func(string) string{
		extractJsonFromMarkdown,
		fixJsonFormat,
		completeJson,
	}

	for _, strategy := range strategies {
		fixed := strategy(jsonStr)
		if isValidJSON(fixed) {
			var parsed interface{}
			if err := json.Unmarshal([]byte(fixed), &parsed); err == nil {
				return Result{Success: true, Result: formatJSON(parsed), ExtractedJson: parsed}
			}
		}
	}

	// 4. 组合策略
	fixed := fixJsonFormat(completeJson(jsonStr))
	if isValidJSON(fixed) {
		var parsed interface{}
		if err := json.Unmarshal([]byte(fixed), &parsed); err == nil {
			return Result{Success: true, Result: formatJSON(parsed), ExtractedJson: parsed}
		}
	}

	return Result{Success: false, Error: "无法解析 JSON，请检查输入格式是否正确"}
}

func extractJsonFromMarkdown(text string) string {
	re := regexp.MustCompile("(?s)```(?:json)?\\s*([\\s\\S]*?)```")
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	re2 := regexp.MustCompile("(?s)```\\s*([\\s\\S]*?)```")
	matches2 := re2.FindStringSubmatch(text)
	if len(matches2) >= 2 {
		return strings.TrimSpace(matches2[1])
	}
	return text
}

func fixJsonFormat(jsonStr string) string {
	fixed := jsonStr
	fixed = regexp.MustCompile("//.*$").ReplaceAllString(fixed, "")
	fixed = strings.ReplaceAll(fixed, "'", "\"")
	fixed = regexp.MustCompile(",(\\s*[}\\]])").ReplaceAllString(fixed, "$1")
	fixed = regexp.MustCompile(`([{,\s]+)([a-zA-Z_][a-zA-Z0-9_]*)\s*:`).ReplaceAllString(fixed, `$1"$2":`)
	fixed = strings.ReplaceAll(fixed, "undefined", "null")
	return fixed
}

func completeJson(jsonStr string) string {
	fixed := strings.TrimSpace(jsonStr)
	if strings.HasPrefix(fixed, "[") && !strings.HasSuffix(fixed, "]") {
		depth := 0
		for i, ch := range fixed {
			if ch == '[' {
				depth++
			} else if ch == ']' {
				depth--
				if depth == 0 {
					return fixed[:i+1]
				}
			}
		}
	}
	if strings.HasPrefix(fixed, "{") && !strings.HasSuffix(fixed, "}") {
		lastComplete := strings.LastIndex(fixed, ",")
		if lastComplete != -1 {
			fixed = fixed[:lastComplete] + "}"
		}
	}
	return fixed
}

func isValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

func formatJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
