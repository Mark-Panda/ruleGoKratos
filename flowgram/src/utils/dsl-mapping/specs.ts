import type { NodeMappingSpec } from './types';
import {
  transformAiLlmConfigIn,
  transformAiLlmConfigOut,
  transformCursorCliConfigIn,
  transformFeishuWebhookConfigIn,
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
    {
      inputKey: 'userPrompt',
      dslKey: 'userPrompt',
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
    { inputKey: 'key', dslKey: 'key', valueType: 'constant', defaultValue: '' },
    { inputKey: 'url', dslKey: 'url', valueType: 'constant', defaultValue: '' },
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
      inputKey: 'userPrompt',
      dslKey: 'userPrompt',
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

/**
 * x/cursorCli：调用官方 Cursor CLI（可执行文件为 agent，见安装/概览文档）。
 * 仍兼容历史 DSL 键 cursorPath（经 transformIn 合并到 agentPath）。
 * 无头主路径：printMode、prompt、outputFormat（--output-format）、model；args 仅追加额外 argv。
 */
export const cursorCliMappingSpec: NodeMappingSpec = {
  nodeType: 'x/cursorCli',
  fields: [
    { inputKey: 'agentPath', dslKey: 'agentPath', valueType: 'constant', defaultValue: 'agent' },
    { inputKey: 'args', dslKey: 'args', valueType: 'json', defaultValue: [] },
    { inputKey: 'printMode', dslKey: 'printMode', valueType: 'boolean', defaultValue: false },
    { inputKey: 'prompt', dslKey: 'prompt', valueType: 'template', defaultValue: '' },
    // 与 -p 同时生效；仅允许 text / json / stream-json，缺省由后端按 text 写入 argv。
    { inputKey: 'outputFormat', dslKey: 'outputFormat', valueType: 'constant', defaultValue: 'text' },
    { inputKey: 'model', dslKey: 'model', valueType: 'template', defaultValue: '' },
    // 非空时插入 --api-key；留空则运行时读 CURSOR_API_KEY；可用 ${metadata.xxx}，勿硬编码进仓库。
    { inputKey: 'apiKey', dslKey: 'apiKey', valueType: 'template', defaultValue: '' },
    // 非空时插入 --workspace（仓库根 / 代码上下文）；与 workDir（进程 cwd）不同。
    { inputKey: 'workspacePath', dslKey: 'workspacePath', valueType: 'template', defaultValue: '' },
    { inputKey: 'log', dslKey: 'log', valueType: 'boolean', defaultValue: false },
    { inputKey: 'replaceData', dslKey: 'replaceData', valueType: 'boolean', defaultValue: true },
    { inputKey: 'workDir', dslKey: 'workDir', valueType: 'template', defaultValue: '' },
    { inputKey: 'timeoutMs', dslKey: 'timeoutMs', valueType: 'number', defaultValue: 0 },
  ],
  transformIn: transformCursorCliConfigIn,
};

/**
 * x/cursorAcp：以 stdio 启动 agent acp（JSON-RPC 每行一条）。
 * 用户任务/说明：写在 stdinLines 中对应 RPC（如 session/prompt）的 JSON 行内，无单独 prompt 键。
 */
/**
 * x/feishuWebhook：飞书自定义机器人 Webhook（v2）。
 * post：postSplitByLine / postAtAllBefore|After / postMentionUserIds 为友好勾选与列表。
 * interactive：interactivePreset=notice_card 时用 cardNoticeTitle + cardNoticeMarkdown 组装通知卡。
 */
export const feishuWebhookMappingSpec: NodeMappingSpec = {
  nodeType: 'x/feishuWebhook',
  fields: [
    { inputKey: 'msgType', dslKey: 'msgType', valueType: 'constant', defaultValue: 'text' },
    { inputKey: 'webhookUrl', dslKey: 'webhookUrl', valueType: 'template', defaultValue: '' },
    { inputKey: 'text', dslKey: 'text', valueType: 'template', defaultValue: '' },
    { inputKey: 'postTitle', dslKey: 'postTitle', valueType: 'template', defaultValue: '' },
    { inputKey: 'postBody', dslKey: 'postBody', valueType: 'template', defaultValue: '' },
    { inputKey: 'postLang', dslKey: 'postLang', valueType: 'constant', defaultValue: 'zh_cn' },
    { inputKey: 'postSplitByLine', dslKey: 'postSplitByLine', valueType: 'boolean', defaultValue: false },
    { inputKey: 'postAtAllBefore', dslKey: 'postAtAllBefore', valueType: 'boolean', defaultValue: false },
    { inputKey: 'postAtAllAfter', dslKey: 'postAtAllAfter', valueType: 'boolean', defaultValue: false },
    { inputKey: 'postMentionUserIds', dslKey: 'postMentionUserIds', valueType: 'json', defaultValue: [] },
    {
      inputKey: 'interactivePreset',
      dslKey: 'interactivePreset',
      valueType: 'constant',
      defaultValue: 'card_json',
    },
    { inputKey: 'cardNoticeTitle', dslKey: 'cardNoticeTitle', valueType: 'template', defaultValue: '' },
    {
      inputKey: 'cardNoticeMarkdown',
      dslKey: 'cardNoticeMarkdown',
      valueType: 'template',
      defaultValue: '',
    },
    { inputKey: 'cardJson', dslKey: 'cardJson', valueType: 'template', defaultValue: '' },
    { inputKey: 'rawJson', dslKey: 'rawJson', valueType: 'template', defaultValue: '' },
    { inputKey: 'timeoutMs', dslKey: 'timeoutMs', valueType: 'number', defaultValue: 15000 },
    { inputKey: 'replaceData', dslKey: 'replaceData', valueType: 'boolean', defaultValue: false },
  ],
  transformIn: transformFeishuWebhookConfigIn,
};

export const cursorAcpMappingSpec: NodeMappingSpec = {
  nodeType: 'x/cursorAcp',
  fields: [
    { inputKey: 'agentPath', dslKey: 'agentPath', valueType: 'constant', defaultValue: 'agent' },
    { inputKey: 'args', dslKey: 'args', valueType: 'json', defaultValue: ['acp'] },
    { inputKey: 'stdinLines', dslKey: 'stdinLines', valueType: 'json', defaultValue: [] },
    { inputKey: 'apiKey', dslKey: 'apiKey', valueType: 'template', defaultValue: '' },
    { inputKey: 'workspacePath', dslKey: 'workspacePath', valueType: 'template', defaultValue: '' },
    { inputKey: 'log', dslKey: 'log', valueType: 'boolean', defaultValue: false },
    { inputKey: 'replaceData', dslKey: 'replaceData', valueType: 'boolean', defaultValue: true },
    { inputKey: 'workDir', dslKey: 'workDir', valueType: 'template', defaultValue: '' },
    { inputKey: 'timeoutMs', dslKey: 'timeoutMs', valueType: 'number', defaultValue: 120000 },
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
  'x/cursorCli': cursorCliMappingSpec,
  'x/cursorAcp': cursorAcpMappingSpec,
  'x/feishuWebhook': feishuWebhookMappingSpec,
};

export function getNodeMappingSpec(nodeType: string): NodeMappingSpec | undefined {
  return SPEC_BY_TYPE[nodeType];
}
