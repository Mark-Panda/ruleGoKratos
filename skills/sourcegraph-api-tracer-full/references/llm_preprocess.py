#!/usr/bin/env python3
"""
LLM 预处理：根据 API 路径和 HTTP 方法，调用大模型生成 Sourcegraph 搜索条件。
供 trace_api.sh 调用，解决 agent CLI 在脚本/无 TTY 下无输出的问题。

用法: python3 llm_preprocess.py <url> <method>
      python3 llm_preprocess.py --prompt-only <url> <method>   # 仅输出 prompt 文本（供 agent CLI 等使用）
示例: python3 llm_preprocess.py /invoice/agency/list/xls get
      python3 llm_preprocess.py /xlsx/:id/get get
      python3 llm_preprocess.py --prompt-only /xlsx/:id/get get

环境变量（或同目录 `config.json` 中的对应字段）:
  OPENAI_API_KEY 或 TRACE_LLM_API_KEY   - API Key（必填，否则输出字面量 fallback）
  OPENAI_BASE_URL 或 TRACE_LLM_BASE_URL - 可选，兼容 OpenAI 的 API 地址
  TRACE_LLM_CHAT_PATH                   - 可选，chat 路径，默认 v1/chat/completions（部分网关为 openai/v1/chat/completions）
  TRACE_LLM_MODEL                       - 模型名，默认 gpt-4o-mini
  TRACE_DEBUG=1                         - 调试时在 stderr 打印请求/错误详情

stdout: 仅输出一行 JSON，形如 {"patternType":"literal|regexp","patterns":[...],"explanation":"..."}
"""

import json
import os
import re
import sys
import urllib.request
import urllib.error
from typing import Optional


def _debug(msg: str) -> None:
    if os.environ.get("TRACE_DEBUG") == "1":
        print(msg, file=sys.stderr)


def load_dotenv() -> bool:
    """已弃用。保留兼容接口，不再读取 .env。"""
    return False


def fallback_literal(url: str) -> dict:
    return {
        "patternType": "literal",
        "patterns": [url],
        "explanation": "未配置 API 或调用失败，使用 URL 字面量搜索",
    }


def extract_json_from_text(text: str) -> Optional[dict]:
    """从 LLM 返回文本中提取第一个合法 JSON 对象。"""
    text = text.strip()
    # 尝试 ```json ... ```
    m = re.search(r"```(?:json)?\s*\n?(.*?)```", text, re.DOTALL)
    if m:
        try:
            return json.loads(m.group(1).strip())
        except json.JSONDecodeError:
            pass
    # 找第一个完整 {...}
    start = text.find("{")
    if start >= 0:
        depth = 0
        for i in range(start, len(text)):
            if text[i] == "{":
                depth += 1
            elif text[i] == "}":
                depth -= 1
                if depth == 0:
                    try:
                        return json.loads(text[start : i + 1])
                    except json.JSONDecodeError:
                        break
    return None


def build_prompt(url: str, method: str) -> str:
    method_upper = method.upper()
    return f"""你是一个代码搜索专家。我需要在 Sourcegraph 中搜索引用了某个 API 路径的代码。

API 路径: {url}
HTTP 方法: {method_upper}

请分析这个 API 路径，判断其中是否包含动态路径参数（如 :id、{{id}}、具体的数字/UUID 等），然后生成最优的 Sourcegraph 搜索 content pattern。

规则：
1. 如果路径没有动态参数（如 /invoice/entry/xlsx），直接返回该路径字符串
2. 如果路径有占位符参数（如 /xlsx/:id/get 或 /xlsx/{{id}}/get），将参数部分替换为正则通配，生成 patternType:regexp 的搜索 pattern
3. 如果路径看起来有具体的动态值（如 /user/123/detail），将数字/UUID 部分替换为正则通配
4. 同时考虑：前端代码中 API 路径通常会用模板字符串（如 `/xlsx/${{id}}/get`）或字符串拼接（"/xlsx/" + id + "/get"），需要让 pattern 能匹配到这些写法
5. 前端代码中也可能通过路由定义来写（如 /xlsx/:id/get），需要也能匹配

你必须严格按以下 JSON 格式返回，不要有任何其他内容：
{{
  "patternType": "literal" 或 "regexp",
  "patterns": ["pattern1", "pattern2"],
  "explanation": "简短解释"
}}

patterns 数组说明：
- 如果 patternType 是 literal，patterns 里放一个精确路径字符串
- 如果 patternType 是 regexp，patterns 里放一个或多个正则表达式，每个 pattern 会单独搜索一次，结果合并去重
- 正则中不需要转义 /，但需要转义 Sourcegraph regexp 的特殊字符

示例：
输入: /invoice/entry/xlsx GET
输出: {{"patternType": "literal", "patterns": ["/invoice/entry/xlsx"], "explanation": "静态路径，直接字面量搜索"}}

输入: /xlsx/:id/get GET
输出: {{"patternType": "regexp", "patterns": ["/xlsx/[^/]+/get", "/xlsx/:id/get"], "explanation": "包含路径参数 :id，用正则匹配动态段，同时搜索路由定义写法"}}

输入: /user/{{userId}}/detail GET
输出: {{"patternType": "regexp", "patterns": ["/user/[^/]+/detail", "/user/\\\\{{userId\\\\}}/detail"], "explanation": "包含路径参数 {{userId}}，用正则匹配，同时搜索大括号模板定义"}}

输入: /xls/get/list GET
输出: {{"patternType": "literal", "patterns": ["/xls/get/list"], "explanation": "静态路径，直接字面量搜索"}}"""


