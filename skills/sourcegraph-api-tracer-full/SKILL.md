---
name: sourcegraph-api-tracer-full
description: 用 Sourcegraph 追踪 HTTP 接口在前端/后端仓库中的定义、调用方与影响范围。Use when 用户问接口影响范围、接口被哪些服务调用、被哪些前端或后端服务调用、谁在调这个 API、某路径在哪个服务实现、路由/Handler 在哪里、需要批量用 Sourcegraph 查接口落点，或想只查后端/先给临时结果/跳过 Token 检查与预处理时；也可在用户显式使用 /sourcegraph-api-tracer-full 时调用。
category: devops
---

# Sourcegraph API Tracer

智能追踪 API 接口在前端或后端代码库中的位置。支持单个或批量处理多个接口。使用 Sourcegraph 搜索，集成 Playwright 自动化 Token 获取。

## 强制执行原则

<HARD-GATE>
本 Skill 是严格顺序工作流，不是可自由挑选的步骤清单。

- 必须从 `Step 1` 开始，严格按 `Step 1 -> Step 2 -> Step 3 -> Step 4 -> Step 5` 顺序执行
- 未完成当前步骤，不得进入下一步，不得并行绕过，不得先给“临时答案”后补前置步骤
- 即使用户只想先看后端、只查单个接口、路径看起来很简单、已知某个仓库大概率会命中，也不能跳过任何前置步骤
- 即使你认为 Token 大概率有效、可手写搜索词、或可直接进本地仓库搜索，也不能跳过 `Token 检查`、`LLM 预处理`、`Sourcegraph 搜索`、`build_routes`
- 某一步失败时，必须停在该步排障或向用户明确说明阻塞点；不得假定后续结果、不得伪造命中仓库、不得跳到后续步骤碰运气
- 违反步骤顺序就是违反本 Skill 的要求；“先查一点再补流程”也属于违规
</HARD-GATE>

## 何时使用本 Skill（触发场景）

在对话中出现以下意图时**应读取并遵循本 Skill**：

- **影响范围**：某接口改了会影响哪些地方、接口影响范围、上下游影响
- **调用方（统称）**：这个接口被哪些**服务**调用、谁调了这个 API、哪些地方会请求这个路径
- **前端调用**：接口被哪些**前端**（页面/应用/仓库）调用、H5/管理端/客户端是否在用
- **后端调用**：接口被哪些**后端**服务调用、服务间调用、其他微服务是否依赖该路由
- **落点定位**：某 `METHOD + path` 在后端哪里注册、前端哪里 `axios/fetch`、与批量追踪多个接口

若用户只给了路径而未给 HTTP 方法，应先与其确认方法或按常见用法（如 GET）先查再说明假设。

## 功能模块

1. **Token 管理**：启动时读 `config.json`，Token 过期（HTTP 401）时重新从 `config.json` 加载；通过 Playwright 获取新 Token 后写回 `config.json`
2. **后端追踪**：在后端代码库（Go/Java/Node 等）搜索 API 定义/路由注册/调用链
3. **前端追踪**：在前端代码库搜索 API 的 UI 入口、请求调用、接口定义
4. **Playwright Token 获取**：Node.js + Playwright，GitLab LDAP 登录并生成 Access Token，自动写回 `config.json`

## 前置依赖

### Node.js Playwright（`references/sourcegraph-token-playwright.js`）

```bash
cd references
npm install
npx playwright install chromium
```

### Sourcegraph CLI（可选）

参考: https://docs.sourcegraph.com/cli

### 配置文件 `references/config.json`

**唯一配置源**，所有脚本执行前均读取此文件；新 Token 也写回此文件。

首次使用时若不存在，参考以下结构创建（`references/config.json` 已在 `.gitignore` 中，勿提交密钥）：

