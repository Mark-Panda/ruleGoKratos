/**
 * 管理端「终端」：WebSocket + PTY 交互式 shell（工作目录白名单与 REST RunTerminal 一致）。
 */
import React, { useCallback, useEffect, useRef, useState } from 'react';

import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { Button, Input, Typography } from '@douyinfe/semi-ui';

import { getApiOrigin, getAuthToken } from '../../services/http';

import '@xterm/xterm/css/xterm.css';

const defaultCwd = '/app';

function buildTerminalWsURL(cwd: string): string {
  const u = new URL(getApiOrigin());
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:';
  const params = new URLSearchParams();
  params.set('cwd', cwd.trim() || defaultCwd);
  const tok = getAuthToken();
  if (tok) {
    params.set('token', tok);
  }
  return `${u.origin}/api/v1/admin/terminal/ws?${params.toString()}`;
}

export const TerminalSection: React.FC = () => {
  const [cwd, setCwd] = useState(defaultCwd);
  const [connected, setConnected] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const resizeObsRef = useRef<ResizeObserver | null>(null);

  const disconnect = useCallback(() => {
    resizeObsRef.current?.disconnect();
    resizeObsRef.current = null;
    try {
      wsRef.current?.close();
    } catch {
      /* ignore */
    }
    wsRef.current = null;
    termRef.current?.dispose();
    termRef.current = null;
    fitRef.current = null;
    setConnected(false);
  }, []);

  const sendResize = useCallback(() => {
    const ws = wsRef.current;
    const fit = fitRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN || !fit) {
      return;
    }
    fit.fit();
    const dims = fit.proposeDimensions();
    if (dims && dims.rows > 0 && dims.cols > 0) {
      ws.send(JSON.stringify({ type: 'resize', rows: dims.rows, cols: dims.cols }));
    }
  }, []);

  const connect = useCallback(() => {
    disconnect();
    const el = containerRef.current;
    if (!el) {
      return;
    }
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      theme: { background: '#0d1117', foreground: '#e6edf3' },
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(el);
    fit.fit();
    termRef.current = term;
    fitRef.current = fit;

    const url = buildTerminalWsURL(cwd);
    const ws = new WebSocket(url);
    ws.binaryType = 'arraybuffer';
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
      sendResize();
    };

    ws.onmessage = (ev: MessageEvent) => {
      if (ev.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(ev.data));
      } else if (typeof ev.data === 'string') {
        term.write(ev.data);
      }
    };

    ws.onerror = () => {
      term.writeln('\r\n\x1b[31m[WebSocket 错误]\x1b[0m');
    };

    ws.onclose = () => {
      setConnected(false);
    };

    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode(data));
      }
    });

    const ro = new ResizeObserver(() => {
      sendResize();
    });
    ro.observe(el);
    resizeObsRef.current = ro;
  }, [cwd, disconnect, sendResize]);

  useEffect(() => () => disconnect(), [disconnect]);

  return (
    <div
      style={{
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        height: '100%',
        boxSizing: 'border-box',
      }}
    >
      <Typography.Title heading={6} style={{ margin: 0 }}>
        服务端终端
      </Typography.Title>
      <Typography.Text type="tertiary" size="small">
        通过 WebSocket + PTY 打开交互式 shell；工作目录须为{' '}
        <Typography.Text code>/app</Typography.Text> 或配置的 Agent 工作区（绝对路径）。经 Nginx
        反代时需已配置 Upgrade。REST{' '}
        <Typography.Text code>POST /admin/terminal/run</Typography.Text> 仍可用于单次命令。
      </Typography.Text>

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, alignItems: 'center' }}>
        <Typography.Text size="small" style={{ minWidth: 72 }}>
          工作目录
        </Typography.Text>
        <Input
          value={cwd}
          onChange={setCwd}
          placeholder="/app 或 /app/mcp/..."
          style={{ flex: '1 1 280px', maxWidth: 560 }}
        />
        <Button theme="solid" type="primary" onClick={connect} disabled={connected}>
          连接
        </Button>
        <Button onClick={disconnect} disabled={!connected}>
          断开
        </Button>
      </div>

      <div
        ref={containerRef}
        style={{
          flex: 1,
          minHeight: 320,
          background: 'rgba(6,7,9,0.92)',
          border: '1px solid rgba(6,7,9,0.2)',
          borderRadius: 4,
          padding: 8,
          overflow: 'hidden',
        }}
      />
    </div>
  );
};

export default TerminalSection;
