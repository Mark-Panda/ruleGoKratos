import type { NodeMappingSpec } from './types';
import {
  transformAgentHarnessConfigIn,
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

/** ai/agentHarness：与 buildDocumentFromRuleChainJSON 中回显默认值一致。 */
export const aiAgentHarnessMappingSpec: NodeMappingSpec = {
  nodeType: 'ai/agentHarness',
  fields: [
    {
      inputKey: 'llmConfigId',
      dslKey: 'llmConfigId',
      valueType: 'number',
      defaultValue: 0,
    },
    {
      inputKey: 'llmModelEntryId',
      dslKey: 'llmModelEntryId',
      valueType: 'number',
      defaultValue: 0,
    },
    {
      inputKey: 'managedAgentId',
      dslKey: 'managedAgentId',
      valueType: 'number',
      defaultValue: 0,
    },
    {
      inputKey: 'workspaceId',
      dslKey: 'workspaceId',
      valueType: 'template',
      defaultValue: '',
    },
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
      inputKey: 'enableUUIDTool',
      dslKey: 'enableUUIDTool',
      valueType: 'boolean',
      defaultValue: true,
    },
    {
      inputKey: 'enableWorkspaceTools',
      dslKey: 'enableWorkspaceTools',
      valueType: 'boolean',
      defaultValue: true,
    },
    {
      inputKey: 'enableSubAgentTool',
      dslKey: 'enableSubAgentTool',
      valueType: 'boolean',
      defaultValue: true,
    },
    {
      inputKey: 'skillAllowlist',
      dslKey: 'skillAllowlist',
      valueType: 'json',
      defaultValue: [],
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
    {
      inputKey: 'gitWorktreeMode',
      dslKey: 'gitWorktreeMode',
      valueType: 'boolean',
      defaultValue: false,
    },
  ],
  transformIn: transformAgentHarnessConfigIn,
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

export const opensearchSearchMappingSpec: NodeMappingSpec = {
  nodeType: 'opensearch/search',
  fields: [
    { inputKey: 'endpoint', dslKey: 'endpoint', valueType: 'template', defaultValue: '' },
    { inputKey: 'index', dslKey: 'index', valueType: 'template', defaultValue: '' },
    { inputKey: 'username', dslKey: 'username', valueType: 'constant', defaultValue: '' },
    { inputKey: 'password', dslKey: 'password', valueType: 'constant', defaultValue: '' },
    {
      inputKey: 'insecureSkipVerify',
      dslKey: 'insecureSkipVerify',
      valueType: 'boolean',
      defaultValue: false,
    },
    { inputKey: 'timeoutSec', dslKey: 'timeoutSec', valueType: 'number', defaultValue: 60 },
    {
      inputKey: 'searchType',
      dslKey: 'searchType',
      valueType: 'constant',
      defaultValue: 'query_then_fetch',
    },
    {
      inputKey: 'ignoreUnavailable',
      dslKey: 'ignoreUnavailable',
      valueType: 'boolean',
      defaultValue: false,
    },
    {
      inputKey: 'defaultSearchBody',
      dslKey: 'defaultSearchBody',
      valueType: 'template',
      defaultValue:
        '{"size":100,"sort":[{"@timestamp":{"order":"desc"}}],"query":{"match_all":{}}}',
    },
  ],
};

export const volcTlsSearchLogsMappingSpec: NodeMappingSpec = {
  nodeType: 'volcTls/searchLogs',
  fields: [
    { inputKey: 'endpoint', dslKey: 'endpoint', valueType: 'constant', defaultValue: '' },
    { inputKey: 'region', dslKey: 'region', valueType: 'constant', defaultValue: '' },
    { inputKey: 'accessKeyId', dslKey: 'accessKeyId', valueType: 'template', defaultValue: '' },
    {
      inputKey: 'secretAccessKey',
      dslKey: 'secretAccessKey',
      valueType: 'template',
      defaultValue: '',
    },
    { inputKey: 'sessionToken', dslKey: 'sessionToken', valueType: 'template', defaultValue: '' },
    { inputKey: 'topicId', dslKey: 'topicId', valueType: 'template', defaultValue: '' },
    { inputKey: 'defaultQuery', dslKey: 'defaultQuery', valueType: 'template', defaultValue: '*' },
    { inputKey: 'limit', dslKey: 'limit', valueType: 'number', defaultValue: 100 },
    { inputKey: 'useApiV3', dslKey: 'useApiV3', valueType: 'boolean', defaultValue: false },
    { inputKey: 'timeoutSec', dslKey: 'timeoutSec', valueType: 'number', defaultValue: 60 },
    {
      inputKey: 'timeRangePreset',
      dslKey: 'timeRangePreset',
      valueType: 'constant',
      defaultValue: 'last_15m',
    },
    {
      inputKey: 'defaultStartTimeMs',
      dslKey: 'defaultStartTimeMs',
      valueType: 'number',
      defaultValue: 0,
    },
    {
      inputKey: 'defaultEndTimeMs',
      dslKey: 'defaultEndTimeMs',
      valueType: 'number',
      defaultValue: 0,
    },
    { inputKey: 'defaultSort', dslKey: 'defaultSort', valueType: 'constant', defaultValue: 'desc' },
    { inputKey: 'highLight', dslKey: 'highLight', valueType: 'boolean', defaultValue: false },
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

/** 包容分支：DSL 形状与 switch 相同（cases），路由语义由引擎区分。 */
export const inclusiveMappingSpec: NodeMappingSpec = {
  nodeType: 'inclusive',
  fields: [{ inputKey: 'cases', dslKey: 'cases', valueType: 'json', defaultValue: [] }],
  transformOut: transformSwitchConfigOut,
  transformIn: transformSwitchConfigIn,
};

export const whileMappingSpec: NodeMappingSpec = {
  nodeType: 'while',
  fields: [
    {
      inputKey: 'condition',
      dslKey: 'condition',
      valueType: 'template',
      defaultValue: '',
    },
    { inputKey: 'do', dslKey: 'do', valueType: 'constant', defaultValue: '' },
  ],
};

export const execMappingSpec: NodeMappingSpec = {
  nodeType: 'exec',
  fields: [
    { inputKey: 'cmd', dslKey: 'cmd', valueType: 'template', defaultValue: '' },
    { inputKey: 'args', dslKey: 'args', valueType: 'json', defaultValue: [] },
    { inputKey: 'log', dslKey: 'log', valueType: 'boolean', defaultValue: false },
    { inputKey: 'replaceData', dslKey: 'replaceData', valueType: 'boolean', defaultValue: false },
  ],
};

export const fileReadMappingSpec: NodeMappingSpec = {
  nodeType: 'x/fileRead',
  fields: [
    { inputKey: 'path', dslKey: 'path', valueType: 'template', defaultValue: '' },
    { inputKey: 'dataType', dslKey: 'dataType', valueType: 'constant', defaultValue: 'text' },
    { inputKey: 'recursive', dslKey: 'recursive', valueType: 'boolean', defaultValue: false },
  ],
};

export const fileWriteMappingSpec: NodeMappingSpec = {
  nodeType: 'x/fileWrite',
  fields: [
    { inputKey: 'path', dslKey: 'path', valueType: 'template', defaultValue: '' },
    { inputKey: 'content', dslKey: 'content', valueType: 'template', defaultValue: '${data}' },
    { inputKey: 'append', dslKey: 'append', valueType: 'boolean', defaultValue: false },
  ],
};

export const fileDeleteMappingSpec: NodeMappingSpec = {
  nodeType: 'x/fileDelete',
  fields: [{ inputKey: 'path', dslKey: 'path', valueType: 'template', defaultValue: '' }],
};

export const fileListMappingSpec: NodeMappingSpec = {
  nodeType: 'x/fileList',
  fields: [
    { inputKey: 'path', dslKey: 'path', valueType: 'template', defaultValue: '' },
    { inputKey: 'recursive', dslKey: 'recursive', valueType: 'boolean', defaultValue: false },
  ],
};

/** 兼容历史：configuration 曾误写入整份 node.data，仅含嵌套 inputsValues 时拆出 source。 */
export const jsonExtractMappingSpec: NodeMappingSpec = {
  nodeType: 'x/jsonExtract',
  transformIn: (cfg) => {
    const c = cfg as Record<string, unknown>;
    // 引擎与正确导出的 configuration 在顶层使用字符串字段；整份 node.data 误入时无 string source，需从 inputsValues 拆出。
    if (typeof c.source === 'string') return cfg;
    const iv = c.inputsValues as Record<string, unknown> | undefined;
    if (!iv || typeof iv !== 'object') return cfg;
    const pick = (v: unknown): unknown => {
      if (v != null && typeof v === 'object' && 'content' in (v as object)) {
        return (v as { content: unknown }).content;
      }
      return v;
    };
    return {
      source: pick(iv.source) ?? '',
    } as Record<string, unknown>;
  },
  fields: [
    { inputKey: 'source', dslKey: 'source', valueType: 'template', defaultValue: '' },
  ],
};

export const gitCloneMappingSpec: NodeMappingSpec = {
  nodeType: 'ci/gitClone',
  fields: [
    { inputKey: 'repository', dslKey: 'repository', valueType: 'template', defaultValue: '' },
    { inputKey: 'directory', dslKey: 'directory', valueType: 'template', defaultValue: '' },
    {
      inputKey: 'reference',
      dslKey: 'reference',
      valueType: 'constant',
      defaultValue: 'refs/heads/main',
    },
    { inputKey: 'authType', dslKey: 'authType', valueType: 'constant', defaultValue: 'token' },
    { inputKey: 'authUser', dslKey: 'authUser', valueType: 'constant', defaultValue: '' },
    { inputKey: 'authPassword', dslKey: 'authPassword', valueType: 'template', defaultValue: '' },
    { inputKey: 'authPemFile', dslKey: 'authPemFile', valueType: 'constant', defaultValue: '' },
    { inputKey: 'proxyUrl', dslKey: 'proxyUrl', valueType: 'constant', defaultValue: '' },
    { inputKey: 'proxyUsername', dslKey: 'proxyUsername', valueType: 'constant', defaultValue: '' },
    { inputKey: 'proxyPassword', dslKey: 'proxyPassword', valueType: 'constant', defaultValue: '' },
  ],
};

export const gitCommitMappingSpec: NodeMappingSpec = {
  nodeType: 'ci/gitCommit',
  fields: [
    { inputKey: 'directory', dslKey: 'directory', valueType: 'template', defaultValue: '' },
    { inputKey: 'pattern', dslKey: 'pattern', valueType: 'constant', defaultValue: '' },
    { inputKey: 'message', dslKey: 'message', valueType: 'template', defaultValue: '' },
    {
      inputKey: 'signature',
      dslKey: 'signature',
      valueType: 'json',
      defaultValue: { authorName: '', authorEmail: '' },
    },
  ],
};

export const gitPushMappingSpec: NodeMappingSpec = {
  nodeType: 'ci/gitPush',
  fields: [
    { inputKey: 'repository', dslKey: 'repository', valueType: 'template', defaultValue: '' },
    { inputKey: 'directory', dslKey: 'directory', valueType: 'template', defaultValue: '' },
    {
      inputKey: 'refSpecs',
      dslKey: 'refSpecs',
      valueType: 'constant',
      defaultValue: 'refs/heads/main:refs/heads/main',
    },
    { inputKey: 'authType', dslKey: 'authType', valueType: 'constant', defaultValue: 'token' },
    { inputKey: 'authUser', dslKey: 'authUser', valueType: 'constant', defaultValue: '' },
    { inputKey: 'authPassword', dslKey: 'authPassword', valueType: 'template', defaultValue: '' },
    { inputKey: 'authPemFile', dslKey: 'authPemFile', valueType: 'constant', defaultValue: '' },
    { inputKey: 'proxyUrl', dslKey: 'proxyUrl', valueType: 'constant', defaultValue: '' },
    { inputKey: 'proxyUsername', dslKey: 'proxyUsername', valueType: 'constant', defaultValue: '' },
    { inputKey: 'proxyPassword', dslKey: 'proxyPassword', valueType: 'constant', defaultValue: '' },
  ],
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
    {
      inputKey: 'outputFormat',
      dslKey: 'outputFormat',
      valueType: 'constant',
      defaultValue: 'text',
    },
    { inputKey: 'model', dslKey: 'model', valueType: 'template', defaultValue: '' },
    // 非空时插入 --workspace（仓库根 / 代码上下文）；与 workDir（进程 cwd）不同。
    { inputKey: 'workspacePath', dslKey: 'workspacePath', valueType: 'template', defaultValue: '' },
    // true 时插入 --worktree（无参数值），让 Agent 在新 Git worktree 中运行；可配合 --workspace 使用。
    { inputKey: 'worktree', dslKey: 'worktree', valueType: 'boolean', defaultValue: false },
    { inputKey: 'force', dslKey: 'force', valueType: 'boolean', defaultValue: true },
    { inputKey: 'log', dslKey: 'log', valueType: 'boolean', defaultValue: false },
    { inputKey: 'replaceData', dslKey: 'replaceData', valueType: 'boolean', defaultValue: true },
    { inputKey: 'workDir', dslKey: 'workDir', valueType: 'template', defaultValue: '' },
    { inputKey: 'timeoutMs', dslKey: 'timeoutMs', valueType: 'number', defaultValue: 0 },
  ],
  transformIn: transformCursorCliConfigIn,
};

export const cursorCliAuthMappingSpec: NodeMappingSpec = {
  nodeType: 'x/cursorCliAuth',
  fields: [
    { inputKey: 'agentPath', dslKey: 'agentPath', valueType: 'constant', defaultValue: 'agent' },
    { inputKey: 'workspacePath', dslKey: 'workspacePath', valueType: 'template', defaultValue: '$HOME' },
    { inputKey: 'worktree', dslKey: 'worktree', valueType: 'boolean', defaultValue: false },
    { inputKey: 'force', dslKey: 'force', valueType: 'boolean', defaultValue: true },
    { inputKey: 'workDir', dslKey: 'workDir', valueType: 'template', defaultValue: '' },
    { inputKey: 'timeoutMs', dslKey: 'timeoutMs', valueType: 'number', defaultValue: 15000 },
    { inputKey: 'replaceData', dslKey: 'replaceData', valueType: 'boolean', defaultValue: true },
  ],
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
    {
      inputKey: 'postSplitByLine',
      dslKey: 'postSplitByLine',
      valueType: 'boolean',
      defaultValue: false,
    },
    {
      inputKey: 'postAtAllBefore',
      dslKey: 'postAtAllBefore',
      valueType: 'boolean',
      defaultValue: false,
    },
    {
      inputKey: 'postAtAllAfter',
      dslKey: 'postAtAllAfter',
      valueType: 'boolean',
      defaultValue: false,
    },
    {
      inputKey: 'postMentionUserIds',
      dslKey: 'postMentionUserIds',
      valueType: 'json',
      defaultValue: [],
    },
    {
      inputKey: 'interactivePreset',
      dslKey: 'interactivePreset',
      valueType: 'constant',
      defaultValue: 'card_json',
    },
    {
      inputKey: 'cardNoticeTitle',
      dslKey: 'cardNoticeTitle',
      valueType: 'template',
      defaultValue: '',
    },
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

export const feishuCliAuthMappingSpec: NodeMappingSpec = {
  nodeType: 'x/feishuCliAuth',
  fields: [
    { inputKey: 'cliPath', dslKey: 'cliPath', valueType: 'constant', defaultValue: 'lark-cli' },
    { inputKey: 'args', dslKey: 'args', valueType: 'json', defaultValue: ['auth', 'status'] },
    { inputKey: 'workDir', dslKey: 'workDir', valueType: 'template', defaultValue: '' },
    { inputKey: 'timeoutMs', dslKey: 'timeoutMs', valueType: 'number', defaultValue: 15000 },
    { inputKey: 'replaceData', dslKey: 'replaceData', valueType: 'boolean', defaultValue: true },
  ],
};

export const workspaceSyncMappingSpec: NodeMappingSpec = {
  nodeType: 'x/workspaceSync',
  fields: [
    { inputKey: 'workspaceId', dslKey: 'workspaceId', valueType: 'constant', defaultValue: '' },
    { inputKey: 'replaceData', dslKey: 'replaceData', valueType: 'boolean', defaultValue: true },
  ],
};

export const apiRouteTracerSourcegraphMappingSpec: NodeMappingSpec = {
  nodeType: 'x/apiRouteTracerSourcegraph',
  fields: [
    { inputKey: 'endpoint', dslKey: 'endpoint', valueType: 'template', defaultValue: '' },
    { inputKey: 'accessToken', dslKey: 'accessToken', valueType: 'template', defaultValue: '' },
    { inputKey: 'timeoutSec', dslKey: 'timeoutSec', valueType: 'number', defaultValue: 30 },
    { inputKey: 'repoScope', dslKey: 'repoScope', valueType: 'constant', defaultValue: '' },
    { inputKey: 'repoFrontend', dslKey: 'repoFrontend', valueType: 'template', defaultValue: '' },
    { inputKey: 'repoBackend', dslKey: 'repoBackend', valueType: 'template', defaultValue: '' },
    {
      inputKey: 'contextGlobal',
      dslKey: 'contextGlobal',
      valueType: 'boolean',
      defaultValue: true,
    },
    { inputKey: 'typeFilter', dslKey: 'typeFilter', valueType: 'template', defaultValue: '' },
    { inputKey: 'displayLimit', dslKey: 'displayLimit', valueType: 'number', defaultValue: 1500 },
    {
      inputKey: 'defaultPatternType',
      dslKey: 'defaultPatternType',
      valueType: 'constant',
      defaultValue: 'literal',
    },
    {
      inputKey: 'defaultPatterns',
      dslKey: 'defaultPatterns',
      valueType: 'template',
      defaultValue: '',
    },
  ],
};

export const cursorAcpMappingSpec: NodeMappingSpec = {
  nodeType: 'x/cursorAcp',
  fields: [
    {
      inputKey: 'acpSimpleMode',
      dslKey: 'acpSimpleMode',
      valueType: 'boolean',
      defaultValue: true,
    },
    { inputKey: 'acpTask', dslKey: 'acpTask', valueType: 'template', defaultValue: '' },
    { inputKey: 'agentPath', dslKey: 'agentPath', valueType: 'constant', defaultValue: 'agent' },
    { inputKey: 'args', dslKey: 'args', valueType: 'json', defaultValue: ['acp'] },
    { inputKey: 'stdinLines', dslKey: 'stdinLines', valueType: 'json', defaultValue: [] },
    { inputKey: 'workspacePath', dslKey: 'workspacePath', valueType: 'template', defaultValue: '' },
    { inputKey: 'worktree', dslKey: 'worktree', valueType: 'boolean', defaultValue: false },
    { inputKey: 'force', dslKey: 'force', valueType: 'boolean', defaultValue: true },
    { inputKey: 'log', dslKey: 'log', valueType: 'boolean', defaultValue: false },
    { inputKey: 'replaceData', dslKey: 'replaceData', valueType: 'boolean', defaultValue: true },
    { inputKey: 'workDir', dslKey: 'workDir', valueType: 'template', defaultValue: '' },
    { inputKey: 'timeoutMs', dslKey: 'timeoutMs', valueType: 'number', defaultValue: 120000 },
  ],
};

const SPEC_BY_TYPE: Record<string, NodeMappingSpec> = {
  'ai/agentHarness': aiAgentHarnessMappingSpec,
  restApiCall: restApiCallMappingSpec,
  flow: flowMappingSpec,
  'transform/yapi': yapiMappingSpec,
  dbClient: dbClientMappingSpec,
  'x/redisClient': redisClientMappingSpec,
  'opensearch/search': opensearchSearchMappingSpec,
  'volcTls/searchLogs': volcTlsSearchLogsMappingSpec,
  'transform/multiNodeOutput': multiNodeOutputMappingSpec,
  switch: switchMappingSpec,
  inclusive: inclusiveMappingSpec,
  while: whileMappingSpec,
  exec: execMappingSpec,
  'x/fileRead': fileReadMappingSpec,
  'x/fileWrite': fileWriteMappingSpec,
  'x/fileDelete': fileDeleteMappingSpec,
  'x/fileList': fileListMappingSpec,
  'x/jsonExtract': jsonExtractMappingSpec,
  'ci/gitClone': gitCloneMappingSpec,
  'ci/gitCommit': gitCommitMappingSpec,
  'ci/gitPush': gitPushMappingSpec,
  jsTransform: jsTransformMappingSpec,
  log: logMappingSpec,
  jsFilter: jsFilterMappingSpec,
  luaTransform: luaTransformMappingSpec,
  'x/cursorCli': cursorCliMappingSpec,
  'x/cursorCliAuth': cursorCliAuthMappingSpec,
  'x/cursorAcp': cursorAcpMappingSpec,
  'x/feishuWebhook': feishuWebhookMappingSpec,
  'x/feishuCliAuth': feishuCliAuthMappingSpec,
  'x/workspaceSync': workspaceSyncMappingSpec,
  'x/apiRouteTracerSourcegraph': apiRouteTracerSourcegraphMappingSpec,
};

export function getNodeMappingSpec(nodeType: string): NodeMappingSpec | undefined {
  return SPEC_BY_TYPE[nodeType];
}
