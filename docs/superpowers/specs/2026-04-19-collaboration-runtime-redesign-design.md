# 协作运行时重构设计（Typed Execution Plan Runtime）

## 背景与目标

当前 Agent Playground 的四种协作模式以 `router.go`、`plan_exec.go`、`supervision.go`、`peer_handoff.go` 四个处理器分别实现。这个方案在功能验证阶段足够直接，但已经暴露出几类结构性问题：

- 模式语义与真实执行不完全一致。前端和注释描述的是“LLM 路由 / 并行监督 / 自主交接”，而后端很多地方仍是固定规则、串行执行或轮询选择。
- 不同模式各自维护执行骨架，公共能力（步骤调度、状态推进、失败恢复、共享上下文）没有统一抽象。
- 方案配置项已经进入 API 与存储，但有些配置并未真正作用于运行时，导致“配置存在但行为不可兑现”。
- 目前更偏“单次跑通”，缺少面向失败恢复、结构化交接、阶段回放和后续 DSL 驱动的稳定内核。

本次设计目标：

1. 将四种协作模式统一收敛到一个 **Typed Execution Plan Runtime**。
2. 保持“协作能力优先”，首版真正支持并发、review、handoff、finalizer 等能力。
3. 采用单机可持久恢复设计：运行中保持自动，失败后支持恢复、重试、跳过、改派。
4. 统一沉淀共享上下文、事件日志与可读 trace，为后续 Debug、回放与画布编排接入打基础。
5. 在不一次性推翻现有 Playground 的前提下，逐步替换现有 handler 内核。

## 设计范围

### In Scope

- 新增统一的执行计划模型：`ExecutionPlan / Step / Run / Artifact / RecoveryAction`。
- 新增统一运行时：步骤调度、状态机推进、依赖解析、失败后恢复。
- 将四种模式从“直接执行逻辑”改造成“Plan Builder”。
- 引入结构化 artifact 和事件日志，区分机器事实源与人类可读 trace。
- 保持单服务部署，但支持运行状态、步骤状态、阶段产物和恢复点持久化。
- 逐步迁移现有 Playground 协作能力到新 runtime。

### Out of Scope

- 首版不做完全自由的 DAG 编排引擎。
- 首版不支持运行中人工接管、暂停后改图、动态插入步骤。
- 首版不直接按分布式 worker / 任务队列 / 多实例协调设计。
- 首版不让前端 Flowgram DSL 直接驱动执行引擎，仍由协作方案编译 plan。
- 首版不引入完整的 event sourcing，只实现结构化事件日志与恢复锚点。

## 核心原则

### 1. 模式负责“编译”，运行时负责“执行”

四种协作模式不再各自维护一套完整 `Execute()` 主流程，而是只负责把协作意图编译成统一的 `ExecutionPlan`。  
真正的执行推进、状态流转、artifact 读写、trace 记录、失败恢复全部下沉到 runtime。

### 2. 先做强类型 Step，不直接上自由图引擎

首版采用 Typed Execution Plan，而不是完全通用的 DAG 工作流引擎。  
原因是当前首要目标是尽快把多 Agent 协作能力做实，并为失败恢复和可观测提供稳定边界。强类型 Step 已足以覆盖当前四种模式，同时避免过早引入过度抽象。

### 3. 运行中自动，失败后恢复

首版首要恢复模型不是人工介入执行中的决策，而是当某个 Step 失败后，将 Run 置于显式的 `waiting_recovery` 状态，并提供一组明确恢复动作：

- `retry_step`
- `reroute_step_to_agent`
- `skip_step`
- `retry_from_checkpoint`

### 4. 事件是事实源，Trace 是展示层

运行时必须先记录结构化事件，再由上层将事件解释为当前 Playground 可读的时间线。  
后续 Debug、筛选、失败定位、回放、控制台恢复操作都基于事件层，而不是直接依赖面向展示的字符串 Trace。

### 5. 渐进迁移，不一次性推翻现有入口

