import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { marked } from 'marked';
import {
  Button,
  Empty,
  Popconfirm,
  Select,
  Spin,
  TextArea,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { IconCopy, IconPlus, IconUpload } from '@douyinfe/semi-icons';

import {
  buildChatAttachmentsFromFiles,
  mergeMessageForChatDisplay,
  streamChat,
  type ChatAttachmentPayload,
} from '../../services/api-chat';
import { listLlmConfigs, type LlmConfigItem } from '../../services/api-agent';
import { listManagedAgents } from '../../services/api-managed-agents';

marked.use({
  breaks: true,
});

const STORAGE_MODEL_KEY = 'flowgram-overview-chat-model-v1';
const STORAGE_CHAT_STORE_KEY = 'flowgram-overview-chat-store-v1';
const STORAGE_MANAGED_AGENT_KEY = 'flowgram-overview-chat-managed-agent-v1';

/** 页面引导：能力与「如何提问」（空状态 + 输入区提示） */
const CHAT_GUIDE = {
  title: 'Code 助手',
  subtitle:
    '对接 RuleGo Agent Harness（与画布「Agent LLM / ai/agentHarness」同源）。在凭证与模型就绪时，可通过工具增强回答（SKILL、MCP、工作区文件与 Shell 等，以服务端实际启用为准）。',
  steps: [
    '在顶部可选「Agent 配置」托管档案（统一系统提示/SKILL/MCP/模型），或选择「对话模型」：条目来自「Agent 管理 → 模型管理」。二者至少选其一。',
    '输入时尽量包含：要解决什么、当前现象或报错、期望交付物（示例代码 / 步骤列表 / 取舍说明）；可粘贴日志或代码。',
    '「清除上下文」：之后请求不再附带此前对话历史，界面记录仍保留。「重置聊天」：清空本会话消息。',
    '可点击「附件」上传多个文件（文本直接嵌入，二进制以 Base64 交由模型按说明解析）；数据保存在本机浏览器。',
  ],
  inputHint:
    '提问建议：背景 → 现状或报错 → 期望结果；可附加代码/日志文件；涉及仓库时请写路径、分支或相关接口名。',
  placeholder: '目标 + 现状/报错 + 期望输出（可贴代码）。Enter 发送，Shift+Enter 换行',
};

const MD_SURFACE_STYLE: React.CSSProperties = {
  fontSize: 14,
  lineHeight: 1.62,
  color: 'rgba(28,32,41,0.92)',
};

type Msg = { role: 'user' | 'assistant'; content: string };

type ModelPick = { configId: number; entryId: number; label: string };

type ChatSession = {
  id: string;
  title: string;
  updatedAt: number;
  messages: Msg[];
  historyStartIndex: number;
};

type ChatStore = {
  sessions: ChatSession[];
  activeId: string | null;
};

function newSessionId(): string {
  return `s_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 11)}`;
}

function deriveTitle(messages: Msg[]): string {
  const firstUser = messages.find((m) => m.role === 'user');
  if (!firstUser?.content?.trim()) return '新会话';
  let t = firstUser.content.trim();
  const cut = t.indexOf('\n\n---\n【附件】');
  if (cut > 0) t = t.slice(0, cut).trim();
  t = t.replace(/\s+/g, ' ');
  return t.length > 22 ? `${t.slice(0, 22)}…` : t;
}

function loadChatStore(): ChatStore {
  try {
    const raw = typeof window !== 'undefined' ? localStorage.getItem(STORAGE_CHAT_STORE_KEY) : null;
    if (raw) {
      const p = JSON.parse(raw) as ChatStore;
      if (
        Array.isArray(p?.sessions) &&
        p.sessions.length > 0 &&
        typeof p.activeId === 'string' &&
        p.sessions.some((s) => s.id === p.activeId)
      ) {
        return {
          sessions: p.sessions.map((s) => ({
            ...s,
            messages: Array.isArray(s.messages) ? s.messages : [],
            historyStartIndex: typeof s.historyStartIndex === 'number' ? s.historyStartIndex : 0,
          })),
          activeId: p.activeId,
        };
      }
    }
  } catch {
    /* ignore */
  }
  const id = newSessionId();
  return {
    sessions: [
      {
        id,
        title: '新会话',
        updatedAt: Date.now(),
        messages: [],
        historyStartIndex: 0,
      },
    ],
    activeId: id,
  };
}

function loadStoredModel(): ModelPick | null {
  try {
    const raw =
      typeof window !== 'undefined' ? window.localStorage.getItem(STORAGE_MODEL_KEY) : null;
    if (!raw) return null;
    const o = JSON.parse(raw) as ModelPick;
    if (typeof o.configId === 'number' && typeof o.entryId === 'number' && o.label) return o;
  } catch {
    /* ignore */
  }
  return null;
}

function saveStoredModel(m: ModelPick) {
  try {
    window.localStorage.setItem(STORAGE_MODEL_KEY, JSON.stringify(m));
  } catch {
    /* ignore */
  }
}

function loadStoredManagedAgentId(): number {
  try {
    const raw = typeof window !== 'undefined' ? localStorage.getItem(STORAGE_MANAGED_AGENT_KEY) : null;
    if (!raw) return 0;
    const n = Number(JSON.parse(raw));
    return Number.isFinite(n) && n > 0 ? n : 0;
  } catch {
    return 0;
  }
}

function saveStoredManagedAgentId(id: number) {
  try {
    if (id > 0) window.localStorage.setItem(STORAGE_MANAGED_AGENT_KEY, JSON.stringify(id));
    else window.localStorage.removeItem(STORAGE_MANAGED_AGENT_KEY);
  } catch {
    /* ignore */
  }
}

function markdownToHtml(raw: string): string {
  try {
    return String(marked.parse(raw || ''));
  } catch {
    return `<p>${(raw || '').replace(/</g, '&lt;').replace(/>/g, '&gt;')}</p>`;
  }
}

async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    try {
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select();
      const ok = document.execCommand('copy');
      document.body.removeChild(ta);
      return ok;
    } catch {
      return false;
    }
  }
}

