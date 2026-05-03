/**
 * 管理端「Cursor CLI」：终端内执行 `agent login`（容器内配合 NO_OPEN_BROWSER 提取浏览器链接）。
 * 参考：https://cursor.com/docs/cli/overview
 */
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
  Banner,
  Button,
  Card,
  Collapse,
  Divider,
  Spin,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { IconCopy, IconDelete, IconLink, IconRefresh, IconStop } from '@douyinfe/semi-icons';

import { getApiOrigin, getAuthToken } from '../../services/http';
import { runTerminal } from '../../services/api-agent';

/* ───── 工具函数 ───── */

function isLoginOkFromStatusText(raw: string): boolean {
  const t = raw.trim();
  if (!t || /^Not logged in\b/i.test(t)) return false;
  return /Login successful|Logged in as\b/i.test(t);
}

function parseLoggedInAccount(raw: string): string | null {
  const m = raw.trim().match(/Logged in as\s+(\S+)/i);
  return m?.[1] ? m[1] : null;
}

function extractCursorHttpsUrls(raw: string): string[] {
  const re = /https:\/\/[^\s\)\]'"]+/g;
  const candidates = raw.match(re) ?? [];
  const filtered = candidates.filter((u) => {
    try {
      const host = new URL(u).hostname;
      return host === 'cursor.com' || host.endsWith('.cursor.com') || host.endsWith('.cursor.sh');
    } catch {
      return false;
    }
  });
  return [...new Set(filtered)];
}

function parseUserCodeFromVerifyUrl(url: string): string | null {
  try {
    const sp = new URL(url).searchParams;
    const c = sp.get('user_code');
    return c && c.trim() ? c.trim() : null;
  } catch {
    return null;
  }
}

async function copyToClipboard(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
    Toast.success('已复制到剪贴板');
  } catch {
    Toast.error('复制失败，请手动选择复制');
  }
}

/* ───── 状态指示圆点 ───── */

function StatusDot({ color }: { color: string }) {
  return (
    <span
      style={{
        display: 'inline-block',
        width: 10,
        height: 10,
        borderRadius: '50%',
        background: color,
        boxShadow: `0 0 6px ${color}60`,
        marginRight: 8,
        flexShrink: 0,
      }}
    />
  );
}

/* ───── 信息行组件 ───── */

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, lineHeight: 1.8 }}>
      <Typography.Text type="tertiary" size="small" style={{ flexShrink: 0, minWidth: 80 }}>
        {label}
      </Typography.Text>
      <Typography.Text style={{ flex: 1 }}>{children}</Typography.Text>
    </div>
  );
}

/* ───── URL 操作卡片 ───── */

function UrlActions({ urls }: { urls: string[] }) {
  if (!urls.length) return null;
  return (
    <div
      style={{
        marginTop: 12,
        padding: 14,
        background: 'linear-gradient(135deg, var(--semi-color-fill-0), var(--semi-color-fill-1))',
        borderRadius: 10,
        border: '1px solid var(--semi-color-border)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 10 }}>
        <IconLink style={{ color: 'var(--semi-color-primary)' }} />
        <Typography.Text strong style={{ fontSize: 13 }}>
          检测到授权链接
        </Typography.Text>
      </div>
      <Typography.Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 10 }}>
        请在浏览器中打开以下链接完成登录授权
      </Typography.Text>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {urls.map((href) => {
          const code = parseUserCodeFromVerifyUrl(href);
          return (
            <div
              key={href}
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: 8,
                alignItems: 'center',
                padding: '10px 12px',
                background: 'var(--semi-color-bg-1)',
                borderRadius: 8,
                border: '1px solid var(--semi-color-border)',
              }}
            >
              <a
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                style={{ wordBreak: 'break-all', flex: '1 1 220px', fontSize: 13 }}
              >
                {href}
              </a>
              {code && (
                <Tag color="blue" size="small">
                  用户码：{code}
                </Tag>
              )}
              <Button size="small" icon={<IconCopy />} onClick={() => void copyToClipboard(href)}>
                复制
              </Button>
            </div>
          );
        })}
      </div>
    </div>
  );
}

/* ───── 终端日志面板 ───── */

const TERMINAL_STYLE: React.CSSProperties = {
  padding: 14,
  background: 'linear-gradient(180deg, #0d1117 0%, #0f1720 100%)',
  color: '#e6edf3',
  borderRadius: 10,
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
  fontSize: 12,
  lineHeight: 1.6,
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
  maxHeight: 380,
  overflow: 'auto',
  border: '1px solid rgba(148,163,184,0.2)',
  boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.15)',
};

