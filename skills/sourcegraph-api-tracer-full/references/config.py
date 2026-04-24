#!/usr/bin/env python3
"""
API Route Tracer 共享配置。

唯一配置源：同目录的 config.json。
所有脚本在执行前均调用 load_config() 读取最新配置；新 Token 也写回 config.json。

支持 os.environ 覆盖：若某 Key 已在进程环境中 export，则以环境变量为准（不被 config.json 覆盖）。
"""

import json
import os
import sys
from typing import Any, Dict, Optional

_SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
CONFIG_JSON_PATH = os.path.join(_SCRIPT_DIR, "config.json")


# ---------------------------------------------------------------------------
# 读 / 写 config.json
# ---------------------------------------------------------------------------

def _read_json() -> Dict[str, Any]:
    """读取 config.json；不存在或解析失败则返回空字典。"""
    try:
        with open(CONFIG_JSON_PATH, "r", encoding="utf-8-sig") as f:
            data = json.load(f)
        return data if isinstance(data, dict) else {}
    except (OSError, json.JSONDecodeError):
        return {}


def _write_json(data: Dict[str, Any]) -> bool:
    """原子地把 data 写入 config.json（覆盖全量）；写入失败返回 False。"""
    try:
        tmp = CONFIG_JSON_PATH + ".tmp"
        with open(tmp, "w", encoding="utf-8", newline="\n") as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
            f.write("\n")
        os.replace(tmp, CONFIG_JSON_PATH)
        return True
    except OSError as e:
        print(f"[config] 写入 config.json 失败: {e}", file=sys.stderr)
        return False


def update_config_json(**kwargs: Any) -> bool:
    """将给定的 key/value 合并到 config.json（其他字段保留）。"""
    data = _read_json()
    data.update(kwargs)
    return _write_json(data)


# ---------------------------------------------------------------------------
# 加载配置
# ---------------------------------------------------------------------------

def _str(v: Any) -> str:
    return "" if v is None else str(v).strip()


def load_config() -> Dict[str, Any]:
    """
    读取 config.json，返回配置字典（os.environ 优先于 config.json 的值）。
    每次调用都会重新读文件，保证获取最新 Token。
    """
    raw = _read_json()
    if os.environ.get("TRACE_DEBUG", "0") == "1":
        print(f"[config] loaded config.json from {CONFIG_JSON_PATH}", file=sys.stderr)

    def _get(key: str, default: Any = "") -> str:
        """os.environ 优先，其次 config.json，最后 default。"""
        if key in os.environ:
            return os.environ[key]
        v = raw.get(key, default)
        return _str(v) if not isinstance(v, bool) else ("1" if v else "0")

    work_dir = os.path.expanduser(_get("WORK_DIR", "~/bizCompareWarehouse"))
    raw_docs = _get("TRACE_DOCS_DIR", "")
    docs_dir = os.path.expanduser(raw_docs) if raw_docs else os.path.join(work_dir, "docs")

    cfg: Dict[str, Any] = {
        "SOURCEGRAPH_URL": _get("SOURCEGRAPH_URL", "https://sourcegraph.yc345.tv").rstrip("/"),
        "SOURCEGRAPH_TOKEN": _get("SOURCEGRAPH_TOKEN", ""),
        "SOURCEGRAPH_REPO_URL_TEMPLATE": _get("SOURCEGRAPH_REPO_URL_TEMPLATE", "https://%s.git"),
        "SOURCEGRAPH_REPO_FRONTEND": _get("SOURCEGRAPH_REPO_FRONTEND", "teacher/fe/.*|frontend/.*"),
        "SOURCEGRAPH_REPO_BACKEND": _get("SOURCEGRAPH_REPO_BACKEND", "teacher/backend/.*|backend/.*"),
        "SOURCEGRAPH_INCLUDE_FORKED": _get("SOURCEGRAPH_INCLUDE_FORKED", "1"),
        "SOURCEGRAPH_CONTEXT_GLOBAL": _get("SOURCEGRAPH_CONTEXT_GLOBAL", "1"),
        "SOURCEGRAPH_TYPE_FILTER": _get("SOURCEGRAPH_TYPE_FILTER", ""),
        "SOURCEGRAPH_STREAM_PATH": _get("SOURCEGRAPH_STREAM_PATH", "search/stream"),
        "SOURCEGRAPH_DISPLAY_LIMIT": int(_get("SOURCEGRAPH_DISPLAY_LIMIT", str(raw.get("SOURCEGRAPH_DISPLAY_LIMIT", 1500)))),
        "WORK_DIR": work_dir,
        "TRACE_DOCS_DIR": docs_dir,
        "TRACE_LLM_API_KEY": _get("TRACE_LLM_API_KEY", "") or _get("OPENAI_API_KEY", ""),
        "TRACE_LLM_BASE_URL": _get("TRACE_LLM_BASE_URL", "") or _get("OPENAI_BASE_URL", "https://api.openai.com"),
        "TRACE_LLM_CHAT_PATH": _get("TRACE_LLM_CHAT_PATH", "v1/chat/completions"),
        "TRACE_LLM_MODEL": _get("TRACE_LLM_MODEL", "gpt-4o-mini"),
        "TRACE_USE_LLM": _get("TRACE_USE_LLM", "1"),
        "TRACE_DEBUG": _get("TRACE_DEBUG", "0"),
        "TRACE_STOP_AFTER": _get("TRACE_STOP_AFTER", ""),
        "AGENT_TIMEOUT": int(_get("AGENT_TIMEOUT", str(raw.get("AGENT_TIMEOUT", 300)))),
        "AGENT_MODEL": (_get("AGENT_MODEL", "auto") or "auto"),
        "MAX_RETRIES": int(_get("MAX_RETRIES", str(raw.get("MAX_RETRIES", 2)))),
        "LLM_PREPROCESS_TIMEOUT": int(_get("LLM_PREPROCESS_TIMEOUT", str(raw.get("LLM_PREPROCESS_TIMEOUT", 60)))),
        # Playwright / LDAP（供 sourcegraph-token-playwright.js 与其他脚本读取）
        "SOURCEGRAPH_LDAP_USERNAME": _get("SOURCEGRAPH_LDAP_USERNAME", ""),
        "SOURCEGRAPH_LDAP_PASSWORD": _get("SOURCEGRAPH_LDAP_PASSWORD", ""),
        "SOURCEGRAPH_GITLAB_HOST": _get("SOURCEGRAPH_GITLAB_HOST", "gitlab.yc345.tv"),
        "PLAYWRIGHT_HEADLESS": _get("PLAYWRIGHT_HEADLESS", "1"),
    }
    return cfg