function patchActiveSession(prev: ChatStore, recipe: (s: ChatSession) => ChatSession): ChatStore {
  const aid = prev.activeId;
  if (!aid) return prev;
  return {
    ...prev,
    sessions: prev.sessions.map((s) => {
      if (s.id !== aid) return s;
      const next = recipe(s);
      return {
        ...next,
        title: deriveTitle(next.messages),
        updatedAt: Date.now(),
      };
    }),
  };
}

export const OverviewChatSection: React.FC = () => {
  const [store, setStore] = useState<ChatStore>(() => loadChatStore());
  const [configs, setConfigs] = useState<LlmConfigItem[]>([]);
  const [managedProfiles, setManagedProfiles] = useState<{ id: number; name: string }[]>([]);
  const [loadingModels, setLoadingModels] = useState(true);
  const [modelPick, setModelPick] = useState<ModelPick | null>(() => loadStoredModel());
  const [managedAgentId, setManagedAgentId] = useState<number>(() => loadStoredManagedAgentId());
  const [input, setInput] = useState('');
  const [streaming, setStreaming] = useState(false);
  const [pendingFiles, setPendingFiles] = useState<File[]>([]);
  const abortRef = useRef<AbortController | null>(null);
  const listEndRef = useRef<HTMLDivElement | null>(null);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const activeSession = useMemo(
    () => store.sessions.find((s) => s.id === store.activeId),
    [store.sessions, store.activeId]
  );
  const messages = activeSession?.messages ?? [];
  const historyStartIndex = activeSession?.historyStartIndex ?? 0;

  useEffect(() => {
    const t = window.setTimeout(() => {
      try {
        localStorage.setItem(STORAGE_CHAT_STORE_KEY, JSON.stringify(store));
      } catch {
        /* quota */
      }
    }, 400);
    return () => window.clearTimeout(t);
  }, [store]);

  const fetchConfigs = useCallback(async () => {
    setLoadingModels(true);
    try {
      const [rows, agents] = await Promise.all([listLlmConfigs(), listManagedAgents()]);
      setConfigs(Array.isArray(rows) ? rows : []);
      setManagedProfiles(
        Array.isArray(agents)
          ? agents.filter((a) => a.enabled !== false).map((a) => ({ id: a.id, name: a.name }))
          : []
      );
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setLoadingModels(false);
    }
  }, []);

  useEffect(() => {
    fetchConfigs();
  }, [fetchConfigs]);

  const modelOptions = useMemo(() => {
    const opts: { value: string; label: string; pick: ModelPick }[] = [];
    for (const c of configs) {
      if (!c.enabled) continue;
      for (const m of c.models || []) {
        if (!m.enabled) continue;
        const label = `${c.name} · ${m.modelName}${m.description ? `（${m.description}）` : ''}`;
        opts.push({
          value: `${c.id}:${m.id}`,
          label,
          pick: { configId: c.id, entryId: m.id, label },
        });
      }
    }
    return opts;
  }, [configs]);

  const selectedValue =
    modelPick != null ? `${modelPick.configId}:${modelPick.entryId}` : undefined;

  useEffect(() => {
    if (!modelPick || modelOptions.length === 0) return;
    const ok = modelOptions.some(
      (o) => o.pick.configId === modelPick.configId && o.pick.entryId === modelPick.entryId
    );
    if (!ok) setModelPick(null);
  }, [modelOptions, modelPick]);

  useEffect(() => {
    if (managedAgentId <= 0 || managedProfiles.length === 0) return;
    const ok = managedProfiles.some((p) => p.id === managedAgentId);
    if (!ok) {
      setManagedAgentId(0);
      saveStoredManagedAgentId(0);
    }
  }, [managedProfiles, managedAgentId]);

  useEffect(() => {
    listEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streaming]);

  const sortedSessions = useMemo(
    () => [...store.sessions].sort((a, b) => b.updatedAt - a.updatedAt),
    [store.sessions]
  );

  const switchSession = (id: string) => {
    abortRef.current?.abort();
    setStreaming(false);
    setStore((prev) => ({ ...prev, activeId: id }));
  };

  const createSession = () => {
    abortRef.current?.abort();
    setStreaming(false);
    const id = newSessionId();
    setStore((prev) => ({
      sessions: [
        {
          id,
          title: '新会话',
          updatedAt: Date.now(),
          messages: [],
          historyStartIndex: 0,
        },
        ...prev.sessions,
      ],
      activeId: id,
    }));
  };

  const deleteSession = (id: string) => {
    abortRef.current?.abort();
    setStreaming(false);
    setStore((prev) => {
      const rest = prev.sessions.filter((s) => s.id !== id);
      if (rest.length === 0) {
        const nid = newSessionId();
        return {
          sessions: [
            {
              id: nid,
              title: '新会话',
              updatedAt: Date.now(),
              messages: [],
              historyStartIndex: 0,
            },
          ],
          activeId: nid,
        };
      }
      let nextActive = prev.activeId;
      if (prev.activeId === id) {
        nextActive = rest[0]?.id ?? null;
      }
      return { sessions: rest, activeId: nextActive };
    });
  };

  const handleSend = async () => {
    const text = input.trim();
    const hasFiles = pendingFiles.length > 0;
    if ((!text && !hasFiles) || streaming) return;
    if (!managedAgentId && !modelPick) {
      Toast.warning({
        content:
          '请在顶部选择对话模型（模型管理），或选择「Agent 配置」托管配置其一',
      });
      return;
    }

    const prior = messages.slice(historyStartIndex);
    const history = prior.map(({ role, content }) => ({ role, content }));

    const filesSnapshot = [...pendingFiles];
    setPendingFiles([]);
    setInput('');
    setStreaming(true);

    let attachments: ChatAttachmentPayload[] = [];
    try {
      attachments = await buildChatAttachmentsFromFiles(filesSnapshot);
    } catch (e) {
      const msg = String((e as Error)?.message ?? e);
      Toast.error({ content: msg || '读取附件失败' });
      setPendingFiles(filesSnapshot);
      setInput(text);
      setStreaming(false);
      return;
    }

    const displayUser = mergeMessageForChatDisplay(text, attachments);
    const userMsg: Msg = { role: 'user', content: displayUser };
    let assistantBuf = '';

    setStore((prev) =>
      patchActiveSession(prev, (s) => ({
        ...s,
        messages: [...s.messages, userMsg, { role: 'assistant', content: '' }],
      }))
    );

    abortRef.current?.abort();
    abortRef.current = new AbortController();

    try {
      await streamChat(
        {
          message: text,
          attachments: attachments.length ? attachments : undefined,
          history,
          llmConfigId: modelPick?.configId ?? 0,
          llmModelEntryId: modelPick?.entryId ?? 0,
          ...(managedAgentId > 0 ? { managedAgentId } : {}),
        },
        (chunk, done, err) => {
          if (err) {
            Toast.error({ content: err });
            return;
          }
          if (chunk) {
            assistantBuf += chunk;
            setStore((prev) =>
              patchActiveSession(prev, (s) => {
                const next = [...s.messages];
                const last = next[next.length - 1];
                if (last?.role === 'assistant') {
                  next[next.length - 1] = { ...last, content: assistantBuf };
                }
                return { ...s, messages: next };
              })
            );
          }
          if (done) {
            setStreaming(false);
          }
        },
        abortRef.current.signal
      );
    } catch (e) {
      const msg = String((e as Error)?.message ?? e);
      Toast.error({ content: msg });
      setStore((prev) =>
        patchActiveSession(prev, (s) => ({
          ...s,
          messages: s.messages.slice(0, -2),
        }))
      );
      setInput(text);
      setPendingFiles(filesSnapshot);
    } finally {
      setStreaming(false);
    }
  };

  const resetChat = () => {
    abortRef.current?.abort();
    setStreaming(false);
    setStore((prev) =>
      patchActiveSession(prev, (s) => ({
        ...s,
        title: '新会话',
        messages: [],
        historyStartIndex: 0,
      }))
    );
    setInput('');
    setPendingFiles([]);
  };

  const clearContextOnly = () => {
    setStore((prev) =>
      patchActiveSession(prev, (s) => ({
        ...s,
        historyStartIndex: s.messages.length,
      }))
    );
    Toast.success({ content: '已标记：后续请求不再携带此前的对话历史（界面记录仍保留）' });
  };

  const onModelChange = (value: string | number | any) => {
    const v = String(value ?? '');
    const found = modelOptions.find((o) => o.value === v);
    if (found) {
      setModelPick(found.pick);
      saveStoredModel(found.pick);
    }
  };

  const handleCopy = async (text: string) => {
    const t = text.trim();
    if (!t) {
      Toast.warning({ content: '暂无内容可复制' });
      return;
    }
    const ok = await copyToClipboard(t);
    if (ok) Toast.success({ content: '已复制到剪贴板' });
    else Toast.error({ content: '复制失败，请手动选择文本' });
  };

  const fmtTime = (ts: number) => {
    try {
      const d = new Date(ts);
      const now = new Date();
      const sameDay =
        d.getFullYear() === now.getFullYear() &&
        d.getMonth() === now.getMonth() &&
        d.getDate() === now.getDate();
      return sameDay
        ? d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
        : `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(
            2,
            '0'
          )}`;
    } catch {
      return '';
    }
  };

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        minHeight: 480,
        background: '#F7F8FA',
      }}
    >
      <div
        style={{
          padding: '12px 24px',
          background: '#fff',
          borderBottom: '1px solid rgba(6,7,9,0.08)',
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: 12,
          flexShrink: 0,
        }}
      >
        <Typography.Text strong style={{ marginRight: 8 }}>
          Agent 配置
        </Typography.Text>
        <Select
          placeholder={
            loadingModels ? '加载中…' : managedProfiles.length ? '可选：托管 Agent 配置' : '暂无托管配置'
          }
          style={{ minWidth: 200, maxWidth: 320 }}
          loading={loadingModels}
          value={managedAgentId > 0 ? String(managedAgentId) : ''}
          optionList={[
            { value: '', label: '不使用（仅模型管理）' },
            ...managedProfiles.map((p) => ({
              value: String(p.id),
              label: `${p.name} (#${p.id})`,
            })),
          ]}
          onChange={(v) => {
            const n = v ? Number(v) : 0;
            const id = Number.isFinite(n) && n > 0 ? n : 0;
            setManagedAgentId(id);
            saveStoredManagedAgentId(id);
          }}
        />
        <Typography.Text strong style={{ marginRight: 8 }}>
          对话模型
        </Typography.Text>
        <Select
          placeholder={loadingModels ? '正在加载模型列表…' : '请选择模型（来自模型管理）'}
          style={{ minWidth: 280, maxWidth: 420 }}
          loading={loadingModels}
          value={selectedValue}
          optionList={modelOptions.map((o) => ({ value: o.value, label: o.label }))}
          onChange={onModelChange}
          filter
        />
        <Button size="small" onClick={() => fetchConfigs()}>
          刷新模型列表
        </Button>
        <div style={{ flex: 1 }} />
        <Button size="small" onClick={clearContextOnly} disabled={streaming}>
          清除上下文
        </Button>
        <Button size="small" type="danger" onClick={resetChat} disabled={streaming}>
          清空本会话
        </Button>
      </div>

      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'row',
          minHeight: 0,
          minWidth: 0,
        }}
      >
        {/* 左侧：会话列表 */}
        <aside
          style={{
            width: 212,
            flexShrink: 0,
            background: '#fff',
            borderRight: '1px solid rgba(6,7,9,0.08)',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              padding: '12px',
              borderBottom: '1px solid rgba(6,7,9,0.06)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 8,
            }}
          >
            <Typography.Text strong>会话</Typography.Text>
            <Button
              icon={<IconPlus />}
              size="small"
              theme="solid"
              type="primary"
              onClick={createSession}
            >
              新建
            </Button>
          </div>
          <div style={{ flex: 1, overflow: 'auto', padding: 8 }}>
            {sortedSessions.map((sess) => {
              const active = sess.id === store.activeId;
              return (
                <div
                  key={sess.id}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    marginBottom: 6,
                    padding: '8px 10px',
                    borderRadius: 8,
                    cursor: 'pointer',
                    background: active ? 'rgba(22,100,255,0.12)' : 'transparent',
                    border: active ? '1px solid rgba(22,100,255,0.25)' : '1px solid transparent',
                  }}
                >
                  <div
                    role="presentation"
                    style={{ flex: 1, minWidth: 0 }}
                    onClick={() => switchSession(sess.id)}
                  >
                    <Typography.Text
                      ellipsis={{ showTooltip: true }}
                      style={{ display: 'block', fontWeight: active ? 600 : 400 }}
                    >
                      {sess.title}
                    </Typography.Text>
                    <Typography.Text type="tertiary" size="small">
                      {fmtTime(sess.updatedAt)}
                    </Typography.Text>
                  </div>
                  <Popconfirm
                    title="删除此会话？"
                    content="本地保存的消息将一并删除。"
                    onConfirm={() => deleteSession(sess.id)}
                  >
                    <Button size="small" type="danger" theme="borderless">
                      删
                    </Button>
                  </Popconfirm>
                </div>
              );
            })}
          </div>
          <Typography.Text
            type="tertiary"
            size="small"
            style={{ padding: '8px 12px', borderTop: '1px solid rgba(6,7,9,0.06)' }}
          >
            会话保存在本机浏览器。清除站点数据会丢失。
          </Typography.Text>
        </aside>

        {/* 中间：对话 */}
        <div
          style={{
            flex: 1,
            minWidth: 0,
            display: 'flex',
            flexDirection: 'column',
            background: '#F7F8FA',
          }}
        >
          <div style={{ flex: 1, overflow: 'auto', padding: '12px 14px', minWidth: 0 }}>
            <style>{`
          .overview-chat-md { word-break: break-word; }
          .overview-chat-md :first-child { margin-top: 0; }
          .overview-chat-md :last-child { margin-bottom: 0; }
          .overview-chat-md h1, .overview-chat-md h2, .overview-chat-md h3 { margin: 0.65em 0 0.4em; font-weight: 600; line-height: 1.35; }
          .overview-chat-md h1 { font-size: 1.35em; }
          .overview-chat-md h2 { font-size: 1.2em; }
          .overview-chat-md h3 { font-size: 1.08em; }
          .overview-chat-md p { margin: 0.45em 0; }
          .overview-chat-md ul, .overview-chat-md ol { margin: 0.45em 0; padding-left: 1.35em; }
          .overview-chat-md blockquote {
            margin: 0.5em 0; padding: 4px 12px; border-left: 3px solid rgba(22,100,255,0.35); background: rgba(6,7,9,0.03); border-radius: 0 6px 6px 0;
          }
          .overview-chat-md code {
            font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
            font-size: 0.92em;
            padding: 1px 6px; border-radius: 4px; background: rgba(6,7,9,0.06);
          }
          .overview-chat-md pre {
            margin: 0.55em 0; padding: 12px 14px; border-radius: 8px;
            background: rgba(6,7,9,0.055); overflow-x: auto; border: 1px solid rgba(6,7,9,0.06);
          }
          .overview-chat-md pre code { padding: 0; background: transparent; font-size: 13px; line-height: 1.5; }
          .overview-chat-md table { border-collapse: collapse; width: 100%; margin: 0.5em 0; font-size: 13px; }
          .overview-chat-md th, .overview-chat-md td { border: 1px solid rgba(6,7,9,0.1); padding: 6px 10px; text-align: left; }
          .overview-chat-md th { background: rgba(6,7,9,0.04); }
          .overview-chat-md a { color: #1664ff; }
        `}</style>
            <div style={{ width: '100%', boxSizing: 'border-box' }}>
              {messages.length === 0 ? (
                <div style={{ paddingTop: 28, paddingBottom: 24 }}>
                  <Empty title={CHAT_GUIDE.title} description="" style={{ marginBottom: 16 }} />
                  <Typography.Paragraph
                    spacing="extended"
                    style={{ margin: '0 0 12px', color: 'rgba(28,32,41,0.78)' }}
                  >
                    {CHAT_GUIDE.subtitle}
                  </Typography.Paragraph>
                  <Typography.Title heading={6} style={{ margin: '16px 0 8px' }}>
                    怎么用
                  </Typography.Title>
                  <ul
                    style={{
                      margin: '0 0 16px',
                      paddingLeft: 20,
                      color: 'rgba(28,32,41,0.78)',
                      lineHeight: 1.75,
                    }}
                  >
                    {CHAT_GUIDE.steps.map((line, idx) => (
                      <li key={`chat-step-${idx}`}>{line}</li>
                    ))}
                  </ul>
                  <Typography.Text
                    type="tertiary"
                    size="small"
                    style={{ display: 'block', lineHeight: 1.65 }}
                  >
                    {CHAT_GUIDE.inputHint}
                  </Typography.Text>
                </div>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12, width: '100%' }}>
                  {messages.map((m, i) => {
                    const isLastAssistantStreaming =
                      streaming && i === messages.length - 1 && m.role === 'assistant';
                    const showPlainStream = isLastAssistantStreaming;
                    const html =
                      !showPlainStream && m.content.trim() !== '' ? markdownToHtml(m.content) : '';
                    const bubbleKey = `${store.activeId}-${i}`;
                    const isUser = m.role === 'user';
                    return (
                      <div
                        key={bubbleKey}
                        style={{
                          display: 'flex',
                          justifyContent: isUser ? 'flex-end' : 'flex-start',
                          width: '100%',
                        }}
                      >
                        <div
                          style={{
                            maxWidth: isUser ? 'min(92%, 960px)' : '100%',
                            flex: isUser ? '0 1 auto' : '1 1 auto',
                            minWidth: 0,
                            padding: '10px 14px',
                            borderRadius: 12,
                            background: isUser ? 'rgba(22,100,255,0.12)' : '#fff',
                            border: '1px solid rgba(6,7,9,0.06)',
                            wordBreak: 'break-word',
                            boxSizing: 'border-box',
                          }}
                        >
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'space-between',
                              gap: 8,
                              marginBottom: 8,
                            }}
                          >
                            <Typography.Text type="tertiary" size="small">
                              {isUser ? '你' : '助手'}
                            </Typography.Text>
                            <Button
                              icon={<IconCopy />}
                              theme="borderless"
                              size="small"
                              type="tertiary"
                              disabled={!m.content.trim()}
                              onClick={() => handleCopy(m.content)}
                            >
                              复制
                            </Button>
                          </div>
                          {showPlainStream ? (
                            <div style={{ ...MD_SURFACE_STYLE, whiteSpace: 'pre-wrap' }}>
                              {m.content}
                            </div>
                          ) : (
                            <div
                              className="overview-chat-md"
                              style={MD_SURFACE_STYLE}
                              dangerouslySetInnerHTML={{ __html: html || '<p></p>' }}
                            />
                          )}
                        </div>
                      </div>
                    );
                  })}
                  <div ref={listEndRef} />
                </div>
              )}
            </div>
          </div>

          <div
            style={{
              padding: '12px 14px 16px',
              background: '#fff',
              borderTop: '1px solid rgba(6,7,9,0.08)',
              flexShrink: 0,
              minWidth: 0,
            }}
          >
            <div style={{ width: '100%', boxSizing: 'border-box' }}>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                accept="image/*,video/*,audio/*,.txt,.md,.markdown,.json,.yaml,.yml,.csv,.xml,.html,.htm,.pdf,.zip"
                style={{ display: 'none' }}
                onChange={(e) => {
                  const list = e.target.files;
                  if (!list?.length) return;
                  setPendingFiles((prev) => [...prev, ...Array.from(list)].slice(0, 12));
                  e.target.value = '';
                }}
              />
              <Spin spinning={streaming}>
                <TextArea
                  value={input}
                  onChange={setInput}
                  placeholder={CHAT_GUIDE.placeholder}
                  autosize={{ minRows: 3, maxRows: 12 }}
                  onEnterPress={(e) => {
                    if (!e.shiftKey && !e.nativeEvent?.isComposing) {
                      e.preventDefault();
                      handleSend();
                    }
                  }}
                  disabled={streaming}
                />
                {pendingFiles.length > 0 && (
                  <div
                    style={{
                      marginTop: 8,
                      display: 'flex',
                      flexWrap: 'wrap',
                      gap: 8,
                      alignItems: 'center',
                    }}
                  >
                    {pendingFiles.map((f, idx) => (
                      <Typography.Text
                        key={`${f.name}_${f.size}_${idx}`}
                        size="small"
                        style={{
                          padding: '4px 10px',
                          background: 'rgba(46,50,56,0.06)',
                          borderRadius: 6,
                          maxWidth: '100%',
                          wordBreak: 'break-all',
                        }}
                      >
                        {f.name}
                        <Button
                          theme="borderless"
                          size="small"
                          type="tertiary"
                          style={{ marginLeft: 6, padding: '0 4px' }}
                          onClick={() =>
                            setPendingFiles((prev) => prev.filter((_, i) => i !== idx))
                          }
                        >
                          移除
                        </Button>
                      </Typography.Text>
                    ))}
                  </div>
                )}
                <Typography.Text
                  type="tertiary"
                  size="small"
                  style={{ display: 'block', marginTop: 8, lineHeight: 1.55 }}
                >
                  {CHAT_GUIDE.inputHint}
                </Typography.Text>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginTop: 10,
                    gap: 10,
                  }}
                >
                  <Button
                    icon={<IconUpload />}
                    onClick={() => fileInputRef.current?.click()}
                    disabled={streaming}
                  >
                    附件
                  </Button>
                  <Button
                    theme="solid"
                    type="primary"
                    onClick={handleSend}
                    disabled={streaming || (!input.trim() && pendingFiles.length === 0)}
                  >
                    发送
                  </Button>
                </div>
              </Spin>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default OverviewChatSection;
