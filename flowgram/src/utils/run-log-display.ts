/**
 * ListRunLogs 返回的 ruleChain 为 RuleGo RuleChain DSL（含嵌套 ruleChain / metadata），
 * 工作流展示名与规则链 ID 在内层 ruleChain.ruleChain。
 */
export function runLogChainDisplay(row: any): { id: string; name: string } {
  const pack = row?.ruleChain;
  const inner = pack?.ruleChain;
  if (inner != null && typeof inner === 'object') {
    return {
      id: String((inner as { id?: unknown }).id ?? ''),
      name: String((inner as { name?: unknown }).name ?? ''),
    };
  }
  return {
    id: String(pack?.id ?? ''),
    name: String(pack?.name ?? ''),
  };
}

/** 接口返回的时间戳可能是 number 或 JSON 序列化后的 string */
export function coerceRunLogTs(v: unknown): number {
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string') {
    const n = Number(v);
    return Number.isFinite(n) ? n : 0;
  }
  return 0;
}

export type ParsedRunLogPayload = {
  id?: string;
  logs: unknown[];
  startTs: number;
  endTs: number;
  metadata: unknown;
  ruleChain: unknown;
};

/** 规范化 GetRunLogByMsgId / ListRunLogs 单条的 JSON（兼容字段名差异） */
export function parseRunLogPayload(raw: unknown): ParsedRunLogPayload {
  if (!raw || typeof raw !== 'object') {
    return { logs: [], startTs: 0, endTs: 0, metadata: undefined, ruleChain: undefined };
  }
  const o = raw as Record<string, unknown>;
  const logsRaw = o.logs ?? (o as { Logs?: unknown }).Logs;
  const logs = Array.isArray(logsRaw) ? logsRaw : [];
  const idVal = o.id ?? (o as { Id?: unknown }).Id;
  const meta = o.metadata ?? (o as { Metadata?: unknown }).Metadata;
  const rc = o.ruleChain ?? (o as { RuleChain?: unknown }).RuleChain;
  return {
    id: idVal != null ? String(idVal) : undefined,
    logs,
    startTs: coerceRunLogTs(o.startTs ?? (o as { start_ts?: unknown }).start_ts),
    endTs: coerceRunLogTs(o.endTs ?? (o as { end_ts?: unknown }).end_ts),
    metadata: meta,
    ruleChain: rc,
  };
}

export function runLogRowsFromPayload(logs: unknown[]): Array<Record<string, unknown>> {
  return logs.map((item, idx) => {
    const row =
      item && typeof item === 'object' && !Array.isArray(item)
        ? (item as Record<string, unknown>)
        : {};
    const nid = String(row.nodeId ?? (row as { Id?: unknown }).Id ?? '');
    return {
      ...row,
      _idx: idx,
      _rowKey: `${idx}-${nid || idx}`,
    };
  });
}

function jsonCompact(v: unknown): string {
  if (typeof v === 'string') return v;
  try {
    return JSON.stringify(v);
  } catch {
    return String(v);
  }
}

/** RuleGo RuleNodeRunLog.inMsg/outMsg（RuleMsg）摘要，用于表格列 */
export function summarizeRuleMsgLike(v: unknown, max = 160): string {
  if (v === undefined || v === null) return '—';
  if (typeof v === 'string') {
    const t = v.trim();
    return t ? truncateText(t, max) : '—';
  }
  if (typeof v !== 'object' || Array.isArray(v)) {
    return truncateText(jsonCompact(v), max);
  }
  const o = v as Record<string, unknown>;
  const typ = typeof o.type === 'string' ? o.type : '';
  let dataPayload: unknown = o.data;
  if (
    dataPayload &&
    typeof dataPayload === 'object' &&
    !Array.isArray(dataPayload) &&
    typeof (dataPayload as Record<string, unknown>).data === 'string'
  ) {
    dataPayload = (dataPayload as Record<string, unknown>).data;
  }
  let body = '';
  if (typeof dataPayload === 'string') {
    body = dataPayload.trim();
  } else if (dataPayload !== undefined && dataPayload !== null) {
    body = jsonCompact(dataPayload);
  } else {
    body = jsonCompact(o);
  }
  const prefix = typ ? `[${typ}] ` : '';
  const combined = `${prefix}${body}`.trim();
  return combined ? truncateText(combined, Math.max(max, prefix.length + 40)) : '—';
}

