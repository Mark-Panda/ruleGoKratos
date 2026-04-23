import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import hljs from 'highlight.js/lib/common';
import 'highlight.js/styles/github.css';
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
  type ChatStreamPayload,
} from '../../services/api-chat';
import { listLlmConfigs, type LlmConfigItem } from '../../services/api-agent';
import { listManagedAgents } from '../../services/api-managed-agents';

function escapeHtml(raw: string): string {
  return raw
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

const markedRenderer = new marked.Renderer();
(markedRenderer as any).code = (code: string, infostring?: string) => {
  const rawLang = (infostring || '').trim().split(/\s+/)[0] || '';
  let lang = rawLang.toLowerCase();
  let highlighted = escapeHtml(code || '');
  try {
    if (lang && hljs.getLanguage(lang)) {
      highlighted = hljs.highlight(code || '', { language: lang, ignoreIllegals: true }).value;
    } else {
      const auto = hljs.highlightAuto(code || '');
      highlighted = auto.value || highlighted;
      if (!lang && auto.language) lang = auto.language;
    }
  } catch {
    highlighted = escapeHtml(code || '');
  }
  if (!lang) lang = 'text';
  const encoded = encodeURIComponent(code || '');
  return `
<div class="overview-chat-code-wrap">
  <div class="overview-chat-code-toolbar">
    <span class="overview-chat-code-lang">${escapeHtml(lang)}</span>
    <button class="overview-chat-code-copy" type="button" data-copy-code="${encoded}">复制代码</button>
  </div>
  <pre><code class="hljs language-${escapeHtml(lang)}">${highlighted}</code></pre>
</div>`;
};

marked.use({
  breaks: true,
  renderer: markedRenderer,
});

const STORAGE_MODEL_KEY = 'flowgram-overview-chat-model-v1';
const STORAGE_CHAT_STORE_KEY = 'flowgram-overview-chat-store-v1';
const STORAGE_MANAGED_AGENT_KEY = 'flowgram-overview-chat-managed-agent-v1';
const STORAGE_LAST_REQUEST_KEY = 'flowgram-overview-chat-last-request-v1';
const RETRY_PAYLOAD_MAX_BYTES = 380000;

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

type RetryPayloadStore = Record<string, ChatStreamPayload>;
type RetrySource = 'persisted' | 'memory';

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

function loadLastRequestStore(): RetryPayloadStore {
  try {
    const raw = typeof window !== 'undefined' ? localStorage.getItem(STORAGE_LAST_REQUEST_KEY) : null;
    if (!raw) return {};
    const parsed = JSON.parse(raw) as RetryPayloadStore;
    if (!parsed || typeof parsed !== 'object') return {};
    return parsed;
  } catch {
    return {};
  }
}

function saveLastRequestStore(store: RetryPayloadStore) {
  try {
    localStorage.setItem(STORAGE_LAST_REQUEST_KEY, JSON.stringify(store));
  } catch {
    /* ignore */
  }
}

function upsertLastRequestStore(sessionId: string, payload: ChatStreamPayload): boolean {
  if (!sessionId) return false;
  try {
    const next = { ...loadLastRequestStore(), [sessionId]: payload };
    const raw = JSON.stringify(next);
    if (new TextEncoder().encode(raw).length > RETRY_PAYLOAD_MAX_BYTES) {
      return false;
    }
    localStorage.setItem(STORAGE_LAST_REQUEST_KEY, raw);
    return true;
  } catch {
    return false;
  }
}

function removeLastRequestStore(sessionId: string) {
  if (!sessionId) return;
  const next = { ...loadLastRequestStore() };
  delete next[sessionId];
  saveLastRequestStore(next);
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
  const initialRetryStore = useMemo(() => loadLastRequestStore(), []);
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
  const stopByUserRef = useRef(false);
  const lastRequestBySessionRef = useRef<RetryPayloadStore>(initialRetryStore);
  const [retrySourceBySession, setRetrySourceBySession] = useState<Record<string, RetrySource>>(() => {
    const source: Record<string, RetrySource> = {};
    for (const sid of Object.keys(initialRetryStore)) {
      source[sid] = 'persisted';
    }
    return source;
  });
  const listEndRef = useRef<HTMLDivElement | null>(null);
  const listWrapRef = useRef<HTMLDivElement | null>(null);
  const autoFollowRef = useRef(true);
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

  useEffect(() => {
    const alive = new Set(store.sessions.map((s) => s.id));
    const all = { ...lastRequestBySessionRef.current };
    let changed = false;
    let sourceChanged = false;
    const nextSource = { ...retrySourceBySession };
    for (const sid of Object.keys(all)) {
      if (!alive.has(sid)) {
        delete all[sid];
        changed = true;
      }
    }
    for (const sid of Object.keys(nextSource)) {
      if (!alive.has(sid)) {
        delete nextSource[sid];
        sourceChanged = true;
      }
    }
    if (changed) {
      lastRequestBySessionRef.current = all;
      saveLastRequestStore(all);
    }
    if (sourceChanged) {
      setRetrySourceBySession(nextSource);
    }
  }, [store.sessions, retrySourceBySession]);

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

  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'auto') => {
    listEndRef.current?.scrollIntoView({ behavior, block: 'end' });
  }, []);

  const handleListScroll = useCallback(() => {
    const el = listWrapRef.current;
    if (!el) return;
    const threshold = 88;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight <= threshold;
    autoFollowRef.current = nearBottom;
  }, []);

  /** 中断当前与服务器的流式请求（与底部按钮共用） */
  const pauseCurrentExecution = useCallback(() => {
    if (!streaming) return;
    stopByUserRef.current = true;
    abortRef.current?.abort();
    setStreaming(false);
  }, [streaming]);

  /** fetch 被 Abort 后整理最后一条助手气泡：用户主动暂停时保留已生成内容并追加说明 */
  const applyAbortErrorToAssistant = useCallback((userRequested: boolean) => {
    setStore((prev) =>
      patchActiveSession(prev, (s) => {
        const next = [...s.messages];
        const last = next[next.length - 1];
        if (last?.role !== 'assistant') {
          return s;
        }
        if (!last.content.trim()) {
          next.pop();
          return { ...s, messages: next };
        }
        if (userRequested) {
          next[next.length - 1] = {
            ...last,
            content:
              last.content.trimEnd() +
              '\n\n> 已暂停执行（已中断流式输出，上文已生成内容已保留）。',
          };
        }
        return { ...s, messages: next };
      })
    );
  }, []);

  useEffect(() => {
    autoFollowRef.current = true;
    scrollToBottom('auto');
  }, [store.activeId, scrollToBottom]);

  useEffect(() => {
    if (!autoFollowRef.current) return;
    scrollToBottom(streaming ? 'auto' : 'smooth');
  }, [messages, streaming, scrollToBottom]);

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
    delete lastRequestBySessionRef.current[id];
    removeLastRequestStore(id);
    setRetrySourceBySession((prev) => {
      if (!prev[id]) return prev;
      const next = { ...prev };
      delete next[id];
      return next;
    });
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
    stopByUserRef.current = false;

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
    const requestPayload: ChatStreamPayload = {
      message: text,
      attachments: attachments.length ? attachments : undefined,
      history,
      llmConfigId: modelPick?.configId ?? 0,
      llmModelEntryId: modelPick?.entryId ?? 0,
      ...(managedAgentId > 0 ? { managedAgentId } : {}),
    };
    if (store.activeId) {
      lastRequestBySessionRef.current[store.activeId] = requestPayload;
      const ok = upsertLastRequestStore(store.activeId, requestPayload);
      setRetrySourceBySession((prev) => ({ ...prev, [store.activeId as string]: ok ? 'persisted' : 'memory' }));
      if (!ok && requestPayload.attachments?.length) {
        Toast.warning({ content: '附件较大：已仅在当前页面保留重试快照，刷新后可能无法重试' });
      }
    }

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
        requestPayload,
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
      const aborted = (e as Error)?.name === 'AbortError';
      if (aborted) {
        applyAbortErrorToAssistant(stopByUserRef.current);
        return;
      }
      const msg = String((e as Error)?.message ?? e) || '请求失败';
      Toast.error({ content: msg });
      setStore((prev) =>
        patchActiveSession(prev, (s) => ({
          ...s,
          messages: (() => {
            const next = [...s.messages];
            const last = next[next.length - 1];
            if (last?.role === 'assistant') {
              const merged = (assistantBuf || '').trim();
              next[next.length - 1] = {
                role: 'assistant',
                content: merged
                  ? `${merged}\n\n> ⚠️ 生成中断：${msg}`
                  : `> ⚠️ 生成失败：${msg}`,
              };
            }
            return next;
          })(),
        }))
      );
    } finally {
      setStreaming(false);
      stopByUserRef.current = false;
    }
  };

  const handleRegenerateLast = async () => {
    if (streaming) return;
    if (messages.length < 2) {
      Toast.warning({ content: '暂无可重新生成的回答' });
      return;
    }
    const last = messages[messages.length - 1];
    const prev = messages[messages.length - 2];
    if (last.role !== 'assistant' || prev.role !== 'user') {
      Toast.warning({ content: '当前仅支持对最近一轮问答重新生成' });
      return;
    }
    if (!managedAgentId && !modelPick) {
      Toast.warning({
        content:
          '请在顶部选择对话模型（模型管理），或选择「Agent 配置」托管配置其一',
      });
      return;
    }

    const sid = store.activeId;
    if (!sid) return;
    const requestPayload = lastRequestBySessionRef.current[sid];
    if (!requestPayload) {
      Toast.warning({ content: '缺少上次请求快照，无法重新生成' });
      return;
    }

    setStreaming(true);
    stopByUserRef.current = false;
    setStore((p) =>
      patchActiveSession(p, (s) => {
        const next = [...s.messages];
        next[next.length - 1] = { role: 'assistant', content: '' };
        return { ...s, messages: next };
      })
    );

    let assistantBuf = '';
    abortRef.current?.abort();
    abortRef.current = new AbortController();

    try {
      await streamChat(
        requestPayload,
        (chunk, done, err) => {
          if (err) {
            Toast.error({ content: err });
            return;
          }
          if (chunk) {
            assistantBuf += chunk;
            setStore((prevStore) =>
              patchActiveSession(prevStore, (s) => {
                const next = [...s.messages];
                const tail = next[next.length - 1];
                if (tail?.role === 'assistant') {
                  next[next.length - 1] = { ...tail, content: assistantBuf };
                }
                return { ...s, messages: next };
              })
            );
          }
          if (done) setStreaming(false);
        },
        abortRef.current.signal
      );
    } catch (e) {
      const aborted = (e as Error)?.name === 'AbortError';
      if (aborted) {
        applyAbortErrorToAssistant(stopByUserRef.current);
      } else {
        const msg = String((e as Error)?.message ?? e) || '请求失败';
        Toast.error({ content: msg });
        setStore((prevStore) =>
          patchActiveSession(prevStore, (s) => {
            const next = [...s.messages];
            const tail = next[next.length - 1];
            if (tail?.role === 'assistant') {
              next[next.length - 1] = {
                role: 'assistant',
                content: assistantBuf.trim()
                  ? `${assistantBuf.trim()}\n\n> ⚠️ 重新生成中断：${msg}`
                  : `> ⚠️ 重新生成失败：${msg}`,
              };
            }
            return { ...s, messages: next };
          })
        );
      }
    } finally {
      setStreaming(false);
      stopByUserRef.current = false;
    }
  };

  const handleRetryLastRequest = async () => {
    if (streaming) return;
    const sid = store.activeId;
    if (!sid) return;
    const requestPayload = lastRequestBySessionRef.current[sid];
    if (!requestPayload) {
      Toast.warning({ content: '暂无可重试的上次请求' });
      return;
    }
    setStreaming(true);
    stopByUserRef.current = false;
    setStore((prev) =>
      patchActiveSession(prev, (s) => {
        const next = [...s.messages];
        if (next.length > 0 && next[next.length - 1]?.role === 'assistant') {
          next[next.length - 1] = { role: 'assistant', content: '' };
        } else {
          next.push({ role: 'assistant', content: '' });
        }
        return { ...s, messages: next };
      })
    );

    let assistantBuf = '';
    abortRef.current?.abort();
    abortRef.current = new AbortController();

    try {
      await streamChat(
        requestPayload,
        (chunk, done, err) => {
          if (err) {
            Toast.error({ content: err });
            return;
          }
          if (chunk) {
            assistantBuf += chunk;
            setStore((prevStore) =>
              patchActiveSession(prevStore, (s) => {
                const next = [...s.messages];
                const tail = next[next.length - 1];
                if (tail?.role === 'assistant') {
                  next[next.length - 1] = { ...tail, content: assistantBuf };
                } else {
                  next.push({ role: 'assistant', content: assistantBuf });
                }
                return { ...s, messages: next };
              })
            );
          }
          if (done) setStreaming(false);
        },
        abortRef.current.signal
      );
    } catch (e) {
      const aborted = (e as Error)?.name === 'AbortError';
      if (aborted) {
        applyAbortErrorToAssistant(stopByUserRef.current);
      } else {
        const msg = String((e as Error)?.message ?? e) || '请求失败';
        Toast.error({ content: msg });
        setStore((prevStore) =>
          patchActiveSession(prevStore, (s) => {
            const next = [...s.messages];
            const tail = next[next.length - 1];
            if (tail?.role === 'assistant') {
              next[next.length - 1] = {
                role: 'assistant',
                content: assistantBuf.trim()
                  ? `${assistantBuf.trim()}\n\n> ⚠️ 重试中断：${msg}`
                  : `> ⚠️ 重试失败：${msg}`,
              };
            }
            return { ...s, messages: next };
          })
        );
      }
    } finally {
      setStreaming(false);
      stopByUserRef.current = false;
    }
  };

  const resetChat = () => {
    abortRef.current?.abort();
    setStreaming(false);
    if (store.activeId) {
      delete lastRequestBySessionRef.current[store.activeId];
      removeLastRequestStore(store.activeId);
      setRetrySourceBySession((prev) => {
        const next = { ...prev };
        delete next[store.activeId as string];
        return next;
      });
    }
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

  const waitingFirstToken =
    streaming &&
    messages.length > 0 &&
    messages[messages.length - 1]?.role === 'assistant' &&
    !messages[messages.length - 1]?.content?.trim();
  const activeRetrySource = store.activeId ? retrySourceBySession[store.activeId] : undefined;
  const canRetryLastRequest =
    !!(store.activeId && lastRequestBySessionRef.current[store.activeId]);

  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        minHeight: 0,
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
        <Button
          size="small"
          type="warning"
          theme={streaming ? 'solid' : 'light'}
          disabled={!streaming}
          onClick={pauseCurrentExecution}
        >
          暂停执行
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
            minHeight: 0,
            display: 'flex',
            flexDirection: 'column',
            background: '#F7F8FA',
          }}
        >
          <div
            ref={listWrapRef}
            onScroll={handleListScroll}
            style={{
              flex: 1,
              minHeight: 0,
              overflow: 'auto',
              padding: '12px 14px',
              minWidth: 0,
            }}
          >
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
          .overview-chat-code-wrap { margin: 0.55em 0; border-radius: 8px; overflow: hidden; border: 1px solid rgba(6,7,9,0.06); }
          .overview-chat-code-toolbar {
            display: flex; align-items: center; justify-content: space-between;
            padding: 6px 10px; background: rgba(6,7,9,0.04); border-bottom: 1px solid rgba(6,7,9,0.06);
          }
          .overview-chat-code-lang { font-size: 12px; color: rgba(28,32,41,0.62); text-transform: lowercase; }
          .overview-chat-code-copy {
            border: 0; background: transparent; cursor: pointer; font-size: 12px;
            color: #1664ff; padding: 0;
          }
          .overview-chat-md pre {
            margin: 0; padding: 12px 14px; border-radius: 0;
            background: rgba(6,7,9,0.055); overflow-x: auto; border: 0;
          }
          .overview-chat-md pre code { padding: 0; background: transparent; font-size: 13px; line-height: 1.5; }
          .overview-chat-md .hljs { background: transparent; color: inherit; }
          .overview-chat-md table { border-collapse: collapse; width: 100%; margin: 0.5em 0; font-size: 13px; }
          .overview-chat-md th, .overview-chat-md td { border: 1px solid rgba(6,7,9,0.1); padding: 6px 10px; text-align: left; }
          .overview-chat-md th { background: rgba(6,7,9,0.04); }
          .overview-chat-md a { color: #1664ff; }
          .overview-chat-stream-cursor {
            display: inline-block;
            width: 8px;
            margin-left: 2px;
            color: rgba(22,100,255,0.92);
            animation: overviewChatBlink 1s steps(1, end) infinite;
          }
          .overview-chat-thinking-dot {
            display: inline-block;
            width: 6px;
            height: 6px;
            margin: 0 3px;
            border-radius: 50%;
            background: rgba(22,100,255,0.75);
            animation: overviewChatThinking 1.1s infinite ease-in-out;
          }
          .overview-chat-thinking-dot:nth-child(2) { animation-delay: .15s; }
          .overview-chat-thinking-dot:nth-child(3) { animation-delay: .3s; }
          @keyframes overviewChatBlink {
            0%, 49% { opacity: 1; }
            50%, 100% { opacity: 0; }
          }
          @keyframes overviewChatThinking {
            0%, 80%, 100% { transform: translateY(0); opacity: .35; }
            40% { transform: translateY(-2px); opacity: 1; }
          }
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
                    const html = m.content.trim() !== '' ? markdownToHtml(m.content) : '';
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
                          <div
                            className="overview-chat-md"
                            style={MD_SURFACE_STYLE}
                              onClick={async (evt) => {
                                const target = evt.target as HTMLElement | null;
                                const btn = target?.closest?.('[data-copy-code]') as HTMLElement | null;
                                if (!btn) return;
                                const encoded = btn.getAttribute('data-copy-code') || '';
                                const code = decodeURIComponent(encoded);
                                const ok = await copyToClipboard(code);
                                if (ok) Toast.success({ content: '代码已复制' });
                                else Toast.error({ content: '代码复制失败' });
                              }}
                            dangerouslySetInnerHTML={{ __html: html || '<p></p>' }}
                          />
                          {isLastAssistantStreaming && <span className="overview-chat-stream-cursor">▍</span>}
                        </div>
                      </div>
                    );
                  })}
                  {waitingFirstToken && (
                    <div style={{ display: 'flex', justifyContent: 'flex-start', width: '100%' }}>
                      <div
                        style={{
                          maxWidth: '100%',
                          padding: '10px 14px',
                          borderRadius: 12,
                          background: '#fff',
                          border: '1px solid rgba(6,7,9,0.06)',
                          color: 'rgba(28,32,41,0.72)',
                          fontSize: 14,
                        }}
                      >
                        助手正在思考
                        <span className="overview-chat-thinking-dot" />
                        <span className="overview-chat-thinking-dot" />
                        <span className="overview-chat-thinking-dot" />
                      </div>
                    </div>
                  )}
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
              {/* Spin spinning 时 semi-spin-block::after 会盖住子节点，按钮须放在 Spin 外 */}
              <Spin spinning={streaming} style={{ width: '100%', display: 'block' }}>
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
              </Spin>
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
                        onClick={() => setPendingFiles((prev) => prev.filter((_, i) => i !== idx))}
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
                {streaming && (
                  <Button type="tertiary" onClick={pauseCurrentExecution}>
                    暂停执行
                  </Button>
                )}
                {!streaming && (
                  <Button type="tertiary" onClick={handleRegenerateLast}>
                    重新生成上一条
                  </Button>
                )}
                {!streaming && (
                  <Button type="tertiary" onClick={handleRetryLastRequest} disabled={!canRetryLastRequest}>
                    {!canRetryLastRequest
                      ? '暂无可重试请求'
                      : activeRetrySource === 'persisted'
                      ? '重试上次请求（含附件）'
                      : '重试上次请求（仅当前页）'}
                  </Button>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default OverviewChatSection;
