#!/usr/bin/env python3
"""
API Route Tracer 主入口。
步骤：LLM 预处理 → Sourcegraph 搜索 → 生成 routes → 对每个仓库 clone/更新 → 调用 agent 分析。

用法: python3 trace_api.py <url> <method> [frontend|backend]
环境变量:
  TRACE_STOP_AFTER=llm|sourcegraph|repos  在此步骤后停止
  TRACE_USE_LLM=0                         跳过 LLM，用字面量
  TRACE_DEBUG=1                           调试输出
  AGENT_MODEL=<model>                     指定 agent 模型
"""

import argparse
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Optional

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from config import load_config
from build_routes import build_routes
from sourcegraph_search import search as sourcegraph_search, SourcegraphUnavailableError

BLUE = "\033[0;34m"
GREEN = "\033[0;32m"
YELLOW = "\033[1;33m"
RED = "\033[0;31m"
NC = "\033[0m"


def log_info(msg: str) -> None:
    print(f"{BLUE}ℹ{NC} {msg}", flush=True)


def log_success(msg: str) -> None:
    print(f"{GREEN}✅{NC} {msg}", flush=True)


def log_warning(msg: str) -> None:
    print(f"{YELLOW}⚠️{NC} {msg}", flush=True)


def log_error(msg: str) -> None:
    print(f"{RED}❌{NC} {msg}", flush=True)


def debug_log(msg: str) -> None:
    if os.environ.get("TRACE_DEBUG") == "1":
        print(f"{YELLOW}[DEBUG]{NC} {msg}", flush=True, file=sys.stderr)


def run_llm_preprocess(url: str, method: str, timeout_s: int) -> dict:
    """调用 llm_preprocess.py，返回 {patternType, patterns, explanation} 或 fallback。"""
    script = SCRIPT_DIR / "llm_preprocess.py"
    if not script.is_file():
        return {"patternType": "literal", "patterns": [url], "explanation": "无 llm_preprocess.py"}
    try:
        out = subprocess.run(
            [sys.executable, str(script), url, method.upper()],
            capture_output=True,
            text=True,
            timeout=timeout_s,
            cwd=str(SCRIPT_DIR),
        )
        raw = (out.stdout or "").strip()
        for line in raw.splitlines():
            line = line.strip()
            if line.startswith("{") and "patternType" in line:
                try:
                    data = json.loads(line)
                    if data.get("patternType") in ("literal", "regexp") and data.get("patterns"):
                        log_info(f"LLM 预处理结果: {json.dumps(data, ensure_ascii=False)}")
                        return data
                except json.JSONDecodeError:
                    pass
        if raw:
            try:
                data = json.loads(raw.splitlines()[-1])
                if data.get("patternType") in ("literal", "regexp") and data.get("patterns"):
                    log_info(f"LLM 预处理结果: {json.dumps(data, ensure_ascii=False)}")
                    return data
            except json.JSONDecodeError:
                pass
    except subprocess.TimeoutExpired:
        log_warning("LLM 预处理超时，使用 URL 字面量")
    except Exception as e:
        debug_log(f"llm_preprocess 调用异常: {e}")
    return {"patternType": "literal", "patterns": [url], "explanation": "fallback 字面量"}


def _normalize_git_url(url: str) -> str:
    return (url or "").strip().rstrip("/").removesuffix(".git")


def _run_git(service_path: Path, args: list[str], timeout: int) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["git", *args],
        cwd=str(service_path),
        capture_output=True,
        text=True,
        timeout=timeout,
    )


def _detect_default_branch(service_path: Path) -> Optional[str]:
    for branch in ("main", "master"):
        result = _run_git(service_path, ["rev-parse", "--verify", f"origin/{branch}"], timeout=10)
        if result.returncode == 0:
            return branch
    result = _run_git(service_path, ["branch", "--show-current"], timeout=10)
    branch = (result.stdout or "").strip()
    return branch or None


