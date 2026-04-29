/**
 * 画布「添加节点」面板：按节点功能推断分组；可在 FlowNodeMeta.panelCategory 覆盖。
 */

/** 分组排序（靠前先展示） */
export const PANEL_CATEGORY_ORDER: string[] = [
  'scheduling',
  'utilities',
  'ai-capability',
  'service-calls',
  'data-processing',
  'flow-control',
  'project-mgmt',
  'data-storage',
  'file-ops',
  'command-dev',
  'other',
];

export const PANEL_CATEGORY_LABELS: Record<string, string> = {
  'flow-control': '流程控制',
  'data-processing': '数据处理',
  'service-calls': '服务调用',
  'data-storage': '数据存储与检索',
  'ai-capability': 'AI 能力',
  'file-ops': '文件操作',
  'command-dev': '命令与开发',
  scheduling: '调度触发',
  'project-mgmt': '项目管理',
  utilities: '辅助与调试',
  other: '其他',
};

/** 调度触发：定时触发、规则链入口 */
const SCHEDULING = new Set(['start']);

/** 流程控制：控制规则链执行流程 */
const FLOW_CONTROL = new Set([
  'end',
  'condition',
  'switch',
  'inclusive',
  'loop',
  'for',
  'while',
  'fork',
  'join',
  'break',
  'block-start',
  'block-end',
  'flow',
]);

/** 数据处理：数据转换、过滤、提取 */
const DATA_PROCESSING = new Set([
  'jsTransform',
  'jsFilter',
  'luaTransform',
  'fetch-node-output',
  'x/jsonExtract',
]);

/** 服务调用：调用外部服务、API、通知 */
const SERVICE_CALLS = new Set([
  'restApiCall',
  'x/feishuWebhook',
  'x/cursorCli',
  'x/cursorAcp',
  'x/cursorCliAuth',
  'x/feishuCliAuth',
]);

/** 数据存储与检索：数据库、缓存、搜索引擎 */
const DATA_STORAGE = new Set([
  'dbClient',
  'x/redisClient',
  'opensearch/search',
  'volcTls/searchLogs',
]);

/** 文件操作：文件读写 */
const FILE_OPS = new Set([
  'x/fileRead',
  'x/fileWrite',
  'x/fileDelete',
  'x/fileList',
]);

/** 命令与开发：命令执行、Git 操作 */
const COMMAND_DEV = new Set([
  'exec',
  'ci/gitClone',
  'ci/gitCommit',
  'ci/gitPush',
]);

/** 项目管理：任务看板、服务管理、工作区 */
const PROJECT_MGMT = new Set([
  'x/taskBoard',
  'x/serviceManagement',
  'x/workspaceSync',
]);

/** 辅助与调试：注释、日志、分组容器 */
const UTILITIES = new Set(['comment', 'log', 'group']);

/** 根据节点 type 推断面板分组 id */
export function inferPanelCategoryKey(nodeType: string): string {
  const t = String(nodeType);
  if (t.startsWith('ai/')) return 'ai-capability';
  if (t.startsWith('transform/')) return 'data-processing';
  if (t.startsWith('endpoint/')) return 'scheduling';
  if (SCHEDULING.has(t)) return 'scheduling';
  if (FLOW_CONTROL.has(t)) return 'flow-control';
  if (DATA_PROCESSING.has(t)) return 'data-processing';
  if (SERVICE_CALLS.has(t)) return 'service-calls';
  if (DATA_STORAGE.has(t)) return 'data-storage';
  if (FILE_OPS.has(t)) return 'file-ops';
  if (COMMAND_DEV.has(t)) return 'command-dev';
  if (PROJECT_MGMT.has(t)) return 'project-mgmt';
  if (UTILITIES.has(t)) return 'utilities';
  return 'other';
}

export function panelCategorySortKey(id: string): number {
  const i = PANEL_CATEGORY_ORDER.indexOf(id);
  return i === -1 ? PANEL_CATEGORY_ORDER.length + 1 : i;
}
