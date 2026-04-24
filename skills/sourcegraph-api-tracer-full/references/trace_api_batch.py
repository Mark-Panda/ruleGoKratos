#!/usr/bin/env python3
"""
批量 API Route Tracer - 支持一次处理多个接口

用法: 
    python3 trace_api_batch.py <api1_url> <api1_method> [<api2_url> <api2_method> ...]
    python3 trace_api_batch.py --file <接口列表文件>

示例:
    # 命令行传入多个接口
    python3 trace_api_batch.py /invoice/entry/xlsx get /user/login post /api/data get
    
    # 从文件读取（每行格式：URL METHOD）
    python3 trace_api_batch.py --file apis.txt

接口列表文件格式（apis.txt）：
    /invoice/entry/xlsx get
    /user/login post
    /api/data get
"""

import argparse
import sys
from pathlib import Path
from typing import List, Tuple
import subprocess

# 颜色
BLUE = "\033[0;34m"
GREEN = "\033[0;32m"
YELLOW = "\033[1;33m"
RED = "\033[0;31m"
NC = "\033[0m"

SCRIPT_DIR = Path(__file__).resolve().parent


def log_info(msg: str) -> None:
    print(f"{BLUE}ℹ{NC} {msg}", flush=True)


def log_success(msg: str) -> None:
    print(f"{GREEN}✅{NC} {msg}", flush=True)


def log_error(msg: str) -> None:
    print(f"{RED}❌{NC} {msg}", flush=True)


def parse_file(file_path: Path) -> List[Tuple[str, str]]:
    """从文件解析接口列表"""
    apis = []
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            for i, line in enumerate(f, 1):
                line = line.strip()
                if not line or line.startswith('#'):
                    continue
                parts = line.split()
                if len(parts) < 2:
                    log_error(f"第 {i} 行格式错误（需要 URL METHOD）: {line}")
                    continue
                url, method = parts[0], parts[1]
                apis.append((url, method))
    except Exception as e:
        log_error(f"读取文件失败: {e}")
        sys.exit(1)
    return apis


def main():
    parser = argparse.ArgumentParser(
        description="批量 API Route Tracer - Frontend",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__
    )
    
    parser.add_argument('apis', nargs='*', help='接口列表（URL METHOD 成对出现）')
    parser.add_argument('--file', '-f', type=Path, help='接口列表文件路径')
    
    args = parser.parse_args()
    
    # 解析接口列表
    apis: List[Tuple[str, str]] = []
    
    # 检查参数
    if args.file and args.apis:
        log_error("不能同时使用命令行参数和文件参数")
        sys.exit(1)
    
    if not args.file and not args.apis:
        log_error("请提供接口列表（命令行或 --file）")
        parser.print_help()
        sys.exit(1)
    
    if args.file:
        if not args.file.exists():
            log_error(f"文件不存在: {args.file}")
            sys.exit(1)
        apis = parse_file(args.file)
    else:
        if len(args.apis) % 2 != 0:
            log_error("接口参数必须成对出现（URL METHOD）")
            print(f"\n{YELLOW}示例:{NC}")
            print("  python3 trace_api_batch.py /api/xxx get /api/yyy post")
            sys.exit(1)
        
        for i in range(0, len(args.apis), 2):
            url = args.apis[i]
            method = args.apis[i + 1]
            apis.append((url, method))
    
    if not apis:
        log_error("未找到有效的接口")
        sys.exit(1)
    
    # 显示汇总
    print("=" * 60, flush=True)
    log_info(f"🎯 批量追踪模式（前端 + 后端）")
    log_info(f"📋 共 {len(apis)} 个接口待处理")
    print("=" * 60, flush=True)
    for i, (url, method) in enumerate(apis, 1):
        print(f"  {i}. {YELLOW}{method.upper()}{NC} {BLUE}{url}{NC}", flush=True)
    print("=" * 60, flush=True)
    print(flush=True)
    
    # 逐个处理
    trace_script = SCRIPT_DIR / "trace_api.py"
    if not trace_script.exists():
        log_error(f"找不到 trace_api.py: {trace_script}")
        sys.exit(1)
    
    success_count = 0
    failed_apis = []
    
    for i, (url, method) in enumerate(apis, 1):
        print("\n" + "━" * 60, flush=True)
        log_info(f"处理接口 [{i}/{len(apis)}]: {method.upper()} {url}")
        print("━" * 60, flush=True)
        print(flush=True)
        
        try:
            result = subprocess.run(
                [sys.executable, str(trace_script), url, method],
                cwd=SCRIPT_DIR,
                check=False
            )
            
            if result.returncode == 0:
                success_count += 1
                log_success(f"✓ [{i}/{len(apis)}] 完成: {method.upper()} {url}")
            else:
                failed_apis.append((url, method))
                log_error(f"✗ [{i}/{len(apis)}] 失败: {method.upper()} {url}")
        except Exception as e:
            failed_apis.append((url, method))
            log_error(f"✗ [{i}/{len(apis)}] 异常: {e}")
        
        print(flush=True)
    
    # 最终汇总
    print("\n" + "=" * 60, flush=True)
    log_info("📊 批量处理完成")
    print("=" * 60, flush=True)
    log_success(f"成功: {success_count}/{len(apis)}")
    if failed_apis:
        log_error(f"失败: {len(failed_apis)}/{len(apis)}")
        print(f"\n{YELLOW}失败的接口：{NC}", flush=True)
        for url, method in failed_apis:
            print(f"  • {method.upper()} {url}", flush=True)
    print("=" * 60, flush=True)
    
    return 0 if success_count == len(apis) else 1


if __name__ == "__main__":
    sys.exit(main())
