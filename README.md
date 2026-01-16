# ruleGoKratos

基于 Kratos 框架的规则引擎服务，集成了 rulego 规则引擎，提供规则链管理、组件管理、工作流执行等功能。

## 项目特性

- 🎯 **规则链管理**: 可视化编辑和管理规则链
- 🔧 **组件系统**: 丰富的组件库和自定义组件支持
- 📝 **工作流引擎**: 支持 Markdown 格式的工作流定义
- 📊 **运行日志**: 完整的执行日志记录和查询
- 🤖 **AI 智能助手**: 基于 Eino 的 AI 对话功能
- 🎨 **可视化编辑器**: 基于 React 的流程图编辑器

## 技术栈

### 后端
- **框架**: Kratos v2
- **规则引擎**: rulego v0.35.0
- **数据库**: PostgreSQL (GORM)
- **依赖注入**: Google Wire
- **API 协议**: gRPC + HTTP (REST)
- **AI 能力**: CloudWeGo Eino, OpenAI API

### 前端
- **框架**: React
- **构建工具**: Rsbuild
- **流程图**: 自定义实现

## 快速开始

### 前置要求

- Go 1.24+
- PostgreSQL
- Node.js 18+ (前端开发)

### 安装依赖

```bash
# 安装 Kratos CLI
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest

# 安装 Wire
go get github.com/google/wire/cmd/wire

# 下载项目依赖
make init
```

### 生成代码

```bash
# 生成 API 代码 (pb.go, grpc, http, validate, swagger)
make api

# 生成 Wire 依赖注入代码
cd cmd/ruleGoKratos
wire

# 生成所有文件
make all
```

### 配置数据库

1. 创建 PostgreSQL 数据库
2. 执行 SQL 脚本初始化表结构：
   ```bash
   psql -U postgres -d your_database -f sql/rule_chain.sql
   psql -U postgres -d your_database -f sql/component_regulation.sql
   psql -U postgres -d your_database -f sql/component_use_rule.sql
   psql -U postgres -d your_database -f sql/md_workflow.sql
   psql -U postgres -d your_database -f sql/run_log.sql
   ```

3. 更新 `configs/config.yaml` 中的数据库配置

### 运行服务

```bash
# 开发模式
go run cmd/ruleGoKratos/main.go -conf configs

# 构建并运行
go build -o ./bin/ruleGoKratos ./cmd/ruleGoKratos
./bin/ruleGoKratos -conf ./configs
```

### Docker 部署

```bash
# 使用 Docker Compose
docker-compose up -d

# 或手动构建和运行
docker build -t rulegokratos .
docker run --rm -p 8000:8000 -p 9000:9000 -v $(pwd)/configs:/data/conf rulegokratos
```

## 项目文档

- 📖 [架构文档](./ARCHITECTURE.md) - 详细的架构说明和设计文档
- 📁 [目录结构](./DIRECTORY_STRUCTURE.md) - 目录结构快速参考

## 项目结构

```
ruleGoKratos/
├── api/              # API 定义层 (Protocol Buffers)
├── cmd/              # 应用程序入口
├── configs/          # 配置文件
├── flowgram/         # 前端可视化编辑器
├── internal/         # 内部业务代码
│   ├── biz/         # 业务逻辑层
│   ├── data/        # 数据访问层
│   ├── service/     # 服务层
│   └── server/      # 服务器层
├── sql/              # 数据库脚本
└── third_party/      # 第三方依赖
```

详细说明请参考 [目录结构文档](./DIRECTORY_STRUCTURE.md)。

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

详细开发指南请参考 [架构文档](./ARCHITECTURE.md)。

## AI 模型支持

项目支持以下 AI 模型：

- `doubao-seed-1-6-thinking-250715` - 多模态深度思考模型，支持结构化输出
- `doubao-1-5-thinking-pro-250415` - 深度思考模型

## 许可证

详见 [LICENSE](./LICENSE) 文件。