function TerminalLogPanel({
  fullLog,
  busy,
  logWrapRef,
}: {
  fullLog: string;
  busy: boolean;
  logWrapRef: React.Ref<HTMLDivElement>;
}) {
  return (
    <div ref={logWrapRef} style={TERMINAL_STYLE}>
      {fullLog ||
        (busy ? '⏳ 正在连接终端，等待输出…' : '等待命令输出… 请先点击「开始 agent login」')}
    </div>
  );
}

/* ───── 常量 ───── */

const LOGIN_CMD = 'NO_OPEN_BROWSER=1 agent login';
const RE_LOGIN_SUCCESS = /Login successful|Logged in as\b|Authentication tokens stored securely/i;
const RE_LOGIN_FAILED = /Login failed or timed out/i;

/* ───── 主组件 ───── */

const PAGE_PAD: React.CSSProperties = {
  padding: '20px clamp(16px, 2.5vw, 40px) 32px',
  width: '100%',
  maxWidth: '100%',
  alignSelf: 'stretch',
  flex: 1,
  minHeight: 0,
  minWidth: 0,
  overflowY: 'auto',
  boxSizing: 'border-box',
  display: 'flex',
  flexDirection: 'column',
  gap: 16,
};

const CARD_STYLE: React.CSSProperties = { borderRadius: 12 };

