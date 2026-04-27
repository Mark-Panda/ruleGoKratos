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
    description:
      '从文本中提取 JSON 并做格式纠错与补全。支持提取 markdown 代码块中的 JSON、自动修复单引号、缺失引号等常见错误。可输出解析后的 JSON 对象或错误信息。',
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
      height: 380,
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
          extractPattern: { type: 'constant', content: 'auto' },
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
                description: '包含 JSON 的文本内容，通常来自 Agent 输出的 markdown 代码块',
              },
            },
            extractPattern: {
              type: 'string',
              extra: {
                label: '提取模式',
                description: 'auto: 自动检测 JSON 代码块 | json: 仅提取标准 JSON | md: 仅提取 markdown 代码块',
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
