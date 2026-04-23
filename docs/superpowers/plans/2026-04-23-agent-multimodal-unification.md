# Agent Multimodal Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 统一 `ChatService` 与 `ai/agentHarness` 节点的附件输入能力，让 Agent 对图片、视频、音频走 Eino 原生多模态，并为通用文件补齐一致的输入/降级行为。

**Architecture:** 复用现有 `HarnessAttachment -> schema.MessageInputPart` 组装链路，不新增第二套协议。`ChatService` 保持现状，`agentHarness` 节点新增从 RuleGo `msg` 数据/元数据中解析附件并注入 `HarnessRequest.Attachments`，同时在多模态组装层补通用文件 part 的能力位和当前 OpenAI 适配器下的安全降级。

**Tech Stack:** Go, Kratos, RuleGo, CloudWeGo Eino, `eino-ext/components/model/openai`, Go tests

---

### Task 1: 固化多模态输入行为

**Files:**
- Modify: `internal/biz/harness_multimodal_test.go`
- Modify: `internal/biz/agent_test.go`
- Test: `internal/biz/harness_multimodal_test.go`
- Test: `internal/biz/agent_test.go`

- [ ] **Step 1: 写失败测试，描述通用文件与节点附件的期望行为**

```go
func TestBuildHarnessInputParts_shouldKeepGenericFileAsFilePart(t *testing.T) {
	parts := buildHarnessInputParts("请阅读附件", []HarnessAttachment{{
		Filename:      "spec.pdf",
		MimeType:      "application/pdf",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("%PDF-1.7 sample")),
	}})

	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(parts))
	}
	if parts[1].Type != schema.ChatMessagePartTypeFileURL || parts[1].File == nil {
		t.Fatalf("expected file part, got %#v", parts[1])
	}
	if parts[1].File.Name != "spec.pdf" {
		t.Fatalf("unexpected file name: %#v", parts[1].File)
	}
}

func TestComposeMessages_shouldEmbedAttachmentsIntoLastUserMessage(t *testing.T) {
	uc := newTestAgentUsecase()
	msgs := uc.composeMessages(&HarnessRequest{}, "", nil, "看这个文件", []HarnessAttachment{{
		Filename:      "spec.pdf",
		MimeType:      "application/pdf",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("pdf")),
	}})

	last := msgs[len(msgs)-1]
	if len(last.UserInputMultiContent) == 0 {
		t.Fatalf("expected multimodal user message, got %#v", last)
	}
}
```

- [ ] **Step 2: 运行测试，确认它们先失败**

Run: `go test ./internal/biz -run 'TestBuildHarnessInputParts_shouldKeepGenericFileAsFilePart|TestComposeMessages_shouldEmbedAttachmentsIntoLastUserMessage'`

Expected: FAIL，报出当前通用文件未生成 `file_url` part，或最后一条用户消息未包含预期附件 part。

- [ ] **Step 3: 以最小改动补通用文件 part**

```go
case strings.HasPrefix(effective, "image/"):
	// existing image branch
case strings.HasPrefix(effective, "video/"):
	// existing video branch
case strings.HasPrefix(effective, "audio/"):
	// existing audio branch
default:
	fm := effective
	if fm == "" {
		fm = mime
	}
	parts = append(parts, schema.MessageInputPart{
		Type: schema.ChatMessagePartTypeFileURL,
		File: &schema.MessageInputFile{
			MessagePartCommon: schema.MessagePartCommon{
				Base64Data: &data,
				MIMEType:   fm,
			},
			Name: fn,
		},
	})
	if txt != "" {
		parts = append(parts, schema.MessageInputPart{
			Type: schema.ChatMessagePartTypeText,
			Text: legacyAttachmentBlock(fn, mime, txt, ""),
		})
	}
```

- [ ] **Step 4: 重新运行测试，确认通过**

Run: `go test ./internal/biz -run 'TestBuildHarnessInputParts_shouldKeepGenericFileAsFilePart|TestComposeMessages_shouldEmbedAttachmentsIntoLastUserMessage'`

Expected: PASS

### Task 2: 给 Agent Harness 节点接入附件提取

**Files:**
- Create: `internal/data/components/agent_harness_llm_test.go`
- Modify: `internal/data/components/agent_harness_llm.go`
- Test: `internal/data/components/agent_harness_llm_test.go`

- [ ] **Step 1: 写失败测试，定义节点如何从 RuleGo 消息读取附件**

```go
func TestExtractHarnessAttachments_shouldReadArrayFromMessageData(t *testing.T) {
	raw := map[string]any{
		"attachments": []any{
			map[string]any{
				"filename":      "demo.png",
				"mimeType":      "image/png",
				"contentBase64": "YWJj",
			},
		},
	}

	atts := extractHarnessAttachments(raw, nil)
	if len(atts) != 1 {
		t.Fatalf("got %d attachments", len(atts))
	}
	if atts[0].Filename != "demo.png" || atts[0].MimeType != "image/png" {
		t.Fatalf("unexpected attachment: %#v", atts[0])
	}
}

func TestExtractHarnessAttachments_shouldFallbackToMetadata(t *testing.T) {
	meta := map[string]any{
		"attachments": []map[string]any{{
			"filename": "clip.mp4",
			"mimeType": "video/mp4",
		}},
	}

	atts := extractHarnessAttachments(nil, meta)
	if len(atts) != 1 || atts[0].Filename != "clip.mp4" {
		t.Fatalf("unexpected attachments: %#v", atts)
	}
}
```