def ensure_repo(work_dir: str, local_repo_dir: str, git_url: str, display_name: str) -> Optional[Path]:
    """若目录存在则校验 origin 并更新；否则 clone。成功时返回仓库路径。"""
    work = Path(work_dir)
    work.mkdir(parents=True, exist_ok=True)
    service_path = work / local_repo_dir

    if service_path.exists():
        if not (service_path / ".git").exists():
            log_error(f"目录已存在但不是 git 仓库，无法安全复用: {service_path}")
            return None
        try:
            remote = _run_git(service_path, ["remote", "get-url", "origin"], timeout=10)
            if remote.returncode != 0:
                log_error(f"读取 origin 失败: {display_name}")
                return None
            current_origin = _normalize_git_url(remote.stdout)
            expected_origin = _normalize_git_url(git_url)
            if current_origin != expected_origin:
                log_error(
                    f"本地仓库 origin 与目标不一致，已停止以避免串仓: "
                    f"{current_origin} != {expected_origin}"
                )
                return None

            fetch = _run_git(service_path, ["fetch", "origin", "--prune"], timeout=60)
            if fetch.returncode != 0:
                log_error(f"git fetch 失败: {display_name}")
                return None

            branch = _detect_default_branch(service_path)
            if branch:
                checkout = _run_git(service_path, ["checkout", branch], timeout=15)
                if checkout.returncode != 0:
                    log_error(f"git checkout {branch} 失败: {display_name}")
                    return None
                pull = _run_git(service_path, ["pull", "--ff-only", "origin", branch], timeout=60)
                if pull.returncode != 0:
                    log_error(f"git pull origin {branch} 失败: {display_name}")
                    return None
            else:
                log_warning(f"未检测到默认分支，保留当前分支继续分析: {display_name}")
        except subprocess.TimeoutExpired:
            log_error(f"更新仓库超时: {display_name}")
            return None
        log_success("代码已更新")
        return service_path

    if not git_url or git_url == "null":
        log_error(f"无 Git 地址，无法 clone: {display_name}")
        return None
    log_info("正在 clone 仓库...")
    r = subprocess.run(["git", "clone", git_url, local_repo_dir], cwd=str(work), timeout=120)
    if r.returncode != 0:
        log_error("clone 失败")
        return None
    log_success("clone 成功")
    return service_path


def _api_slug(url: str, method: str) -> str:
    s = (url or "").strip("/").replace("/", "_")
    s = "".join(c if c.isalnum() or c in "_" else "_" for c in s)
    return (s or "api")[:80] + "_" + (method or "get").upper()


def copy_generated_doc(service_path: Path, output_md: Path, expected_name: str) -> bool:
    for src in [service_path / expected_name, service_path / "docs" / expected_name]:
        if src.is_file():
            try:
                shutil.move(str(src), str(output_md))
                log_success(f"已移动到: {output_md}")
                return True
            except OSError as e:
                log_error(f"移动失败: {e}")
                return False
    log_warning(f"未在仓库内找到生成的文件（{expected_name}），请手动移动")
    return False


def _build_agent_prompt(url: str, method: str, project_type: str,
                        output_md_path: Optional[Path] = None) -> str:
    if output_md_path:
        output_instruction = (
            f"\n请将分析结果汇总成一份 markdown 文件，并保存到以下路径（绝对路径）：\n"
            f"{output_md_path.resolve()}"
        )
    else:
        output_instruction = "\n请将分析结果汇总成 markdown 给我。"

    if project_type == "backend":
        return f"""这是一个后端项目。请帮我分析 API endpoint '{url}' ({method} 方法) 在本仓库中的定义与调用情况。

请整理并输出：
1. **接口定义位置**：该接口在本项目中的实现文件、类/方法、路由注册位置（如 Controller、Router、路由表等）。
2. **调用该接口的服务/模块**：本项目内或依赖中哪些服务、模块、定时任务、消息消费者等会调用该接口（若有）。
3. **文件位置清单**：列出所有相关文件路径及简要说明（定义处、调用处、配置处等）。
4. **调用链概要**：入口 → 中间层 → 该 API 的典型调用路径（若适用）。
{output_instruction}"""

    return f"""这是一个{project_type}项目。请分析 API endpoint '{url}' ({method} 方法) 在本项目中的 UI 入口，并严格按以下格式输出。

**分析要点**（分析时用，不必逐条罗列）：
- 在哪个页面/组件/功能模块使用？通过什么触发？（按钮、页面加载、表单提交等）
- 所属后台名称（如「渠道后台」「运营后台」）、路由前缀、菜单所在位置
- 路由 path、name、权限标识（若有）

**输出格式要求**（必须遵守）：

1. **标题**：一行，如：`# 接口使用情况：\`{url}\` ({method.upper()})`

2. **项目说明**：一段简短说明，包含：项目/后台名称、路由前缀、菜单所在位置。

3. **主表**：一张 Markdown 表格，表头为「| 接口 | 所属后台 | UI 入口 |」，本接口占一行。
   - 接口列：`**{method.upper()}** \`{url}\``
   - 所属后台：如「仅 运营后台」或「渠道后台 + 运营后台」
   - UI 入口：一句话描述入口（菜单路径 + 页内入口如按钮名）

4. **路由与权限速查**：二级标题 `## 路由与权限速查`，按后台分子节，每块列出相关路由 path、name、权限、菜单层级。

不要输出「使用位置」「触发方式」「小结」等长章节，不要贴大段代码；以简洁表格和列表为主。
{output_instruction}"""


