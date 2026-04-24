#!/usr/bin/env python3
import json, os, sys, urllib.request, urllib.error, re

# Load config into env
cfg_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "config.json")
with open(cfg_path, "r", encoding="utf-8-sig") as f:
    data = json.load(f)
for key in ("TRACE_LLM_API_KEY", "TRACE_LLM_BASE_URL", "TRACE_LLM_CHAT_PATH", "TRACE_LLM_MODEL"):
    if key in data and data[key] and key not in os.environ:
        os.environ[key] = str(data[key])

url = sys.argv[1] if len(sys.argv) > 1 else "/entry/businessOrder/cancel"
method = sys.argv[2].upper() if len(sys.argv) > 2 else "PUT"

def build_prompt(url, method):
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
- 正则中不需要转义 /，但需要转义 Sourcegraph regexp 的特殊字符"""

method_upper = method.upper()
api_key = os.environ.get("TRACE_LLM_API_KEY")
base_url = os.environ.get("TRACE_LLM_BASE_URL", "https://api.openai.com").rstrip("/")
chat_path = os.environ.get("TRACE_LLM_CHAT_PATH", "v1/chat/completions").strip("/")
model = os.environ.get("TRACE_LLM_MODEL", "gpt-4o-mini")

chat_url = f"{base_url}/{chat_path}"
body = {
    "model": model,
    "messages": [{"role": "user", "content": build_prompt(url, method)}],
    "temperature": 0.2,
    "max_tokens": 1024,
}
data = json.dumps(body).encode("utf-8")
req = urllib.request.Request(
    chat_url, data=data,
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {api_key}"},
    method="POST"
)

def extract_json(text):
    m = re.search(r"```(?:json)?\s*\n?(.*?)```", text, re.DOTALL)
    if m:
        try:
            return json.loads(m.group(1).strip())
        except json.JSONDecodeError:
            pass
    start = text.find("{")
    if start >= 0:
        depth = 0
        for i in range(start, len(text)):
            if text[i] == "{": depth += 1
            elif text[i] == "}":
                depth -= 1
                if depth == 0:
                    try:
                        return json.loads(text[start:i+1])
                    except json.JSONDecodeError:
                        break
    return None

try:
    with urllib.request.urlopen(req, timeout=60) as resp:
        out = json.loads(resp.read().decode())
except Exception as e:
    print(f"Error: {e}")
    sys.exit(1)

content = (out.get("choices") or [{}])[0].get("message", {}).get("content") or ""
print(f"LLM raw content:\n{content[:500]}", file=sys.stderr)
parsed = extract_json(content)
if parsed and "patternType" in parsed and "patterns" in parsed:
    print(json.dumps(parsed, ensure_ascii=False))
else:
    print(json.dumps({"patternType": "literal", "patterns": [url], "explanation": "fallback"}))