`WorkflowService` 继续作为外部入口，现有 Playground API 先保持兼容。  
先引入新的 plan/runtime/recovery/event 模型，再逐步迁移四种模式，降低重构风险。

## 目标架构

首版运行时推荐采用以下分层：

- **输入层**
  - `CollaborationScheme`
  - 后续可扩展的 Flowgram/DSL 编排输入
- **编排层**
  - `PlanBuilder`：负责将输入编译为 `ExecutionPlan`
- **运行层**
  - `Runtime`：负责调度与状态机推进
  - `StepExecutor`：按 Step 类型实际执行
- **共享上下文层**
  - `ArtifactStore`：持久化结构化中间产物
- **恢复层**
  - `RecoveryService`：在失败后生成和应用恢复动作
- **观测层**
  - `EventLog`：结构化事实源
  - `TraceView`：面向 Playground UI 的可读时间线
- **执行适配层**
  - `HarnessAdapter`：继续复用现有 `RunAgentHarness`、托管 Agent 和工具链能力

### 建议模块划分

建议在 `internal/biz/playground/` 下逐步拆分为：

- `planbuilder/`
- `runtime/`
- `artifact/`
- `recovery/`
- `eventlog/`
- `traceview/`

现有 `internal/biz/playground/collaboration/` 在迁移期保留，但内部角色会逐步调整为：

- `harness_runner.go` 保持为 Agent Step 执行适配层
- 原四个 handler 文件逐步改造成各自的 `PlanBuilder`

## 核心数据模型

### ExecutionPlan

`ExecutionPlan` 表示一次协作运行的静态执行计划。它是编译结果，不记录运行中状态。

建议包含以下核心字段：

- `plan_id`
- `plan_version`
- `source_mode`
- `entry_step_ids`
- `steps`
- `defaults`
  - 默认恢复策略
  - 默认并发策略
  - 默认 finalizer 配置
- `metadata`

职责：

- 描述本次协作有哪些 Step。
- 声明 Step 之间的依赖关系和执行顺序。
- 保留模式语义，但不承载运行态。

### Run

`Run` 表示某个 `ExecutionPlan` 的一次具体执行实例。

建议包含：

- `run_id`
- `plan_id`
- `scheme_id`
- `status`
- `input_artifact_id`
- `current_step_ids`
- `last_checkpoint_id`
- `failure_summary`
- `started_at`
- `finished_at`
- `metadata`

推荐 `RunState`：

- `pending`
- `ready`
- `running`
- `waiting_recovery`
- `completed`
- `failed`
- `cancelled`

其中 `waiting_recovery` 必须是显式状态，不能只通过一段错误信息隐式表示。

### Step

`Step` 是运行时最小执行单元。每个 Step 必须有稳定 ID，且单一职责明确。

建议首版支持 6 种 `StepKind`：

- `route`
- `agent`
- `parallel`
- `review`
- `handoff`
- `finalize`

每个 Step 建议包含：

- `step_id`
- `kind`
- `name`
- `depends_on`
- `agent_binding`（如适用）
- `input_refs`
- `output_ref`
- `config`
- `retry_policy`
- `checkpoint_policy`

推荐 `StepState`：

- `pending`
- `ready`
- `running`
- `succeeded`
- `failed`
- `skipped`

### Artifact

`Artifact` 是结构化共享上下文载体。  
首版应避免继续主要依赖自由文本拼接来完成多 Agent 交接。

建议 artifact 类型至少覆盖：

- `user_input`
- `task_brief`
- `plan_outline`
- `worker_result`
- `review_result`
- `handoff_payload`
- `final_answer`

每个 artifact 建议包含：

- `artifact_id`
- `run_id`
- `type`
- `producer_step_id`
- `schema_version`
- `payload`
- `summary`
- `created_at`

### RecoveryAction

`RecoveryAction` 表示失败后允许用户触发的恢复操作。

首版支持：

