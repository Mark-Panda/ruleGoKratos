#!/usr/bin/env bash
# 将本机 ~/.cursor 同步到构建上下文 deploy/docker-cursor-overlay/.cursor，
# 供 Dockerfile.all COPY 至镜像 /root/.cursor/
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DST="${ROOT}/deploy/docker-cursor-overlay/.cursor"

mkdir -p "${DST}"

SRC="${HOME}/.cursor"
if [[ ! -d "${SRC}" ]]; then
  echo "prepare-docker-cursor: 未找到 ${SRC}，跳过同步（镜像内仅保留占位）。" >&2
  exit 0
fi

rsync -a "${SRC}/" "${DST}/"
echo "prepare-docker-cursor: 已同步 ${SRC}/ → ${DST}/"
