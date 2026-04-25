# 规则链 Skill 生成设计

## 背景与目标

当前流程管理中的主规则链已经具备以下可复用信息：

- 规则链基础描述：`additionalInfo.description`
- Flowgram 入参/出参说明：`configuration.flowgram.io`
- 规则链关联的 skill 目录名：`configuration.flowgram.skill.dir_name`
- 管理端已有 Skill 文件浏览与编辑能力
- 后端已有托管 Agent、`run_skill`、`managed_agent_id` 与同步 Harness 执行能力

本次需求的目标不是“给用户生成一份说明文档”，而是为每个主规则链生成一个**供其他 Agent 更容易正确触发和调用的 Skill**。该 Skill 主要服务于 Agent 的自主决策与执行，用户若要手动调用规则链会直接调接口，不依赖 Skill。

本次设计目标：

1. 在流程管理中为每个主规则链增加“创建技能 / 更新技能”入口。
2. 由服务端同步调用选中的托管 Agent，为当前规则链生成对应 Skill，并写入 `/app/skills/<dir_name>/`。
3. 生成的 Skill 必须面向 Agent 使用，重点描述：何时适合触发、如何整理 `metadata/data`、如何调用当前规则链的同步执行接口、如何解释返回结果。
4. 当规则链被删除时，对应 Skill 目录同步删除。
5. 当规则链描述或入/出参发生变化时，Skill 状态变为待更新。
6. 当 Skill 文件被手工删除时，按钮状态自动回退为“创建技能”。

## 设计范围

### In Scope

- 主规则链的 Skill 状态判定与管理入口。
- 后端新增规则链 Skill 状态查询与同步生成接口。
- 规则链配置中的 `flowgram.skill` 元数据扩展。
- 服务端基于托管 Agent + `skill-creator-0.1.0` 生成 Skill。
- 删除规则链时同步删除 Skill 目录。
- Skill 文件存在性缺失时的状态回退。

### Out of Scope

- 子规则链的 Skill 管理。
- 用户直接通过 UI 执行 Skill。
- 将 Skill 生成为 zip 包或发布到外部技能市场。
- 设计 Agent 如何在运行时做最终触发决策；本次只保证 Skill 写法足够清晰、易于被 Agent 正确使用。

## 核心原则

### 1. 主规则链才有独立 Skill

只有主规则链具备独立业务语义边界，才适合作为 Skill 的直接载体。子规则链通常是复用性内部流程，不在本次范围内。

### 2. 后端是 Skill 状态的单一事实源

前端不自行推断 Skill 是否存在、是否待更新，而是以后端返回的状态为准。  
原因是状态同时依赖：

- 规则链配置中的 Skill 元数据
- `/app/skills/<dir_name>/SKILL.md` 的真实文件存在性
- 当前规则链描述与入/出参摘要签名

### 3. Skill 面向 Agent，而不是面向用户

生成的 `SKILL.md` 不应写成“教用户如何点按钮 / 手动调接口”的说明文档，而应写成：

- 这个能力适合处理什么请求
- 如何从自然语言整理结构化输入
- 如何调用规则链同步执行接口
- 如何解释输出与处理失败

### 4. 生成结果必须服务端验收

不能只根据 Agent 文本回复判断生成成功。服务端必须校验：

- 目标目录存在
- `SKILL.md` 存在且非空
- 关键约束段落存在
- 回写配置成功

### 5. 规则链删除与 Skill 删除保持强一致

用户删除规则链时，应同步删除其 Skill 目录；若 Skill 删除失败，则规则链删除也应失败，避免留下脱离规则链生命周期的孤儿 Skill。

## 用户界面设计

### 入口位置

建议共用一套后端接口，在两个位置提供入口：

- `flowgram/src/management/sections/WorkflowSection.tsx`
  - 在列表“操作”列中，仅主流程显示 Skill 按钮
- `flowgram/src/management/rule-detail.tsx`
  - 在规则链详情页基础信息区域显示 Skill 按钮

### 按钮展示规则

仅主规则链显示 Skill 操作。子规则链不显示。

按钮主文案建议为：

- `创建技能`
- `更新技能`

若正在请求，则显示：

- `创建中...`
- `更新中...`

为避免阻止用户手动重建，状态为已同步时仍可显示 `更新技能`，但同时增加一个只读状态标记：

- `已同步`
- `待更新`
- `缺失`

### Agent 选择交互

本次约束已明确：默认复用管理端当前选中的托管 Agent。

但流程管理页本身不一定天然持有该上下文，因此交互建议如下：

1. 点击“创建技能 / 更新技能”
2. 若当前上下文中已有默认托管 Agent，则直接调用后端接口
3. 若当前上下文无默认托管 Agent，则弹出托管 Agent 选择器
4. 用户确认后调用后端同步接口

前端不保存 Skill 生成逻辑，只负责传递：

- `ruleChainId`
- `managedAgentId`

### 状态刷新

以下场景需要刷新 Skill 状态：

- 打开规则链详情页
- 列表页加载主流程列表
- 保存规则链基础信息成功后
- Skill 创建 / 更新成功后
- 删除规则链成功后刷新列表