- `retry_step(step_id)`
- `reroute_step(step_id, target_agent_id)`
- `skip_step(step_id)`
- `retry_from_checkpoint(checkpoint_id)`

每个恢复动作都必须：

- 对应明确失败点
- 可审计
- 有事件记录
- 可在前端控制台中直接展示

## 四种模式到 Plan Builder 的映射

四种模式的职责统一改为：从方案配置编译出 `ExecutionPlan`。

### RouterExpertPlanBuilder

编译结果：

- `route`
- `agent`
- `finalize`

说明：

- `route` Step 负责做目标 Agent 选择，并输出结构化选择结果：
  - `selected_agent`
  - `candidate_agents`
  - `reason`
  - `confidence`
- `agent` Step 只执行被选中的单个 Agent。
- `finalize` Step 对最终结果做统一整理。

该模式的核心差异是“单路径快速分流”。

### PlanExecPlanBuilder

编译结果：

- `agent(planner)`
- 顺序 `agent(step_1...step_n)`
- `finalize`

说明：

- planner Step 负责生成结构化执行大纲，或对静态步骤补全任务说明。
- 顺序执行的各个 Agent Step 通过 artifact 传递上下文，而不是仅通过自由文本拼接。
- finalizer 统一整理多阶段结果。

该模式的核心差异是“强顺序、强上下文串联”。

### SupervisionPlanBuilder

编译结果：

- `review/supervisor_assign`
- `parallel(worker_1...worker_n)`
- `review(supervisor_review)`
- `finalize`

说明：

- supervisor 的第一步负责定义 worker 分工或确认分工。
- `parallel` Step 负责真正 fan-out/fan-in。
- 第二个 `review` Step 汇总 worker 结果、审查质量、决定是否需要补救。
- finalizer 输出统一答案。

该模式的核心差异是“并发产出 + 集中复核”。

### PeerHandoffPlanBuilder

编译结果：

- `agent(entry)`
- `handoff`
- `agent(next)`
- `handoff`
- ...（直到终止条件）
- `finalize`

说明：

- `handoff` Step 必须输出结构化交接决策：
  - `next_agent`
  - `handoff_reason`
  - `payload_summary`
  - `stop_or_continue`
- runtime 根据交接决策继续追加或激活后续 Step。
- `max_handoffs` 仍作为硬性保护。

该模式的核心差异是“链式自治接力”。

## 运行时与状态机设计

### 执行推进规则

Runtime 的核心职责：

1. 装载 `ExecutionPlan`
2. 创建 `Run`
3. 初始化可执行 Step
4. 按依赖关系推进 Step 状态
5. 调用对应 `StepExecutor`
6. 写入 artifact 与 event
7. 在失败时生成恢复动作
8. 在成功时完成 Run 并输出最终结果

### Step 执行职责

建议每种 Step 都由单独执行器负责：

- `RouteStepExecutor`
- `AgentStepExecutor`
- `ParallelStepExecutor`
- `ReviewStepExecutor`
- `HandoffStepExecutor`
- `FinalizeStepExecutor`

`AgentStepExecutor` 继续通过现有 `RunAgentHarness` 执行模型和工具链，不重写已有 Harness 能力。

### Checkpoint 设计

首版不需要每个 Step 都建立 checkpoint，但建议在以下阶段创建恢复锚点：

- `parallel` fan-out 全部完成后
- `review` 结束后
- `handoff` 决策完成后
- `finalize` 之前

Checkpoint 的目标是支持“从稳定阶段重试”，而不是每次失败都从 Run 起点重跑。

## 恢复与可观测设计

### 失败后的运行语义

当某个 Step 执行失败时：

1. 写入 `step_failed` 事件
2. 记录失败摘要、最近输入、最近输出、错误类型
3. 将该 Step 标记为 `failed`
4. 将 Run 标记为 `waiting_recovery`
5. 生成可选 `RecoveryAction`

如果用户应用恢复动作：

