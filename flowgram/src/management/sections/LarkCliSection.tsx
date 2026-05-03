/**
 * 管理端「飞书 CLI」：终端交互式配置（config init --new -> auth login --recommend）。
 * 参考：https://github.com/larksuite/cli/blob/main/README.zh.md
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

/* ───── 类型 & 工具函数 ───── */

interface AuthStatusParsed {
  tokenStatus?: string;
  ok?: boolean;
  userName?: string;
  identity?: string;
  grantedAt?: string;
  appId?: string;
  brand?: string;
}

function parseAuthStdout(raw: string): AuthStatusParsed | null {
  const s = raw.trim();
  if (!s) return null;
  try {
    return JSON.parse(s) as AuthStatusParsed;
  } catch {
    return null;
  }
}

function isConfigured(parsed: AuthStatusParsed | null): boolean {
  return parsed?.tokenStatus === 'valid';
}

function formatToCnTime(raw?: string): string {
  if (!raw) return '—';
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return raw;
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(d);
}

function extractLarkHttpsUrls(raw: string): string[] {
  const re = /https:\/\/[^\s\)\]'"]+/g;
  const candidates = raw.match(re) ?? [];
  const filtered = candidates.filter((u) => {
    try {
      const host = new URL(u).hostname;
      return (
        host.endsWith('feishu.cn') || host.endsWith('larksuite.com') || host.endsWith('feishu.net')
      );
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

/* ───── 步骤指示器 ───── */

interface StepInfo {
  key: string;
  label: string;
  desc: string;
}

const SETUP_STEPS: StepInfo[] = [
  { key: 'config', label: '应用配置', desc: 'lark-cli config init --new' },
  { key: 'auth', label: '授权登录', desc: 'lark-cli auth login --recommend' },
];

function StepIndicator({ current }: { current: string }) {
  const idx = current === 'auth' ? 1 : current === 'config' ? 0 : -1;
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 0, marginTop: 12, marginBottom: 4 }}>
      {SETUP_STEPS.map((step, i) => {
        const done = idx > i || current === 'done';
        const active = idx === i;
        return (
          <React.Fragment key={step.key}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: '50%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 12,
                  fontWeight: 600,
                  background: done
                    ? 'var(--semi-color-success)'
                    : active
                    ? 'var(--semi-color-primary)'
                    : 'var(--semi-color-fill-1)',
                  color: done || active ? '#fff' : 'var(--semi-color-text-2)',
                  transition: 'all 0.25s',
                }}
              >
                {done ? '✓' : i + 1}
              </div>
              <div>
                <Typography.Text
                  strong={active}
                  style={{
                    color: done
                      ? 'var(--semi-color-success)'
                      : active
                      ? 'var(--semi-color-primary)'
                      : 'var(--semi-color-text-2)',
                    fontSize: 13,
                  }}
                >
                  {step.label}
                </Typography.Text>
                <Typography.Text
                  type="tertiary"
                  size="small"
                  style={{ display: 'block', marginTop: -2 }}
                >
                  {step.desc}
                </Typography.Text>
              </div>
            </div>
            {i < SETUP_STEPS.length - 1 && (
              <div
                style={{
                  flex: '0 0 32px',
                  height: 2,
                  background:
                    done || active ? 'var(--semi-color-primary)' : 'var(--semi-color-fill-1)',
                  borderRadius: 1,
                  margin: '0 12px',
                  alignSelf: 'center',
                  transition: 'background 0.25s',
                }}
              />
            )}
          </React.Fragment>
        );
      })}
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
        终端中的二维码无法在网页中扫码，请优先用浏览器打开下列链接
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
  maxHeight: 420,
  overflow: 'auto',
  border: '1px solid rgba(148,163,184,0.2)',
  boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.15)',
};

function TerminalLogPanel({
  fullLog,
  busy,
  lastCommandLabel,
  lastExit,
  logWrapRef,
}: {
  fullLog: string;
  busy: boolean;
  lastCommandLabel: string;
  lastExit: number | null;
  logWrapRef: React.Ref<HTMLDivElement>;
}) {
  return (
    <div ref={logWrapRef} style={TERMINAL_STYLE}>
      {fullLog ||
        (busy ? '⏳ 正在连接终端，等待输出…' : '等待命令输出… 请先点击「开始自动交互式配置」')}
    </div>
  );
}

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