export const CursorCliSection: React.FC = () => {
  const [loadingStatus, setLoadingStatus] = useState(false);
  const [statusText, setStatusText] = useState('');

  const [busyLogin, setBusyLogin] = useState(false);
  const [loginStage, setLoginStage] = useState('');

  const [lastStdout, setLastStdout] = useState('');
  const [lastStderr, setLastStderr] = useState('');
  const [lastExit, setLastExit] = useState<number | null>(null);
  const [lastCommandLabel, setLastCommandLabel] = useState('');
  const [autoFollowLog, setAutoFollowLog] = useState(true);
  const wsRef = useRef<WebSocket | null>(null);
  const logWrapRef = useRef<HTMLDivElement | null>(null);
  const logBufRef = useRef('');
  const sentExitRef = useRef(false);
  const doneRef = useRef(false);
  const failNotifiedRef = useRef(false);

  const extractedUrls = useMemo(
    () => extractCursorHttpsUrls([lastStdout, lastStderr].join('\n')),
    [lastStdout, lastStderr]
  );

  const configured = isLoginOkFromStatusText(statusText);
  const loggedInAccount = useMemo(() => parseLoggedInAccount(statusText), [statusText]);

  const refreshStatus = useCallback(async () => {
    setLoadingStatus(true);
    try {
      const res = await runTerminal('agent status 2>&1', '/app');
      const out = [res.stdout, res.stderr].filter(Boolean).join('\n');
      setStatusText(out);
    } catch (e) {
      setStatusText(e instanceof Error ? e.message : String(e));
    } finally {
      setLoadingStatus(false);
    }
  }, []);

  useEffect(() => {
    void refreshStatus();
  }, [refreshStatus]);

  useEffect(() => {
    if (!configured || !autoFollowLog) return;
    if (!logWrapRef.current) return;
    logWrapRef.current.scrollTop = logWrapRef.current.scrollHeight;
  }, [lastStdout, lastStderr, autoFollowLog, configured]);

  const closeLoginSocket = useCallback(() => {
    try {
      wsRef.current?.close();
    } catch {
      // ignore
    }
    wsRef.current = null;
  }, []);

  useEffect(() => () => closeLoginSocket(), [closeLoginSocket]);

  const buildTerminalWsURL = useCallback((cwd: string): string => {
    const u = new URL(getApiOrigin());
    u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:';
    const params = new URLSearchParams();
    params.set('cwd', cwd);
    const tok = getAuthToken();
    if (tok) params.set('token', tok);
    return `${u.origin}/api/v1/admin/terminal/ws?${params.toString()}`;
  }, []);

  const finishSuccess = useCallback(() => {
    if (doneRef.current) return;
    doneRef.current = true;
    setLoginStage('登录成功');
    setLastExit(0);
    Toast.success('Cursor CLI 登录成功。');
    void refreshStatus();
  }, [refreshStatus]);

  const appendConsole = useCallback(
    (rawChunk: string) => {
      const clean = rawChunk.replace(/\[[0-9;?]*[ -/]*[@-~]/g, '');
      logBufRef.current += clean;
      setLastStdout(logBufRef.current);

      if (!failNotifiedRef.current && RE_LOGIN_FAILED.test(logBufRef.current)) {
        failNotifiedRef.current = true;
        setLoginStage('登录失败或超时');
        setLastExit(1);
        Toast.error('检测到登录失败或超时，请查看日志后重试。');
      }

      if (!doneRef.current && RE_LOGIN_SUCCESS.test(logBufRef.current)) {
        finishSuccess();
        if (!sentExitRef.current) {
          sentExitRef.current = true;
          setTimeout(() => {
            try {
              wsRef.current?.send(new TextEncoder().encode('\nexit\n'));
            } catch {
              // ignore
            }
            setTimeout(() => {
              closeLoginSocket();
              setBusyLogin(false);
              setLoginStage('');
            }, 200);
          }, 400);
        }
      }
    },
    [closeLoginSocket, finishSuccess]
  );

  const runAgentLogin = useCallback(async () => {
    setBusyLogin(true);
    setLoginStage(`执行 ${LOGIN_CMD}`);
    logBufRef.current = `$ ${LOGIN_CMD}\n`;
    setLastStdout(logBufRef.current);
    setLastStderr('');
    setLastExit(null);
    setLastCommandLabel(LOGIN_CMD);
    sentExitRef.current = false;
    doneRef.current = false;
    failNotifiedRef.current = false;
    closeLoginSocket();
    try {
      const ws = new WebSocket(buildTerminalWsURL('/app'));
      ws.binaryType = 'arraybuffer';
      wsRef.current = ws;
      ws.onopen = () => {
        try {
          ws.send(new TextEncoder().encode(`${LOGIN_CMD}\n`));
        } catch (e) {
          setLastStderr(e instanceof Error ? e.message : String(e));
        }
      };
      ws.onmessage = (ev: MessageEvent) => {
        const chunk =
          ev.data instanceof ArrayBuffer
            ? new TextDecoder().decode(new Uint8Array(ev.data))
            : typeof ev.data === 'string'
            ? ev.data
            : '';
        if (!chunk) return;
        appendConsole(chunk);
      };
      ws.onerror = () => {
        setLastStderr((prev) => `${prev}${prev ? '\n' : ''}[WebSocket 错误]`);
      };
      ws.onclose = () => {
        wsRef.current = null;
        setBusyLogin(false);
        setLoginStage('');
        if (doneRef.current) {
          void refreshStatus();
        } else if (!failNotifiedRef.current) {
          Toast.warning('终端连接已关闭；若未完成登录请重试。');
        }
        void refreshStatus();
      };
    } catch (e) {
      setLastStderr(e instanceof Error ? e.message : String(e));
      setLastExit(-1);
      Toast.error('无法建立终端连接');
      setBusyLogin(false);
    }
  }, [appendConsole, buildTerminalWsURL, closeLoginSocket, refreshStatus]);

  const fullLog = `${lastStdout}${lastStderr ? `${lastStdout ? '\n' : ''}${lastStderr}` : ''}`;

  const clearConsole = useCallback(() => {
    logBufRef.current = '';
    setLastStdout('');
    setLastStderr('');
    setLastExit(null);
    setLastCommandLabel('');
  }, []);

  /* ───── 渲染 ───── */
  return (
    <div style={PAGE_PAD}>
      {/* 页面标题 & 状态概览 */}
      <Card style={CARD_STYLE} bodyStyle={{ padding: '16px 20px' }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            flexWrap: 'wrap',
            gap: 12,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <div
              style={{
                width: 36,
                height: 36,
                borderRadius: 10,
                background: 'linear-gradient(135deg, #7c3aed, #a78bfa)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#fff',
                fontSize: 18,
                fontWeight: 700,
                flexShrink: 0,
              }}
            >
              C
            </div>
            <div>
              <Typography.Title heading={6} style={{ margin: 0 }}>
                Cursor CLI
              </Typography.Title>
              <Typography.Text type="tertiary" size="small">
                agent 登录与授权管理
              </Typography.Text>
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <StatusDot
              color={configured ? 'var(--semi-color-success)' : 'var(--semi-color-warning)'}
            />
            <Tag color={configured ? 'green' : 'orange'} size="small">
              {configured ? '已登录' : '未登录'}
            </Tag>
          </div>
        </div>
      </Card>

      {configured ? (
        <>
          {/* 已登录 — 账号信息卡 */}
          <Card style={CARD_STYLE} bodyStyle={{ padding: 0 }}>
            <div
              style={{
                background: 'linear-gradient(135deg, rgba(124,58,237,0.06), rgba(124,58,237,0.02))',
                padding: '18px 20px',
                borderRadius: '12px 12px 0 0',
                borderBottom: '1px solid var(--semi-color-border)',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <StatusDot color="var(--semi-color-success)" />
                <Typography.Text strong style={{ fontSize: 14 }}>
                  已登录
                </Typography.Text>
              </div>
            </div>
            <div style={{ padding: '16px 20px' }}>
              {loggedInAccount && (
                <InfoRow label="账号">
                  <Typography.Text strong style={{ fontSize: 14 }}>
                    {loggedInAccount}
                  </Typography.Text>
                </InfoRow>
              )}
              <InfoRow label="状态输出">
                <Typography.Text type="tertiary" size="small" style={{ lineHeight: 1.5 }}>
                  详见下方 <Typography.Text code>agent status</Typography.Text> 原始输出
                </Typography.Text>
              </InfoRow>
            </div>
            <div
              style={{
                padding: '12px 20px',
                borderTop: '1px solid var(--semi-color-border)',
                display: 'flex',
                gap: 8,
              }}
            >
              <Button
                icon={<IconRefresh />}
                loading={loadingStatus}
                onClick={() => void refreshStatus()}
              >
                刷新状态
              </Button>
            </div>
          </Card>

          {/* 原始状态（折叠） */}
          <Collapse style={{ borderRadius: 12 }}>
            <Collapse.Panel header="原始状态输出（agent status）" itemKey="status">
              <pre
                style={{
                  margin: 0,
                  padding: 12,
                  background: 'rgba(6,7,9,0.03)',
                  borderRadius: 8,
                  fontSize: 12,
                  maxHeight: 160,
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  border: '1px solid var(--semi-color-border)',
                }}
              >
                {loadingStatus ? '加载中…' : statusText || '（暂无）'}
              </pre>
            </Collapse.Panel>
          </Collapse>

          {/* 提示 */}
          <Banner
            type="info"
            fullMode={false}
            title="说明"
            description={
              <div style={{ lineHeight: 1.65 }}>
                <div>
                  更换账号请先在环境执行 <Typography.Text code>agent logout</Typography.Text>
                  ，再刷新本页按未登录流程操作。
                </div>
                <div style={{ marginTop: 6 }}>
                  Docker 持久化：可将宿主机 <Typography.Text code>~/.cursor</Typography.Text>{' '}
                  挂载到容器同路径。
                </div>
              </div>
            }
            style={{ borderRadius: 10 }}
          />
        </>
      ) : (
        <>
          {/* 未登录告警 */}
          <Banner
            type="warning"
            fullMode={false}
            title="尚未检测到有效登录"
            description="点击下方「开始 agent login」，在浏览器中打开授权链接完成登录。"
            style={{ borderRadius: 10 }}
          />

          {/* 登录操作卡 */}
          <Card style={CARD_STYLE} bodyStyle={{ padding: 0 }}>
            <div style={{ padding: '18px 20px' }}>
              <Typography.Title heading={6} style={{ margin: 0, marginBottom: 4 }}>
                Agent 登录
              </Typography.Title>
              <Typography.Text type="tertiary" size="small">
                执行 <Typography.Text code>{LOGIN_CMD}</Typography.Text>，提取浏览器授权链接
              </Typography.Text>
            </div>
            <Divider margin="4px" />
            <div style={{ padding: '16px 20px' }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  marginBottom: 12,
                  flexWrap: 'wrap',
                }}
              >
                <Tag color={busyLogin ? 'blue' : 'orange'} size="small">
                  {busyLogin ? loginStage || '处理中' : '待操作'}
                </Tag>
                {lastExit !== null && (
                  <Tag color={lastExit === 0 ? 'green' : 'red'} size="small">
                    exit {lastExit}
                  </Tag>
                )}
              </div>
              {busyLogin && (
                <Banner
                  type="info"
                  fullMode={false}
                  closeIcon={null}
                  icon={<Spin size="small" />}
                  title="登录进行中"
                  description="请在浏览器完成授权，命令会自动检测登录结果。"
                  style={{ marginBottom: 12, borderRadius: 8 }}
                />
              )}
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                <Button
                  theme="solid"
                  type="primary"
                  size="large"
                  loading={busyLogin}
                  disabled={busyLogin}
                  onClick={() => void runAgentLogin()}
                  style={{ borderRadius: 8, fontWeight: 600 }}
                >
                  开始 agent login
                </Button>
                <Button
                  icon={<IconRefresh />}
                  loading={loadingStatus}
                  disabled={busyLogin}
                  onClick={() => void refreshStatus()}
                >
                  刷新状态
                </Button>
                <Button
                  icon={<IconStop />}
                  disabled={!busyLogin}
                  type="danger"
                  onClick={() => {
                    closeLoginSocket();
                    setBusyLogin(false);
                    setLoginStage('');
                  }}
                >
                  停止
                </Button>
              </div>
            </div>
          </Card>

          {/* 终端日志 */}
          <Card style={CARD_STYLE} bodyStyle={{ padding: 0 }}>
            <div
              style={{
                padding: '14px 20px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                borderBottom: '1px solid var(--semi-color-border)',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <div
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: '50%',
                    background: busyLogin
                      ? 'var(--semi-color-success)'
                      : 'var(--semi-color-fill-2)',
                    boxShadow: busyLogin ? '0 0 6px var(--semi-color-success)' : 'none',
                    transition: 'all 0.3s',
                  }}
                />
                <Typography.Text strong style={{ fontSize: 13 }}>
                  实时终端日志
                </Typography.Text>
              </div>
              <div style={{ display: 'flex', gap: 4 }}>
                <Button
                  size="small"
                  icon={<IconCopy />}
                  disabled={!fullLog}
                  onClick={() => void copyToClipboard(fullLog)}
                />
                <Button
                  size="small"
                  icon={<IconDelete />}
                  disabled={busyLogin}
                  onClick={clearConsole}
                />
                <Button
                  size="small"
                  onClick={() => setAutoFollowLog((v) => !v)}
                  style={{ fontSize: 11 }}
                >
                  {autoFollowLog ? '🔒 跟随' : '🔓 自由'}
                </Button>
              </div>
            </div>
            <div style={{ padding: '14px 20px' }}>
              {lastCommandLabel && (
                <Typography.Text
                  type="tertiary"
                  size="small"
                  style={{ display: 'block', marginBottom: 8 }}
                >
                  {lastCommandLabel}
                </Typography.Text>
              )}
              <UrlActions urls={extractedUrls} />
              <TerminalLogPanel fullLog={fullLog} busy={busyLogin} logWrapRef={logWrapRef} />
            </div>
          </Card>

          {/* 当前探测（折叠） */}
          <Collapse style={{ borderRadius: 12 }}>
            <Collapse.Panel header="当前探测（agent status）" itemKey="status">
              <pre
                style={{
                  margin: 0,
                  padding: 12,
                  background: 'rgba(6,7,9,0.03)',
                  borderRadius: 8,
                  fontSize: 12,
                  maxHeight: 120,
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  border: '1px solid var(--semi-color-border)',
                }}
              >
                {loadingStatus ? '加载中…' : statusText || '（暂无）'}
              </pre>
            </Collapse.Panel>
          </Collapse>

          {/* 说明 */}
          <Banner
            type="info"
            fullMode={false}
            title="说明"
            description={
              <div style={{ lineHeight: 1.65 }}>
                <div>
                  非交互命令超时见配置项{' '}
                  <Typography.Text code>agent.terminal_exec_timeout</Typography.Text>； 本页{' '}
                  <Typography.Text code>agent login</Typography.Text>{' '}
                  走交互式终端，一般不受该上限约束。
                </div>
                <div style={{ marginTop: 6 }}>
                  若长时间无输出，请确认 Nginx 对{' '}
                  <Typography.Text code>/api/v1/admin/terminal/ws</Typography.Text> 已配置{' '}
                  <Typography.Text code>Upgrade: websocket</Typography.Text>，且{' '}
                  <Typography.Text code>PATH</Typography.Text> 含{' '}
                  <Typography.Text code>agent</Typography.Text>。
                </div>
                <div style={{ marginTop: 6 }}>
                  持久化：可将宿主机 <Typography.Text code>~/.cursor</Typography.Text>{' '}
                  挂载到容器内同路径。
                </div>
              </div>
            }
            style={{ borderRadius: 10 }}
          />
        </>
      )}
    </div>
  );
};

export default CursorCliSection;
