/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { Divider } from '@douyinfe/semi-ui';

import { useIsSidebar, useNodeRenderContext } from '../../../hooks';
import { LuaCodeEditor } from '../../../form-components/code-editor/lua';

export function LuaTransformCode() {
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
          const FIXED_HEADER = 'function Transform(msg, metadata, msgType, dataType)';
          const FIXED_FOOTER = 'end';
          const SIGNATURE_START =
            /\bfunction\s+Transform\s*\(\s*msg\s*,\s*metadata\s*,\s*msgType\s*,\s*dataType\s*\)\s*/m;
          const SIGNATURE_END = /\bend\s*$/m;

          const enforceSignature = (src: string): string => {
            let text = String(src ?? '');
            if (!SIGNATURE_START.test(text)) {
              // prepend fixed header if missing or wrong
              const firstLineEnd = text.indexOf('\n');
              const body = firstLineEnd >= 0 ? text.slice(firstLineEnd + 1) : text;
              text = `${FIXED_HEADER}\n${body}`;
            } else {
              // normalize header line to fixed signature
              text = text.replace(SIGNATURE_START, `${FIXED_HEADER}\n`);
            }
            if (!SIGNATURE_END.test(text)) {
              // append end if missing
              if (!text.endsWith('\n')) text += '\n';
              text += FIXED_FOOTER;
            }
            return text;
          };

          return (
            <LuaCodeEditor
              value={field.value}
              onChange={(value) => field.onChange(enforceSignature(String(value ?? '')))}
              readonly={readonly}
            />
          );
        }}
      </Field>
    </>
  );
}
