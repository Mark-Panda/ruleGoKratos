# DSL 映射统一化设计（单一配置源）

## 背景与目标

当前前端 DSL 映射集中在 `flowgram/src/utils/rulechain-builder.ts`，存在如下问题：

- 同一个节点的 `toDSL` 与 `fromDSL` 逻辑分散在大 `switch` 中，重复多且易漂移。
- 节点表单 schema、默认值、字段映射规则分散在多个位置，不是单一事实源。
- 新增节点时，容易出现“UI 可编辑但 DSL 不完整”或“DSL 可导入但 UI 回显缺字段”的问题。

本次设计目标：

1. 建立可复用的公共映射引擎，降低重复逻辑。
2. 将节点映射规则统一为单一配置源（`NodeMappingSpec`）。
3. 采用渐进迁移，先覆盖 `ai/agentHarness`、`ai/llm`、`restApiCall`。
4. 将新增组件的实现规范沉淀为项目级 Skill，确保后续组件按同一规则实现。

## 设计范围

### In Scope

- 新增 DSL 映射引擎目录与类型定义。
- 在 `rulechain-builder.ts` 中接入引擎分发（命中 spec 走新逻辑，未命中沿用旧逻辑）。
- 首批迁移三个节点：`ai/agentHarness`、`ai/llm`、`restApiCall`。
- 为首批节点补充 round-trip 测试。
- 新增项目级 Skill：约束未来新增组件必须同步完成映射与测试。

### Out of Scope

- 一次性迁移所有节点。
- 变更节点渲染 UI 与交互体验。
- 修改后端 DSL 协议结构。

## 目标架构

新增目录：`flowgram/src/utils/dsl-mapping/`

- `types.ts`
  - 定义 `NodeMappingSpec`、字段描述、值类型、默认值策略、`transform.in/out` 钩子类型。
- `engine.ts`
  - 提供通用 `mapNodeToDsl` 与 `mapDslToNodeData` 能力。
  - 处理路径读写、类型归一化、默认值回填。
- `specs.ts`
  - 维护节点映射配置注册表（单一配置源）。
  - 首批放入 `ai/agentHarness`、`ai/llm`、`restApiCall`。
- `specializers.ts`
  - 仅放复杂特例转换（例如 URL query 拼装/拆分、messages 结构映射）。

## 数据模型与映射规则

### NodeMappingSpec 核心字段（概念）

- `nodeType`: 节点类型，例如 `ai/llm`。
- `fields`: 字段映射列表，描述 `nodePath <-> dslPath`。
- `valueKind`: 值类别（`template`、`constant`、`number`、`boolean`、`json`）。
- `defaultValue`: 缺失值默认值。
- `required`: 是否关键字段。
- `transformOut`: `toDSL` 阶段的字段后处理。
- `transformIn`: `fromDSL` 阶段的字段后处理。

### 统一原则

- 优先从 `inputsValues` 取值，避免与 `inputs` schema 混杂。
- 默认值策略统一在 spec 中声明，避免散落在业务代码。
- 允许节点特例，但特例必须最小化并集中在 `specializers.ts`。

## 执行流程

### toDSL（画布 -> DSL）

1. 根据节点 `type` 查询 `spec`。
2. 命中时调用 `engine.mapNodeToDsl(node, spec)`：
   - 按字段映射读取 `inputsValues`。
   - 按 `valueKind` 归一化类型。
   - 回填 `defaultValue`。
3. 执行 `transformOut`（如 `restApiCall` URL query 合成、`ai/llm.messages` 组装）。
4. 输出标准 DSL `configuration`。
5. 未命中时降级走现有 `switch` 逻辑。

### fromDSL（DSL -> 画布）

1. 根据节点 `type` 查询 `spec`。
2. 命中时调用 `engine.mapDslToNodeData(dslNode, spec)`：
   - 从 `configuration` 提取字段。
   - 按 spec 回填 `inputsValues`（含值类型）。
3. 执行 `transformIn`（如 URL query 拆分到 `paramsValues`）。
4. 生成节点 `data`。
5. 未命中时沿用现有逻辑。

## 错误处理与兼容性

- 映射失败不抛出阻断性异常，使用 `warn` 并降级到安全默认值，保证画布可打开。
- 关键字段（如 `requestMethod`、`model`、`url`）提供最小默认值，避免导入后无法编辑。
- 采用“按节点迁移”策略，未迁移节点不受影响，控制回归风险。

## 测试策略

新增测试目录：`flowgram/src/utils/dsl-mapping/__tests__/`

每个已迁移节点至少包含：

1. `toDSL` 用例：`inputsValues -> DSL.configuration`。
2. `fromDSL` 用例：`DSL.configuration -> inputsValues`。
3. round-trip 用例：`node -> dsl -> node` 关键字段一致。

回归重点：

- `ai/llm` 的 `messages/params` 双向一致性。
- `ai/agentHarness` 的开关字段与数值字段默认值一致性。
- `restApiCall` 的 URL 与 query params 双向一致性。

## 落地计划（渐进）

### 阶段 1

- 新增 `dsl-mapping` 目录与通用引擎。
- 接入分发逻辑（新旧并存）。
- 迁移 `ai/agentHarness`、`ai/llm`。

### 阶段 2

- 迁移 `restApiCall`（含 query 特例）。
- 补齐测试与回归。
- 清理已被引擎替代的重复 helper 代码。

## 项目级 Skill 设计

为保证后续新增组件统一遵循映射规范，新增项目级 Skill：

- 路径：`.cursor/skills/dsl-component-mapping/SKILL.md`
- 名称建议：`dsl-component-mapping`
- 触发条件：新增节点组件、修改节点 DSL、修复节点回显/导出问题时自动使用。

Skill 约束内容：

1. 新增组件必须同时提交：
   - 节点注册与 UI schema。
   - `NodeMappingSpec` 映射定义（`toDSL/fromDSL`）。
   - 至少 1 组 round-trip 测试。
2. 特例转换必须写在 `specializers`，并说明原因。
3. PR 检查若缺少映射或测试，视为未完成。

## 风险与缓解

- 风险：首次抽象过度，导致理解成本上升。  
  缓解：先迁移 3 个节点验证模式，再扩展。

- 风险：历史节点行为依赖隐式默认值。  
  缓解：spec 显式声明默认值，并通过 round-trip 锁定行为。

- 风险：后续新增节点绕过规范。  
  缓解：以项目级 Skill 强化流程约束，并在 CR 模板中加入检查项。

## 验收标准

- 已迁移节点不再依赖大 `switch` 的字段拼装逻辑。
- 三个节点的 `toDSL/fromDSL` 与 round-trip 测试通过。
- 新增项目级 Skill 已落地，可指导后续新增组件按统一规则实现。
- 对未迁移节点无行为回归。
