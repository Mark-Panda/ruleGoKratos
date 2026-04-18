package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/maps"
	"github.com/rulego/rulego/utils/str"
	"github.com/sashabaranov/go-openai"
)

var reThinkStrip = regexp.MustCompile(`<think>[\s\S]*?</think>`)

// ManagedTextGenerateNode 替代 rulego-components-ai 的 ai/llm：凭证仅从模型管理加载。
type ManagedTextGenerateNode struct {
	Config managedOpenAIConfiguration

	systemPromptTemplate str.Template
	chatMessageTemplates []managedChatMsgTpl
	imagesTemplates      []str.Template
	hasVar               bool
	responseFormat       openai.ChatCompletionResponseFormatType
}

type managedOpenAIConfiguration struct {
	LlmConfigID      int64 `json:"llmConfigId"`
	LlmModelEntryID int64 `json:"llmModelEntryId"`

	Url          string              `json:"url"`
	Key          string              `json:"key"`
	Model        string              `json:"model"`
	SystemPrompt string              `json:"systemPrompt"`
	Messages     []managedChatMsg    `json:"messages"`
	Images       []string            `json:"images"`
	Params       managedOpenAIParams `json:"params"`
}

type managedOpenAIParams struct {
	Temperature      float32  `json:"temperature"`
	TopP             float32  `json:"topP"`
	PresencePenalty  float32  `json:"presencePenalty"`
	FrequencyPenalty float32  `json:"frequencyPenalty"`
	MaxTokens        int      `json:"maxTokens"`
	Stop             []string `json:"stop"`
	ResponseFormat   string   `json:"responseFormat"`
	JsonSchema       string   `json:"jsonSchema"`
	KeepThink        bool     `json:"keepThink"`
}

type managedChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type managedChatMsgTpl struct {
	Role            string
	ContentTemplate str.Template
}

func (x *ManagedTextGenerateNode) Type() string {
	return "ai/llm"
}

func (x *ManagedTextGenerateNode) New() types.Node {
	return &ManagedTextGenerateNode{
		Config: managedOpenAIConfiguration{
			Params: managedOpenAIParams{
				Temperature: 0.6,
				TopP:        0.75,
			},
		},
	}
}

func (x *ManagedTextGenerateNode) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "ai/llm",
		Desc:  "Text generation（凭证来自模型管理）",
	}
}

func (x *ManagedTextGenerateNode) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &x.Config); err != nil {
		return err
	}
	if x.Config.LlmConfigID <= 0 || x.Config.LlmModelEntryID <= 0 {
		return fmt.Errorf("请选择模型管理中的 LLM 配置与模型（llmConfigId / llmModelEntryId）")
	}

	x.systemPromptTemplate = str.NewTemplate(x.Config.SystemPrompt)
	if !x.systemPromptTemplate.IsNotVar() {
		x.hasVar = true
	}
	for _, item := range x.Config.Messages {
		tmpl := str.NewTemplate(item.Content)
		if !tmpl.IsNotVar() {
			x.hasVar = true
		}
		role := strings.TrimSpace(item.Role)
		if role == "" {
			role = openai.ChatMessageRoleUser
		}
		x.chatMessageTemplates = append(x.chatMessageTemplates, managedChatMsgTpl{
			Role:            role,
			ContentTemplate: tmpl,
		})
	}
	for _, item := range x.Config.Images {
		tmpl := str.NewTemplate(item)
		if !tmpl.IsNotVar() {
			x.hasVar = true
		}
		x.imagesTemplates = append(x.imagesTemplates, tmpl)
	}

	x.Config.Params.ResponseFormat = strings.TrimSpace(x.Config.Params.ResponseFormat)
	if x.Config.Params.ResponseFormat == "" {
		x.Config.Params.ResponseFormat = string(openai.ChatCompletionResponseFormatTypeText)
	}
	x.responseFormat = openai.ChatCompletionResponseFormatType(x.Config.Params.ResponseFormat)
	if x.responseFormat != openai.ChatCompletionResponseFormatTypeText &&
		x.responseFormat != openai.ChatCompletionResponseFormatTypeJSONObject &&
		x.responseFormat != openai.ChatCompletionResponseFormatTypeJSONSchema {
		x.Config.Params.ResponseFormat = string(openai.ChatCompletionResponseFormatTypeText)
		x.responseFormat = openai.ChatCompletionResponseFormatTypeText
	}
	return nil
}

