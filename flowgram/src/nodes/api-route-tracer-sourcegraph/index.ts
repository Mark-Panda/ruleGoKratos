import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconApi from '../../assets/icon_api.svg';

let index = 0;

const ACTION_OPTIONS = [
  { label: 'gitPrepare：准备仓库', value: 'gitPrepare' },
  { label: 'queryBuild：构建查询', value: 'queryBuild' },
  { label: 'search：执行搜索', value: 'search' },
];

export const ApiRouteTracerSourcegraphNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.ApiRouteTracerSourcegraph,
  info: {
    icon: iconApi,
    description:
      'API 路由追踪一体节点：按 action 在 gitPrepare / queryBuild / search 间切换。',
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
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.ApiRouteTracerSourcegraph,
      data: {
        title: `ApiRouteTracerSourcegraph_${++index}`,
        positionType: 'middle',
        inputsValues: {
          action: { type: 'constant', content: 'queryBuild' },
          gitlabUrl: { type: 'template', content: '' },
          workDir: { type: 'template', content: '' },
          endpoint: { type: 'template', content: '' },
          accessToken: { type: 'template', content: '' },
          timeoutSec: { type: 'constant', content: 30 },
          defaultSearchQuery: { type: 'template', content: '' },
          repoScope: { type: 'constant', content: '' },
          repoFrontend: { type: 'template', content: '' },
          repoBackend: { type: 'template', content: '' },
          contextGlobal: { type: 'constant', content: true },
          typeFilter: { type: 'template', content: '' },
          includeForked: { type: 'constant', content: true },
          displayLimit: { type: 'constant', content: 1500 },
          defaultPatternType: { type: 'constant', content: 'literal' },
          defaultPatterns: { type: 'template', content: '' },
        },
        inputs: {
          type: 'object',
          required: ['action'],
          properties: {
            action: {
              type: 'string',
              enum: ['gitPrepare', 'queryBuild', 'search'],
              default: { type: 'constant', content: 'queryBuild' } as any,
              extra: {
                label: '动作',
                formComponent: 'enum-select',
                options: ACTION_OPTIONS,
                description: '同一节点按动作切换行为：仓库准备 / 查询构建 / Sourcegraph 搜索',
              },
            },
            gitlabUrl: {
              type: 'string',
              extra: {
                label: 'GitLab URL（gitPrepare）',
                formComponent: 'prompt-editor',
                description: '仓库地址（不含 .git），支持 ${...} 模板',
              },
            },
            workDir: {
              type: 'string',
              extra: {
                label: '工作目录（gitPrepare）',
                formComponent: 'prompt-editor',
                description: '本地拉取目录，支持 ~ 与 ${...}',
              },
            },
            endpoint: {
              type: 'string',
              extra: {
                label: 'Sourcegraph Endpoint（search）',
                formComponent: 'prompt-editor',
                description: '例如 https://sourcegraph.com',
              },
            },
            accessToken: {
              type: 'string',
              extra: {
                label: 'Access Token（search）',
                formComponent: 'prompt-editor',
                description: '可留空（匿名实例）',
              },
            },
            timeoutSec: { type: 'number', extra: { label: '超时（秒）' } },
            defaultSearchQuery: {
              type: 'string',
              extra: {
                label: '默认查询（search）',
                formComponent: 'prompt-editor',
                description: '当消息 data 为空时回退该查询',
              },
            },
            repoScope: {
              type: 'string',
              enum: ['', 'frontend', 'backend'],
              default: { type: 'constant', content: '' } as any,
              extra: {
                label: '仓库范围（queryBuild）',
                formComponent: 'enum-select',
                options: [
                  { label: '全部', value: '' },
                  { label: 'frontend', value: 'frontend' },
                  { label: 'backend', value: 'backend' },
                ],
              },
            },
            repoFrontend: {
              type: 'string',
              extra: {
                label: '前端 repo 过滤（queryBuild）',
                formComponent: 'prompt-editor',
                description: 'repo 正则，不填使用内置默认',
              },
            },
            repoBackend: {
              type: 'string',
              extra: {
                label: '后端 repo 过滤（queryBuild）',
                formComponent: 'prompt-editor',
                description: 'repo 正则，不填使用内置默认',
              },
            },
            contextGlobal: {
              type: 'boolean',
              extra: { label: '追加 context:global（queryBuild）' },
            },
            typeFilter: {
              type: 'string',
              extra: {
                label: '类型过滤（queryBuild）',
                formComponent: 'prompt-editor',
                description: '例如 lang:go',
              },
            },
            includeForked: {
              type: 'boolean',
              extra: { label: '包含 fork 仓库（queryBuild）' },
            },
            displayLimit: {
              type: 'number',
              extra: { label: '结果数量上限（queryBuild）' },
            },
            defaultPatternType: {
              type: 'string',
              enum: ['literal', 'regexp'],
              default: { type: 'constant', content: 'literal' } as any,
              extra: {
                label: '默认 pattern 类型（queryBuild）',
                formComponent: 'enum-select',
              },
            },
            defaultPatterns: {
              type: 'string',
              extra: {
                label: '默认 patterns（queryBuild）',
                formComponent: 'prompt-editor',
                description: '每行一个 pattern；消息未传入时使用',
              },
            },
          },
        },
      },
    } as any;
  },
};
