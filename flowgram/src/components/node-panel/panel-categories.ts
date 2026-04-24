/**
 * 画布「添加节点」面板：按节点 type 推断分组；可在 FlowNodeMeta.panelCategory 覆盖。
 */

/** 分组排序（靠前先展示） */
export const PANEL_CATEGORY_ORDER: string[] = [
  'flow-base',
  'flow-control',
  'data-transform',
  'http-external',
  'ai-agent',
  'integration-x',
  'storage',
  'files',
  'exec-shell',
  'ci-cd',
  'schedule',
  'subflow',
  'debug',
  'other',
];

export const PANEL_CATEGORY_LABELS: Record<string, string> = {
  'flow-base': '流程与画布',
  'flow-control': '分支与循环',
  'data-transform': '数据转换',
  'http-external': 'HTTP',
  'ai-agent': 'AI / Agent',
  'integration-x': '扩展集成',
  storage: '数据库',
  files: '文件读写',
  'exec-shell': '命令与 Shell',
  'ci-cd': 'CI / Git',
  schedule: '调度',
  subflow: '子规则链',
  debug: '调试',
  other: '其他',
};

const FLOW_BASE = new Set(['start', 'end', 'comment', 'group']);
const FLOW_CONTROL = new Set([
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
]);
const DATA_TRANSFORM = new Set(['jsTransform', 'jsFilter', 'luaTransform', 'fetch-node-output']);

/** 根据节点 type 推断面板分组 id */
export function inferPanelCategoryKey(nodeType: string): string {
  const t = String(nodeType);
  if (t.startsWith('ai/')) return 'ai-agent';
  if (t.startsWith('transform/')) return 'data-transform';
  if (t.startsWith('ci/')) return 'ci-cd';
  if (t.startsWith('endpoint/')) return 'schedule';
  if (t === 'restApiCall') return 'http-external';
  if (t.startsWith('x/')) {
    if (/^x\/file/i.test(t)) return 'files';
    return 'integration-x';
  }
  if (t.startsWith('opensearch/') || t.startsWith('volcTls/')) return 'integration-x';
  if (FLOW_BASE.has(t)) return 'flow-base';
  if (FLOW_CONTROL.has(t)) return 'flow-control';
  if (DATA_TRANSFORM.has(t)) return 'data-transform';
  if (t === 'dbClient') return 'storage';
  if (t === 'log') return 'debug';
  if (t === 'exec') return 'exec-shell';
  if (t === 'flow') return 'subflow';
  return 'other';
}

export function panelCategorySortKey(id: string): number {
  const i = PANEL_CATEGORY_ORDER.indexOf(id);
  return i === -1 ? PANEL_CATEGORY_ORDER.length + 1 : i;
}