func defaultOpenAIv1Base(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return "https://api.openai.com/v1"
	}
	return u
}

func (x *ManagedTextGenerateNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	if ruleGoAgentUsecase == nil {
		ctx.TellFailure(msg, errors.New("Agent/LLM 服务未注入"))
		return
	}
	modelName, apiKey, baseURL, err := ruleGoAgentUsecase.ResolveManagedLLM(
		ctx.GetContext(), x.Config.LlmConfigID, x.Config.LlmModelEntryID)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = defaultOpenAIv1Base(baseURL)
	client := openai.NewClientWithConfig(cfg)

	var evn map[string]interface{}
	if x.hasVar {
		evn = base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	}
	systemPrompt := x.systemPromptTemplate.Execute(evn)

	content, err := x.sendCompletion(ctx, client, evn, modelName, systemPrompt)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	if x.responseFormat == openai.ChatCompletionResponseFormatTypeJSONObject ||
		x.responseFormat == openai.ChatCompletionResponseFormatTypeJSONSchema {
		msg.DataType = types.JSON
	} else {
		msg.DataType = types.TEXT
	}
	msg.SetData(content)
	ctx.TellSuccess(msg)
}

func (x *ManagedTextGenerateNode) sendCompletion(
	ctx types.RuleContext,
	client *openai.Client,
	evn map[string]interface{},
	resolvedModel string,
	systemPrompt string,
) (string, error) {
	var messages []openai.ChatCompletionMessage
	if systemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}
	messageLen := len(x.chatMessageTemplates)
	imageLen := len(x.imagesTemplates)
	for index, item := range x.chatMessageTemplates {
		content := item.ContentTemplate.Execute(evn)
		if index == (messageLen-1) && imageLen > 0 {
			var multiContent []openai.ChatMessagePart
			for _, imageItemTpl := range x.imagesTemplates {
				imageURL := imageItemTpl.Execute(evn)
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						URL:    imageURL,
						Detail: openai.ImageURLDetailAuto,
					},
				})
			}
			multiContent = append(multiContent, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: content,
			})
			messages = append(messages, openai.ChatCompletionMessage{
				Role:         item.Role,
				MultiContent: multiContent,
			})
		} else {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    item.Role,
				Content: content,
			})
		}
	}

	var responseFormatJSONSchema *openai.ChatCompletionResponseFormatJSONSchema
	if x.Config.Params.JsonSchema != "" {
		var schemaRawMessage = json.RawMessage{}
		if err := json.Unmarshal([]byte(x.Config.Params.JsonSchema), &schemaRawMessage); err != nil {
			return "", err
		}
		responseFormatJSONSchema = &openai.ChatCompletionResponseFormatJSONSchema{
			Schema: schemaRawMessage,
		}
	}

	resp, err := client.CreateChatCompletion(
		ctx.GetContext(),
		openai.ChatCompletionRequest{
			Model:            resolvedModel,
			Messages:         messages,
			Temperature:      x.Config.Params.Temperature,
			TopP:             x.Config.Params.TopP,
			PresencePenalty:  x.Config.Params.PresencePenalty,
			FrequencyPenalty: x.Config.Params.FrequencyPenalty,
			MaxTokens:        x.Config.Params.MaxTokens,
			Stop:             x.Config.Params.Stop,
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type:       openai.ChatCompletionResponseFormatType(x.Config.Params.ResponseFormat),
				JSONSchema: responseFormatJSONSchema,
			},
		},
	)
	if err != nil {
		return "", err
	}
	var combined string
	for _, choice := range resp.Choices {
		combined += choice.Message.Content
	}

	if x.responseFormat == openai.ChatCompletionResponseFormatTypeText {
		if !x.Config.Params.KeepThink {
			combined = strings.TrimLeft(reThinkStrip.ReplaceAllString(combined, ""), "\n")
		}
	} else {
		combined = strings.TrimLeft(reThinkStrip.ReplaceAllString(combined, ""), "\n")
		combined = strings.TrimPrefix(combined, "```json\n")
		combined = strings.TrimSuffix(combined, "\n```")
	}
	return combined, nil
}

func (x *ManagedTextGenerateNode) Destroy() {}
