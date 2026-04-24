#!/usr/bin/env python3
"""
根据仓库列表 + URL + Method 生成 routes JSON（与原有 trace_api.sh 输出格式一致）。
可被 trace_api.py 调用，也可单独运行。

用法: python3 build_routes.py <url> <method> [project_type]
      stdin: 每行一个仓库名（如 gitlab.com/org/repo）
      stdout: {"routes": [...]}
"""

import json
import os
import re
import sys
from typing import List

try:
    from config import get_config, load_dotenv
except ImportError:
    _SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
    sys.path.insert(0, _SCRIPT_DIR)
    from config import get_config, load_dotenv


def repo_to_project_name(repo_name: str) -> str:
    return repo_name.split("/")[-1] if "/" in repo_name else repo_name


def repo_to_local_repo_dir(repo_name: str) -> str:
    repo_name = repo_name.strip().strip("/")
    safe = re.sub(r"[^A-Za-z0-9._-]+", "__", repo_name)
    return safe[:120] or repo_to_project_name(repo_name)


def repo_to_clone_url(repo_name: str, template: str = "https://%s.git") -> str:
    """
    根据模板生成克隆 URL。
    template 为 "ssh" 或形如 "git@..." 时，自动转换为 SSH URL：
      gitlab.yc345.tv/teacher/fe/repo → git@gitlab.yc345.tv:teacher/fe/repo.git
    否则按 printf 风格替换 %s。
    """
    t = template.strip()
    if t.lower() == "ssh" or t.startswith("git@"):
        parts = repo_name.strip("/").split("/", 1)
        if len(parts) == 2:
            return f"git@{parts[0]}:{parts[1]}.git"
    return template % repo_name


def infer_project_type(
    repo_name: str,
    explicit_project_type: str = "code",
    frontend_pattern: str = "",
    backend_pattern: str = "",
) -> str:
    if explicit_project_type in ("frontend", "backend"):
        return explicit_project_type

    try:
        if frontend_pattern and re.search(frontend_pattern, repo_name):
            return "frontend"
    except re.error:
        pass

    try:
        if backend_pattern and re.search(backend_pattern, repo_name):
            return "backend"
    except re.error:
        pass

    repo_name_lower = repo_name.lower()
    if "/fe/" in repo_name_lower or "/frontend/" in repo_name_lower:
        return "frontend"
    if "/backend/" in repo_name_lower:
        return "backend"
    return explicit_project_type or "code"


def build_routes(
    repo_names: List[str],
    url: str,
    method: str,
    project_type: str = "code",
    repo_url_template: str = "https://%s.git",
    frontend_pattern: str = "",
    backend_pattern: str = "",
) -> dict:
    method_upper = method.upper()
    routes = []
    for repo_name in repo_names:
        repo_name = repo_name.strip()
        if not repo_name:
            continue
        routes.append({
            "projectName": repo_to_project_name(repo_name),
            "repoName": repo_name,
            "localRepoDir": repo_to_local_repo_dir(repo_name),
            "projectGitRepoUrl": repo_to_clone_url(repo_name, repo_url_template),
            "projectType": infer_project_type(
                repo_name,
                explicit_project_type=project_type,
                frontend_pattern=frontend_pattern,
                backend_pattern=backend_pattern,
            ),
            "route": {"path": url, "method": method_upper},
        })
    return {"routes": routes}


def main() -> None:
    load_dotenv()
    cfg = get_config()
    if len(sys.argv) < 3:
        print("Usage: build_routes.py <url> <method> [project_type]", file=sys.stderr)
        sys.exit(1)
    url = sys.argv[1]
    method = sys.argv[2]
    project_type = (sys.argv[3] if len(sys.argv) > 3 else "").strip() or "code"
    repo_names = [line.strip() for line in sys.stdin if line.strip()]
    template = cfg.get("SOURCEGRAPH_REPO_URL_TEMPLATE", "https://%s.git")
    out = build_routes(
        repo_names,
        url,
        method,
        project_type=project_type,
        repo_url_template=template,
        frontend_pattern=cfg.get("SOURCEGRAPH_REPO_FRONTEND", ""),
        backend_pattern=cfg.get("SOURCEGRAPH_REPO_BACKEND", ""),
    )
    print(json.dumps(out, ensure_ascii=False))


if __name__ == "__main__":
    main()