```json
{
  "SOURCEGRAPH_URL": "https://sourcegraph.yc345.tv",
  "SOURCEGRAPH_TOKEN": "",
  "SOURCEGRAPH_REPO_FRONTEND": "teacher/fe/.*|frontend/.*",
  "SOURCEGRAPH_REPO_BACKEND":  "teacher/backend/.*|backend/.*",
  "SOURCEGRAPH_INCLUDE_FORKED": "1",
  "SOURCEGRAPH_CONTEXT_GLOBAL": "1",
  "SOURCEGRAPH_STREAM_PATH": "search/stream",
  "SOURCEGRAPH_DISPLAY_LIMIT": 1500,
  "WORK_DIR": "~/bizCompareWarehouse",
  "TRACE_LLM_API_KEY": "",
  "TRACE_LLM_BASE_URL": "https://api.openai.com",
  "TRACE_LLM_CHAT_PATH": "v1/chat/completions",
  "TRACE_LLM_MODEL": "gpt-4o-mini",
  "TRACE_USE_LLM": "1",
  "TRACE_DEBUG": "0",
  "AGENT_TIMEOUT": 300,
  "AGENT_MODEL": "auto",
  "MAX_RETRIES": 2,
  "LLM_PREPROCESS_TIMEOUT": 60,
  "SOURCEGRAPH_LDAP_USERNAME": "",
  "SOURCEGRAPH_LDAP_PASSWORD": "",
  "SOURCEGRAPH_GITLAB_HOST": "gitlab.yc345.tv",
  "PLAYWRIGHT_HEADLESS": "1"
}
```

**环境变量优先**：若某字段已在 shell 中 `export`，则以环境变量为准，不被 `config.json` 覆盖。

## 使用方式

### 单接口追踪

```
/sourcegraph-api-tracer-full GET /agency-qr/:qrId/detail
/sourcegraph-api-tracer-full POST /users/login
/sourcegraph-api-tracer-full DELETE /agency-qr/:qrId
```

### 批量追踪

在会话中列出多个接口，一次性处理：

```
/sourcegraph-api-tracer-full
GET /agency-qr/:qrId/detail
POST /users/login
PUT /products/:id
DELETE /orders/:id
```

### 指定代码库位置

```
/sourcegraph-api-tracer-full GET /api/users --backend=/path/to/backend --frontend=/path/to/frontend
```

### 仅获取/刷新 Token

```
/sourcegraph-api-tracer-full --refresh-token
```

---

## 工作流程（必须按顺序执行）

### Step 1: Token 检查与获取

1. 所有脚本在启动时调用 `config.load_config()`，**每次调用均重新读取** `references/config.json`，确保获取最新 Token。
2. **HTTP 401 自动重试**：`sourcegraph_search.py` 遇到 401 时调用 `refresh_token_from_config_after_401()`——重新读 `config.json`，若其中 Token 与当前进程不同则更新并**重试一次** stream 请求。这处理了「Playwright 已把新 Token 写回 config.json，但本进程仍持有旧 Token」的情况。
3. **Playwright 刷新**：运行 `node sourcegraph-token-playwright.js` 后，新 Token 通过 `writeTokenToConfigJson()` 原子写入 `config.json`（tmp 文件 + rename），下一次 `load_config()` 或 401 重试即可取到。
4. **os.environ 优先**：若在 shell 中显式 `export SOURCEGRAPH_TOKEN=xxx`，则该值优先于 `config.json`，`refresh_token_from_config_after_401()` 在 Token 相同时也会跳过（避免死循环）。
5. 若 401 重试后仍失败，报错并提示用户运行 `node sourcegraph-token-playwright.js` 刷新。

**进入 Step 2 的前提：**

- 已完成 `load_config()`，当前执行上下文持有最新配置
- 若搜索阶段出现 401，已按规则执行“重读 `config.json` 并重试一次”
- 若仍无法获得有效 Token，必须停止后续步骤并向用户说明；不得继续后面的预处理、搜索或分析

### Step 2: LLM 预处理（`llm_preprocess.py`）

分析 URL 路径结构，生成最优的 Sourcegraph 搜索 pattern：
- 静态路径（如 `/invoice/entry/xlsx`）→ `patternType: literal`
- 含动态参数（如 `/room/:roomRef/order/:orderId`）→ `patternType: regexp`，同时生成正则通配版与路由定义版两个 pattern

**进入 Step 3 的前提：**

- 已完成路径分析并产出 pattern
- 动态路径已完成参数位处理，不能用“肉眼直接写关键词”替代
- 即使只查后端，也必须先完成该步，因为后续搜索依赖这里生成的 pattern

