/**
 * 画布节点内紧凑预览：统一两行截断（webkit line-clamp）+ Flow 值摘要。
 * 避免 white-space: pre-wrap 与 line-clamp 冲突导致「仍然占很多行」。
 */

import type { IFlowValue } from '@flowgram.ai/form-materials';

/** 画布字段两行预览（webkit line-clamp），与 maxHeight 双保险 */
export const CANVAS_TWO_LINE_BOX_STYLE = {
  padding: '6px 8px',
  background: '#f5f5f5',
  borderRadius: '4px',
  fontSize: '12px',
  color: '#666',
  whiteSpace: 'normal',
  wordBreak: 'break-word',
  overflow: 'hidden',
  display: '-webkit-box',
  WebkitBoxOrient: 'vertical',
  WebkitLineClamp: 2,
  lineHeight: '18px',
  maxHeight: '36px',
  minHeight: '36px',
} as const;

/** 画布摘要：折叠空白与换行，便于两行展示 */
export function normalizeCanvasPreviewText(s: string): string {
  return String(s ?? '')
    .replace(/\s+/g, ' ')
    .trim();
}

export function truncateCanvasText(s: string, maxChars: number): string {
  const t = normalizeCanvasPreviewText(s);
  if (!t) return '';
  if (t.length <= maxChars) return t;
  return `${t.slice(0, Math.max(0, maxChars - 1))}…`;
}

export function summarizeFlowValue(v: IFlowValue | undefined): string {
  if (v == null) return '';
  const any = v as { type?: string; content?: unknown };
  if (any.type === 'constant') {
    if (typeof any.content === 'string') return any.content;
    if (typeof any.content === 'number' || typeof any.content === 'boolean') {
      return String(any.content);
    }
    try {
      return JSON.stringify(any.content);
    } catch {
      return String(any.content);
    }
  }
  if (any.type === 'template') {
    return typeof any.content === 'string' ? any.content : '';
  }
  return '';
}

/** 详细多行摘要（不建议直接用于画布，内容多时仍会很长） */
export function summarizeFlowValuesRecord(
  rec: Record<string, IFlowValue | undefined> | undefined
): string {
  if (!rec || Object.keys(rec).length === 0) return '';
  return Object.entries(rec)
    .map(([k, val]) => `${k}: ${summarizeFlowValue(val)}`)
    .join('\n');
}

/**
 * 画布专用：单行语义拼接 + 字符上限，再放入两行 clamp 容器。
 */
export function summarizeFlowValuesRecordCompact(
  rec: Record<string, IFlowValue | undefined> | undefined,
  maxChars = 160
): string {
  if (!rec || Object.keys(rec).length === 0) return '';
  const parts: string[] = [];
  for (const [k, v] of Object.entries(rec)) {
    parts.push(`${k}:${summarizeFlowValue(v)}`);
  }
  return truncateCanvasText(parts.join(' · '), maxChars);
}

export function canvasSchemaPreviewText(schema: unknown, maxChars = 200): string {
  if (schema == null) return '';
  try {
    return truncateCanvasText(JSON.stringify(schema), maxChars);
  } catch {
    return truncateCanvasText(String(schema), maxChars);
  }
}
