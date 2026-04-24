#!/bin/bash
# API Route Tracer - 主入口
#
# 用法：
#   单接口: ./trace_api.sh <url> <method> [frontend|backend]
#   批量（多参数）: ./trace_api.sh <url1> <method1> <url2> <method2> ...
#   文件批量: ./trace_api.sh --file apis.txt
#
# 环境变量:
#   TRACE_STOP_AFTER=llm|sourcegraph|repos   在此步骤后停止（调试用）
#   TRACE_DEBUG=1                            开启调试输出
#   TRACE_USE_LLM=0                          跳过 LLM，直接字面量搜索
#   AGENT_MODEL=<model>                      指定 agent 模型

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if [[ "${1:-}" == "--file" ]] || [[ "${1:-}" == "-f" ]]; then
    echo "🎯 批量模式（文件）..."
    exec python3 "$SCRIPT_DIR/trace_api_batch.py" "$@"
fi

if [ $# -lt 2 ]; then
    echo "用法: $0 <url> <method> [frontend|backend]"
    echo "   或: $0 <url1> <method1> <url2> <method2> ...  (批量模式)"
    echo "   或: $0 --file apis.txt                        (文件批量模式)"
    echo "示例: $0 /room/:roomRef/order/:orderId GET"
    echo "      $0 /room/:roomRef/order/:orderId GET backend"
    exit 1
fi

if [ $# -gt 3 ]; then
    echo "🎯 批量模式（检测到 $# 个参数）..."
    exec python3 "$SCRIPT_DIR/trace_api_batch.py" "$@"
else
    exec python3 "$SCRIPT_DIR/trace_api.py" "$@"
fi