def call_llm_api(url: str, method: str) -> dict:
    api_key = os.environ.get("TRACE_LLM_API_KEY") or os.environ.get("OPENAI_API_KEY")
    base_url = (
        os.environ.get("TRACE_LLM_BASE_URL") or os.environ.get("OPENAI_BASE_URL") or "https://api.openai.com"
    ).rstrip("/")
    chat_path = os.environ.get("TRACE_LLM_CHAT_PATH", "v1/chat/completions").strip("/")
    model = os.environ.get("TRACE_LLM_MODEL", "gpt-4o-mini")

    _debug(f"TRACE_LLM: base_url={base_url} chat_path={chat_path} model={model} has_key={bool(api_key)}")

    if not api_key:
        _debug("TRACE_LLM: 未读取到 API Key，请检查 config.json 或环境变量 TRACE_LLM_API_KEY / OPENAI_API_KEY")
        print("TRACE_LLM: 未读取到 API Key，使用字面量。请检查 references/config.json 或设 TRACE_DEBUG=1 查看详情。", file=sys.stderr)
        return fallback_literal(url)

    chat_url = f"{base_url}/{chat_path}"
    body = {
        "model": model,
        "messages": [
            {"role": "user", "content": build_prompt(url, method)},
        ],
        "temperature": 0.2,
        "max_tokens": 1024,
    }
    data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(
        chat_url,
        data=data,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {api_key}",
        },
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            out = json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        err_body = e.read().decode(errors="replace")[:500]
        _debug(f"TRACE_LLM: HTTP 错误 {e.code} {e.reason} url={chat_url} body={err_body}")
        if e.code == 503 and "model_not_found" in err_body:
            print(
                f"TRACE_LLM: 网关无可用模型通道（当前 TRACE_LLM_MODEL={model}）。"
                "请在 config.json 中改为网关支持的模型，如 gpt-3.5-turbo、gpt-4 等。",
                file=sys.stderr,
            )
        else:
            print(f"TRACE_LLM: API 请求失败 HTTP {e.code}。TRACE_DEBUG=1 查看详情（url/body）。", file=sys.stderr)
        return fallback_literal(url)
    except (urllib.error.URLError, json.JSONDecodeError, OSError) as e:
        _debug(f"TRACE_LLM: 请求或解析失败: {e}")
        print("TRACE_LLM: 使用字面量 fallback（网络或解析失败）。TRACE_DEBUG=1 查看原因。", file=sys.stderr)
        return fallback_literal(url)

    content = (out.get("choices") or [{}])[0].get("message", {}).get("content") or ""
    parsed = extract_json_from_text(content)
    if not parsed or "patternType" not in parsed or "patterns" not in parsed:
        _debug(f"TRACE_LLM: 模型返回无法解析为有效 JSON 或缺少 patternType/patterns，content 前 300 字: {content[:300]!r}")
        print("TRACE_LLM: 使用字面量 fallback（模型返回格式不符）。TRACE_DEBUG=1 查看原因。", file=sys.stderr)
        return fallback_literal(url)

    if parsed.get("patternType") not in ("literal", "regexp"):
        parsed["patternType"] = "literal"
    if not isinstance(parsed.get("patterns"), list) or not parsed["patterns"]:
        parsed["patterns"] = [url]
    parsed.setdefault("explanation", "")
    return parsed


def _load_config_json_into_env() -> None:
    """从同目录 config.json 读取 LLM 相关配置写入 os.environ（不覆盖已有环境变量）。"""
    import json as _json
    cfg_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "config.json")
    try:
        with open(cfg_path, "r", encoding="utf-8-sig") as f:
            data = _json.load(f)
        for key in ("TRACE_LLM_API_KEY", "TRACE_LLM_BASE_URL", "TRACE_LLM_CHAT_PATH",
                    "TRACE_LLM_MODEL", "OPENAI_API_KEY", "OPENAI_BASE_URL"):
            if key in data and data[key] and key not in os.environ:
                os.environ[key] = str(data[key])
    except (OSError, _json.JSONDecodeError):
        pass


def main() -> None:
    _load_config_json_into_env()
    load_dotenv()

    if len(sys.argv) >= 2 and sys.argv[1] == "--prompt-only":
        if len(sys.argv) < 4:
            print("Usage: llm_preprocess.py --prompt-only <url> <method>", file=sys.stderr)
            sys.exit(1)
        print(build_prompt(sys.argv[2], sys.argv[3]))
        sys.exit(0)

    if len(sys.argv) < 3:
        print(json.dumps({"patternType": "literal", "patterns": ["/"], "explanation": "缺少参数"}, ensure_ascii=False))
        sys.exit(0)

    url_arg = sys.argv[1]
    method_arg = sys.argv[2].upper()

    result = call_llm_api(url_arg, method_arg)
    # 打印返回结果到 stderr，便于查看；stdout 保持单行 JSON 供调用方解析
    print(
        f"[llm_preprocess] 返回: patternType={result.get('patternType')}, patterns={result.get('patterns')!r}, explanation={result.get('explanation', '')}",
        file=sys.stderr,
    )
    print(json.dumps(result, ensure_ascii=False))
    sys.exit(0)


if __name__ == "__main__":
    main()