# 兼容旧接口（其他脚本可能调用 get_config / load_dotenv）
def get_config() -> Dict[str, Any]:
    return load_config()


def load_dotenv() -> bool:
    """已弃用（保留兼容），内部不再读 .env。"""
    return False


def get(key: str, default: str = "") -> str:
    return os.environ.get(key, default)


# ---------------------------------------------------------------------------
# Token 刷新写回
# ---------------------------------------------------------------------------

def save_token_to_config(token: str, url: Optional[str] = None) -> bool:
    """
    将新 Token（及可选的 URL）写回 config.json，同时刷新 os.environ。
    供 Playwright 取到新 Token 后、以及 401 自动刷新后调用。
    """
    kw: Dict[str, Any] = {"SOURCEGRAPH_TOKEN": token}
    if url:
        kw["SOURCEGRAPH_URL"] = url.rstrip("/")
    ok = update_config_json(**kw)
    if ok:
        os.environ["SOURCEGRAPH_TOKEN"] = token
        if url:
            os.environ["SOURCEGRAPH_URL"] = url.rstrip("/")
        if os.environ.get("TRACE_DEBUG", "0") == "1":
            print(f"[config] SOURCEGRAPH_TOKEN 已写回 config.json", file=sys.stderr)
    return ok


def refresh_token_from_config_after_401() -> bool:
    """
    Sourcegraph 返回 401 后调用：重新读 config.json，若其中 Token 与当前进程不同则更新 os.environ 并返回 True。
    （Token 可能已由其他进程/Agent 刷新写入 config.json，这里只做重读，不触发 Playwright。）
    """
    raw = _read_json()
    new_token = (raw.get("SOURCEGRAPH_TOKEN") or "").strip()
    cur = (os.environ.get("SOURCEGRAPH_TOKEN") or "").strip()
    if not new_token or new_token == cur:
        return False
    os.environ["SOURCEGRAPH_TOKEN"] = new_token
    url = (raw.get("SOURCEGRAPH_URL") or "").strip().rstrip("/")
    if url:
        os.environ["SOURCEGRAPH_URL"] = url
    if os.environ.get("TRACE_DEBUG", "0") == "1":
        print(f"[config] 401 后从 config.json 重新加载 Token", file=sys.stderr)
    return True
