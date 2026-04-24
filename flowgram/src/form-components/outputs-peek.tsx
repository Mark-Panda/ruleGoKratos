/**
 * 侧栏：完整输出端口（DisplayOutputs）；画布：outputs 字段两行 JSON 摘要，避免节点被撑高。
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { DisplayOutputs } from '@flowgram.ai/form-materials';

import { CANVAS_TWO_LINE_BOX_STYLE, canvasSchemaPreviewText } from '../utils/canvas-node-preview';
import { WorkflowNodeType } from '../nodes/constants';
import { useIsSidebar, useNodeRenderContext } from '../hooks';

export function OutputsPeek() {
  const isSidebar = useIsSidebar();
  const { node } = useNodeRenderContext();
  if (isSidebar) {
    return <DisplayOutputs displayFromScope />;
  }
  if (node.flowNodeType !== WorkflowNodeType.For) {
    return null;
  }
  return (
    <Field name="outputs">
      {({ field }) => {
        const text = canvasSchemaPreviewText(field.value, 220);
        if (!text.trim()) return null;
        return (
          <div
            style={{
              ...CANVAS_TWO_LINE_BOX_STYLE,
              fontFamily: 'monospace',
              fontSize: '11px',
            }}
          >
            {text}
          </div>
        );
      }}
    </Field>
  );
}
