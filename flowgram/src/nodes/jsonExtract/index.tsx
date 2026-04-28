import { WorkflowNodeType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconCode from '../../assets/icon-script.png';
import { formMeta } from './form-meta';

let index = 0;

export const JsonExtractNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.JsonExtract,
  info: {
    icon: iconCode,
    description: '从文本中提取 JSON（固定最严格模式，自动走完整提取链路）',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: 'Success' },
      { type: 'output', location: 'bottom', portID: 'Failure' },
    ],
    size: {
      width: 360,
      height: 260,
    },
    defaultExpanded: false,
    expandable: true,
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.JsonExtract,
      data: {
        title: `JsonExtract_${++index}`,
        positionType: 'middle',
        inputsValues: {
          source: { type: 'template', content: '' },
        },
        inputs: {
          type: 'object',
          required: ['source'],
          properties: {
            source: {
              type: 'string',
              extra: {
                label: '输入文本',
                formComponent: 'prompt-editor',
                description: '包含 JSON 的文本内容，组件将以固定最严格模式自动提取。',
              },
            },
          },
        },
        outputs: {
          type: 'object',
          properties: {
            result: {
              type: 'string',
            },
            extractedJson: {
              type: 'object',
            },
            error: {
              type: 'string',
            },
            success: {
              type: 'boolean',
            },
          },
        },
      },
    };
  },
  formMeta,
};
