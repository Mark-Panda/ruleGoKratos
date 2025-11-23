/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { Divider } from '@douyinfe/semi-ui';

import { useIsSidebar, useNodeRenderContext } from '../../../hooks';
import { CodeEditorWithFormat } from '../../../components/code-editor-with-format';

export function TransformCode() {
  const isSidebar = useIsSidebar();
  const { readonly } = useNodeRenderContext();

  if (!isSidebar) {
    return null;
  }

  return (
    <>
      <Divider />
      <Field<string> name="script.content">
        {({ field }) => {
          const FIXED_HEADER = 'async function Transform(msg, metadata, msgType, dataType) {';
          const SIGNATURE_TRANSFORM = /^(?:\s*async\s+)?function\s+Transform\s*\([^)]*\)\s*\{/m;
          const SIGNATURE_STRICT =
            /^(?:\s*async\s+)?function\s+Transform\s*\(\s*msg\s*,\s*metadata\s*,\s*msgType\s*,\s*dataType\s*\)\s*\{/m;

          const enforceSignature = (src: string): string => {
            if (SIGNATURE_STRICT.test(src)) return src; // already correct
            if (SIGNATURE_TRANSFORM.test(src)) {
              return src.replace(SIGNATURE_TRANSFORM, FIXED_HEADER);
            }
            // fallback: replace first top-level function declaration
            const ANY_FUNC = /^(?:\s*async\s+)?function\s+[A-Za-z_$][\w$]*\s*\([^)]*\)\s*\{/m;
            if (ANY_FUNC.test(src)) {
              return src.replace(ANY_FUNC, FIXED_HEADER);
            }
            // if user deletes header entirely, prepend fixed header
            return `${FIXED_HEADER}\n` + src;
          };

          return (
            <CodeEditorWithFormat
              value={field.value}
              onChange={(value: string) => field.onChange(enforceSignature(String(value ?? '')))}
              readonly={readonly}
            />
          );
        }}
      </Field>
    </>
  );
}
