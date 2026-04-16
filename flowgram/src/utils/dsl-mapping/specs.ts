import type { NodeMappingSpec } from './types';
import {
  transformAiLlmConfigIn,
  transformAiLlmConfigOut,
  transformRestApiCallIn,
  transformRestApiCallOut,
  transformSwitchConfigIn,
  transformSwitchConfigOut,
} from './specializers';

/**
 * 节点映射规格注册表。
 *
 * 缺字段 / 空值时的回填以各字段 `defaultValue` 为准（与 `engine.ts` 中「spec 默认值语义」一致）；
 * 勿在调用方另写一套隐式默认。
 */

/** ai/llm：画布 inputsValues 与 DSL configuration（含 messages、params）之间的字段规格。 */
export const aiLlmMappingSpec: NodeMappingSpec = {
  nodeType: 'ai/llm',
  fields: [
    {
      inputKey: 'model',
      dslKey: 'model',
      valueType: 'constant',
      defaultValue: '',
    },
    { inputKey: 'key', dslKey: 'key', valueType: 'constant', defaultValue: '' },
    { inputKey: 'url', dslKey: 'url', valueType: 'constant', defaultValue: '' },
    {
      inputKey: 'systemPrompt',
      dslKey: 'systemPrompt',
      valueType: 'template',
      defaultValue: '',
    },
    {
      inputKey: 'userPrompt',
      dslKey: 'userPrompt',
      valueType: 'template',
      defaultValue: '',
    },
    {
      inputKey: 'temperature',
      dslKey: 'temperature',
      valueType: 'number',
      defaultValue: 0.5,
    },
    {
      inputKey: 'topP',
      dslKey: 'topP',
      valueType: 'number',
      defaultValue: 0.5,
    },
    {
      inputKey: 'maxTokens',
      dslKey: 'maxTokens',
      valueType: 'number',
      defaultValue: 0,
    },
    {
      inputKey: 'responseFormat',
      dslKey: 'responseFormat',
      valueType: 'constant',
      defaultValue: 'text',
    },
  ],
  transformOut: transformAiLlmConfigOut,
  transformIn: transformAiLlmConfigIn,
};

/** ai/agentHarness：与 buildDocumentFromRuleChainJSON 中回显默认值一致。 */
export const aiAgentHarnessMappingSpec: NodeMappingSpec = {
  nodeType: 'ai/agentHarness',
  fields: [
    {
      inputKey: 'model',
      dslKey: 'model',
      valueType: 'template',
      defaultValue: '',
    },
    {
      inputKey: 'systemPrompt',
      dslKey: 'systemPrompt',
      valueType: 'template',
      defaultValue: '',
    },
    {
      inputKey: 'userPrompt',
      dslKey: 'userPrompt',
      valueType: 'template',
      defaultValue: '',
    },
    {
      inputKey: 'enableSkillTool',
      dslKey: 'enableSkillTool',
      valueType: 'boolean',
      defaultValue: true,
    },
    {
      inputKey: 'enableMcpTool',
      dslKey: 'enableMcpTool',
      valueType: 'boolean',
      defaultValue: true,
    },
    {
      inputKey: 'enableUUIDTool',
      dslKey: 'enableUUIDTool',
      valueType: 'boolean',
      defaultValue: true,
    },
    {
      inputKey: 'enableWorkspaceTools',
      dslKey: 'enableWorkspaceTools',
      valueType: 'boolean',
      defaultValue: false,
    },
    {
      inputKey: 'skillAllowlist',
      dslKey: 'skillAllowlist',
      valueType: 'template',
      defaultValue: '',
    },
    {
      inputKey: 'mcpAllowlist',
      dslKey: 'mcpAllowlist',
      valueType: 'template',
      defaultValue: '',
    },
    {
      inputKey: 'maxIterations',
      dslKey: 'maxIterations',
      valueType: 'number',
      defaultValue: 0,
    },
    {
      inputKey: 'maxToolCalls',
      dslKey: 'maxToolCalls',
      valueType: 'number',
      defaultValue: 0,
    },
    {
      inputKey: 'toolTimeoutSecs',
      dslKey: 'toolTimeoutSecs',
      valueType: 'number',
      defaultValue: 0,
    },
  ],
};

/**
 * restApiCall：画布侧用 `url`（无 query 的基址）+ `params`（JSON 对象）等；DSL 侧为
 * `restEndpointUrlPattern`（可含 query）+ `params`（对象）。query 与 params 的折叠/展开由 specializer 完成。
 */
export const restApiCallMappingSpec: NodeMappingSpec = {
  nodeType: 'restApiCall',
  fields: [
    {
      inputKey: 'url',
      dslKey: 'restUrlBase',
      valueType: 'template',
      defaultValue: '',
    },
    {
      inputKey: 'requestMethod',
      dslKey: 'requestMethod',
      valueType: 'constant',
      defaultValue: 'GET',
    },
    {
      inputKey: 'params',
      dslKey: 'params',
      valueType: 'json',
      defaultValue: {},
    },
    {
      inputKey: 'headers',
      dslKey: 'headers',
      valueType: 'json',
      defaultValue: {},
    },
    {
      inputKey: 'body',
      dslKey: 'body',
      valueType: 'template',
      defaultValue: undefined,
    },
    {
      inputKey: 'readTimeoutMs',
      dslKey: 'readTimeoutMs',
      valueType: 'number',
      defaultValue: undefined,
    },
  ],
  transformOut: transformRestApiCallOut,
  transformIn: transformRestApiCallIn,
};

