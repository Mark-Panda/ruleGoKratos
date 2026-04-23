# 项目开发规范

## 管理端 API 规范

- **强制**：新增或修改管理端接口时，必须先在 `api/rulego/v1/admin.proto` 中定义 `rpc` 与 `google.api.http` 路由映射。
- **强制**：禁止在 `internal/service` 或 `internal/server` 中新增绕过 proto 的手写 REST 路由作为正式接口。
- **强制**：接口契约字段统一使用 proto message，保持前后端与网关生成代码一致。
- **建议**：完成 proto 变更后，优先执行针对性生成（如 `admin.proto`），避免被其他无关 proto 问题阻塞。

## 迁移与兼容

- 已存量的手写路由可逐步迁移到 proto 契约，不要求一次性全量重构。
- 新需求必须遵循“先 proto，后实现”的流程，避免出现双轨接口。
