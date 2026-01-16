# 项目架构文档

本文档详细描述了 ruleGoKratos 项目的目录结构和各个文件的作用。

## 项目概述

ruleGoKratos 是一个基于 Kratos 框架的规则引擎服务，集成了 rulego 规则引擎，提供规则链管理、组件管理、工作流执行等功能。项目采用分层架构设计，包含前端可视化编辑器和后端服务。

## 目录结构

```
ruleGoKratos/
├── api/                    # API 定义层
├── cmd/                    # 应用程序入口
├── configs/                # 配置文件
├── flowgram/               # 前端可视化编辑器
├── internal/               # 内部业务代码
├── sql/                    # 数据库脚本
├── third_party/            # 第三方依赖
├── volumes/                # Docker 数据卷
├── pgvectot_sql/           # 向量数据库初始化脚本
└── schema_json/            # JSON Schema 定义
```

---

## 详细目录说明

### `/api` - API 定义层

API 层使用 Protocol Buffers 定义服务接口，支持 gRPC 和 HTTP 两种协议。

#### `/api/rulego/v1/`

包含所有 API 的 protobuf 定义文件：

- **`rules.proto`** - 规则链相关 API 定义
  - 规则链的创建、查询、更新、删除
  - 规则链的执行和管理
  
- **`components.proto`** - 组件相关 API 定义
  - 组件注册、查询、管理
  - 组件使用规则管理
  
- **`md_workflow.proto`** - Markdown 工作流 API 定义
  - 工作流的创建和执行
  - 工作流模板管理
  
- **`run_log.proto`** - 运行日志 API 定义
  - 规则链执行日志查询
  - 日志统计和分析
  
- **`chat.proto`** - 聊天相关 API 定义
  - AI 对话接口
  - 智能助手功能
  
- **`error_reason.proto`** - 错误原因定义
  - 统一错误码和错误信息定义

**生成文件说明：**
- `*.pb.go` - protobuf 生成的 Go 代码
- `*_grpc.pb.go` - gRPC 服务端和客户端代码
- `*_http.pb.go` - HTTP 路由和处理器代码
- `*.pb.validate.go` - 参数验证代码

---

### `/cmd` - 应用程序入口

#### `/cmd/ruleGoKratos/`

应用程序的主入口和依赖注入配置：

- **`main.go`** - 应用程序主入口
  - 初始化配置、日志、数据库连接
  - 启动 gRPC 和 HTTP 服务器
  - 使用 wire 进行依赖注入
  
- **`wire.go`** - Wire 依赖注入配置
  - 定义依赖注入的 Provider 集合
  
- **`wire_gen.go`** - Wire 自动生成的依赖注入代码
  - 由 `wire.go` 自动生成，包含完整的依赖注入逻辑

---

### `/configs` - 配置文件

- **`config.yaml`** - 应用程序配置文件
  - 数据库连接配置
  - 服务器端口配置
  - 日志配置
  - 规则引擎配置

---

### `/internal` - 内部业务代码

遵循 Kratos 框架的分层架构设计：

#### `/internal/biz` - 业务逻辑层

业务逻辑层包含核心业务用例（UseCase）：

- **`biz.go`** - 业务层 Provider 集合定义
  - 使用 Wire 管理所有 UseCase 的依赖注入

- **`rule_chain.go`** - 规则链业务逻辑
  - 规则链的创建、更新、查询、删除
  - 规则链的执行和状态管理

- **`component_regulation.go`** - 组件规范业务逻辑
  - 组件规范的注册和管理

- **`component_use_rule.go`** - 组件使用规则业务逻辑
  - 组件使用规则的配置和管理

- **`md_workflow.go`** - Markdown 工作流业务逻辑
  - 工作流的解析和执行

- **`run_log.go`** - 运行日志业务逻辑
  - 日志的查询和统计

- **`agent.go`** / **`eino_agent.go`** - AI Agent 业务逻辑
  - AI 智能助手功能实现

