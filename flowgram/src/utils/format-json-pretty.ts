/**
 * 将合法 JSON 字符串格式化为 2 空格缩进；非法则返回错误信息。
 * 不支持 JSONC 注释（含 // 时需先移除后再格式化）。
 */

export type FormatJsonPrettyResult = { ok: true; text: string } | { ok: false; error: string };

export function tryFormatJsonPretty(raw: string): FormatJsonPrettyResult {
  const t = raw?.trim() ?? '';
  if (t === '') {
    return { ok: true, text: '' };
  }
  try {
    const parsed = JSON.parse(t);
    return { ok: true, text: JSON.stringify(parsed, null, 2) };
  } catch (e) {
    return {
      ok: false,
      error: e instanceof Error ? e.message : String(e),
    };
  }
}
