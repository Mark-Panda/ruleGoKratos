/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { Divider } from '@douyinfe/semi-ui';

import { useIsSidebar, useNodeRenderContext } from '../../../hooks';
import { CodeEditorWithFormat } from '../../../components/code-editor-with-format';

export function Code() {
  const isSidebar = useIsSidebar();
  const { readonly } = useNodeRenderContext();

  if (!isSidebar) {
    return null;
  }

  return (
    <>
      <Divider />
      <Field<string> name="script.content">
        {({ field }) => (
          <CodeEditorWithFormat
            value={field.value}
            onChange={(value: string) => field.onChange(value)}
            readonly={readonly}
          />
        )}
      </Field>
    </>
  );
}