- **`prompts.go`** - 提示词管理
  - AI 提示词的配置和管理

- **`logger.go`** - 日志工具
  - 日志记录辅助函数

- **`entity/`** - 业务实体定义
  - `rule_chain.go` - 规则链实体
  - `component_regulation.go` - 组件规范实体
  - `component_use_rule.go` - 组件使用规则实体
  - `md_workflow.go` - 工作流实体
  - `run_log.go` - 运行日志实体
  - `tpl.go` - 模板实体

- **`prompt/`** - 提示词模板文件
  - 包含各种 `.tpl` 模板文件，用于 AI 提示词生成

#### `/internal/data` - 数据访问层

数据访问层负责数据库操作和外部服务调用：

- **`data.go`** - 数据层 Provider 集合和数据库初始化
  - 数据库连接初始化
  - 规则引擎初始化
  - 规则链加载逻辑

- **`rule_chain.go`** - 规则链数据访问
  - 规则链的数据库 CRUD 操作

- **`component_regulation.go`** - 组件规范数据访问
  - 组件规范的数据库操作

- **`component_use_rule.go`** - 组件使用规则数据访问
  - 组件使用规则的数据库操作

- **`md_workflow.go`** - 工作流数据访问
  - 工作流的数据库操作

- **`run_log.go`** - 运行日志数据访问
  - 运行日志的数据库操作

- **`components/`** - 自定义组件实现
  - `yapi.go` - YApi 接口调用组件
  - 其他自定义 rulego 组件实现

- **`dao/`** - 数据访问对象
  - 封装数据库操作的 DAO 层

#### `/internal/service` - 服务层

服务层实现 gRPC 和 HTTP 接口：

- **`service.go`** - 服务层 Provider 集合定义

- **`rules.go`** - 规则链服务实现
  - 实现 `rules.proto` 定义的接口

- **`components.go`** - 组件服务实现
  - 实现 `components.proto` 定义的接口

- **`md_workflow.go`** - 工作流服务实现
  - 实现 `md_workflow.proto` 定义的接口

- **`run_log.go`** - 运行日志服务实现
  - 实现 `run_log.proto` 定义的接口

- **`chat.go`** - 聊天服务实现
  - 实现 `chat.proto` 定义的接口

#### `/internal/server` - 服务器层

服务器层负责启动和管理 gRPC 和 HTTP 服务器：

- **`server.go`** - 服务器 Provider 集合定义

- **`grpc.go`** - gRPC 服务器配置和启动
  - gRPC 中间件配置
  - 服务注册

- **`http.go`** - HTTP 服务器配置和启动
  - HTTP 路由配置
  - 中间件配置
  - Swagger 文档生成

#### `/internal/conf` - 配置定义

- **`conf.proto`** - 配置结构定义
  - 使用 protobuf 定义配置结构

- **`conf.pb.go`** - 配置结构生成的 Go 代码

---

### `/flowgram` - 前端可视化编辑器

基于 React 的流程图编辑器，用于可视化编辑规则链：

#### 主要目录：

- **`src/`** - 源代码目录
  - `app.tsx` - 应用主入口
  - `editor.tsx` - 编辑器主组件
  - `components/` - React 组件
    - `base-node/` - 基础节点组件
    - `add-node/` - 添加节点组件
    - `node-panel/` - 节点配置面板
    - `sidebar/` - 侧边栏组件
    - `testrun/` - 测试运行组件
    - `comment/` - 注释组件
    - `group/` - 分组组件
    - 等等
  - `nodes/` - 节点类型定义和实现
  - `form-components/` - 表单组件
  - `hooks/` - React Hooks
  - `services/` - API 服务调用
  - `utils/` - 工具函数
  - `styles/` - 样式文件
  - `typings/` - TypeScript 类型定义

- **`package.json`** - 前端依赖配置
- **`rsbuild.config.ts`** - 构建工具配置
- **`index.html`** - HTML 入口文件