## Skill 状态模型

建议由后端统一返回如下三态：

- `missing`
  - 未配置 `dir_name`，或 `/app/skills/<dir_name>/SKILL.md` 不存在
- `stale`
  - Skill 文件存在，但当前规则链语义签名与最近一次生成签名不一致
- `ready`
  - Skill 文件存在，且签名一致

### 状态判定依据

Skill 状态不应基于规则链全量配置，而只基于会影响 Skill 语义的字段：

1. 描述：`additionalInfo.description`
2. `metadata` 入参定义：`configuration.flowgram.io.request_metadata_params`
3. `data` 入参定义：`configuration.flowgram.io.request_message_body_params`
4. 出参定义：`configuration.flowgram.io.response_message_body_params`

节点编排本身变化不会自动将 Skill 标记为待更新。本次需求中，Skill 的更新边界只看以上四项。

## 配置存储设计

沿用现有 `configuration.flowgram.skill`，扩展以下字段：

- `dir_name`
  - Skill 目录名，对应 `/app/skills/<dir_name>/`
- `status`
  - 最近一次后端判定状态：`missing | stale | ready`
- `signature`
  - 对描述 + 入/出参定义做稳定序列化后的摘要
- `generated_at`
  - 最近一次成功生成时间
- `generated_by_managed_agent_id`
  - 最近一次成功生成所使用的托管 Agent ID
- `skill_entry_file`
  - 固定记录为 `SKILL.md`
- `last_error`
  - 最近一次生成失败时的简要错误信息

### 目录名策略

目录名不是由前端人工输入，而是由服务端生成与规范化：

1. 若已有 `dir_name` 且本次是更新，则优先复用
2. 若没有 `dir_name`，则要求 Agent 基于规则链描述给出英文目录名建议
3. 服务端再做最终规范化：
   - 小写
   - 仅允许字母、数字、连字符
   - 去除前后分隔符
   - 限长
   - 冲突时追加短后缀

目录名一旦生成并成功落盘，后续更新应稳定复用，避免目录漂移。

## 后端接口设计

建议新增规则链 Skill 管理接口：

- `GET /api/v1/rules/{id}/skill/status`
- `POST /api/v1/rules/{id}/skill/generate`

`DELETE /api/v1/rules/{id}/skill` 不是必需接口，因为规则链删除已要求强一致同步删除 Skill。

### GET /rules/{id}/skill/status

职责：

1. 读取规则链详情
2. 解析 `configuration.flowgram.skill`
3. 检查 `/app/skills/<dir_name>/SKILL.md` 是否存在
4. 计算当前规则链 Skill 语义签名
5. 返回 `missing / stale / ready`

建议响应至少包含：

- `status`
- `dirName`
- `entryFile`
- `signature`
- `generatedAt`
- `generatedByManagedAgentId`
- `lastError`

### POST /rules/{id}/skill/generate

请求建议包含：

- `managedAgentId`

职责：

1. 校验规则链存在且为主规则链
2. 读取规则链描述与入/出参定义
3. 生成或复用目标 `dir_name`
4. 组装 Agent 上下文
5. 使用指定 `managed_agent_id` 同步执行 Harness
6. 强制 Agent 使用 `run_skill("skill-creator-0.1.0")`
7. 校验产物
8. 回写 `configuration.flowgram.skill`
9. 返回最新状态与目录信息

该接口为同步接口，前端点击后直接等待结果，不额外引入任务中心。

## Agent 生成编排设计

### 运行方式

服务端不暴露通用聊天 prompt 给前端，而是在后端内部构造一个受控请求，走现有托管 Agent + Harness 同步执行链路。

输入上下文至少包括：

- 规则链 ID
- 规则链名称
- 描述
- `metadata` 入参定义
- `data` 入参定义
- 出参定义
- 目标目录 `/app/skills/<dir_name>/`
- 生成目标：Agent-oriented Skill
- 规则链同步执行接口约定

### 对 Agent 的强约束

服务端应通过系统提示或任务模板明确要求：

1. 必须调用 `run_skill`
2. `skill_name` 固定为 `skill-creator-0.1.0`
3. 生成结果必须落到 `/app/skills/<dir_name>/SKILL.md`
4. 生成内容面向 Agent，不面向手工终端用户
5. 重点写清：
   - 能力边界
   - 触发线索
   - 自然语言到 `metadata/data` 的整理规则
   - 当前规则链同步执行接口调用方式
   - 返回结果解释
   - 错误与缺参兜底

### 服务端验收要求

生成结束后，服务端必须至少检查：

- `/app/skills/<dir_name>/SKILL.md` 存在
- 内容非空
- 内容中出现当前规则链 ID
- 内容中出现规则链同步执行接口路径
- 内容中包含 `metadata`、`data`、返回结果解释相关段落

若任一项不满足，生成视为失败，不更新 `generated_at` 和 `signature`，但可更新 `last_error`。

## 生成的 SKILL.md 结构约束

生成的 Skill 不需要教用户怎么点按钮，而应帮助 Agent 更容易在后续会话中正确使用该规则链能力。

