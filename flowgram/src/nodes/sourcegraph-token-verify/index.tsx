import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconApi from '../../assets/icon_api.svg';
import { sourcegraphTokenVerifyFormMeta } from './form-meta';

let index = 0;

export const SourcegraphTokenVerifyNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.SourcegraphTokenVerify,
  info: {
    icon: iconApi,
    description: '校验 SourceGraph API Token 有效性；有效走 Success，无效走 Failure。',
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
      width: 320,
      height: 240,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: sourcegraphTokenVerifyFormMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.SourcegraphTokenVerify,
      data: {
        title: `SG_Token校验_${++index}`,
        positionType: 'middle',
        inputsValues: {
          endpoint: { type: 'template', content: '' },
          accessToken: { type: 'template', content: '' },
          timeoutSec: { type: 'constant', content: 15 },
        },
        inputs: {
          type: 'object',
          required: ['endpoint', 'accessToken'],
          properties: {
            endpoint: {
              type: 'string',
              extra: {
                label: '请求地址',
                formComponent: 'prompt-editor',
                description: '例如 https://sourcegraph.xxxx.tv',
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
              extra: { label: '超时（秒）', description: '默认 15' },
            },
          },
        },
      },
    } as any;
  },
};
