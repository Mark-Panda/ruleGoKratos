# 定时任务管理设计

## 背景与目标

当前项目已有规则链管理、主规则链启停与规则链执行能力，但缺少独立的定时任务管理入口。现有 Flowgram 中存在 `endpoint/schedule` 类节点与 DSL 映射，但它属于规则链编排内部能力，不提供面向运维的任务列表、数据库持久化、启停控制和执行历史。

本次需求目标：

1. 在管理后台新增“定时任务”菜单。
2. 支持定时任务列表的增删改查、开启、关闭。
3. 支持 cron 表达式，并将常规 cron 场景翻译成可视化配置。
4. 定时任务持久化到数据库，创建时默认关闭。
5. 定时任务绑定某条已启用的主规则链。
6. 定时触发时向规则链发送固定系统事件：`{"trigger":"schedule","taskId":"..."}`。
7. 新增执行历史表，支持按定时任务查看历史记录。

## 设计范围

### In Scope

- 后端新增定时任务与执行历史数据模型。
- 后端新增定时任务 CRUD、启停、执行历史查询接口。
- 后端引入单实例内置 cron 调度器。
- 服务启动时恢复已开启的定时任务。
- 开启任务时校验绑定规则链必须是已启用主规则链。
- 触发时若绑定规则链不可用，自动关闭定时任务并记录失败历史。
- 前端新增“定时任务”菜单与列表页面。
- 前端新增 cron 常规场景可视化配置与高级 cron 输入。

### Out of Scope

- 分布式调度与多实例去重。
- 外部任务队列、Redis 锁或独立调度服务。
- 定时任务 payload 自定义配置。
- 对 Flowgram 画布内 `endpoint/schedule` 节点能力做协议改造。
- 复杂 cron 全字段图形化编辑器。

## 核心原则

### 1. 数据库是任务配置事实源

定时任务的名称、cron 表达式、可视化配置、绑定规则链、启停状态和最近运行状态均以数据库为准。内存调度器只保存当前进程中的运行态 job，服务重启后必须从数据库恢复。

### 2. 创建默认关闭，开启时做强校验

任务创建后默认 `disabled=true`，不会立即注册到 cron 调度器。用户点击开启时，后端校验绑定规则链必须满足：

- `root=true`
- `disabled=false`
- 当前规则链引擎中可执行

校验通过后才更新数据库状态并注册 job。

### 3. 触发失败不静默丢失

每次触发都写入 `scheduled_task_runs`。规则链执行成功记录 `success`；执行失败记录 `failed`；如果绑定规则链已删除或停用，则自动关闭任务，并写入失败历史，失败原因说明任务已被系统关闭。

### 4. 可视化配置服务于回显，执行以 cron 为准

前端保存时统一生成 cron 表达式。后端执行只依赖 `cron_expr`。`schedule_config` 保存前端可视化配置 JSON，用于编辑时准确回显用户选择。

## 后端设计

### 数据表

新增 `scheduled_tasks`：

- `id`：任务 ID。
- `name`：任务名称。
- `description`：任务描述。
- `rule_chain_id`：绑定的规则链 ID。
- `cron_expr`：cron 表达式。
- `schedule_type`：可视化类型，例如 `every_minutes`、`every_hours`、`daily`、`weekly`、`monthly`、`advanced`。
- `schedule_config`：可视化配置 JSON。
- `disabled`：是否关闭，默认 `true`。
- `last_run_at`：最近运行时间。
- `last_status`：最近运行结果，例如 `success`、`failed`。
- `last_error`：最近失败原因。
- `created_at` / `updated_at` / `deleted_at`。

新增 `scheduled_task_runs`：

- `id`：执行历史 ID。
- `task_id`：定时任务 ID。
- `rule_chain_id`：触发时绑定的规则链 ID。
- `status`：执行结果，例如 `success`、`failed`。
- `trigger_payload`：固定系统事件 JSON 快照。
- `error_message`：失败原因。
- `started_at`：开始时间。
- `finished_at`：结束时间。
- `created_at`。

建议在 `sql/` 下新增建表脚本，并在 `internal/data/dao/dao.go` 中按现有项目习惯补充 `AutoMigrate` 或独立迁移函数。

### API

新增 `ScheduledTaskService`，保持与 `TaskBoardService` 类似的 proto + HTTP 风格：

- `ListScheduledTasks`
- `GetScheduledTask`
- `CreateScheduledTask`
- `UpdateScheduledTask`
- `DeleteScheduledTask`
- `EnableScheduledTask`
- `DisableScheduledTask`
- `ListScheduledTaskRuns`

列表接口支持按名称、启停状态、绑定规则链过滤，并返回分页信息。执行历史接口按 `task_id` 查询，支持分页。

### 分层实现

沿用现有 Kratos 分层：

- `api/rulego/v1/scheduled_task_service.proto`
- `internal/service/scheduled_task_service.go`
- `internal/biz/scheduled_task.go`
- `internal/biz/entity/scheduled_task.go`
- `internal/data/scheduled_task.go`
- `internal/data/dao/scheduled_task.go`

