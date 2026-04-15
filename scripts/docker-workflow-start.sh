#!/usr/bin/env bash
set -eo pipefail
cd /app

if [ ! -x ./server ]; then
  echo "FATAL: /app/server missing or not executable" >&2
  exit 1
fi

./server -conf /data/conf &
backend_pid=$!

for _ in $(seq 1 90); do
  if ! kill -0 "$backend_pid" 2>/dev/null; then
    echo "FATAL: backend exited before HTTP became ready" >&2
    wait "$backend_pid" || true
    exit 1
  fi
  if curl -s -o /dev/null --connect-timeout 1 --max-time 3 "http://127.0.0.1:8000/" 2>/dev/null; then
    exec nginx -g "daemon off;"
  fi
  sleep 1
done

echo "FATAL: backend did not accept HTTP on 127.0.0.1:8000 within 90s" >&2
kill "$backend_pid" 2>/dev/null || true
exit 1
