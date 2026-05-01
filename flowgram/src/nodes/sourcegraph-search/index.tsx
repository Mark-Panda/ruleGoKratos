import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconApi from '../../assets/icon_api.svg';
import { formMeta } from './form-meta';

let index = 0;

export const SourcegraphSearchNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.SourcegraphSearch,
  info: {
    icon: iconApi,
    description:
      'Sourcegraph 搜索：根据搜索路径组装查询并执行搜索，返回结果 JSON。',
  },
  meta: {
    panelCategory: 'service-calls',
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 380,
      height: 520,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.SourcegraphSearch,
      data: {
        title: `Sourcegraph搜索_${++index}`,
        positionType: 'middle',
        inputsValues: {
          endpoint: { type: 'template', content: '' },
          accessToken: { type: 'template', content: '' },
          timeoutSec: { type: 'constant', content: 30 },
          repoScope: { type: 'constant', content: '' },
          repoFrontend: { type: 'template', content: '' },
          repoBackend: { type: 'template', content: '' },
          contextGlobal: { type: 'constant', content: true },
          typeFilter: { type: 'template', content: '' },
          displayLimit: { type: 'constant', content: 1500 },
          defaultPatternType: { type: 'constant', content: 'literal' },
          defaultPatterns: { type: 'template', content: '' },
        },
        inputs: {
          type: 'object',
          required: ['endpoint'],
          properties: {
            endpoint: {
              type: 'string',
              extra: {
                label: '请求地址',
                formComponent: 'prompt-editor',
                description: '例如 https://sourcegraph.example.com',
              },
            },
            accessToken: {
              type: 'string',
              extra: {
                label: 'Access Token',
                formComponent: 'prompt-editor',
                description: 'Sourcegraph API 访问令牌',
              },
            },
            timeoutSec: {
              type: 'number',
              extra: { label: '超时（秒）', description: '默认 30' },
            },
            repoScope: {
              type: 'string',
              enum: ['', 'frontend', 'backend'],
              default: { type: 'constant', content: '' } as any,
              extra: {
                label: '仓库范围',
                formComponent: 'enum-select',
                options: [
                  { label: '不限制', value: '' },
                  { label: '仅前端仓库', value: 'frontend' },
                  { label: '仅后端仓库', value: 'backend' },
                ],
              },
            },
            repoFrontend: {
              type: 'string',
              extra: {
                label: '前端仓库正则',
                formComponent: 'prompt-editor',
                description: '默认 teacher/fe/.*|frontend/.*',
              },
            },
            repoBackend: {
              type: 'string',
              extra: {
                label: '后端仓库正则',
                formComponent: 'prompt-editor',
                description: '默认 teacher/backend/.*|backend/.*',
              },
            },
            contextGlobal: {
              type: 'boolean',
              extra: { label: '搜索全部仓库', description: '添加 context:global 条件，默认开启' },
            },
            typeFilter: {
              type: 'string',
              extra: {
                label: '文件类型过滤',
                formComponent: 'prompt-editor',
                description: '例如 lang:Go 或 file:\\.go$',
              },
            },
            displayLimit: {
              type: 'number',
              extra: { label: '结果数量上限', description: '默认 1500' },
            },
            defaultPatternType: {
              type: 'string',
              enum: ['literal', 'regexp'],
              default: { type: 'constant', content: 'literal' } as any,
              extra: {
                label: '匹配模式',
                formComponent: 'enum-select',
                options: [
                  { label: '精确匹配（literal）', value: 'literal' },
                  { label: '正则匹配（regexp）', value: 'regexp' },
                ],
              },
            },
            defaultPatterns: {
              type: 'string',
              extra: {
                label: '默认搜索路径',
                formComponent: 'prompt-editor',
                description: '换行分隔，例如 /api/user/list\\n/api/order/detail',
              },
            },
          },
        },
      },
    } as any;
  },
};