export const LarkCliSection: React.FC = () => {
  const [loadingStatus, setLoadingStatus] = useState(false);
  const [statusText, setStatusText] = useState('');
  const [statusParsed, setStatusParsed] = useState<AuthStatusParsed | null>(null);

  const [busyAutoSetup, setBusyAutoSetup] = useState(false);
  const [autoSetupStage, setAutoSetupStage] = useState('');

  const [lastStdout, setLastStdout] = useState('');
  const [lastStderr, setLastStderr] = useState('');
  const [lastExit, setLastExit] = useState<number | null>(null);
  const [lastCommandLabel, setLastCommandLabel] = useState('');
  const [autoFollowLog, setAutoFollowLog] = useState(true);
  const wsRef = useRef<WebSocket | null>(null);
  const logWrapRef = useRef<HTMLDivElement | null>(null);
  const logBufRef = useRef('');
  const sentAuthRef = useRef(false);
  const sentExitRef = useRef(false);
  const doneRef = useRef(false);

  const extractedUrls = useMemo(
    () => extractLarkHttpsUrls([lastStdout, lastStderr].join('\n')),
    [lastStdout, lastStderr]
  );

  const refreshStatus = useCallback(async () => {
    setLoadingStatus(true);
    try {
      const res = await runTerminal('lark-cli auth status 2>&1', '/app');
      const out = [res.stdout, res.stderr].filter(Boolean).join('\n');
      setStatusText(out);
      setStatusParsed(parseAuthStdout(out));
    } catch (e) {
      setStatusText(e instanceof Error ? e.message : String(e));
      setStatusParsed(null);
    } finally {
      setLoadingStatus(false);
    }
  }, []);

  useEffect(() => {
    void refreshStatus();
  }, [refreshStatus]);

  useEffect(() => {
    if (!autoFollowLog) return;
    if (!logWrapRef.current) return;
    logWrapRef.current.scrollTop = logWrapRef.current.scrollHeight;
  }, [lastStdout, lastStderr, autoFollowLog]);

  const configured = isConfigured(statusParsed);

  const closeAutoSetupSocket = useCallback(() => {
    try {
      wsRef.current?.close();
    } catch {
      // ignore
    }
    wsRef.current = null;
  }, []);

  useEffect(() => () => closeAutoSetupSocket(), [closeAutoSetupSocket]);

  const buildTerminalWsURL = useCallback((cwd: string): string => {
    const u = new URL(getApiOrigin());
    u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:';
    const params = new URLSearchParams();
    params.set('cwd', cwd);
    const tok = getAuthToken();
    if (tok) params.set('token', tok);
    return `${u.origin}/api/v1/admin/terminal/ws?${params.toString()}`;
  }, []);

  const appendConsole = useCallback(
    (rawChunk: string) => {
      const clean = rawChunk.replace(/\[[0-9;?]*[ -/]*[@-~]/g, '');
      logBufRef.current += clean;
      setLastStdout(logBufRef.current);
      if (!sentAuthRef.current && /OK:\s*应用配置成功/i.test(logBufRef.current)) {
        sentAuthRef.current = true;
        setAutoSetupStage('auth');
        try {
          const ws = wsRef.current;
          if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(new TextEncoder().encode('\n# auto-next\nlark-cli auth login --recommend\n'));
            logBufRef.current += '\n$ lark-cli auth login --recommend\n';
            setLastStdout(logBufRef.current);
          }
        } catch {}
      }
      if (!doneRef.current && /OK:\s*授权成功/i.test(logBufRef.current)) {
        doneRef.current = true;
        setAutoSetupStage('done');
        setLastExit(0);
        Toast.success('自动交互式配置完成，授权成功。');
        void refreshStatus();
      }
    },
    [refreshStatus]
  );

  const runAutoInteractiveSetup = useCallback(async () => {
    setBusyAutoSetup(true);
    setAutoSetupStage('config');
    logBufRef.current = '$ lark-cli config init --new\n';
    setLastStdout(logBufRef.current);
    setLastStderr('');
    setLastExit(null);
    setLastCommandLabel('自动配置（config init --new -> auth login --recommend）');
    sentAuthRef.current = false;
    sentExitRef.current = false;
    doneRef.current = false;
    closeAutoSetupSocket();
    try {
      const ws = new WebSocket(buildTerminalWsURL('/app'));
      ws.binaryType = 'arraybuffer';
      wsRef.current = ws;
      ws.onopen = () => {
        try {
          ws.send(new TextEncoder().encode('lark-cli config init --new\n'));
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
        if (
          doneRef.current &&
          !sentExitRef.current &&
          (/OK:\s*授权成功/i.test(logBufRef.current) || /\$|#\s*$/.test(chunk))
        ) {
          sentExitRef.current = true;
          try {
            ws.send(new TextEncoder().encode('\nexit\n'));
          } catch {}
          setTimeout(() => {
            closeAutoSetupSocket();
            setBusyAutoSetup(false);
            setAutoSetupStage('');
          }, 200);
        }
      };
      ws.onerror = () => {
        setLastStderr((prev) => `${prev}${prev ? '\n' : ''}[WebSocket 错误]`);
      };
      ws.onclose = () => {
        wsRef.current = null;
        setBusyAutoSetup(false);
        if (doneRef.current) {
          setAutoSetupStage('');
          void refreshStatus();
        } else {
          Toast.warning('终端连接已关闭；请检查输出或重试。');
        }
      };
    } catch (e) {
      setLastStderr(e instanceof Error ? e.message : String(e));
      setLastExit(-1);
      Toast.error('自动流程执行失败');
    }
  }, [appendConsole, buildTerminalWsURL, closeAutoSetupSocket]);

  const anyLongRunning = busyAutoSetup;
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
                background: 'linear-gradient(135deg, #3370ff, #5e8cff)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#fff',
                fontSize: 18,
                fontWeight: 700,
                flexShrink: 0,
              }}
            >
              L
            </div>
            <div>
              <Typography.Title heading={6} style={{ margin: 0 }}>
                飞书 CLI
              </Typography.Title>
              <Typography.Text type="tertiary" size="small">
                lark-cli 配置与授权管理
              </Typography.Text>
            </div>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <StatusDot
              color={configured ? 'var(--semi-color-success)' : 'var(--semi-color-warning)'}
            />
            <Tag color={configured ? 'green' : 'orange'} size="small">
              {configured ? '已授权' : '未授权'}
            </Tag>
          </div>
        </div>
      </Card>

      {/* 已配置 — 状态信息卡 */}
      {configured && statusParsed?.tokenStatus === 'valid' && (
        <Card style={CARD_STYLE} bodyStyle={{ padding: 0 }}>
          <div
            style={{
              background: 'linear-gradient(135deg, rgba(51,112,255,0.06), rgba(51,112,255,0.02))',
              padding: '18px 20px',
              borderRadius: '12px 12px 0 0',
              borderBottom: '1px solid var(--semi-color-border)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <StatusDot color="var(--semi-color-success)" />
              <Typography.Text strong style={{ fontSize: 14 }}>
                令牌有效
              </Typography.Text>
            </div>
          </div>
          <div style={{ padding: '16px 20px' }}>
            <InfoRow label="用户">{statusParsed.userName ?? '—'}</InfoRow>
            <InfoRow label="身份">{statusParsed.identity ?? '—'}</InfoRow>
            <InfoRow label="应用 ID">
              <Typography.Text code>{statusParsed.appId ?? '—'}</Typography.Text>
            </InfoRow>
            <InfoRow label="品牌">{statusParsed.brand ?? '—'}</InfoRow>
            <InfoRow label="授权时间">{formatToCnTime(statusParsed.grantedAt)}</InfoRow>
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
      )}

      {/* 未配置 — 告警 */}
      {!configured && (
        <Banner
          type="warning"
          fullMode={false}
          title="尚未检测到有效登录"
          description="点击下方「开始自动交互式配置」，完成应用配置与授权登录。"
          style={{ borderRadius: 10 }}
        />
      )}

      {/* 一键配置卡片 */}
      <Card style={CARD_STYLE} bodyStyle={{ padding: 0 }}>
        <div style={{ padding: '18px 20px' }}>
          <Typography.Title heading={6} style={{ margin: 0, marginBottom: 4 }}>
            一键终端交互式配置
          </Typography.Title>
          <Typography.Text type="tertiary" size="small">
            严格按顺序执行两步操作，自动衔接
          </Typography.Text>
        </div>
        <StepIndicator current={autoSetupStage} />
        <Divider margin="12px" />
        <div style={{ padding: '0 20px 16px' }}>
          {busyAutoSetup && (
            <Banner
              type="info"
              fullMode={false}
              closeIcon={null}
              icon={<Spin size="small" />}
              title={
                autoSetupStage === 'config'
                  ? '步骤 1/2：应用配置中'
                  : autoSetupStage === 'auth'
                  ? '步骤 2/2：等待授权'
                  : '处理中'
              }
              description={
                autoSetupStage === 'config'
                  ? '请根据终端输出，在浏览器中打开链接完成应用创建'
                  : '请在浏览器/飞书完成扫码授权，命令会自动继续'
              }
              style={{ marginBottom: 12, borderRadius: 8 }}
            />
          )}
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            <Button
              theme="solid"
              type="primary"
              size="large"
              loading={busyAutoSetup}
              disabled={anyLongRunning && !busyAutoSetup}
              onClick={() => void runAutoInteractiveSetup()}
              style={{ borderRadius: 8, fontWeight: 600 }}
            >
              {busyAutoSetup ? '配置进行中…' : '开始自动交互式配置'}
            </Button>
            <Button
              icon={<IconRefresh />}
              loading={loadingStatus}
              disabled={anyLongRunning}
              onClick={() => void refreshStatus()}
            >
              刷新状态
            </Button>
            <Button
              icon={<IconStop />}
              disabled={!busyAutoSetup}
              type="danger"
              onClick={() => {
                closeAutoSetupSocket();
                setBusyAutoSetup(false);
                setAutoSetupStage('');
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
                background: busyAutoSetup
                  ? 'var(--semi-color-success)'
                  : 'var(--semi-color-fill-2)',
                boxShadow: busyAutoSetup ? '0 0 6px var(--semi-color-success)' : 'none',
                transition: 'all 0.3s',
              }}
            />
            <Typography.Text strong style={{ fontSize: 13 }}>
              实时终端日志
            </Typography.Text>
            {lastExit !== null && (
              <Tag color={lastExit === 0 ? 'green' : 'red'} size="small">
                exit {lastExit}
              </Tag>
            )}
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
              disabled={anyLongRunning}
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
          <TerminalLogPanel
            fullLog={fullLog}
            busy={busyAutoSetup}
            lastCommandLabel={lastCommandLabel}
            lastExit={lastExit}
            logWrapRef={logWrapRef}
          />
        </div>
      </Card>

      {/* 原始状态输出（折叠） */}
      <Collapse style={{ borderRadius: 12 }}>
        <Collapse.Panel header="原始状态输出（lark-cli auth status）" itemKey="status">
          <pre
            style={{
              margin: 0,
              padding: 12,
              background: 'rgba(6,7,9,0.03)',
              borderRadius: 8,
              fontSize: 12,
              maxHeight: 200,
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
        title="流程说明"
        description={
          <div style={{ lineHeight: 1.65 }}>
            <div>
              <strong>步骤 1</strong>{' '}
              会实时打印终端字符二维码与配置链接；链接也可在下方日志区的「检测到授权链接」面板中一键复制。
            </div>
            <div style={{ marginTop: 6 }}>
              <strong>步骤 2</strong> 会输出授权链接并等待扫码；完成授权后自动检测成功。
            </div>
            <div style={{ marginTop: 6 }}>
              服务端单次命令超时由{' '}
              <Typography.Text code>agent.terminal_exec_timeout</Typography.Text>{' '}
              控制；如授权耗时较长可适当调大。
            </div>
          </div>
        }
        style={{ borderRadius: 10 }}
      />
    </div>
  );
};

export default LarkCliSection;