### Step 3: Sourcegraph 搜索（`sourcegraph_search.py`）

用 pattern 查询 Sourcegraph stream API，返回命中的仓库名列表（支持 `frontend` / `backend` 范围限定）。

**进入 Step 4 的前提：**

- 已使用 Step 2 的产出执行 Sourcegraph 搜索
- 已得到命中的仓库列表，或确认无命中
- 不允许跳过该步直接去本地仓库 `rg` / `grep` / agent 分析，也不允许仅凭经验猜测“应该是哪个仓库”

### Step 4: 生成 routes（`build_routes.py`）

将仓库列表组装为结构化 JSON，包含 `projectName`、`projectGitRepoUrl`、`projectType`、`route` 字段。

**进入 Step 5 的前提：**

- 已完成结构化 `routes` 生成
- 若搜索无命中，应在此处停止并报告“无命中”；不得虚构可能仓库继续分析
- 即使用户只要后端结果，也只能在这一步之后基于 `projectType` 缩小分析范围，不能跳过本步

### Step 5: Clone/更新 + Cursor agent 分析（`trace_api.py`）

对每个命中仓库：
1. 若本地不存在则 `git clone`；否则 `git pull` 更新到最新
2. 在仓库目录下调用 `agent --print --trust "<分析 prompt>"` 让 Cursor CLI 分析代码
3. 根据项目类型（前端/后端）使用不同 prompt：
   - **后端**：定位路由注册、Handler、调用链、文件清单
   - **前端**：定位 UI 入口、所属后台、触发方式、路由与权限速查
4. 分析结果保存到 `WORK_DIR/docs/<接口slug>_<项目名>.md`

**Step 5 补充约束 — 搜索关键词传递规则（强制，禁止违反）：**

-传给 Cursor/Codex 的搜索关键词**只允许使用 `route` 字段的原始值**，不得拆分、截取、拼接或添加任何其他词
- 示例 ✅：`route = "/goods/:group/list"` → 搜索 `/goods/:group/list`
- 示例 ❌：搜索 `/goods/:group/list` + `/goods/` + `:group/list`（拆分）
- 示例 ❌：搜索 `/goods/:group/list` + `/list`（添加后缀）
- 示例 ❌：搜索 `/api/goods/:group/list`（添加前缀）
- 如路径含动态参数（如 `:group`），保留原样，不得自行替换为具体值或移除
- prompt 中只需给出 route 值，由 Cursor 自行决定搜索策略（如正则、精确匹配等），**不得**在 prompt 里额外规定"搜索正则应该是xxx"

**完成判定：**

- 每个命中仓库都已按 `projectType` 完成分析
- 输出结论来自已完成的仓库分析，而不是猜测
- 只有在完成该步后，才允许向用户汇总“后端落点 / 前端入口 / 影响范围”结果

## 允许缩小范围，不允许跳过步骤

- 可以通过 `frontend` / `backend` 范围限定，缩小搜索与分析范围
- 可以只汇报后端结果，但前提仍然是完整执行 `Step 1 -> Step 5`
- “只查后端”代表缩小 `Step 3` 和 `Step 5` 的范围，不代表可跳过 `Step 1`、`Step 2`、`Step 4`

## 中断与恢复规则

- 任一步失败时，停在该步处理，不得假定后续结果
- 恢复时只能从“最近一个已完成步骤”继续；若上游输入已变更，则从受影响的最早步骤重新开始
- 典型示例：
  - Token 刷新后，至少重新执行 `Step 1` 后再继续
  - URL/path 变化后，必须重新执行 `Step 2` 以及其后的所有步骤
  - Sourcegraph 命中仓库变化后，必须重新执行 `Step 4` 和 `Step 5`

## 常见跳步借口（禁止）

| 借口                             | 规则                                                               |
| -------------------------------- | ------------------------------------------------------------------ |
| “用户只想先看后端”               | 可以限制范围到后端，但仍必须完整执行 `Step 1 -> Step 5`。          |
| “Token 大概率还有效”             | 仍必须先读取最新配置并按 401 规则处理，不能靠猜测跳过。            |
| “路径很简单，我直接写搜索词就行” | 不允许。动态参数和路由定义 pattern 必须经过 `LLM 预处理` 生成。    |
| “我已经知道大概率是哪个仓库”     | 不允许。必须先做 `Sourcegraph 搜索` 与 `build_routes` 确认命中集。 |
| “先给个临时答案，后面再补步骤”   | 不允许。未完成完整链路前，不得输出结论性答案。                     |

