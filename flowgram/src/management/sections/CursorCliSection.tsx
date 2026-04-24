/**
 * 管理端「Cursor CLI」：终端内执行 `agent login`（容器内配合 NO_OPEN_BROWSER 提取浏览器链接）。
 * 参考：https://cursor.com/docs/cli/overview
 */
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { Banner, Button, Card, Spin, Tag, Toast, Typography } from '@douyinfe/semi-ui';

import { getApiOrigin, getAuthToken } from '../../services/http';
import { runTerminal } from '../../services/api-agent';

/** 与容器内 agent status 一致：✓ Logged in as …；旧版 CLI 可能仍输出 Login successful */
function isLoginOkFromStatusText(raw: string): boolean {
  const t = raw.trim();
  if (!t || /^Not logged in\b/i.test(t)) return false;
  return /Login successful|Logged in as\b/i.test(t);
}

/** 从 agent status 输出解析账号展示用文案 */
function parseLoggedInAccount(raw: string): string | null {
  const m = raw.trim().match(/Logged in as\s+(\S+)/i);
  return m?.[1] ? m[1] : null;
}

/** 从输出中提取 Cursor 登录相关 https 链接 */
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

/** 从 device verify URL 解析 user_code（飞书 OAuth）；Cursor 链接通常无此项 */
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

function UrlActions({ urls }: { urls: string[] }) {
  if (!urls.length) return null;
  return (
    <div
      style={{
        marginTop: 12,
        padding: 12,
        background: 'var(--semi-color-fill-0)',
        borderRadius: 6,
        border: '1px solid rgba(28,31,35,0.08)',
      }}
    >
      <Typography.Text strong style={{ display: 'block', marginBottom: 8 }}>
        从此输出解析到的链接（请在浏览器中打开完成登录）
      </Typography.Text>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
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
                padding: '8px 10px',
                background: '#fff',
                borderRadius: 4,
                border: '1px solid rgba(28,31,35,0.06)',
              }}
            >
              <a
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                style={{ wordBreak: 'break-all', flex: '1 1 220px' }}
              >
                {href}
              </a>
              {code && (
                <Typography.Text type="tertiary" size="small">
                  用户码：{code}
                </Typography.Text>
              )}
              <Button size="small" onClick={() => void copyToClipboard(href)}>
                复制链接
              </Button>
            </div>
          );
        })}
      </div>
    </div>
  );
}

const LOGIN_CMD = 'NO_OPEN_BROWSER=1 agent login';

/** 与容器内 agent login 成功回显一致（见 loginDeepControl 流程） */
const RE_LOGIN_SUCCESS =
  /Login successful|Logged in as\b|Authentication tokens stored securely/i;