`ScheduledTaskUsecase` 负责业务规则、启停校验、触发执行与历史写入。`ScheduledTaskRepo` 负责持久化读写。调度器作为独立组件注入 usecase，避免把 cron job 管理逻辑散落在 service 层。

### 调度器生命周期

服务启动时：

1. 初始化 cron scheduler。
2. 查询所有 `disabled=false` 的定时任务。
3. 对每个任务校验 cron 表达式。
4. 注册到内存 scheduler。
5. 启动 scheduler。

启用任务时：

1. 查询任务与绑定规则链。
2. 校验规则链是已启用主规则链。
3. 校验 cron 表达式合法。
4. 更新 `disabled=false`。
5. 注册或替换内存 job。

关闭任务时：

1. 更新 `disabled=true`。
2. 从内存 scheduler 移除 job。

删除任务时：

1. 若任务已开启，先移除内存 job。
2. 删除任务记录。
3. 执行历史保留，便于审计。

触发任务时：

1. 再次读取任务与绑定规则链，避免使用陈旧内存状态。
2. 若任务已关闭，直接跳过。
3. 若规则链不存在、不是主规则链或已停用，自动关闭任务，移除 job，写失败历史。
4. 构造固定系统事件：`{"trigger":"schedule","taskId":"..."}`。
5. 调用现有规则链执行入口。
6. 写入执行历史，并更新任务最近运行状态。

## 前端设计

### 菜单入口

在 `flowgram/src/management/admin-panel.tsx` 中新增菜单项“定时任务”。沿用现有 hash 菜单模式，新增 `MenuKey`、hash 解析、页面标题与 `renderPage` 映射。

### 列表页面

新增 `ScheduledTaskSection`，风格参考 `TaskBoardSection` 与 `WorkflowSection`：

- 顶部筛选：任务名称、启停状态、绑定规则链。
- 表格字段：任务名称、绑定主规则链、cron 描述、启停状态、最近运行时间、最近结果、最近错误、更新时间。
- 操作：编辑、开启、关闭、查看历史、删除。
- 创建 / 编辑使用 `Modal + Form`。

### cron 可视化配置

表单支持以下常规场景：

- 每 N 分钟：生成 `*/N * * * *`。
- 每 N 小时：生成 `0 */N * * *`。
- 每天某时：生成 `M H * * *`。
- 每周某天某时：生成 `M H * * D`。
- 每月某日某时：生成 `M H D * *`。
- 高级 cron 表达式：用户直接输入。

前端保存时同时提交：

- `cronExpr`
- `scheduleType`
- `scheduleConfig`

编辑回显时优先使用 `scheduleType + scheduleConfig` 还原表单；如果历史数据缺少配置，可退回到高级 cron 输入展示。

### 规则链选择

绑定规则链下拉框只展示主规则链。开启任务时后端仍会强校验规则链是否已启用，前端不作为唯一约束来源。

### 执行历史

在列表操作中提供“执行历史”，打开抽屉或弹窗展示：

- 执行时间
- 执行状态
- 规则链 ID
- 失败原因
- 触发 payload

历史列表按 `task_id` 分页查询。

## 错误处理

- cron 表达式非法：创建或更新时返回参数错误。
- 开启任务但绑定规则链不可用：返回业务错误，不改变任务状态。
- 触发时规则链不可用：自动关闭任务并记录失败历史。
- 规则链执行失败：任务保持开启，记录失败历史与最近失败原因。
- 调度器注册失败：开启失败，任务状态不应变为开启。

## 测试策略

### 后端

- 创建任务默认 `disabled=true`。
- 更新任务时校验 cron 表达式。
- 开启任务时校验绑定规则链必须是已启用主规则链。
- 服务启动恢复已开启任务。
- 关闭任务会移除内存 job。
- 触发成功写入 `scheduled_task_runs` 并更新最近运行状态。
- 规则链不可用时自动关闭任务并写入失败历史。
- 删除任务时保留历史记录。

### 前端

- cron 可视化配置生成正确表达式。
- `scheduleType + scheduleConfig` 可正确回显。
- 高级 cron 输入可保存和回显。
- 列表 CRUD、启停、历史查看的 API 调用参数正确。

## 落地计划

### 阶段 1：后端模型与 API

- 新增 proto、DAO、Repo、Usecase、Service。
- 新增数据库表与迁移。
- 接入 wire、HTTP、gRPC 注册。
- 完成 CRUD、启停、历史查询。

### 阶段 2：调度器与规则链执行

- 引入 cron 调度库。
- 实现服务启动恢复。
- 实现任务触发、执行历史写入、规则链不可用自动关闭。
- 补充后端单元测试。

### 阶段 3：前端页面

- 新增菜单与 `ScheduledTaskSection`。
- 新增 API service。
- 实现 cron 可视化表单、任务 CRUD、启停、历史抽屉。
- 补充前端 cron 转换测试。

## 风险与后续扩展

- 当前方案按单实例设计，多实例部署会导致同一任务重复触发。后续如需多副本部署，应增加分布式锁或迁移到外部调度服务。
- 执行历史可能持续增长，后续可增加保留天数、归档或清理任务。
- 高级 cron 表达式的语义说明需要与后端 cron 库支持的字段数量保持一致，避免前端允许的表达式与后端解析规则不一致。
