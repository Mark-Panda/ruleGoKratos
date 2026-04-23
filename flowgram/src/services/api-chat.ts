import { getApiOrigin } from './http';

export interface ChatHistoryItem {
  role: string;
  content: string;
}

/** 与 rulego.v1.ChatAttachment 的 protojson 字段一致 */
export interface ChatAttachmentPayload {
  filename: string;
  mimeType?: string;
  text?: string;
  contentBase64?: string;
}

export interface ChatStreamPayload {
  message: string;
  model?: string;
  history: ChatHistoryItem[];
  llmConfigId: number;
  llmModelEntryId: number;
  /** 非零时使用「Agent 配置」统一管理模型/SKILL/MCP（与顶部模型选择可同时传，服务端以托管配置为准） */
  managedAgentId?: number;
  attachments?: ChatAttachmentPayload[];
}

const MAX_ATTACHMENT_TEXT_RUNES = 120000;
const MAX_ATTACHMENT_BASE64 = 350000;
const MAX_MERGED_USER_BYTES = 480000;
/** 单文件二进制上限：Base64 长度不超过 MAX_ATTACHMENT_BASE64（约 260KiB 原始） */
const MAX_RAW_BINARY_BYTES = 260 * 1024;
const MAX_FILES = 12;

/**
 * 生成会话界面 / localStorage 中的用户气泡文案：包含正文与附件说明，**不包含**二进制 Base64（避免刷屏）。
 * 实际附件仍通过 streamChat 的 attachments 字段送达后端。
 */
export function mergeMessageForChatDisplay(
  msg: string,
  attachments: ChatAttachmentPayload[] | undefined
): string {
  msg = msg.trim();
  if (!attachments?.length) {
    return msg;
  }
  let b = msg;
  for (const a of attachments) {
    if (!a) continue;
    const fn = (a.filename || '').trim() || '(未命名)';
    const mt = (a.mimeType || '').trim();
    b += '\n\n---\n【附件】 ' + fn;
    if (mt) b += ' • ' + mt;
    b += '\n';
    const txt = (a.text || '').trim();
    if (txt) {
      let t = txt;
      const runes = [...t];
      if (runes.length > MAX_ATTACHMENT_TEXT_RUNES) {
        t = runes.slice(0, MAX_ATTACHMENT_TEXT_RUNES).join('') + '\n…（本附件文本已截断）';
      }
      b += t + '\n';
    }
    if ((a.contentBase64 || '').trim()) {
      b += '（二进制内容已通过多模态请求发送给模型，此处不展示编码）\n';
    }
    if (!txt && !(a.contentBase64 || '').trim()) {
      b += '（附件无内容）\n';
    }
  }
  const enc = new TextEncoder();
  const bytes = enc.encode(b);
  if (bytes.length > MAX_MERGED_USER_BYTES) {
    const cut = bytes.slice(0, MAX_MERGED_USER_BYTES);
    return new TextDecoder('utf-8', { fatal: false }).decode(cut) + '\n…（用户消息过长已截断）';
  }
  return b;
}

const TEXT_LIKE_EXT =
  /\.(txt|md|markdown|json|yaml|yml|csv|ts|tsx|jsx?|mjs|cjs|go|py|rs|java|kt|sql|xml|html?|css|less|scss|sass|vue|svelte|sh|bash|zsh|env|proto|graphql|toml|ini|cfg|conf|gitignore|dockerignore)$/i;

/** 明确视为图/音视频（即使被误标成 text/plain，也不能走 file.text()） */
const BINARY_MEDIA_EXT =
  /\.(png|apng|jpe?g|jfif|pjpeg|webp|gif|bmp|ico|svg|heic|heif|avif|tiff?|mp4|m4v|webm|mov|mkv|avi|wmv|mp3|wav|ogg|m4a|aac|flac|opus)$/i;

function isBinaryMediaFile(file: File): boolean {
  const m = (file.type || '').toLowerCase();
  if (m.startsWith('image/') || m.startsWith('video/') || m.startsWith('audio/')) return true;
  return BINARY_MEDIA_EXT.test(file.name);
}

