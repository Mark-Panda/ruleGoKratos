# Agent 子 Agent 快速上手（简版）

这是一份面向日常使用者的简版说明，重点是“怎么开、怎么用、怎么看效果”。

详细技术说明请看：`docs/agent-subagent-usage.md`

## 1) 你能在哪些地方用

- Agent 管理（托管 Agent）
- Agent Playground
- 规则链 `AgentHarness` 节点
- Code 助手

## 2) 一分钟开启

## 规则链（推荐先用这里验证）

1. 打开 `AgentHarness` 节点
2. 确认开关 `启用 run_sub_agent` = 开启
3. 在系统提示词里加一句：
   - “遇到可拆分任务时，优先调用 run_sub_agent”
4. 运行一次任务观察结果

## Agent 管理 / Playground

1. 打开对应 Agent 配置
2. 在系统提示词里加入拆分策略（示例见下）
3. 保存后在 Playground 跑一次任务

## 3) 推荐提示词（可复制）

```text
当任务可以拆成 2 个及以上独立子任务时，请调用 run_sub_agent，并优先使用 sub_tasks_json。
当任务强耦合时，使用单 task。
当任务很简单时，直接完成，不要委派。
调用 run_sub_agent 时，要求子 Agent 返回 JSON：summary, findings, next_steps。
```

## 4) 常见调用方式

## 单任务

```json
{
  "task": "分析需求并给出实现建议"
}
```

## 批量任务（自动并发）

```json
{
  "sub_tasks_json": "[\"拆解需求\",\"输出接口草案\",\"输出测试点\"]"
}
```

## 批量任务（手动并发）

```json
{
  "sub_tasks_json": "[\"任务A\",\"任务B\",\"任务C\"]",
  "max_concurrency": 2
}
```

## 5) 怎么判断是否生效

看运行结果里是否出现：

- `task_count`
- `effective_concurrency`
- `concurrency_reason`

在 Playground 的 Trace 里会直接显示：

- `tasks X`
- `conc Y`
- `reason ...`

## 6) 常见问题

## 没有自动创建子 Agent

通常是提示词没引导，或模型认为任务不需要拆分。先把“何时拆分”的规则写清楚。

## 并发不符合预期

可能被系统保护裁剪（太小会拉到 1，太大上限为 8），看 `concurrency_reason` 即可。

## 子任务失败

先看 Trace 的 `TOOL_RESULT`，再检查：

- 子任务描述是否清楚
- 权限是否足够（Skill/MCP/Workspace）
- 超时与迭代参数是否过小

