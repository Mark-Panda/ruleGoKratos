package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&FeishuWebhookDsl{})
}

// FeishuWebhookDsl 飞书自定义机器人 Webhook（v2）：text / post / interactive / raw。
// 文档：https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
type FeishuWebhookDsl struct {
	Config                FeishuWebhookConfiguration
	urlTpl                el.Template
	textTpl               el.Template
	postTitleTpl          el.Template
	postBodyTpl           el.Template
	cardJsonTpl           el.Template
	rawJsonTpl            el.Template
	cardNoticeTitleTpl    el.Template
	cardNoticeMarkdownTpl el.Template
	postAtUserTpls        []el.Template
}

// FeishuWebhookConfiguration 与 flowgram DSL configuration 对齐。
type FeishuWebhookConfiguration struct {
	MsgType    string `json:"msgType"` // text | post | interactive | raw
	WebhookURL string `json:"webhookUrl"`
	Text       string `json:"text"`
	PostTitle  string `json:"postTitle"`
	PostBody   string `json:"postBody"`
	PostLang   string `json:"postLang"` // zh_cn | en_us | ja_jp
	// Post 友好选项：按行拆段、@所有人（前/后）、@指定成员（每 id 单独一行）
	PostSplitByLine    bool     `json:"postSplitByLine"`
	PostAtAllBefore    bool     `json:"postAtAllBefore"`
	PostAtAllAfter     bool     `json:"postAtAllAfter"`
	PostMentionUserIds []string `json:"postMentionUserIds"`
	CardJSON           string   `json:"cardJson"`
	RawJSON            string   `json:"rawJson"`
	// interactivePreset：card_json 手写卡片 JSON；notice_card 用标题+Markdown 组装常见通知卡
	InteractivePreset  string `json:"interactivePreset"`
	CardNoticeTitle    string `json:"cardNoticeTitle"`
	CardNoticeMarkdown string `json:"cardNoticeMarkdown"`
	TimeoutMs          int    `json:"timeoutMs"`
	ReplaceData        bool   `json:"replaceData"`
}

type feishuWebhookAPIResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (c *FeishuWebhookDsl) New() types.Node {
	return &FeishuWebhookDsl{Config: FeishuWebhookConfiguration{
		MsgType:            "text",
		PostLang:           "zh_cn",
		PostSplitByLine:    false,
		InteractivePreset:  "card_json",
		TimeoutMs:          15000,
		ReplaceData:        false,
		PostMentionUserIds: nil,
	}}
}

func (c *FeishuWebhookDsl) Type() string {
	return "x/feishuWebhook"
}

func (c *FeishuWebhookDsl) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "feishuWebhook",
		Desc:  "飞书机器人 Webhook：text / post（勾选分段与@）/ interactive（JSON 或通知卡）/ raw",
	}
}

func mustTemplate(s string) (el.Template, error) {
	return el.NewTemplate(s)
}

func (c *FeishuWebhookDsl) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}
	var err error
	if c.urlTpl, err = mustTemplate(c.Config.WebhookURL); err != nil {
		return err
	}
	if c.textTpl, err = mustTemplate(c.Config.Text); err != nil {
		return err
	}
	if c.postTitleTpl, err = mustTemplate(c.Config.PostTitle); err != nil {
		return err
	}
	if c.postBodyTpl, err = mustTemplate(c.Config.PostBody); err != nil {
		return err
	}
	if c.cardJsonTpl, err = mustTemplate(c.Config.CardJSON); err != nil {
		return err
	}
	if c.rawJsonTpl, err = mustTemplate(c.Config.RawJSON); err != nil {
		return err
	}
	if c.cardNoticeTitleTpl, err = mustTemplate(c.Config.CardNoticeTitle); err != nil {
		return err
	}
	if c.cardNoticeMarkdownTpl, err = mustTemplate(c.Config.CardNoticeMarkdown); err != nil {
		return err
	}
	c.postAtUserTpls = nil
	for _, raw := range c.Config.PostMentionUserIds {
		tpl, terr := mustTemplate(raw)
		if terr != nil {
			return terr
		}
		c.postAtUserTpls = append(c.postAtUserTpls, tpl)
	}
	return nil
}

func normalizeMsgType(s string) string {
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return "text"
	}
	return t
}

func normalizePostLang(s string) string {
	t := strings.TrimSpace(s)
	switch strings.ToLower(t) {
	case "zh_cn", "en_us", "ja_jp":
		return strings.ToLower(t)
	default:
		return "zh_cn"
	}
}

func atAllRow() []map[string]string {
	return []map[string]string{{"tag": "at", "user_id": "all"}}
}

func textRow(s string) []map[string]string {
	return []map[string]string{{"tag": "text", "text": s}}
}

func atUserRow(userID string) []map[string]string {
	return []map[string]string{{"tag": "at", "user_id": userID}}
}

