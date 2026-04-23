# Agent 子 Agent（run_sub_agent）使用说明

本文档用于说明当前项目中 `run_sub_agent` 的能力、适用入口、配置方法与排障方式。

## 1. 能力概览

`run_sub_agent` 是 Agent 运行时工具之一，用于让主 Agent 把任务拆分给子 Agent 执行，再聚合结果。

当前已支持：

- 单任务委派：`task`
- 批量任务委派：`sub_tasks_json`（JSON 数组字符串）
- 并发执行：`max_concurrency`（可选，不传时自动估算）
- 结构化结果归一化：统一输出 `summary / findings / next_steps`
- 观测字段：`task_count / requested_concurrency / effective_concurrency / concurrency_reason`
- Trace 展示：Playground Runtime Trace 中可直接看到 `tasks / conc / reason`

注意：这是“可自动调用”的能力，不是“强制一定调用”。是否真的创建子 Agent 由提示词策略 + 模型决策决定。

## 2. 适用入口与默认行为

### 2.1 Agent 管理（Managed Agent）

- 可用：是
- 默认：`EnableSubAgentTool=true`（托管 Agent 注入时开启）
- 是否自动创建：取决于该 Agent 的 `systemPrompt` 是否引导拆分

### 2.2 Agent Playground

- 可用：是
- 默认：通过托管 Agent 路径可用（同 Agent 管理）
- 是否自动创建：取决于托管 Agent 提示词与模型决策

### 2.3 规则链 AgentHarness 节点

- 可用：是
- 默认：`enableSubAgentTool=true`
- 特点：节点默认 `systemPrompt` 已内置拆分策略，更容易触发自动委派

### 2.4 Code 助手（Chat）

- 可用：是
- 默认：走全量工具注册，包含 `run_sub_agent`
- 是否自动创建：取决于对话提示词与模型决策

## 3. 规则链节点配置说明

节点：`ai/agentHarness`

关键字段：

- `enableSubAgentTool`：是否启用 `run_sub_agent`（默认 `true`）
- `systemPrompt`：建议显式写明拆分策略（见下方模板）
- `maxIterations / maxToolCalls / toolTimeoutSecs`：主 Agent 工具循环与超时控制

推荐 `systemPrompt` 策略片段（可直接复用）：

```text
如果任务可拆成 2 个及以上互相独立的子任务，则调用 run_sub_agent，并优先传 sub_tasks_json；
如果任务强耦合，使用单 task；
如果任务很简单，直接完成，不要委派。
调用 run_sub_agent 时，要求子 Agent 返回 JSON：summary, findings, next_steps。
```

## 4. run_sub_agent 参数说明

## 必选/可选

- `task`（可选）：单个子任务文本
- `sub_tasks_json`（可选）：子任务数组 JSON 字符串，例如 `["任务A","任务B"]`
- `system_prompt`（可选）：覆盖子 Agent 系统提示词
- `managed_agent_id`（可选）：指定子 Agent 使用的托管 Agent 配置
- `max_iterations`（可选）：覆盖子 Agent 最大迭代轮次
- `max_tool_calls`（可选）：覆盖子 Agent 最大工具调用次数
- `tool_timeout_secs`（可选）：覆盖子 Agent 单次工具超时
- `max_concurrency`（可选）：并发度，范围建议 `1~8`；不传则自动估算

约束：

- `task` 与 `sub_tasks_json` 不能同时为空
- 存在递归深度保护，防止无限“子 Agent 套娃”

## 5. 调用示例

### 5.1 单任务

```json
{
  "task": "分析这个接口变更对前端的影响并给出改造建议"
}
```

### 5.2 批量任务（自动并发）

```json
{
  "sub_tasks_json": "[\"梳理影响面\",\"修改 service 层\",\"补充测试用例\"]"
}
```

### 5.3 批量任务（指定并发）

```json
{
  "sub_tasks_json": "[\"任务A\",\"任务B\",\"任务C\",\"任务D\"]",
  "max_concurrency": 3
}
```

## 6. 返回结果说明

`run_sub_agent` 返回 JSON（归一化后）常见字段：

- `summary`: 聚合总结
- `findings`: 发现列表
- `next_steps`: 下一步建议
- `sub_results`: 子任务明细（批量时）
- `task_count`: 子任务总数
- `requested_concurrency`: 请求并发（未传通常为 0）
- `effective_concurrency`: 实际并发
- `concurrency_reason`: 并发决策原因

`concurrency_reason` 常见值：

- `single_task_forced_1`
- `single_task_ignores_user_concurrency`
- `user_specified`
- `auto_estimated_by_task_count`
- `clamped_to_min_1`
- `clamped_to_max_8`

## 7. Trace 与可观测性

在 Agent Playground Runtime Trace 中，`TOOL_RESULT` 且工具名为 `run_sub_agent` 时：

- 事件消息会包含：子任务数、实际并发、原因
- 事件详情会带 metadata：
  - `subAgentTaskCount`
  - `subAgentEffectiveConcurrency`
  - `subAgentConcurrencyReason`

前端 Trace 面板会直接显示标签：

- `tasks X`
- `conc Y`
- `reason ...`

## 8. 常见问题（FAQ）

### Q1：为什么没有自动创建子 Agent？

通常是模型判断无需拆分，或提示词未明确引导拆分策略。建议在 `systemPrompt` 写清楚“何时拆分/何时并发”。

### Q2：为什么调用了 run_sub_agent 但失败？

优先检查：

- 当前入口是否启用了 `run_sub_agent`
- 子任务描述是否过于模糊
- 工具权限（Skill/MCP/Workspace）是否满足子任务执行需求
- 超时与迭代参数是否过紧

### Q3：为什么并发不是我传的值？

可能被边界保护裁剪（最小 1、最大 8），或单任务场景强制并发 1。可通过返回字段 `concurrency_reason` 判断。

## 9. 最佳实践

- 先拆“独立子任务”，再并发；不要把强耦合步骤硬并发
- 子任务粒度建议 5~20 分钟可完成，避免过大
- 先用自动并发，观察 `effective_concurrency` 后再精调
- 在托管 Agent 中固化一版稳定的拆分提示词模板