1. 写入 `recovery_applied` 事件
2. 更新 Step / Run 状态
3. 从失败点或 checkpoint 继续推进

### Event Log

Event Log 是机器事实源，建议最少覆盖以下事件：

- `plan_compiled`
- `run_started`
- `step_ready`
- `step_started`
- `artifact_written`
- `step_succeeded`
- `step_failed`
- `checkpoint_created`
- `run_waiting_recovery`
- `recovery_applied`
- `run_completed`
- `run_failed`

### Trace View

Trace View 是 Event Log 的解释视图，用于当前 Playground UI 展示。

原则：

- Event 保持结构化、稳定、机器可消费。
- Trace 可自由组织描述文案，但必须从 Event 派生。
- 不允许只有 Trace 没有底层 Event。

### 前端控制台首版要求

首版控制台至少应能展示：

- 当前 Run 状态
- 当前或最近失败的 Step
- Step 所属 Agent、开始时间、失败原因
- Step 的输入摘要和输出摘要
- Step 读写的 artifact 列表
- 当前可执行的恢复动作
- 恢复动作被应用后的新事件链

## 配置策略

方案配置依然保留在 `CollaborationScheme` 侧，但其职责需调整为“编译输入”，而不是直接驱动各个 handler 的分支逻辑。

建议按以下原则处理：

- 与模式语义强绑定的配置在 `PlanBuilder` 阶段消费。
- 与运行时控制强绑定的配置在 runtime 层统一消费。
- 不能生效的配置不要继续只存不跑。

示例：

- `ExecutionOrder`：由 `PlanExecPlanBuilder` 消费
- `WorkerAgents`：由 `SupervisionPlanBuilder` 消费
- `EntryAgent / MeshAgents / HandoffRules`：由 `PeerHandoffPlanBuilder` 消费
- `EnableFinalizer / FinalizerPrompt`：由 runtime/finalizer 层消费
- `MaxIterations / MaxToolCalls / TimeoutSeconds`：继续统一透传给 Harness 配置

## 前端界面与交互设计调整

方案 C 不只是后端 runtime 重构，前端也需要从“模式示意 + 一次运行 + 看 Trace”的交互模型，升级为“计划执行 + 运行状态 + 失败后恢复”的交互模型。

首版不需要整站重做视觉风格，但必须围绕 Playground 的运行页和方案编辑页进行中等规模调整。

### 设计原则

- 前端展示应优先反映真实的 `ExecutionPlan` 与 `RunState`，而不是固定模式宣传图。
- 失败后恢复必须是一等交互，而不是只在 Trace 里显示一条错误文本。
- Event 与 Artifact 都应有可视入口，避免 runtime 增强后 UI 仍停留在“看字符串日志”。
- 方案编辑页首版只补足 plan builder 所需必要配置，不一次性做成完整编排器。

### 1. 运行页重构为 Plan / Run / Recovery 视图

当前运行页的三栏布局可以保留，但三栏语义应调整：

- 左栏：从静态 `WorkflowGraph` 升级为 **Execution Plan 视图**
- 中栏：从“输入 + 最终结果”升级为 **Run Console + Recovery Console**
- 右栏：从单一 Trace 升级为 **Trace / Artifacts / Recovery Log** 多视图

首版建议仍保留三栏结构，避免一次性重做页面框架，但每一栏的职责需要变化。

### 2. 左栏：静态模式图改为动态 Plan 图

当前 `workflow-graph.tsx` 主要按四种模式写死展示拓扑和文案。  
方案 C 后，左栏应该展示本次运行的真实 `ExecutionPlan`：

- 展示 Step 列表或轻量拓扑图
- 每个 Step 显示：
  - `step_kind`
  - `step_name`
  - `agent`
  - `step_state`
- 支持高亮当前运行 Step
- 支持点击 Step 查看详情

首版不要求完整 DAG 交互编辑，但至少应支持：

- 顺序流展示
- 并行组展示
- handoff 链展示
- finalize 结束节点展示