func (c *FeishuWebhookDsl) buildPostPayload(evn map[string]interface{}) (map[string]interface{}, error) {
	langKey := normalizePostLang(c.Config.PostLang)
	title := strings.TrimSpace(c.postTitleTpl.ExecuteAsString(evn))
	bodyRendered := c.postBodyTpl.ExecuteAsString(evn)

	var rows [][]map[string]string
	if c.Config.PostAtAllBefore {
		rows = append(rows, atAllRow())
	}

	if c.Config.PostSplitByLine {
		for _, line := range strings.Split(bodyRendered, "\n") {
			line = strings.TrimRight(line, "\r")
			if strings.TrimSpace(line) == "" {
				continue
			}
			rows = append(rows, textRow(line))
		}
	} else {
		rows = append(rows, textRow(bodyRendered))
	}

	if c.Config.PostAtAllAfter {
		rows = append(rows, atAllRow())
	}

	for _, tpl := range c.postAtUserTpls {
		uid := strings.TrimSpace(tpl.ExecuteAsString(evn))
		if uid == "" {
			continue
		}
		rows = append(rows, atUserRow(uid))
	}

	if len(rows) == 0 {
		rows = [][]map[string]string{textRow("")}
	}

	inner := map[string]interface{}{
		"title":   title,
		"content": rows,
	}
	postRoot := map[string]interface{}{langKey: inner}
	return map[string]interface{}{
		"msg_type": "post",
		"content": map[string]interface{}{
			"post": postRoot,
		},
	}, nil
}

func buildNoticeFeishuCard(title, markdown string) map[string]interface{} {
	card := map[string]interface{}{
		"config": map[string]interface{}{"wide_screen_mode": true},
		"elements": []interface{}{
			map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"tag":     "lark_md",
					"content": markdown,
				},
			},
		},
	}
	if strings.TrimSpace(title) != "" {
		card["header"] = map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": title,
			},
		}
	}
	return card
}

func normalizeInteractivePreset(s string) string {
	t := strings.ToLower(strings.TrimSpace(s))
	switch t {
	case "notice_card":
		return "notice_card"
	default:
		return "card_json"
	}
}

func (c *FeishuWebhookDsl) buildPayload(evn map[string]interface{}) (map[string]interface{}, error) {
	mt := normalizeMsgType(c.Config.MsgType)
	switch mt {
	case "text":
		return map[string]interface{}{
			"msg_type": "text",
			"content": map[string]string{
				"text": c.textTpl.ExecuteAsString(evn),
			},
		}, nil
	case "post":
		return c.buildPostPayload(evn)
	case "interactive":
		switch normalizeInteractivePreset(c.Config.InteractivePreset) {
		case "notice_card":
			title := strings.TrimSpace(c.cardNoticeTitleTpl.ExecuteAsString(evn))
			md := strings.TrimSpace(c.cardNoticeMarkdownTpl.ExecuteAsString(evn))
			if title == "" && md == "" {
				return nil, errors.New("feishuWebhook: 通知卡片模式需填写「卡片标题」或「卡片 Markdown」至少一项")
			}
			if md == "" {
				md = " "
			}
			card := buildNoticeFeishuCard(title, md)
			return map[string]interface{}{
				"msg_type": "interactive",
				"card":     card,
			}, nil
		default:
			raw := strings.TrimSpace(c.cardJsonTpl.ExecuteAsString(evn))
			if raw == "" {
				return nil, errors.New("feishuWebhook: 自定义卡片模式需填写 cardJson（卡片 JSON）")
			}
			var card map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &card); err != nil {
				return nil, fmt.Errorf("feishuWebhook: cardJson 解析失败: %w", err)
			}
			return map[string]interface{}{
				"msg_type": "interactive",
				"card":     card,
			}, nil
		}
	case "raw":
		raw := strings.TrimSpace(c.rawJsonTpl.ExecuteAsString(evn))
		if raw == "" {
			return nil, errors.New("feishuWebhook: raw 模式需要填写 rawJson（完整请求体 JSON）")
		}
		var root map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &root); err != nil {
			return nil, fmt.Errorf("feishuWebhook: rawJson 须为 JSON 对象: %w", err)
		}
		return root, nil
	default:
		return nil, fmt.Errorf("feishuWebhook: 未知 msgType %q（支持 text/post/interactive/raw）", mt)
	}
}

func (c *FeishuWebhookDsl) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	evn := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	urlStr := strings.TrimSpace(c.urlTpl.ExecuteAsString(evn))
	if urlStr == "" {
		ctx.TellFailure(msg, errors.New("feishuWebhook: webhookUrl 为空"))
		return
	}
	lower := strings.ToLower(urlStr)
	if !strings.HasPrefix(lower, "https://") {
		ctx.TellFailure(msg, errors.New("feishuWebhook: 仅允许 https 的 Webhook 地址"))
		return
	}

	payload, err := c.buildPayload(evn)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}

	timeoutMs := c.Config.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 15000
	}
	reqCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, urlStr, bytes.NewReader(raw))
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("feishuWebhook: 请求失败: %w", err))
		return
	}
	defer resp.Body.Close()
	respBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		ctx.TellFailure(msg, readErr)
		return
	}
	respStr := string(respBytes)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		ctx.TellFailure(msg, fmt.Errorf("feishuWebhook: HTTP %d: %s", resp.StatusCode, respStr))
		return
	}

	var api feishuWebhookAPIResp
	if err := json.Unmarshal(respBytes, &api); err != nil {
		if c.Config.ReplaceData {
			msg.SetData(respStr)
		}
		ctx.TellSuccess(msg)
		return
	}
	if api.Code != 0 {
		ctx.TellFailure(msg, fmt.Errorf("feishuWebhook: code=%d msg=%s", api.Code, api.Msg))
		return
	}
	if c.Config.ReplaceData {
		msg.SetData(respStr)
	}
	ctx.TellSuccess(msg)
}

func (c *FeishuWebhookDsl) Destroy() {}
