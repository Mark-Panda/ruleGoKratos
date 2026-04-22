/**
 * 管理端「飞书 CLI」：仅保留终端交互式配置（config init --new -> auth login --recommend）。
 * 参考：https://github.com/larksuite/cli/blob/main/README.zh.md
 */
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
  Banner,
  Button,
  Card,
  Spin,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';

import { runTerminal } from '../../services/api-agent';
import { getApiOrigin, getAuthToken } from '../../services/http';

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

/** 从命令输出中提取飞书开放平台 / 账号相关的 https 链接（供浏览器打开） */
function extractLarkHttpsUrls(raw: string): string[] {
  const re =
    /https:\/\/[^\s\)\]'"]+/g;
  const candidates = raw.match(re) ?? [];
  const filtered = candidates.filter((u) => {
    try {
      const host = new URL(u).hostname;
      return (
        host.endsWith('feishu.cn') ||
        host.endsWith('larksuite.com') ||
        host.endsWith('feishu.net')
      );
    } catch {
      return false;
    }
  });
  return [...new Set(filtered)];
}

/** 从 device verify URL 解析 user_code，便于对照终端截图 */
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
        从此输出解析到的链接（终端里的二维码无法在网页中扫码，请优先用浏览器打开下列链接）
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
              <a href={href} target="_blank" rel="noopener noreferrer" style={{ wordBreak: 'break-all', flex: '1 1 220px' }}>
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

  const extractedUrls = useMemo(() => extractLarkHttpsUrls([lastStdout, lastStderr].join('\n')), [lastStdout, lastStderr]);

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
      // 去除 ANSI 颜色控制序列，保留字符画二维码主体，便于复制链接/阅读日志
      const clean = rawChunk.replace(/\u001b\[[0-9;?]*[ -/]*[@-~]/g, '');
      logBufRef.current += clean;
      setLastStdout(logBufRef.current);
      if (!sentAuthRef.current && /OK:\s*应用配置成功/i.test(logBufRef.current)) {
        sentAuthRef.current = true;
        setAutoSetupStage('步骤 2/2：执行 lark-cli auth login --recommend');
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
        setAutoSetupStage('配置完成');
        setLastExit(0);
        Toast.success('自动交互式配置完成，授权成功。');
        void refreshStatus();
      }
    },
    [refreshStatus]
  );

  const runAutoInteractiveSetup = useCallback(async () => {
    setBusyAutoSetup(true);
    setAutoSetupStage('步骤 1/2：执行 lark-cli config init --new');
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

  return (
    <div style={{ padding: 24, maxWidth: 960 }}>
      <Typography.Title heading={6} style={{ margin: 0 }}>
        飞书 CLI（lark-cli）配置
      </Typography.Title>

      {configured && statusParsed?.tokenStatus === 'valid' && (
        <Banner
          type="success"
          fullMode={false}
          title="飞书 CLI 已配置且令牌有效"
          description={
            <div style={{ lineHeight: 1.6 }}>
              <div>
                用户：<strong>{statusParsed.userName ?? '—'}</strong>（{statusParsed.identity ?? '—'}）
              </div>
              <div>应用 ID：{statusParsed.appId ?? '—'} · 品牌：{statusParsed.brand ?? '—'}</div>
              <div>令牌创建时间（东八区）：{formatToCnTime(statusParsed.grantedAt)}</div>
            </div>
          }
          style={{ marginTop: 12 }}
        />
      )}

      {!configured && (
        <Banner
          type="warning"
          fullMode={false}
          title="尚未检测到有效登录"
          description="点击下方“一键自动交互式配置”，先完成应用配置，再自动进入授权。完成后点“刷新状态”。"
          style={{ marginTop: 12 }}
        />
      )}

      <Card title="一键终端交互式配置（推荐）" style={{ marginTop: 16 }}>
        <Typography.Paragraph type="tertiary" size="small" style={{ marginTop: 0 }}>
          严格按顺序执行：先 <Typography.Text code>lark-cli config init --new</Typography.Text>，
          检测到「OK: 应用配置成功」后，再自动执行{' '}
          <Typography.Text code>lark-cli auth login --recommend</Typography.Text> 并等待授权完成。
        </Typography.Paragraph>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10, flexWrap: 'wrap' }}>
          <Tag color={busyAutoSetup ? 'blue' : configured ? 'green' : 'orange'}>
            {busyAutoSetup ? `运行中：${autoSetupStage || '处理中'}` : configured ? '已登录' : '未登录'}
          </Tag>
          {lastExit !== null && <Tag color={lastExit === 0 ? 'green' : 'red'}>最近退出码：{lastExit}</Tag>}
        </div>
        {busyAutoSetup && (
          <Banner
            type="info"
            fullMode={false}
            closeIcon={null}
            icon={<Spin size="small" />}
            title="自动交互式配置进行中"
            description={autoSetupStage || '请在浏览器/飞书完成扫码和授权，命令会自动继续。'}
            style={{ marginBottom: 10 }}
          />
        )}
        <Button
          theme="solid"
          type="primary"
          loading={busyAutoSetup}
          disabled={anyLongRunning && !busyAutoSetup}
          onClick={() => void runAutoInteractiveSetup()}
        >
          开始自动交互式配置
        </Button>
        <div style={{ marginTop: 12, display: 'flex', flexWrap: 'wrap', gap: 8 }}>
          <Button loading={loadingStatus} disabled={anyLongRunning} onClick={() => void refreshStatus()}>
            刷新状态（lark-cli auth status）
          </Button>
          <Button
            disabled={!busyAutoSetup}
            onClick={() => {
              closeAutoSetupSocket();
              setBusyAutoSetup(false);
              setAutoSetupStage('');
            }}
          >
            停止当前流程
          </Button>
        </div>
      </Card>

      <Banner
        type="info"
        fullMode={false}
        title="与命令行一致的流程说明"
        description={
          <div style={{ lineHeight: 1.65 }}>
            <div>
              <strong>步骤 1</strong> 会实时打印终端字符二维码与配置链接；可直接在下方日志区复制链接
              （二维码在网页字体下可能不够清晰时，优先使用链接）：
              <Typography.Text code>open.feishu.cn/page/cli</Typography.Text> 链接在浏览器完成应用创建。
            </div>
            <div style={{ marginTop: 8 }}>
              <strong>步骤 2</strong> 会输出「在浏览器中打开以下链接进行认证」并进入「等待用户授权…」；
              与你在容器里手动执行的体验保持一致。
            </div>
            <div style={{ marginTop: 8 }}>
              服务端单次命令超时由 <Typography.Text code>agent.terminal_exec_timeout</Typography.Text>{' '}
              控制；如授权耗时较长可适当调大。
            </div>
          </div>
        }
        style={{ marginTop: 16 }}
      />

      <Card
        title="当前登录状态（lark-cli auth status）"
        style={{ marginTop: 16 }}
        bodyStyle={{ paddingTop: 12 }}
      >
        <pre
          style={{
            margin: 0,
            padding: 10,
            background: 'rgba(6,7,9,0.04)',
            borderRadius: 4,
            fontSize: 12,
            maxHeight: 200,
            overflow: 'auto',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}
        >
          {loadingStatus ? '加载中…' : statusText || '（暂无）'}
        </pre>
      </Card>

      <Card title="实时终端日志" style={{ marginTop: 16 }}>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center', marginBottom: 8 }}>
          <Button
            size="small"
            disabled={!fullLog}
            onClick={() => void copyToClipboard(fullLog)}
          >
            复制全部日志
          </Button>
          <Button size="small" disabled={anyLongRunning} onClick={clearConsole}>
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

        <UrlActions urls={extractedUrls} />

        <div
          ref={logWrapRef}
          style={{
            marginTop: 12,
            padding: 12,
            background: '#0f1720',
            color: '#e6edf3',
            borderRadius: 6,
            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
            fontSize: 11,
            lineHeight: 1.45,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            maxHeight: 420,
            overflow: 'auto',
            border: '1px solid rgba(148,163,184,0.25)',
          }}
        >
          {fullLog || '等待命令输出...'}
        </div>
      </Card>
    </div>
  );
};

export default LarkCliSection;
