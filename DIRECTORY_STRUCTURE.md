# 目录结构快速参考

本文档提供项目目录结构的快速参考，详细说明请参考 [ARCHITECTURE.md](./ARCHITECTURE.md)。

## 根目录文件

| 文件/目录 | 说明 |
|---------|------|
| `README.md` | 项目说明文档 |
| `ARCHITECTURE.md` | 详细架构文档 |
| `DIRECTORY_STRUCTURE.md` | 本文件 - 目录结构快速参考 |
| `go.mod` / `go.sum` | Go 模块依赖管理 |
| `Makefile` | 构建和开发命令 |
| `Dockerfile` | Docker 镜像构建文件 |
| `docker-compose.yml` | Docker Compose 配置 |
| `openapi.yaml` | OpenAPI 规范定义 |
| `LICENSE` | 许可证文件 |

## 主要目录

### `/api` - API 定义层
```
api/
└── rulego/v1/          # API 版本 1
    ├── *.proto         # Protocol Buffers 定义文件
    ├── *.pb.go         # 生成的 Go 代码
    ├── *_grpc.pb.go    # gRPC 代码
    └── *_http.pb.go    # HTTP 代码
```

**主要文件：**
- `rules.proto` - 规则链 API
- `components.proto` - 组件 API
- `md_workflow.proto` - 工作流 API
- `run_log.proto` - 日志 API
- `chat.proto` - 聊天 API
- `error_reason.proto` - 错误定义

### `/cmd` - 应用程序入口
```
cmd/ruleGoKratos/
├── main.go          # 主程序入口
├── wire.go          # Wire 依赖注入配置
└── wire_gen.go      # Wire 生成的代码
```

### `/configs` - 配置文件
```
configs/
└── config.yaml      # 应用配置文件
```

### `/internal` - 内部业务代码

#### `/internal/biz` - 业务逻辑层
```
internal/biz/
├── biz.go                    # Provider 集合
├── rule_chain.go             # 规则链业务逻辑
├── component_regulation.go   # 组件规范业务逻辑
├── component_use_rule.go     # 组件使用规则业务逻辑
├── md_workflow.go            # 工作流业务逻辑
├── run_log.go                # 日志业务逻辑
├── agent.go                  # AI Agent 业务逻辑
├── eino_agent.go             # Eino Agent 实现
├── prompts.go                # 提示词管理
├── logger.go                 # 日志工具
├── entity/                   # 业务实体
│   ├── rule_chain.go
│   ├── component_regulation.go
│   ├── component_use_rule.go
│   ├── md_workflow.go
│   ├── run_log.go
│   └── tpl.go
└── prompt/                   # 提示词模板
    └── *.tpl
```

#### `/internal/data` - 数据访问层
```
internal/data/
├── data.go                   # Provider 集合和数据库初始化
├── rule_chain.go             # 规则链数据访问
├── component_regulation.go   # 组件规范数据访问
├── component_use_rule.go     # 组件使用规则数据访问
├── md_workflow.go            # 工作流数据访问
├── run_log.go                # 日志数据访问
├── components/               # 自定义组件
│   ├── yapi.go              # YApi 组件
│   └── ...
└── dao/                      # 数据访问对象
    └── ...
```

#### `/internal/service` - 服务层
```
internal/service/
├── service.go        # Provider 集合
├── rules.go          # 规则链服务
├── components.go     # 组件服务
├── md_workflow.go    # 工作流服务
├── run_log.go        # 日志服务
└── chat.go           # 聊天服务
```

#### `/internal/server` - 服务器层
```
internal/server/
├── server.go    # Provider 集合
├── grpc.go      # gRPC 服务器
└── http.go      # HTTP 服务器
```

#### `/internal/conf` - 配置定义
```
internal/conf/
├── conf.proto   # 配置结构定义
└── conf.pb.go   # 生成的配置代码
```

### `/flowgram` - 前端可视化编辑器
```
flowgram/
├── src/                    # 源代码
│   ├── app.tsx            # 应用入口
│   ├── editor.tsx         # 编辑器主组件
│   ├── components/        # React 组件
│   ├── nodes/             # 节点类型
│   ├── form-components/   # 表单组件
│   ├── hooks/             # React Hooks
│   ├── services/          # API 服务
│   ├── utils/             # 工具函数
│   └── styles/            # 样式文件
├── package.json           # 依赖配置
├── rsbuild.config.ts      # 构建配置
└── index.html             # HTML 入口
```

### `/sql` - 数据库脚本
```
sql/
├── rule_chain.sql            # 规则链表
├── component_regulation.sql  # 组件规范表
├── component_use_rule.sql    # 组件使用规则表
├── md_workflow.sql           # 工作流表
└── run_log.sql               # 日志表
```

### `/third_party` - 第三方依赖
```
third_party/
├── google/        # Google protobuf 定义
├── openapi/       # OpenAPI 定义
├── validate/      # 验证定义
└── errors/        # 错误定义
```

### 其他目录

| 目录 | 说明 |
|-----|------|
| `/pgvectot_sql` | 向量数据库初始化脚本 |
| `/schema_json` | JSON Schema 定义 |
| `/volumes` | Docker 数据卷（数据库数据） |

## 文件命名规范

### Go 文件
- `*_repo.go` - 数据仓库实现
- `*_usecase.go` - 业务用例实现
- `*_service.go` - 服务实现
- `*_server.go` - 服务器实现

### Proto 文件
- `*.proto` - Protocol Buffers 定义
- `*.pb.go` - 生成的 Go 代码
- `*_grpc.pb.go` - gRPC 代码
- `*_http.pb.go` - HTTP 代码
- `*.pb.validate.go` - 验证代码

### 前端文件
- `*.tsx` - React 组件
- `*.ts` - TypeScript 代码
- `*.less` / `*.css` - 样式文件

## 依赖关系

```
main.go
  ↓
wire_gen.go (依赖注入)
  ↓
server (grpc/http)
  ↓
service (业务服务)
  ↓
biz (业务逻辑)
  ↓
data (数据访问)
  ↓
database / external services
```

## 快速查找

### 添加新 API
1. 编辑 `/api/rulego/v1/*.proto`
2. 运行 `make api`
3. 实现 `/internal/service/*.go`
4. 实现 `/internal/biz/*.go`
5. 实现 `/internal/data/*.go`

### 添加新组件
1. 在 `/internal/data/components/` 创建组件文件
2. 实现 `types.Node` 接口
3. 在 `init()` 中注册组件

### 修改数据库
1. 更新 `/sql/*.sql`
2. 执行 SQL 脚本
3. 更新 `/internal/data/dao/` 中的 DAO

### 修改前端
1. 编辑 `/flowgram/src/` 中的文件
2. 运行前端开发服务器

## 相关文档

- [ARCHITECTURE.md](./ARCHITECTURE.md) - 详细架构文档
- [README.md](./README.md) - 项目说明
