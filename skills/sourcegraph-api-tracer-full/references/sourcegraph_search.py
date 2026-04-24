#!/usr/bin/env python3
"""
Sourcegraph 搜索：根据 patternType + patterns 查询，返回命中的仓库列表。
可被 trace_api.py 调用，也可单独运行（stdin 为 LLM 输出的 JSON，stdout 为每行一个仓库名）。

用法: echo '{"patternType":"literal","patterns":["/invoice/agency/list/xls"]}' | python3 sourcegraph_search.py [frontend|backend]
      或作为模块: sourcegraph_search.search(pattern_type, patterns, repo_scope="")
"""

import json
import os
import re
import sys
import urllib.parse
import urllib.request
import urllib.error
from typing import List, Optional

# 加载同目录 config（若存在）
try:
    from config import get_config, load_config, load_dotenv, refresh_token_from_config_after_401
except ImportError:
    _SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
    sys.path.insert(0, _SCRIPT_DIR)
    from config import get_config, load_config, load_dotenv, refresh_token_from_config_after_401


class SourcegraphUnavailableError(Exception):
    """Sourcegraph 请求失败（网络不可达、超时、配置错误等），供调用方提示用户配置。"""
    def __init__(self, message: str = "Sourcegraph 不可访问", cause: Optional[Exception] = None):
        self.cause = cause
        super().__init__(message)


def _debug(msg: str) -> None:
    if os.environ.get("TRACE_DEBUG") == "1":
        print(f"[DEBUG] {msg}", file=sys.stderr)


def _search_stream_once(cfg: dict, pattern_type: str, patterns: List[str], repo_scope: str) -> List[str]:
    """单次使用当前 cfg/token 完成所有 pattern 的 stream 请求；遇 HTTP 401 向外抛出 urllib.error.HTTPError。"""
    base_url = cfg["SOURCEGRAPH_URL"]
    stream_path = cfg["SOURCEGRAPH_STREAM_PATH"].lstrip("/")
    stream_url = f"{base_url}/{stream_path}"
    token = cfg["SOURCEGRAPH_TOKEN"]
    display_limit = cfg["SOURCEGRAPH_DISPLAY_LIMIT"]
    context_global = "context:global" if cfg["SOURCEGRAPH_CONTEXT_GLOBAL"] == "1" else ""
    type_filter = (cfg["SOURCEGRAPH_TYPE_FILTER"] or "").strip()
    include_forked = cfg["SOURCEGRAPH_INCLUDE_FORKED"] == "1"
    fork_filter = "fork:yes" if include_forked else ""

    if repo_scope == "frontend":
        repo_filter = f"repo:({cfg['SOURCEGRAPH_REPO_FRONTEND']})"
    elif repo_scope == "backend":
        repo_filter = f"repo:({cfg['SOURCEGRAPH_REPO_BACKEND']})"
    else:
        repo_filter = ""

    all_repos: set = set()

    for pattern in patterns:
        if pattern_type == "regexp":
            search_query = " ".join(
                filter(None, [context_global, pattern, type_filter, "patternType:regexp", repo_filter, fork_filter])
            )
        else:
            search_query = " ".join(
                filter(None, [context_global, pattern, type_filter, repo_filter, fork_filter])
            )
        search_query = re.sub(r"  +", " ", search_query.strip())
        search_query += f" count:{display_limit}"

        _debug(f"SEARCH_QUERY={search_query}")

        params = urllib.parse.urlencode({
            "q": search_query,
            "v": "V3",
            "t": "standard",
            "display": display_limit,
        })
        req = urllib.request.Request(
            f"{stream_url}?{params}",
            headers={
                "Accept": "text/event-stream",
                **({"Authorization": f"token {token}"} if token else {}),
            },
            method="GET",
        )
        try:
            with urllib.request.urlopen(req, timeout=60) as resp:
                body = resp.read().decode(errors="replace")
        except urllib.error.HTTPError as e:
            _debug(f"Request failed: {e}")
            if e.code == 401:
                raise
            raise SourcegraphUnavailableError(
                "Sourcegraph 不可访问，请检查网络与配置（SOURCEGRAPH_URL、SOURCEGRAPH_TOKEN 等）",
                cause=e,
            ) from e
        except (urllib.error.URLError, OSError) as e:
            _debug(f"Request failed: {e}")
            raise SourcegraphUnavailableError(
                "Sourcegraph 不可访问，请检查网络与配置（SOURCEGRAPH_URL、SOURCEGRAPH_TOKEN 等）",
                cause=e,
            ) from e

        # 解析 SSE：只处理 event: matches 后的 data
        ev = None
        for line in body.splitlines():
            if line.startswith("event: "):
                ev = line[7:].strip()
                continue
            if line.startswith("data: ") and ev == "matches":
                data_str = line[6:]
                try:
                    arr = json.loads(data_str)
                    for item in arr if isinstance(arr, list) else []:
                        repo = item.get("repository")
                        if repo:
                            all_repos.add(repo)
                except json.JSONDecodeError:
                    pass
                ev = None

    return sorted(all_repos)


