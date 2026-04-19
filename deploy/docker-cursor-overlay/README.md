# Docker 构建用 Cursor 目录覆盖

镜像内 Agent 使用的 Cursor 配置与扩展等，需在**构建镜像前**把本机 `~/.cursor/` 同步到此处，再由 `Dockerfile.all` 复制到容器内的 `/root/.cursor/`。

```bash
# 在仓库根目录执行（会写入本目录下的 .cursor/）
bash scripts/prepare-docker-cursor.sh

# 再构建镜像
docker build -f Dockerfile.all -t your-image:tag .
```

未执行同步时仍可通过构建：仅打包占位目录（内容接近空）。勿将含密钥的本机 `.cursor` 提交到 Git（见 `.gitignore`）。