---

### `/sql` - 数据库脚本

包含数据库表结构的 SQL 脚本：

- **`rule_chain.sql`** - 规则链表结构
- **`component_regulation.sql`** - 组件规范表结构
- **`component_use_rule.sql`** - 组件使用规则表结构
- **`md_workflow.sql`** - 工作流表结构
- **`run_log.sql`** - 运行日志表结构

---

### `/third_party` - 第三方依赖

包含第三方 protobuf 定义文件：

- **`google/`** - Google 官方 protobuf 定义
  - `api/` - Google API 定义
  - `protobuf/` - Protocol Buffers 标准定义

- **`openapi/`** - OpenAPI 相关定义

- **`validate/`** - 参数验证定义

- **`errors/`** - 错误定义

---

### `/pgvectot_sql` - 向量数据库

- **`init_vector.sql`** - 向量数据库初始化脚本
  - 用于支持向量搜索功能

---

### `/schema_json` - JSON Schema

包含 JSON Schema 定义文件，用于数据验证和文档生成。

---

### `/volumes` - Docker 数据卷

Docker 容器的数据持久化目录，包含数据库数据文件。

---

## 技术栈

### 后端
- **框架**: Kratos v2
- **规则引擎**: rulego v0.35.0
- **数据库**: PostgreSQL (使用 GORM)
- **依赖注入**: Google Wire
- **API 协议**: gRPC + HTTP (REST)
- **AI 能力**: 
  - CloudWeGo Eino
  - OpenAI API

### 前端
- **框架**: React
- **构建工具**: Rsbuild
- **流程图**: 自定义实现

### 基础设施
- **容器化**: Docker
- **数据库**: PostgreSQL
- **向量数据库**: pgvector (可选)

---

## 数据流

```
客户端请求
    ↓
HTTP/gRPC Server (internal/server)
    ↓
Service Layer (internal/service)
    ↓
Business Logic Layer (internal/biz)
    ↓
Data Access Layer (internal/data)
    ↓
Database (PostgreSQL)
```

---

## 核心功能模块

### 1. 规则链管理
- 规则链的创建、编辑、删除
- 规则链的可视化编辑（前端 flowgram）
- 规则链的执行和调试
- 规则链版本管理

### 2. 组件管理
- 组件注册和规范定义
- 组件使用规则配置
- 自定义组件开发（如 YApi 组件）

### 3. 工作流管理
- Markdown 格式的工作流定义
- 工作流执行引擎
- 工作流模板管理

### 4. 运行日志
- 规则链执行日志记录
- 日志查询和统计
- 错误追踪和调试

### 5. AI 智能助手
- 基于 Eino 的 AI 对话功能
- 智能提示和辅助

---

## 开发指南

### 添加新的 API

1. 在 `/api/rulego/v1/` 中定义新的 `.proto` 文件
2. 运行 `make api` 生成代码
3. 在 `/internal/service` 中实现服务接口
4. 在 `/internal/biz` 中实现业务逻辑
5. 在 `/internal/data` 中实现数据访问

### 添加新的组件

1. 在 `/internal/data/components/` 中实现组件
2. 组件需要实现 `types.Node` 接口
3. 在 `init()` 函数中注册组件到 rulego

### 数据库迁移

1. 在 `/sql/` 中创建或更新 SQL 脚本
2. 执行 SQL 脚本更新数据库结构

---

## 部署

### Docker 部署

```bash
# 构建镜像
docker build -t rulegokratos .

# 运行容器
docker-compose up -d
```

### 本地开发

```bash
# 安装依赖
make init

# 生成 API 代码
make api

# 运行服务
go run cmd/ruleGoKratos/main.go -conf configs
```

---

## 相关文档

- [Kratos 框架文档](https://go-kratos.dev/)
- [rulego 规则引擎文档](https://rulego.cc/)
- [项目 README](./README.md)