export const flowMappingSpec: NodeMappingSpec = {
  nodeType: 'flow',
  fields: [
    {
      inputKey: 'targetId',
      dslKey: 'targetId',
      valueType: 'constant',
      defaultValue: '',
    },
    {
      inputKey: 'extend',
      dslKey: 'extend',
      valueType: 'boolean',
      defaultValue: false,
    },
  ],
};

export const yapiMappingSpec: NodeMappingSpec = {
  nodeType: 'transform/yapi',
  fields: [
    {
      inputKey: 'baseUrl',
      dslKey: 'baseUrl',
      valueType: 'constant',
      defaultValue: '',
    },
    {
      inputKey: 'userName',
      dslKey: 'userName',
      valueType: 'constant',
      defaultValue: '',
    },
    {
      inputKey: 'password',
      dslKey: 'password',
      valueType: 'constant',
      defaultValue: '',
    },
    {
      inputKey: 'interfacePath',
      dslKey: 'interfacePath',
      valueType: 'constant',
      defaultValue: '',
    },
    {
      inputKey: 'loginType',
      dslKey: 'loginType',
      valueType: 'constant',
      defaultValue: 'ldap',
    },
  ],
};

export const dbClientMappingSpec: NodeMappingSpec = {
  nodeType: 'dbClient',
  fields: [
    { inputKey: 'driverName', dslKey: 'driverName', valueType: 'constant', defaultValue: 'mysql' },
    { inputKey: 'dsn', dslKey: 'dsn', valueType: 'template', defaultValue: '' },
    { inputKey: 'sql', dslKey: 'sql', valueType: 'template', defaultValue: '' },
    { inputKey: 'params', dslKey: 'params', valueType: 'json', defaultValue: [] },
    { inputKey: 'getOne', dslKey: 'getOne', valueType: 'boolean', defaultValue: false },
    { inputKey: 'poolSize', dslKey: 'poolSize', valueType: 'number', defaultValue: 0 },
  ],
};

export const redisClientMappingSpec: NodeMappingSpec = {
  nodeType: 'x/redisClient',
  fields: [
    { inputKey: 'server', dslKey: 'server', valueType: 'constant', defaultValue: '' },
    { inputKey: 'password', dslKey: 'password', valueType: 'constant', defaultValue: '' },
    { inputKey: 'poolSize', dslKey: 'poolSize', valueType: 'number', defaultValue: 0 },
    { inputKey: 'db', dslKey: 'db', valueType: 'number', defaultValue: 0 },
    { inputKey: 'cmd', dslKey: 'cmd', valueType: 'template', defaultValue: '' },
    { inputKey: 'params', dslKey: 'params', valueType: 'json', defaultValue: [] },
  ],
};

export const multiNodeOutputMappingSpec: NodeMappingSpec = {
  nodeType: 'transform/multiNodeOutput',
  fields: [{ inputKey: 'nodeIds', dslKey: 'nodeIds', valueType: 'json', defaultValue: [] }],
};

export const switchMappingSpec: NodeMappingSpec = {
  nodeType: 'switch',
  fields: [{ inputKey: 'cases', dslKey: 'cases', valueType: 'json', defaultValue: [] }],
  transformOut: transformSwitchConfigOut,
  transformIn: transformSwitchConfigIn,
};

export const jsTransformMappingSpec: NodeMappingSpec = {
  nodeType: 'jsTransform',
  fields: [{ inputKey: 'scriptBody', dslKey: 'jsScript', valueType: 'template', defaultValue: '' }],
};

export const logMappingSpec: NodeMappingSpec = {
  nodeType: 'log',
  fields: [{ inputKey: 'scriptBody', dslKey: 'jsScript', valueType: 'template', defaultValue: '' }],
};

export const jsFilterMappingSpec: NodeMappingSpec = {
  nodeType: 'jsFilter',
  fields: [{ inputKey: 'scriptBody', dslKey: 'jsScript', valueType: 'template', defaultValue: '' }],
};

export const luaTransformMappingSpec: NodeMappingSpec = {
  nodeType: 'luaTransform',
  fields: [
    { inputKey: 'scriptBody', dslKey: 'luaScript', valueType: 'template', defaultValue: '' },
  ],
};

const SPEC_BY_TYPE: Record<string, NodeMappingSpec> = {
  'ai/llm': aiLlmMappingSpec,
  'ai/agentHarness': aiAgentHarnessMappingSpec,
  restApiCall: restApiCallMappingSpec,
  flow: flowMappingSpec,
  'transform/yapi': yapiMappingSpec,
  dbClient: dbClientMappingSpec,
  'x/redisClient': redisClientMappingSpec,
  'transform/multiNodeOutput': multiNodeOutputMappingSpec,
  switch: switchMappingSpec,
  jsTransform: jsTransformMappingSpec,
  log: logMappingSpec,
  jsFilter: jsFilterMappingSpec,
  luaTransform: luaTransformMappingSpec,
};

export function getNodeMappingSpec(nodeType: string): NodeMappingSpec | undefined {
  return SPEC_BY_TYPE[nodeType];
}
