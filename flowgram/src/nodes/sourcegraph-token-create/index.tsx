import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconApi from '../../assets/icon_api.svg';
import { sourcegraphTokenCreateFormMeta } from './form-meta';

let index = 0;

export const SourcegraphTokenCreateNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.SourcegraphTokenCreate,
  info: {
    icon: iconApi,
    description: '通过 LDAP 凭证登录 Sourcegraph 并创建 Access Token；无需已有 Token。',
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
      width: 360,
      height: 400,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: sourcegraphTokenCreateFormMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.SourcegraphTokenCreate,
      data: {
        title: `SG_Token创建_${++index}`,
        positionType: 'middle',
        inputsValues: {
          endpoint: { type: 'template', content: '' },
          ldapUsername: { type: 'template', content: '' },
          ldapPassword: { type: 'template', content: '' },
          gitlabHost: { type: 'constant', content: 'gitlab.xxxx.tv' },
          note: { type: 'template', content: 'cli-token' },
          expiresAt: { type: 'template', content: '' },
          scope: { type: 'constant', content: '' },
          headless: { type: 'constant', content: 'true' },
          timeoutMs: { type: 'constant', content: 60000 },
        },
        inputs: {
          type: 'object',
          required: ['endpoint', 'ldapUsername', 'ldapPassword'],
          properties: {
            endpoint: {
              type: 'string',
              extra: {
                label: '请求地址',
                formComponent: 'prompt-editor',
                description: '例如 https://sourcegraph.xxxx.tv',
              },
            },
            ldapUsername: {
              type: 'string',
              extra: {
                label: 'LDAP 用户名',
                formComponent: 'prompt-editor',
                description: 'GitLab LDAP 用户名',
              },
            },
            ldapPassword: {
              type: 'string',
              extra: {
                label: 'LDAP 密码',
                formComponent: 'prompt-editor',
                description: 'GitLab LDAP 密码',
              },
            },
            gitlabHost: {
              type: 'string',
              extra: {
                label: 'GitLab 主机',
                formComponent: 'prompt-editor',
                description: '默认 gitlab.xxxx.tv',
              },
            },
            note: {
              type: 'string',
              extra: {
                label: 'Token 备注',
                formComponent: 'prompt-editor',
                description: 'Token 名称/备注，默认 cli-token',
              },
            },
            expiresAt: {
              type: 'string',
              extra: {
                label: '过期时间',
                formComponent: 'prompt-editor',
                description: 'ISO 8601 格式，如 2029-05-01T00:00:00Z；空则默认 3 年后',
              },
            },
            scope: {
              type: 'string',
              enum: ['', 'USER', 'SITE_ADMIN'],
              default: { type: 'constant', content: '' } as any,
              extra: {
                label: '权限范围',
                formComponent: 'enum-select',
                options: [
                  { label: '默认', value: '' },
                  { label: 'USER', value: 'USER' },
                  { label: 'SITE_ADMIN', value: 'SITE_ADMIN' },
                ],
              },
            },
            headless: {
              type: 'string',
              extra: {
                label: 'Headless 模式',
                formComponent: 'prompt-editor',
                description: '默认 true',
              },
            },
            timeoutMs: {
              type: 'number',
              extra: { label: '超时（毫秒）', description: '默认 60000' },
            },
          },
        },
      },
    } as any;
  },
};
