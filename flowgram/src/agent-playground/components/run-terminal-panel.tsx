/**
 * Run Terminal Panel - 嵌入式终端面板（适配 Playground Run 工作区）
 */
import React, { useCallback, useEffect, useRef } from 'react';

import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { Button, Typography } from '@douyinfe/semi-ui';
import { IconClose } from '@douyinfe/semi-icons';

import { getApiOrigin, getAuthToken } from '../../services/http';

import '@xterm/xterm/css/xterm.css';

function buildTerminalWsURL(cwd: string): string {
  const u = new URL(getApiOrigin());
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:';
  const params = new URLSearchParams();
  params.set('cwd', cwd.trim());
  const tok = getAuthToken();
  if (tok) {
    params.set('token', tok);
  }
  return `${u.origin}/api/v1/admin/terminal/ws?${params.toString()}`;
}

interface RunTerminalPanelProps {
  cwd: string;
  onClose?: () => void;
}

export const RunTerminalPanel: React.FC<RunTerminalPanelProps> = ({ cwd, onClose }) => {
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

  useEffect(() => {
    const el = containerRef.current;
    if (!el || !cwd) return;

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
      term.writeln('\r\n\x1b[33m[连接已断开]\x1b[0m');
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

    return () => disconnect();
  }, [cwd, disconnect, sendResize]);

  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
        background: '#fff',
        borderRadius: 8,
        border: '1px solid rgba(28,31,35,0.08)',
        overflow: 'hidden',
      }}
    >
      {/* 顶部栏 */}
      <div
        style={{
          padding: '8px 12px',
          borderBottom: '1px solid rgba(28,31,35,0.06)',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          background: 'rgba(28,31,35,0.02)',
        }}
      >
        <Typography.Text
          style={{ flex: 1, fontSize: 13, fontFamily: 'monospace' }}
          ellipsis={{ showTooltip: true }}
        >
          终端: {cwd}
        </Typography.Text>
        {onClose && (
          <Button size="small" type="tertiary" icon={<IconClose />} onClick={onClose}>
            关闭
          </Button>
        )}
      </div>

      {/* 终端容器 */}
      <div
        ref={containerRef}
        style={{
          flex: 1,
          minHeight: 200,
          background: 'rgba(6,7,9,0.92)',
          borderRadius: '0 0 8px 8px',
          padding: 8,
          overflow: 'hidden',
        }}
      />
    </div>
  );
};
