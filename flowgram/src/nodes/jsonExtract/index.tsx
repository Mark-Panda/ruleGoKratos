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
      '从文本中提取 JSON 并做格式纠错与补全。支持 markdown/标签/片段提取、结构补全与 schema 补齐；可选输出 repair report。',
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
          parseMode: { type: 'constant', content: 'auto' },
          schemaPaths: { type: 'constant', content: '' },
          emitReport: { type: 'constant', content: false },
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
            parseMode: {
              type: 'string',
              enum: ['strict', 'auto', 'aggressive'],
              extra: {
                label: '解析强度',
                description: 'strict: 仅轻量解析 | auto: 默认修复与补全 | aggressive: 激进修复（NaN/Infinity/分号等）',
              },
            },
            schemaPaths: {
              type: 'string',
              extra: {
                label: 'Schema 路径约束',
                description:
                  '逗号/分号/换行分隔，如 data[].name,data[].spaceName,data[].manager。用于评分优选与缺失字段补齐。',
              },
            },
            emitReport: {
              type: 'boolean',
              extra: {
                label: '输出修复报告',
                description: '开启后输出 source_strategy、repair_strategies、schema_missing 等诊断信息。',
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