建议强制包含以下章节：

### 1. 能力定义

- 该 Skill 对应哪个规则链
- 解决什么问题
- 适合处理哪些类型的请求
- 不适合处理哪些类型的请求

### 2. 触发线索

- 常见意图或任务信号
- 哪些上下文说明该 Skill 值得优先考虑
- 当上下文不足时先补问哪些信息

这里不是写死 Agent 决策规则，而是提供足够清晰的语义边界与线索。

### 3. 参数整理规则

- 如何从自然语言提取 `metadata`
- 如何从自然语言提取 `data`
- 缺少字段时哪些可以推断，哪些必须追问
- 如何对齐规则链的输入结构

### 4. 规则链同步执行方式

该部分必须明确：

- 当前规则链 ID
- 使用同步执行接口
- 请求体结构
- `metadata/data` 如何组装
- 返回值如何读取

### 5. 结果解释

- 哪些返回字段可直接使用
- 哪些字段应先总结再回答用户
- 空结果或异常结果时如何反馈

### 6. 失败与兜底

- 参数不足时先追问
- 执行失败时如何说明
- 不可盲猜规则链未返回的信息

## 同步执行接口设计要求

本次需求要求 Skill 明确描述如何触发当前规则链，并依赖规则链的 `metadata` 与 `data` 入参定义。

因此同步执行接口的设计需要满足以下约束：

1. Skill 说明中必须能明确表达 `metadata` 与 `data` 的来源与写法
2. 规则链执行入口应允许 Skill 清晰表达这两类输入

当前 `ExecuteRuleChainSync` 的 proto 仅显式包含 `data`，不足以自然承载“Skill 以 `metadata/data` 为两类输入”的描述模型。  
本次设计建议在实现阶段评估并优先采用以下方向之一：

- 方向 A：扩同步执行接口，显式支持 `metadata`
- 方向 B：约定 `data` 内封装 `{ metadata, data }`，并在规则链入口统一解析

推荐方向为 **A**，因为它与规则链现有的 Skill 语义模型一致，生成出来的 `SKILL.md` 也更清晰。

## 删除同步设计

当删除规则链时，应在现有 `DeleteRuleChain` 逻辑中增加如下步骤：

1. 先读取规则链详情
2. 解析 `configuration.flowgram.skill.dir_name`
3. 若存在目录名，则尝试删除 `/app/skills/<dir_name>/`
4. Skill 删除成功后，再删除规则链引擎与数据库记录

若 Skill 删除失败，则整个规则链删除返回失败，不做部分成功。

这样可保证规则链与对应 Skill 的生命周期一致，不会留下孤儿 Skill。

## 文件丢失回退设计

若 Skill 目录或 `SKILL.md` 被人工删除，不需要额外清理数据库。  
后端每次查询状态时都重新做真实文件检查：

- 目录不存在或 `SKILL.md` 不存在 -> `missing`
- 文件存在但签名不匹配 -> `stale`
- 文件存在且签名匹配 -> `ready`

这保证了按钮状态总能自动回退到正确状态，无需依赖额外定时任务。

## 错误处理与可观测

### 生成失败

生成接口失败时：

- 返回用户可理解的错误
- 不更新 `generated_at`
- 保留或更新 `last_error`
- 状态建议返回 `missing` 或 `stale`，取决于历史产物是否仍存在

### 删除失败

删除 Skill 目录失败时：

- 中止规则链删除
- 返回明确错误
- 不允许出现“规则链已删但 Skill 保留”的部分成功状态

### 可观测建议

服务端日志中建议记录：

- `rule_chain_id`
- `managed_agent_id`
- `skill_dir_name`
- 生成开始时间 / 结束时间
- 生成结果
- 验收失败原因

## 风险与缓解

### 风险 1：Agent 生成的 Skill 不稳定

缓解：

- 服务端提供强约束提示
- 生成后做固定结构验收
- 状态不直接依赖 Agent 自述成功

### 风险 2：目录名漂移导致旧 Skill 无法更新

缓解：

- 首次成功生成后稳定复用 `dir_name`
- 更新时禁止重新换目录，除非目录缺失且用户主动重建

### 风险 3：规则链输入模型与同步执行接口不一致

缓解：

- 在实现阶段优先统一 `metadata/data` 的执行入口表达
- 以接口契约清晰可表达为目标约束 Skill 内容

### 风险 4：前端自行推断状态导致错乱

缓解：

- 状态完全以后端接口为准
- 前端仅负责展示与触发

## 验收标准

- 主规则链在列表页和详情页均可看到 Skill 管理入口。
- Skill 状态由后端统一返回 `missing / stale / ready`。
- 创建 Skill 时使用指定托管 Agent 同步执行生成。
- 生成的 Skill 写入 `/app/skills/<dir_name>/SKILL.md`。
- Skill 内容明确面向 Agent，包含参数整理、规则链同步执行与结果解释。
- 描述或入/出参变更后，Skill 状态变为 `stale`。
- 手工删除 Skill 文件后，状态自动回退为 `missing`。
- 删除规则链时，对应 Skill 目录同步删除且保持强一致。
