# Agent 多模态附件接入说明

本文档说明当前项目中 Agent 多模态附件的统一输入约定，适用于以下两类入口：

- 管理端聊天接口 `POST /api/v1/chat/stream`
- 规则链节点 `ai/agentHarness`

目标是让两条入口都复用同一套附件结构：图片、视频、音频优先走 Eino 原生多模态；普通文件保持统一输入格式，并在当前模型适配层能力不足时自动降级。

## 1. 统一附件结构

统一附件对象字段如下：

```json
{
  "filename": "demo.png",
  "mimeType": "image/png",
  "text": "",
  "contentBase64": "iVBORw0KGgoAAA..."
}
```

字段说明：

- `filename`：文件名，可为空，但建议始终传
- `mimeType`：MIME 类型，建议显式传入，例如 `image/png`、`video/mp4`、`application/pdf`
- `text`：文本内容。适合纯文本、代码、Markdown、JSON 等可直接内联的附件
- `contentBase64`：二进制内容的 Base64。适合图片、视频、音频，以及暂时无法内联为文本的其它文件

约束说明：

- `text` 与 `contentBase64` 可以二选一，也可以同时存在
- 若两者都为空，该附件会被忽略
- `mimeType` 缺失时，后端会尝试根据二进制魔数或文件扩展名补全

## 2. 管理端聊天接口用法

管理端聊天接口直接支持 `attachments` 字段，结构与上文一致。

请求示例：

```json
{
  "message": "请分析这张图和这个 PDF",
  "history": [],
  "llmConfigId": 1,
  "llmModelEntryId": 2,
  "attachments": [
    {
      "filename": "screen.png",
      "mimeType": "image/png",
      "contentBase64": "iVBORw0KGgoAAA..."
    },
    {
      "filename": "spec.pdf",
      "mimeType": "application/pdf",
      "contentBase64": "JVBERi0xLjQKJ..."
    }
  ]
}
```

补充说明：

- 若 `message` 中包含公网 `https` 图片或视频直链，服务端会尝试拉取并自动追加到 `attachments`
- 前端 `flowgram/src/services/api-chat.ts` 已按该结构发送附件

## 3. 规则链 `ai/agentHarness` 节点用法

`ai/agentHarness` 节点当前不会自己上传文件，但会从进入节点的消息中提取 `attachments`，并注入到 `HarnessRequest.Attachments`。

当前支持两种来源：

### 3.1 放在 `msg.data` JSON 中

适合上游节点输出 JSON 结构数据时使用。

```json
{
  "query": "请总结这个文件",
  "attachments": [
    {
      "filename": "report.pdf",
      "mimeType": "application/pdf",
      "contentBase64": "JVBERi0xLjQKJ..."
    }
  ]
}
```

说明：

- `agentHarness` 会优先尝试把 `msg.data` 当作 JSON 解析
- 若 JSON 顶层存在 `attachments` 字段，会直接读取

### 3.2 放在 `metadata.attachments` 中

适合上游节点不方便改 `msg.data` 主体，但可以追加元数据时使用。

示例概念结构：

```json
{
  "metadata": {
    "attachments": [
      {
        "filename": "clip.mp4",
        "mimeType": "video/mp4",
        "contentBase64": "AAAAIGZ0eXBpc29..."
      }
    ]
  }
}
```

说明：

- 节点运行时会检查环境中的 `metadata.attachments`
- 如果 `msg.data` 中没有可用附件，会回退到 `metadata.attachments`

## 4. 各类型附件的运行时行为

### 4.1 图片

常见 `mimeType`：

- `image/png`
- `image/jpeg`
- `image/webp`
- `image/gif`
- `image/bmp`

行为：

- 走 Eino `UserInputMultiContent`
- 以图片多模态 part 发送给模型

### 4.2 视频

常见 `mimeType`：

- `video/mp4`
- `video/webm`

行为：

- 走 Eino `UserInputMultiContent`
- 以视频多模态 part 发送给模型

### 4.3 音频

当前优先支持以下可直接进入多模态通道的 MIME：

- `audio/wav`
- `audio/vnd.wav`
- `audio/wave`
- `audio/x-wav`
- `audio/mpeg`
- `audio/mp3`

行为：

- 命中支持列表时，走音频多模态 part
- 未命中支持列表时，回退为文本附件块

### 4.4 普通文件

例如：

- `application/pdf`
- `application/zip`
- `application/octet-stream`
- 其它非图/音/视频文件

行为分两层理解：

1. 内部统一模型里，后端已经能把普通文件组装成 Eino `file_url` part
2. 但当前项目使用的 `eino-ext/components/model/openai` 适配层尚未消费通用 `file_url` part，因此在线请求阶段默认会把普通文件降级为文本附件块

这意味着：

- 接入层已经统一，不需要为不同入口发明第二套文件结构
- 图片、视频、音频是真正的原生多模态
- 普通文件目前仍会以“文件名 + MIME + 文本内容或 Base64 摘要”的形式提供给模型

## 5. 推荐接入策略

建议按以下原则构造附件：

- 纯文本文件：优先写入 `text`
- 图片、视频、音频：优先写入 `contentBase64`，并尽量带正确的 `mimeType`
- PDF、压缩包、Office 文件等普通文件：当前先走 `contentBase64`
- 如果浏览器或上游系统给不出可靠的 `mimeType`，至少保证 `filename` 带正确扩展名

推荐示例：

```json
[
  {
    "filename": "query.md",
    "mimeType": "text/markdown",
    "text": "# 背景\n请总结下面内容"
  },
  {
    "filename": "screen.png",
    "mimeType": "image/png",
    "contentBase64": "iVBORw0KGgoAAA..."
  },
  {
    "filename": "spec.pdf",
    "mimeType": "application/pdf",
    "contentBase64": "JVBERi0xLjQKJ..."
  }
]
```

## 6. 排障建议

### Q1：为什么图片没有按多模态理解？

优先检查：

- `contentBase64` 是否真的是图片原始字节
- `mimeType` 是否为 `image/*`
- 文件扩展名是否正确

### Q2：为什么普通文件没有“像图片一样”被模型直接理解？

这是当前模型适配层的已知边界。后端内部已经保留了 Eino `file_url` 能力位，但在线请求时为了兼容现有适配器，会默认降级成文本附件块。

### Q3：为什么节点没有拿到附件？

优先检查：

- `msg.data` 是否是合法 JSON
- `attachments` 是否位于 JSON 顶层
- 或者 `metadata.attachments` 是否存在
- 字段名是否使用 `filename / mimeType / text / contentBase64`

## 7. 当前结论

当前项目的统一约定可以概括为：

- Chat 与 `ai/agentHarness` 节点共用同一套 `attachments` 结构
- 图片、视频、音频优先走 Eino 原生多模态
- 普通文件先统一接入，当前默认安全降级
- 后续若模型适配层补齐通用 `file_url` 支持，无需改调用方结构，只需放开后端降级策略