function looksTextLike(file: File): boolean {
  if (isBinaryMediaFile(file)) return false;
  const m = file.type || '';
  if (m.startsWith('text/')) return true;
  if (m === 'application/json' || m.includes('javascript') || m === 'application/xml') return true;
  return TEXT_LIKE_EXT.test(file.name);
}

/** 与后端 SniffBinaryMIME 对齐，用于扩展名/type 不可靠时强行识别图/视频 */
function sniffMimeFromBytes(u8: Uint8Array): string | undefined {
  if (u8.length < 8) return undefined;
  if (u8[0] === 0x89 && u8[1] === 0x50 && u8[2] === 0x4e && u8[3] === 0x47) return 'image/png';
  if (u8.length >= 3 && u8[0] === 0xff && u8[1] === 0xd8 && u8[2] === 0xff) return 'image/jpeg';
  if (u8.length >= 6) {
    const sig = String.fromCharCode(...u8.slice(0, 6));
    if (sig === 'GIF87a' || sig === 'GIF89a') return 'image/gif';
  }
  if (
    u8.length >= 12 &&
    String.fromCharCode(u8[0], u8[1], u8[2], u8[3]) === 'RIFF' &&
    String.fromCharCode(u8[8], u8[9], u8[10], u8[11]) === 'WEBP'
  ) {
    return 'image/webp';
  }
  if (u8.length >= 2 && u8[0] === 0x42 && u8[1] === 0x4d) return 'image/bmp';
  if (u8.length >= 4 && u8[0] === 0x1a && u8[1] === 0x45 && u8[2] === 0xdf && u8[3] === 0xa3)
    return 'video/webm';
  if (u8.length >= 5 && String.fromCharCode(...u8.slice(0, 5)) === '%PDF-')
    return 'application/pdf';
  const lim = Math.min(u8.length, 4096);
  for (let i = 0; i <= lim - 4; i++) {
    if (u8[i] === 0x66 && u8[i + 1] === 0x74 && u8[i + 2] === 0x79 && u8[i + 3] === 0x70)
      return 'video/mp4';
  }
  return undefined;
}

function pickMimeForBinaryPayload(file: File, buf: ArrayBuffer): string {
  const raw = (file.type || '').trim();
  const rawLower = raw.toLowerCase();
  if (rawLower && rawLower !== 'application/octet-stream') return raw;
  const magic = sniffMimeFromBytes(new Uint8Array(buf));
  if (magic) return magic;
  return inferMimeFromFilename(file.name) ?? 'application/octet-stream';
}

/** 部分浏览器对图片给出空 type 或 application/octet-stream，按扩展名补全以便后端走视觉多模态 */
function inferMimeFromFilename(name: string): string | undefined {
  const i = name.lastIndexOf('.');
  if (i < 0) return undefined;
  const ext = name.slice(i).toLowerCase();
  const map: Record<string, string> = {
    '.png': 'image/png',
    '.jpg': 'image/jpeg',
    '.jpeg': 'image/jpeg',
    '.webp': 'image/webp',
    '.gif': 'image/gif',
    '.bmp': 'image/bmp',
    '.mp4': 'video/mp4',
    '.webm': 'video/webm',
    '.mp3': 'audio/mpeg',
    '.wav': 'audio/wav',
  };
  return map[ext];
}

function arrayBufferToBase64(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let binary = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  return btoa(binary);
}

/**
 * 将浏览器 File 转为接口附件：文本类读 UTF-8；二进制转 Base64（超限则占位说明）。
 */