这样用户看到的是“本次 Run 的真实执行结构”，而不是“某种模式的示意海报”。

### 3. 中栏：引入失败后恢复交互

当前 `RunConsole` 更像一个单次提问和结果查看区。  
方案 C 后，中栏必须支持以下运行语义：

- 输入任务并启动 Run
- 运行中展示当前状态与当前 Step
- 运行成功时展示最终输出
- 运行失败时展示失败摘要和恢复操作

首版中栏建议新增一个明确的 **Recovery 区块**，至少包含：

- 失败 Step 名称
- 失败原因摘要
- 最近一次输入摘要
- 当前可执行恢复动作

恢复动作首版支持以下 UI 操作入口：

- 重试当前 Step
- 跳过当前 Step
- 改派到其他 Agent
- 从最近 checkpoint 重试

其中“改派到其他 Agent”需要一个轻量选择器，但首版只需支持从当前方案绑定 Agent 中选择。

### 4. 右栏：Trace 旁增加 Artifact 与恢复视图

当前右栏更偏时间线阅读器。  
方案 C 后，右栏应改造成可切换的多视图面板，建议首版包含：

- `Trace`
  - 展示可读时间线
- `Artifacts`
  - 按 Step 查看输入摘要、输出摘要、结构化 artifact
- `Recovery`
  - 展示失败点、恢复动作应用记录、checkpoint 记录

设计目标：

- Trace 继续满足“快速看过程”
- Artifact 满足“看结构化上下文和交接”
- Recovery 满足“看失败点和恢复历史”

这样前端才能真正体现 runtime 中 Event、Artifact、Checkpoint 的价值。

### 5. 方案编辑页：补必要配置，不做全量编排器

当前方案编辑页仍是轻量表单。  
方案 C 首版不建议把它一次性改造成图形编排器，但需要补上 plan builder 最低限度所需配置。

首版建议增加：

- 模式专属配置区
  - `router_expert`：路由目标范围、默认 fallback
  - `plan_exec`：planner、执行顺序
  - `supervision`：supervisor、worker 范围
  - `peer_handoff`：entry agent、mesh 范围、handoff 规则摘要
- finalizer 配置
- 默认恢复策略配置（可后置为高级设置）

不建议首版加入：

- 可视化拖拽编排
- 任意 DAG 编辑
- 运行中改图

这些能力应等 runtime 跑稳后，再与 Flowgram/DSL 直接编译能力一起推进。

### 6. 首版前端改动优先级

#### P0：必须改

- `run` 页改成真实 plan/run/recovery 语义
- `workflow-graph` 改成 Execution Plan 视图
- `run-console` 增加失败后恢复交互
- `trace` 面板增加 artifact / recovery 视角

#### P1：应尽快补

- 方案编辑页增加模式专属配置区
- Run 详情页支持按 Step 查看输入输出摘要
- Plan 图支持并行组和 handoff 链的更清晰展示

#### P2：后续再做

- Flowgram/DSL 直接驱动 plan
- 更强的 Run 回放与 checkpoint 浏览
- 图形化编排与局部重跑联动

### 7. 与后端 runtime 的契约要求

前端升级依赖后端提供更结构化的数据，不应继续只依赖 `finalOutput + events` 的组合。

首版建议后端为前端提供：

- `ExecutionPlan` 查询接口
- `Run` 当前状态与失败摘要
- `Step` 列表与状态
- `Artifact` 摘要列表
- `RecoveryAction` 列表
- 恢复动作执行接口
- Event/Trace 查询接口

前端原则上不自己“猜”当前模式的执行结构，而是以 runtime 返回的数据为准。

## 兼容与迁移策略

### 保持外层入口稳定

`WorkflowService` 保持 Playground 的统一入口角色，对外接口先不改。  
内部执行流程由：

- 获取方案
- 获取 handler
- 直接 `Execute`

逐步迁移为：

- 获取方案
- 编译 `ExecutionPlan`
- 调用 `Runtime.Run(plan)`

