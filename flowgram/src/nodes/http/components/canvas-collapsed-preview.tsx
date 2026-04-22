/**
 * HTTP 节点在画布上的紧凑摘要（完整表单仅在侧栏）。
 */

import { Field } from '@flowgram.ai/free-layout-editor';

import {
  CANVAS_TWO_LINE_BOX_STYLE,
  normalizeCanvasPreviewText,
  truncateCanvasText,
} from '../../../utils/canvas-node-preview';

export function HttpCanvasCollapsedPreview() {
  return (
    <Field<string> name="api.method">
      {({ field: m }) => (
        <Field<string> name="body.bodyType">
          {({ field: b }) => (
            <Field name="api.url">
              {({ field: u }) => (
                <Field<number> name="timeout.timeout">
                  {({ field: to }) => (
                    <Field<number> name="timeout.retryTimes">
                      {({ field: tr }) => {
                        const urlRaw = String((u.value as { content?: unknown })?.content ?? '');
                        const urlPart =
                          urlRaw.trim() === ''
                            ? '（未填 URL）'
                            : truncateCanvasText(normalizeCanvasPreviewText(urlRaw), 72);
                        const line = `${String(m.value ?? 'GET')} · ${String(b.value ?? 'none')} · ${urlPart} · ${to.value ?? '—'}ms ×${tr.value ?? '—'}`;
                        return (
                          <div style={{ margin: '0 10px 8px' }}>
                            <div
                              style={{
                                ...CANVAS_TWO_LINE_BOX_STYLE,
                                background: '#f7f8fa',
                                color: '#1d2129',
                                fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
                              }}
                            >
                              {truncateCanvasText(line, 220)}
                            </div>
                          </div>
                        );
                      }}
                    </Field>
                  )}
                </Field>
              )}
            </Field>
          )}
        </Field>
      )}
    </Field>
  );
}