## Red Flags - STOP

- 试图从 `Step 3`、`Step 4` 或 `Step 5` 开始
- 试图直接进入本地仓库搜索来替代 `Sourcegraph 搜索`
- 试图先输出后端/前端结论，再回补 `Token 检查`、`LLM 预处理` 或 `build_routes`
- 试图用“用户很急”“只想先看一部分”“我大概知道答案”作为跳步理由

**出现以上任一情况：停止当前动作，回到最近一个未完成的前置步骤。**

---

## Playwright Token 获取流程

### 脚本位置

`references/sourcegraph-token-playwright.js`（依赖同目录 `package.json`：`npm install` 后运行）

**Token 生成后自动写回 `references/config.json`，无需手动粘贴。**

### 使用方法

**交互模式**（从 `config.json` 读取 LDAP 账号，缺省再提示）:

```bash
cd references
node sourcegraph-token-playwright.js
```

**仅指定 Token 名称**（账号密码来自 `config.json`）:

```bash
node sourcegraph-token-playwright.js my-token
```

**命令行参数模式**:

```bash
node sourcegraph-token-playwright.js <USERNAME> <PASSWORD> [TOKEN_NAME] [EXPIRES_AT]
```

### 参数说明

| 参数       | 说明                  | 默认值                                                  |
| ---------- | --------------------- | ------------------------------------------------------- |
| USERNAME   | GitLab LDAP 用户名    | 读 `config.json` 的 `SOURCEGRAPH_LDAP_USERNAME`，再交互 |
| PASSWORD   | GitLab LDAP 密码      | 读 `config.json` 的 `SOURCEGRAPH_LDAP_PASSWORD`，再交互 |
| TOKEN_NAME | Access Token 名称     | `cli-token`                                             |
| EXPIRES_AT | 过期日期 (YYYY-MM-DD) | 约 3 年后                                               |

### 输出

- 新 Token：**直接原子写入 `references/config.json`**
- 日志：控制台
- 失败截图：`sourcegraph-token-debug.png`（当前工作目录）

### 流程步骤

1. 读取 `config.json` 中的 LDAP 账号等参数
2. 启动 Playwright 浏览器（headless）
3. 导航至 Sourcegraph 登录页
4. 点击 "GitLab LDAP" 登录方式
5. 填写 GitLab 用户名/密码
6. 授权 Sourcegraph 访问（若出现授权页）
7. 提取页面中的 Access Token
8. **写回 `config.json`**，关闭浏览器

---

## 自定义修改

直接编辑 `references/config.json`，例如适配其他 Sourcegraph 实例：

```json
{
  "SOURCEGRAPH_URL": "https://your-sourcegraph.example.com",
  "SOURCEGRAPH_GITLAB_HOST": "gitlab.yourcompany.com"
}
```

---

## 常见问题

**Q: Token 过期了怎么办？**
A: 分步执行：
1. 进入目录：`cd references`
2. 直接运行脚本：`node sourcegraph-token-playwright.js`

新 Token 会自动写回 `config.json`；下一次搜索即生效（不需要重启任何进程）。某些执行器不接受 `cd ... && node ...` 这种组合命令时，就按这两步分别执行。

**Q: 搜索时仍然 401？**
A: `sourcegraph_search.py` 遇 401 会自动重读 `config.json` 重试一次。若仍失败，说明 `config.json` 中的 Token 也已失效，需再次运行 Playwright 脚本刷新。

**Q: 如何处理需要 MFA 的 GitLab 账号？**
A: Playwright 脚本需扩展 MFA 输入步骤，或使用 Service Account（无 MFA）登录。

**Q: 支持哪些框架的后端路由识别？**
A: Go (chi, gin, echo)、Java (Spring Boot)、Node.js (Express, Koa)、Python (Flask, FastAPI)。

**Q: 批量追踪有数量限制吗？**
A: 建议单次不超过 20 个接口，超过时分批处理。