def search(
    pattern_type: str,
    patterns: List[str],
    repo_scope: str = "",
    cfg: Optional[dict] = None,
) -> List[str]:
    """
    对每个 pattern 请求 Sourcegraph stream API，合并去重后返回仓库名列表。
    若返回 HTTP 401，从 config.json 重读 Token 并重试一次。
    若 cfg 由调用方传入，401 重试成功后会就地更新其中的 SOURCEGRAPH_TOKEN，
    避免调用方持有过期 Token 继续发起后续请求。
    """
    if cfg is None:
        cfg = load_config()

    try:
        return _search_stream_once(cfg, pattern_type, patterns, repo_scope)
    except urllib.error.HTTPError as e:
        if e.code != 401:
            raise SourcegraphUnavailableError(
                "Sourcegraph 不可访问，请检查网络与配置（SOURCEGRAPH_URL、SOURCEGRAPH_TOKEN 等）",
                cause=e,
            ) from e
        _debug("HTTP 401，重新从 config.json 读取 Token 后重试")
        if refresh_token_from_config_after_401():
            cfg2 = load_config()
            # 就地更新传入的 cfg，防止调用方后续仍持有旧 Token
            cfg.update({k: cfg2[k] for k in ("SOURCEGRAPH_TOKEN", "SOURCEGRAPH_URL") if k in cfg2})
            try:
                return _search_stream_once(cfg2, pattern_type, patterns, repo_scope)
            except urllib.error.HTTPError as e2:
                raise SourcegraphUnavailableError(
                    "Sourcegraph 返回 401（重试后仍失败）。请运行 node sourcegraph-token-playwright.js 刷新 Token。",
                    cause=e2,
                ) from e2
        raise SourcegraphUnavailableError(
            "Sourcegraph 返回 401，且 config.json 中无更新的 Token。请运行 node sourcegraph-token-playwright.js 重新获取。",
            cause=e,
        ) from e


def main() -> None:
    cfg = load_config()
    repo_scope = (sys.argv[1] if len(sys.argv) > 1 else "").strip().lower()
    if repo_scope and repo_scope not in ("frontend", "backend"):
        repo_scope = ""

    try:
        raw = sys.stdin.read().strip()
        data = json.loads(raw)
    except (json.JSONDecodeError, EOFError):
        print("sourcegraph_search: invalid JSON on stdin", file=sys.stderr)
        sys.exit(1)

    pattern_type = data.get("patternType", "literal")
    patterns = data.get("patterns") or []
    if not patterns:
        print("sourcegraph_search: no patterns in JSON", file=sys.stderr)
        sys.exit(1)

    try:
        repos = search(pattern_type, patterns, repo_scope=repo_scope, cfg=cfg)
    except SourcegraphUnavailableError as e:
        print("Sourcegraph 不可访问，请配置。请检查 SOURCEGRAPH_URL、SOURCEGRAPH_TOKEN 与网络。", file=sys.stderr)
        sys.exit(1)
    for r in repos:
        print(r)
    if not repos:
        sys.exit(1)


if __name__ == "__main__":
    main()