- [ ] **Step 2: 运行测试，确认先失败**

Run: `go test ./internal/data/components -run 'TestExtractHarnessAttachments_'`

Expected: FAIL，提示 `extractHarnessAttachments` 尚不存在或不返回预期附件。

- [ ] **Step 3: 最小实现附件提取与请求注入**

```go
func extractHarnessAttachments(msgData any, metadata map[string]any) []biz.HarnessAttachment {
	// 依次尝试：
	// 1. map 数据体中的 attachments
	// 2. metadata 中的 attachments
	// 3. 兼容 []map[string]any / []any / JSON string
}

req := biz.HarnessRequest{
	Model:           strings.TrimSpace(modelName),
	History:         nil,
	Input:           userPrompt,
	Attachments:     extractHarnessAttachments(msg.GetData(), env),
	SystemPrompt:    systemPrompt,
	ConfigOverride:  cfgOverride,
	ToolOptions:     toolOpts,
	ManagedAgentID:  x.Config.ManagedAgentID,
	LlmConfigID:     x.Config.LlmConfigID,
	LlmModelEntryID: x.Config.LlmModelEntryID,
}
```

- [ ] **Step 4: 重新运行节点测试，确认通过**

Run: `go test ./internal/data/components -run 'TestExtractHarnessAttachments_'`

Expected: PASS

### Task 3: 兼容当前 OpenAI 适配器的安全降级

**Files:**
- Modify: `internal/biz/harness_multimodal_test.go`
- Modify: `internal/biz/agent.go`
- Modify: `internal/biz/harness_multimodal.go`
- Test: `internal/biz/harness_multimodal_test.go`

- [ ] **Step 1: 写失败测试，验证不支持 file part 时会转成文本兜底**

```go
func TestBuildHarnessInputParts_shouldFallbackFileToLegacyTextWhenRequested(t *testing.T) {
	parts := buildHarnessInputPartsWithOptions("读文件", []HarnessAttachment{{
		Filename:      "notes.pdf",
		MimeType:      "application/pdf",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte("pdf")),
	}}, HarnessMultimodalOptions{DisableGenericFilePart: true})

	if len(parts) < 2 {
		t.Fatalf("expected text fallback parts, got %#v", parts)
	}
	last := parts[len(parts)-1]
	if last.Type != schema.ChatMessagePartTypeText || !strings.Contains(last.Text, "notes.pdf") {
		t.Fatalf("expected legacy text fallback, got %#v", last)
	}
}
```

- [ ] **Step 2: 运行测试，确认先失败**

Run: `go test ./internal/biz -run 'TestBuildHarnessInputParts_shouldFallbackFileToLegacyTextWhenRequested'`

Expected: FAIL，因为还没有可控降级开关。

- [ ] **Step 3: 引入最小选项结构，让 `composeMessages` 按模型适配能力决定是否禁用通用 file part**

```go
type HarnessMultimodalOptions struct {
	DisableGenericFilePart bool
}

func buildHarnessInputPartsWithOptions(userText string, attachments []HarnessAttachment, opts HarnessMultimodalOptions) []schema.MessageInputPart {
	// 现有 buildHarnessInputParts 逻辑迁入这里；
	// 当 opts.DisableGenericFilePart=true 时，普通文件不走 file_url，而是回退 legacy 文本块。
}

func buildHarnessInputParts(userText string, attachments []HarnessAttachment) []schema.MessageInputPart {
	return buildHarnessInputPartsWithOptions(userText, attachments, HarnessMultimodalOptions{})
}
```

- [ ] **Step 4: 重新运行测试，确认降级逻辑通过**

Run: `go test ./internal/biz -run 'TestBuildHarnessInputParts_shouldFallbackFileToLegacyTextWhenRequested'`

Expected: PASS

### Task 4: 全量回归与整理

**Files:**
- Modify: `internal/biz/harness_multimodal_test.go`
- Modify: `internal/biz/agent_test.go`
- Modify: `internal/data/components/agent_harness_llm_test.go`

- [ ] **Step 1: 运行多模态与节点相关测试集**

Run: `go test ./internal/biz ./internal/data/components`

Expected: PASS，`internal/biz` 与 `internal/data/components` 全绿。

- [ ] **Step 2: 检查格式化与导入**

Run: `gofmt -w internal/biz/harness_multimodal.go internal/biz/harness_multimodal_test.go internal/biz/agent.go internal/biz/agent_test.go internal/data/components/agent_harness_llm.go internal/data/components/agent_harness_llm_test.go`

Expected: 命令成功，无格式错误。

- [ ] **Step 3: 再跑一次目标测试，确认格式化未引入回归**

Run: `go test ./internal/biz ./internal/data/components`

Expected: PASS