export async function buildChatAttachmentsFromFiles(
  files: File[]
): Promise<ChatAttachmentPayload[]> {
  const list = files.slice(0, MAX_FILES);
  const out: ChatAttachmentPayload[] = [];

  for (const file of list) {
    const name = file.name;

    if (file.size > MAX_RAW_BINARY_BYTES) {
      const mimeFallback =
        (file.type || '').trim() || inferMimeFromFilename(file.name) || 'application/octet-stream';
      out.push({
        filename: name,
        mimeType: mimeFallback,
        text: `[文件过大未上传：${file.size} 字节，单文件二进制上限约 ${MAX_RAW_BINARY_BYTES} 字节]`,
      });
      continue;
    }

    // 文本附件：排除图/音视频（避免误把 PNG 当 UTF-8 文本读坏）
    if (looksTextLike(file)) {
      const raw = (file.type || '').trim();
      const mime =
        raw && raw !== 'application/octet-stream'
          ? raw
          : inferMimeFromFilename(file.name) ?? 'text/plain';
      let t = await file.text();
      const runes = [...t];
      if (runes.length > MAX_ATTACHMENT_TEXT_RUNES) {
        t = runes.slice(0, MAX_ATTACHMENT_TEXT_RUNES).join('') + '\n…（本附件文本已截断）';
      }
      out.push({ filename: name, mimeType: mime, text: t });
      continue;
    }

    const buf = await file.arrayBuffer();
    const mime = pickMimeForBinaryPayload(file, buf);
    let b64 = arrayBufferToBase64(buf);
    if (b64.length > MAX_ATTACHMENT_BASE64) {
      b64 = b64.slice(0, MAX_ATTACHMENT_BASE64) + '\n…（base64 过长已截断）';
    }
    out.push({ filename: name, mimeType: mime, contentBase64: b64 });
  }

  return out;
}

/**
 * POST /api/v1/chat/stream（SSE：每行 `data: {"content","done","error"}`）
 */
export async function streamChat(
  payload: ChatStreamPayload,
  onChunk: (content: string, done: boolean, error?: string) => void,
  signal?: AbortSignal
): Promise<void> {
  const token =
    typeof window !== 'undefined'
      ? window.localStorage.getItem('AUTH_TOKEN') || window.localStorage.getItem('token')
      : '';
  const url = `${getApiOrigin()}/api/v1/chat/stream`;
  const body: Record<string, unknown> = {
    message: payload.message,
    model: payload.model ?? '',
    history: payload.history,
    llmConfigId: payload.llmConfigId,
    llmModelEntryId: payload.llmModelEntryId,
  };
  if (payload.managedAgentId != null && payload.managedAgentId > 0) {
    body.managedAgentId = payload.managedAgentId;
  }
  if (payload.attachments?.length) {
    // 始终带 mimeType 字段，避免 protojson/网关对「缺省 type」与 application/octet-stream 处理不一致
    body.attachments = payload.attachments.map((a) => {
      const row: Record<string, unknown> = {
        filename: a.filename,
        mimeType: a.mimeType ?? '',
      };
      if (a.text !== undefined && a.text !== '') row.text = a.text;
      if (a.contentBase64) row.contentBase64 = a.contentBase64;
      return row;
    });
  }

  const res = await fetch(url, {
    method: 'POST',
    headers: {
      Accept: 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
    signal,
  });
  if (!res.ok) {
    const t = await res.text();
    throw new Error(t || `HTTP ${res.status}`);
  }
  const reader = res.body?.getReader();
  if (!reader) {
    throw new Error('响应不支持流式读取');
  }
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split('\n\n');
    buffer = parts.pop() ?? '';
    for (const block of parts) {
      const line = block.trim();
      if (!line.startsWith('data:')) continue;
      const jsonStr = line.slice(5).trim();
      if (!jsonStr) continue;
      try {
        const obj = JSON.parse(jsonStr) as { content?: string; done?: boolean; error?: string };
        onChunk(obj.content ?? '', !!obj.done, obj.error);
      } catch {
        /* ignore malformed chunk */
      }
    }
  }
  if (buffer.trim()) {
    const line = buffer.trim();
    if (line.startsWith('data:')) {
      const jsonStr = line.slice(5).trim();
      try {
        const obj = JSON.parse(jsonStr) as { content?: string; done?: boolean; error?: string };
        onChunk(obj.content ?? '', !!obj.done, obj.error);
      } catch {
        /* ignore */
      }
    }
  }
}