export type CanvasNodeMaps = {
  /** nodeId -> 画布标题（优先 flowgramUI node.data.title） */
  labels: Map<string, string>;
  /** nodeId -> DSL 节点类型（flowgramUI node.type） */
  types: Map<string, string>;
};

function mergeNodeEntry(
  labels: Map<string, string>,
  types: Map<string, string>,
  id: string,
  label: string,
  nodeType: string,
  overwriteLabel: boolean
): void {
  if (!id) return;
  if (overwriteLabel || !labels.has(id)) {
    const t = label.trim();
    if (t) labels.set(id, t);
  }
  if (nodeType && (overwriteLabel || !types.has(id))) types.set(id, nodeType);
}

/** 递归收集子画布 blocks 内节点 */
function walkFlowgramNodes(
  nodes: unknown,
  labels: Map<string, string>,
  types: Map<string, string>,
  overwrite: boolean
): void {
  if (!Array.isArray(nodes)) return;
  for (const n of nodes) {
    if (!n || typeof n !== 'object' || Array.isArray(n)) continue;
    const raw = n as Record<string, unknown>;
    const id = String(raw.id ?? '');
    const nodeType = String(raw.type ?? '');
    const title =
      raw.data && typeof raw.data === 'object' && !Array.isArray(raw.data)
        ? String((raw.data as Record<string, unknown>).title ?? '').trim()
        : '';
    const label = title || id;
    mergeNodeEntry(labels, types, id, label, nodeType, overwrite);
    const blocks = raw.blocks;
    if (Array.isArray(blocks)) walkFlowgramNodes(blocks, labels, types, overwrite);
  }
}

/** 从「流程详情」metadata.flowgramUI / metadata.nodes 推导节点展示名（与画布一致） */
export function buildCanvasNodeMapsFromRuleDetail(detail: unknown): CanvasNodeMaps {
  const labels = new Map<string, string>();
  const types = new Map<string, string>();
  if (!detail || typeof detail !== 'object') return { labels, types };

  const md = (detail as { metadata?: unknown }).metadata;
  if (!md || typeof md !== 'object' || Array.isArray(md)) return { labels, types };

  const meta = md as Record<string, unknown>;

  const ui = meta.flowgramUI;
  if (ui && typeof ui === 'object' && !Array.isArray(ui)) {
    const nodes = (ui as Record<string, unknown>).nodes;
    walkFlowgramNodes(nodes, labels, types, true);
  }

  const dslNodes = meta.nodes;
  if (Array.isArray(dslNodes)) {
    for (const n of dslNodes) {
      if (!n || typeof n !== 'object' || Array.isArray(n)) continue;
      const raw = n as Record<string, unknown>;
      const id = String(raw.id ?? '');
      const nodeType = String(raw.type ?? '');
      const name = String(raw.name ?? '').trim();
      mergeNodeEntry(labels, types, id, name || id, nodeType, false);
    }
  }

  return { labels, types };
}

export function formatNodeCellDisplay(
  nodeId: string,
  maps: CanvasNodeMaps
): { titleLine: string; subLine: string } {
  const id = nodeId.trim();
  const title = maps.labels.get(id);
  const nt = maps.types.get(id);
  const titleLine = title && title !== id ? title : id ? id : '—';
  const subLine = [id ? `ID ${id}` : '', nt ? nt : ''].filter(Boolean).join(' · ');
  return { titleLine, subLine: subLine || '—' };
}

export function summarizeRunLogErr(logs: unknown[]): boolean {
  if (!Array.isArray(logs)) return false;
  return logs.some((item) => {
    const e =
      item && typeof item === 'object' && !Array.isArray(item)
        ? (item as Record<string, unknown>).err
        : undefined;
    return typeof e === 'string' && e.trim().length > 0;
  });
}

export function truncateText(s: string, max = 160): string {
  if (s.length <= max) return s;
  return `${s.slice(0, max)}…`;
}
