/**
 * 侧栏：完整输出端口（DisplayOutputs）；画布：outputs 字段两行 JSON 摘要，避免节点被撑高。
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { DisplayOutputs } from '@flowgram.ai/form-materials';

import { useIsSidebar } from '../hooks';
import { CANVAS_TWO_LINE_BOX_STYLE, canvasSchemaPreviewText } from '../utils/canvas-node-preview';

export function OutputsPeek() {
  const isSidebar = useIsSidebar();
  if (isSidebar) {
    return <DisplayOutputs displayFromScope />;
  }
  return (
    <Field name="outputs">
      {({ field }) => (
        <div
          style={{
            ...CANVAS_TWO_LINE_BOX_STYLE,
            fontFamily: 'monospace',
            fontSize: '11px',
          }}
        >
          {canvasSchemaPreviewText(field.value, 220)}
        </div>
      )}
    </Field>
  );
}