### 分阶段迁移顺序

#### 阶段 1：引入骨架模型

- 新增 `ExecutionPlan / Run / Step / Artifact / RecoveryAction`
- 新增事件模型与持久化接口
- 不替换现有全部模式，只先建立骨架

#### 阶段 2：新 Runtime 跑通最简单路径

- 先支持 `agent + finalize`
- 将 `router_expert` 迁移为首个通过新 runtime 执行的模式
- 验证计划编译、状态推进、artifact 读写、事件链路

#### 阶段 3：迁移 `plan_exec`

- 支持顺序 `agent` Step
- 引入结构化上游产物传递
- 补 checkpoint 与 finalizer 能力

#### 阶段 4：迁移 `supervision`

- 引入 `parallel` 与 `review`
- 让并发 worker 与 supervisor 复核真正成立

#### 阶段 5：迁移 `peer_handoff`

- 引入 `handoff`
- 支持动态交接决策、终止条件和失败恢复

#### 阶段 6：接入 DSL / 画布编排

- 在协作方案到 plan 跑稳之后
- 再让 Flowgram/DSL 直接编译 plan

## 测试策略

测试应从“能跑通”升级为“模式语义正确”。

### 1. Plan Builder 测试

验证四种模式是否正确编译为目标 Step 序列：

- `router_expert` 应产出 `route -> agent -> finalize`
- `plan_exec` 应产出顺序 Agent Step
- `supervision` 应产出 `review -> parallel -> review -> finalize`
- `peer_handoff` 应产出带 `handoff` 的链式结构

### 2. Runtime 状态机测试

验证：

- Step 依赖是否正确推进
- Run/Step 状态流转是否符合定义
- 失败后是否进入 `waiting_recovery`
- 恢复动作执行后是否重新推进

### 3. Artifact 测试

验证：

- Step 输出是否以结构化 artifact 落盘
- 下游 Step 是否读取预期 artifact
- checkpoint 是否在预期节点创建

### 4. Event / Trace 测试

验证：

- Event 顺序是否正确
- `TraceView` 是否从 Event 派生
- 恢复动作前后事件链是否可回放

### 5. Harness 集成测试

验证：

- `AgentStepExecutor` 能继续复用现有 Harness
- `MaxIterations / MaxToolCalls / TimeoutSeconds` 等 runtime 配置仍生效

## 风险与缓解

### 风险 1：过度抽象，首版推进过慢

缓解：

- 首版只支持 6 种 StepKind
- 不同时做自由 DAG 与画布直接驱动
- 先迁移最简单模式验证 runtime

### 风险 2：新旧两套执行逻辑并存时间较长

缓解：

- 明确迁移阶段
- 每迁移一个模式就补 Builder + Runtime 测试
- 控制兼容层生命周期，避免长期“双写”

### 风险 3：Artifact 设计不稳，后续频繁改 schema

缓解：

- 对关键 artifact 定义 `schema_version`
- 首版先覆盖少量高价值类型
- 保持 Step 输入输出边界清晰

### 风险 4：失败恢复复杂度快速膨胀

缓解：

- 首版恢复动作只做 4 类
- 不支持运行中介入
- 不做复杂时间旅行式回放

## 验收标准

- 四种模式不再直接依赖四套独立执行骨架，而是通过 `PlanBuilder + Runtime` 跑通。
- 运行时显式支持 `RunState` 与 `StepState`，失败后可进入 `waiting_recovery`。
- 首版支持 `retry_step / reroute_step / skip_step / retry_from_checkpoint` 四类恢复动作。
- 多 Agent 交接主要通过结构化 artifact 而不是自由文本拼接完成。
- Event Log 成为事实源，Trace 由 Event 派生。
- 现有 Harness 能力继续可复用，且 Playground 外层入口保持兼容。
- 后续 Flowgram/DSL 可以在不推翻 runtime 的前提下，直接编译为 `ExecutionPlan`。