def run_agent_analysis(service_path: Path, project_name: str, url: str, method: str,
                       project_type: str, timeout_s: int, max_retries: int,
                       output_md_path: Optional[Path] = None,
                       agent_model: str = "") -> bool:
    """在仓库目录下执行 agent 分析，带超时与重试。"""
    if not service_path.is_dir():
        return False
    prompt = _build_agent_prompt(url, method, project_type, output_md_path)
    agent_args = ["agent", "--print", "--trust"]
    if agent_model:
        agent_args.extend(["--model", agent_model])
    agent_args.append(prompt)

    for attempt in range(max_retries + 1):
        if attempt > 0:
            log_warning(f"重试 {attempt}/{max_retries}...")
        try:
            r = subprocess.run(agent_args, cwd=str(service_path), timeout=timeout_s)
            if r.returncode == 0:
                log_success(f"分析完成: {project_name}")
                return True
            log_error(f"agent 退出码: {r.returncode}")
        except subprocess.TimeoutExpired:
            log_error(f"agent 超时（{timeout_s}s）")
    return False


def main() -> int:
    cfg = load_config()

    parser = argparse.ArgumentParser(description="API Route Tracer")
    parser.add_argument("url", help="API 路径，如 /room/:roomRef/order/:orderId")
    parser.add_argument("method", help="HTTP 方法，如 GET")
    parser.add_argument("scope", nargs="?", default="",
                        help="限定 frontend 或 backend 仓库")
    parser.add_argument("--stop-after", default=os.environ.get("TRACE_STOP_AFTER", ""),
                        choices=["llm", "sourcegraph", "repos", ""],
                        help="执行到此步骤后停止")
    args = parser.parse_args()

    url = args.url
    method = (args.method or "get").lower()
    scope = (args.scope or cfg.get("REPO_SCOPE", "")).lower()
    if scope and scope not in ("frontend", "backend"):
        scope = ""
    stop_after = (args.stop_after or cfg.get("TRACE_STOP_AFTER", "")).strip()

    log_info(f"Starting API route trace for: {BLUE}{url}{NC} ({YELLOW}{method.upper()}{NC})")
    if scope:
        log_info(f"Scope: {YELLOW}{scope} only{NC}")
    print(flush=True)

    # ----- 1) LLM 预处理 -----
    if cfg.get("TRACE_USE_LLM", "1") == "0":
        llm_result = {"patternType": "literal", "patterns": [url], "explanation": "TRACE_USE_LLM=0"}
    else:
        log_info("生成 Sourcegraph 搜索 pattern（llm_preprocess.py）...")
        llm_result = run_llm_preprocess(url, method, cfg["LLM_PREPROCESS_TIMEOUT"])

    pattern_type = llm_result.get("patternType", "literal")
    patterns = llm_result.get("patterns") or [url]
    log_success(f"LLM: {llm_result.get('explanation', '')}")
    log_info(f"patternType={pattern_type}，共 {len(patterns)} 个 pattern:")
    for p in patterns:
        print(f"   🔍 {p}", flush=True)
    print(flush=True)

    if stop_after == "llm":
        log_info("TRACE_STOP_AFTER=llm → 停止")
        return 0

    # ----- 2) Sourcegraph 搜索 -----
    log_info("查询 Sourcegraph...")
    try:
        repos = sourcegraph_search(pattern_type, patterns, repo_scope=scope, cfg=cfg)
    except SourcegraphUnavailableError as e:
        log_error("Sourcegraph 服务无法访问")
        print(flush=True)
        log_warning("可能原因：")
        print("   1. SOURCEGRAPH_TOKEN 已过期或无效（运行 node sourcegraph-token-playwright.js 刷新）", flush=True)
        print("   2. Sourcegraph 服务地址不可达", flush=True)
        print("   3. 网络连接问题或防火墙限制", flush=True)
        if cfg.get("TRACE_DEBUG") == "1" and getattr(e, "cause", None):
            print(f"   详细错误: {e.cause}", flush=True)
        return 1

    if not repos:
        log_error("Sourcegraph 未找到包含该 API 路径的仓库")
        return 1

    log_success(f"找到 {len(repos)} 个仓库:")
    for r in repos:
        print(f"   - {r}", flush=True)
    print(flush=True)

    if stop_after == "sourcegraph":
        log_info("TRACE_STOP_AFTER=sourcegraph → 停止")
        return 0

    # ----- 3) 生成 routes -----
    routes_data = build_routes(
        repos, url, method,
        project_type=scope or "code",
        repo_url_template=cfg.get("SOURCEGRAPH_REPO_URL_TEMPLATE", "https://%s.git"),
        frontend_pattern=cfg.get("SOURCEGRAPH_REPO_FRONTEND", ""),
        backend_pattern=cfg.get("SOURCEGRAPH_REPO_BACKEND", ""),
    )
    routes = routes_data.get("routes") or []

    if stop_after == "repos":
        log_info("TRACE_STOP_AFTER=repos → 停止")
        print(json.dumps(routes_data, ensure_ascii=False, indent=2))
        return 0

    # ----- 4) 每个仓库：clone/更新 + agent 分析 -----
    work_dir = cfg["WORK_DIR"]
    agent_timeout = cfg["AGENT_TIMEOUT"]
    max_retries = cfg["MAX_RETRIES"]
    docs_dir = Path(cfg.get("TRACE_DOCS_DIR", str(Path(work_dir) / "docs")))
    docs_dir.mkdir(parents=True, exist_ok=True)
    log_info(f"生成文档将保存到: {docs_dir.resolve()}")
    agent_model = (cfg.get("AGENT_MODEL") or "").strip()
    if agent_model:
        log_info(f"Agent 模型: {agent_model}")
    print(flush=True)

    for i, route in enumerate(routes):
        project_name = route.get("projectName") or ""
        repo_name = route.get("repoName") or project_name
        local_repo_dir = route.get("localRepoDir") or project_name
        git_url = route.get("projectGitRepoUrl") or ""
        route_path = (route.get("route") or {}).get("path", url)
        route_method = (route.get("route") or {}).get("method", method.upper())
        ptype = route.get("projectType", "code")
        if not project_name:
            continue

        slug = _api_slug(url, method)
        safe_name = "".join(c if c.isalnum() or c in "._-" else "_" for c in local_repo_dir)[:100]
        output_md = docs_dir / f"{slug}_{safe_name}.md"

        if len(routes) > 1:
            print("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", flush=True)
            log_info(f"处理仓库 {i + 1}/{len(routes)}")
            print("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", flush=True)

        log_info(f"Project: {project_name} | Repo: {repo_name} | Type: {ptype} | Route: {route_method} {route_path}")
        log_info(f"Git: {git_url}")
        log_info(f"本地目录: {Path(work_dir) / local_repo_dir}")
        log_info(f"文档输出: {output_md}")
        print(flush=True)

        service_path = ensure_repo(work_dir, local_repo_dir, git_url, repo_name)
        if not service_path:
            continue

        log_info("启动 agent 分析...")
        success = run_agent_analysis(
            service_path, project_name, url, method, ptype,
            agent_timeout, max_retries,
            output_md_path=output_md,
            agent_model=agent_model,
        )
        if success:
            copy_generated_doc(service_path, output_md, output_md.name)
        else:
            log_error(f"{project_name} 分析失败")
        print(flush=True)

    if len(routes) > 1:
        print("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", flush=True)
        log_success(f"全部 {len(routes)} 个仓库处理完成！")
    return 0


if __name__ == "__main__":
    sys.exit(main())
