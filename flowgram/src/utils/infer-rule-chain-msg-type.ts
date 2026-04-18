/**
 * 根据规则链详情推断异步/同步 API 路径中的 msgType（/notify/{msgType}）。
 * 优先级：flowgram.entry_msg_type → additionalInfo 约定字段 → metadata.endpoints 路由 path → 默认值 CHAIN。
 */

export const DEFAULT_INFERRED_NOTIFY_MSG_TYPE = 'CHAIN';

const ADDITIONAL_INFO_KEYS = [
  'entryMsgType',
  'msgType',
  'defaultMsgType',
  'notifyMsgType',
] as const;

/** 定时任务 endpoint，from.path 为 Cron 表达式而非 HTTP path */
const SKIP_ENDPOINT_TYPES = new Set(['endpoint/schedule']);

/** 粗略判断 Quartz/Cron 表达式，避免把 cron 当成 HTTP path */
export function looksLikeCronExpression(path: string): boolean {
  const t = path.trim();
  if (!t) return false;
  const spaceCount = (t.match(/\s/g) || []).length;
  if (spaceCount >= 4) return true;
  if (/^[\d\*\-\/,\s]+$/.test(t) && spaceCount >= 1) return true;
  return false;
}

function pathToMsgTypeCandidate(path: string): string | null {
  const p = path.trim();
  if (!p || looksLikeCronExpression(p)) return null;
  if (p.startsWith('/')) {
    const parts = p.replace(/\/+$/, '').split('/').filter(Boolean);
    const last = parts[parts.length - 1];
    if (!last) return null;
    const noBrace = last.replace(/\{[^}]+\}/g, '').trim();
    if (!noBrace || noBrace.includes('/')) return null;
    return noBrace;
  }
  const segments = p.split('/').filter((s) => s && s !== '+' && s !== '#' && !s.includes('*'));
  if (segments.length > 0) {
    return segments[segments.length - 1] ?? null;
  }
  if (/^[a-zA-Z0-9_.\-]+$/.test(p)) return p;
  return null;
}

function inferFromEndpoints(metadata: unknown): string | null {
  const endpoints = (metadata as { endpoints?: unknown })?.endpoints;
  if (!Array.isArray(endpoints) || endpoints.length === 0) return null;

  const sorted = [...endpoints].sort((a, b) => {
    const rank = (t: string) => {
      const s = String(t);
      if (s.includes('rest') || s.includes('http')) return 0;
      return 1;
    };
    return (
      rank(String((a as { type?: string }).type ?? '')) -
      rank(String((b as { type?: string }).type ?? ''))
    );
  });

  for (const ep of sorted) {
    const typ = String((ep as { type?: string }).type ?? '');
    if (SKIP_ENDPOINT_TYPES.has(typ)) continue;
    const routers = (ep as { routers?: unknown }).routers;
    if (!Array.isArray(routers)) continue;
    for (const r of routers) {
      const path = (r as { from?: { path?: string } })?.from?.path;
      if (typeof path !== 'string') continue;
      const c = pathToMsgTypeCandidate(path);
      if (c) return c;
    }
  }
  return null;
}

function inferFromAdditionalInfo(ruleChain: unknown): string | null {
  const add = (ruleChain as { additionalInfo?: unknown })?.additionalInfo;
  if (!add || typeof add !== 'object' || Array.isArray(add)) return null;
  const o = add as Record<string, unknown>;
  for (const key of ADDITIONAL_INFO_KEYS) {
    const v = o[key];
    if (typeof v === 'string' && v.trim()) return v.trim();
  }
  return null;
}

/**
 * @param detail getRuleDetail 返回体：{ ruleChain, metadata }
 * @param flowgramEntryMsg parseRuleChainFlowgramFromConfiguration(d).entryMsgType
 */
export function inferMsgTypeFromRuleDetail(detail: unknown, flowgramEntryMsg?: string): string {
  const hint = typeof flowgramEntryMsg === 'string' ? flowgramEntryMsg.trim() : '';
  if (hint) return hint;

  const rc = (detail as { ruleChain?: unknown })?.ruleChain;
  if (rc) {
    const fromAdd = inferFromAdditionalInfo(rc);
    if (fromAdd) return fromAdd;
  }

  const meta = (detail as { metadata?: unknown })?.metadata;
  const fromEp = inferFromEndpoints(meta);
  if (fromEp) return fromEp;

  return DEFAULT_INFERRED_NOTIFY_MSG_TYPE;
}