const RE_LOGIN_FAILED = /Login failed or timed out/i;

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
  const showTerminalPanel = !configured;

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
    if (!showTerminalPanel || !autoFollowLog) return;
    if (!logWrapRef.current) return;
    logWrapRef.current.scrollTop = logWrapRef.current.scrollHeight;
  }, [lastStdout, lastStderr, autoFollowLog, showTerminalPanel]);

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
      const clean = rawChunk.replace(/\u001b\[[0-9;?]*[ -/]*[@-~]/g, '');
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

  const preBoxStyle: React.CSSProperties = {
    margin: 0,
    padding: '12px 14px',
    background: 'rgba(6,7,9,0.04)',
    borderRadius: 8,
    fontSize: 12,
    overflow: 'auto',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
    border: '1px solid rgba(28,31,35,0.06)',
  };

  return (
    <div
      style={{
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
      }}
    >
      <div>
        <Typography.Title heading={6} style={{ margin: 0 }}>
          Cursor CLI（agent）
        </Typography.Title>
        <Typography.Text type="tertiary" size="small" style={{ display: 'block', marginTop: 6 }}>
          {configured
            ? '当前环境已与 Cursor 账号绑定，可直接使用 agent 相关能力。'
            : '通过 WebSocket 终端完成浏览器授权登录。'}
        </Typography.Text>
      </div>

      {configured ? (
        <Card
          title="登录状态"
          style={{ borderRadius: 10, boxShadow: '0 1px 0 rgba(28,31,35,0.04)' }}
          bodyStyle={{ paddingTop: 8 }}
        >
          <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 10, marginBottom: 12 }}>
            <Tag color="green">已登录</Tag>
            {loggedInAccount && (
              <Typography.Text strong style={{ fontSize: 14 }}>
                {loggedInAccount}
              </Typography.Text>
            )}
          </div>
          <Typography.Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 10 }}>
            <Typography.Text code>agent status</Typography.Text> 输出如下。更换账号请先在环境执行{' '}
            <Typography.Text code>agent logout</Typography.Text>，再刷新本页按未登录流程操作。Docker
            持久化可将宿主机 <Typography.Text code>~/.cursor</Typography.Text> 挂载到容器同路径。
          </Typography.Text>
          <pre style={{ ...preBoxStyle, maxHeight: 160 }}>{loadingStatus ? '加载中…' : statusText || '（暂无）'}</pre>
          <div style={{ marginTop: 14, display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            <Button loading={loadingStatus} onClick={() => void refreshStatus()}>
              刷新状态
            </Button>
          </div>
        </Card>
      ) : (
        <>
          <Banner
            type="warning"
            fullMode={false}
            title="尚未检测到有效登录"
            description="在下方发起登录后，请在「实时终端日志」中打开解析出的链接完成浏览器授权（已设置 NO_OPEN_BROWSER）。"
            style={{ borderRadius: 10 }}
          />

          <Card
            title="登录操作"
            style={{ borderRadius: 10, boxShadow: '0 1px 0 rgba(28,31,35,0.04)' }}
            bodyStyle={{ paddingTop: 8 }}
          >
            <Typography.Paragraph type="tertiary" size="small" style={{ marginTop: 0, marginBottom: 12 }}>
              将执行 <Typography.Text code>{LOGIN_CMD}</Typography.Text>；回显中的{' '}
              <Typography.Text code>loginDeepControl</Typography.Text> 链接也会单独列出。
            </Typography.Paragraph>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12, flexWrap: 'wrap' }}>
              <Tag color={busyLogin ? 'blue' : 'orange'}>
                {busyLogin ? `运行中：${loginStage || '处理中'}` : '未登录'}
              </Tag>
              {lastExit !== null && (
                <Tag color={lastExit === 0 ? 'green' : 'red'}>最近退出码：{lastExit}</Tag>
              )}
            </div>
            {busyLogin && (
              <Banner
                type="info"
                fullMode={false}
                closeIcon={null}
                icon={<Spin size="small" />}
                title="登录进行中"
                description={loginStage || '请在浏览器完成授权。'}
                style={{ marginBottom: 12, borderRadius: 8 }}
              />
            )}
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
              <Button
                theme="solid"
                type="primary"
                loading={busyLogin}
                disabled={busyLogin}
                onClick={() => void runAgentLogin()}
              >
                开始 agent login
              </Button>
              <Button loading={loadingStatus} disabled={busyLogin} onClick={() => void refreshStatus()}>
                刷新状态
              </Button>
              <Button
                disabled={!busyLogin}
                onClick={() => {
                  closeLoginSocket();
                  setBusyLogin(false);
                  setLoginStage('');
                }}
              >
                停止当前流程
              </Button>
            </div>
          </Card>

          <Card
            title="实时终端日志（WebSocket PTY）"
            style={{ borderRadius: 10, boxShadow: '0 1px 0 rgba(28,31,35,0.04)' }}
            bodyStyle={{ paddingTop: 8 }}
          >
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center', marginBottom: 8 }}>
              <Button size="small" disabled={!fullLog} onClick={() => void copyToClipboard(fullLog)}>
                复制全部日志
              </Button>
              <Button size="small" disabled={busyLogin} onClick={clearConsole}>
                清空日志
              </Button>
              <Button size="small" onClick={() => setAutoFollowLog((v) => !v)}>
                {autoFollowLog ? '关闭自动滚动' : '开启自动滚动'}
              </Button>
            </div>
            <Typography.Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 8 }}>
              {lastCommandLabel ? `命令：${lastCommandLabel}` : '命令：尚未开始'}
              {lastExit !== null ? ` · exit code: ${lastExit}` : ''}
            </Typography.Text>
            <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 10 }}>
              若长时间无输出，请确认 Nginx 对 <Typography.Text code>/api/v1/admin/terminal/ws</Typography.Text>{' '}
              已配置 <Typography.Text code>Upgrade: websocket</Typography.Text>，且 <Typography.Text code>PATH</Typography.Text>{' '}
              含 <Typography.Text code>agent</Typography.Text>。
            </Typography.Paragraph>

            <UrlActions urls={extractedUrls} />

            <div
              ref={logWrapRef}
              style={{
                marginTop: 10,
                padding: 12,
                background: '#0b1220',
                color: '#e2e8f0',
                borderRadius: 8,
                fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
                fontSize: 11,
                lineHeight: 1.5,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                maxHeight: 380,
                overflow: 'auto',
                border: '1px solid rgba(100,116,139,0.35)',
              }}
            >
              {fullLog ||
                (busyLogin ? '正在连接终端，等待输出…' : '等待命令输出… 请先点击「开始 agent login」')}
            </div>
          </Card>

          <Card title="当前探测（agent status）" style={{ borderRadius: 10 }} bodyStyle={{ paddingTop: 8 }}>
            <pre style={{ ...preBoxStyle, maxHeight: 120 }}>
              {loadingStatus ? '加载中…' : statusText || '（暂无）'}
            </pre>
          </Card>
        </>
      )}

      {!configured && (
        <Banner
          type="info"
          fullMode={false}
          title="说明"
          description={
            <div style={{ lineHeight: 1.65 }}>
              <div>
                非交互命令超时见配置项 <Typography.Text code>agent.terminal_exec_timeout</Typography.Text>；本页{' '}
                <Typography.Text code>agent login</Typography.Text> 走交互式终端，一般不受该上限约束。
              </div>
              <div style={{ marginTop: 8 }}>
                持久化：可将宿主机 <Typography.Text code>~/.cursor</Typography.Text> 挂载到容器内同路径。
              </div>
            </div>
          }
          style={{ borderRadius: 10 }}
        />
      )}
    </div>
  );
};

export default CursorCliSection;
