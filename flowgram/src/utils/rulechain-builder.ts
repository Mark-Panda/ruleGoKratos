/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowDocument, WorkflowNodeEntity } from '@flowgram.ai/free-layout-editor';

import { FlowDocumentJSON, FlowNodeJSON } from '../typings/node';
import { WorkflowNodeType } from '../nodes/constants';
import { alphaNanoid } from './index';

export interface RuleChainBaseInfo {
  id: string;
  name: string;
  debugMode: boolean;
  root: boolean;
  disabled: boolean;
  configuration?: Record<string, any>;
  additionalInfo?: Record<string, any>;
}

interface RuleNodeRC {
  id: string;
  additionalInfo?: Record<string, any>;
  type: string;
  name: string;
  debugMode: boolean;
  configuration: Record<string, any>;
}

interface NodeConnectionRC {
  fromId: string;
  toId: string;
  type: string;
  label?: string;
}

interface FromDsl {
  path: string;
  configuration: Record<string, any>;
  processors: string[];
}

interface ToDsl {
  path: string;
  configuration: Record<string, any>;
  wait: boolean;
  processors: string[];
}

interface RouterDsl {
  id: string;
  params: any[];
  from: FromDsl;
  to: ToDsl;
  additionalInfo?: Record<string, any>;
}

type EndpointDsl = RuleNodeRC & {
  processors?: string[];
  routers?: RouterDsl[];
};

interface RuleMetadataRC {
  firstNodeIndex: number;
  endpoints?: EndpointDsl[];
  nodes: RuleNodeRC[];
  connections: NodeConnectionRC[];
  ruleChainConnections?: Array<{ fromId: string; toId: string; type: string }>;
}

interface RuleChainRC {
  ruleChain: RuleChainBaseInfo;
  metadata: RuleMetadataRC;
}

export function buildRuleChainJSONFromDocument(
  document: WorkflowDocument,
  baseOverride?: Partial<RuleChainBaseInfo>
): string {
  const raw = document.toJSON() as any;

  const flattened: any[] = Array.isArray(raw.nodes) ? raw.nodes.slice() : [];

  const connectionsRC: NodeConnectionRC[] = [];
  if (Array.isArray(raw.edges)) {
    for (const e of raw.edges as any[]) {
      const conn = buildRuleChainMetaConnection(e);
      if (conn) connectionsRC.push(conn);
    }
  }
  const endpointTypes = new Set(['endpoint/schedule']);
  const endpoiontsRc: EndpointDsl[] = flattened
    .filter((n: any) => endpointTypes.has(String(n.type)))
    .map((n: any) => {
      const nodeType = String(n.type);
      const base: EndpointDsl = {
        id: n.id,
        additionalInfo: n.meta ? { meta: n.meta } : undefined,
        type: nodeType,
        name: n.data?.title ?? nodeType,
        debugMode: false,
        configuration: {},
      };
      switch (nodeType) {
        case 'endpoint/schedule':
          if (n.data?.inputs && n.data?.inputsValues) {
            base.routers = [
              {
                id: alphaNanoid(16),
                params: [],
                from: {
                  path: n.data?.inputsValues.cron.content,
                  configuration: {},
                  processors: [],
                },
                to: {
                  path: baseOverride?.id + ':' + flattened[0].id,
                  configuration: {},
                  wait: false,
                  processors: [],
                },
              },
            ];
          }
          break;
        default:
          break;
      }
      return base;
    });

  const nodesRC: RuleNodeRC[] = [];
  for (const n of flattened) {
    buildRuleChainMetaNodes(n, nodesRC, connectionsRC);
  }

  const startIndex = nodesRC.findIndex((n) => n.type === 'start');
  const ruleChain: RuleChainRC = {
    ruleChain: {
      id: String(baseOverride?.id ?? raw.id ?? 'workflow'),
      name: String(baseOverride?.name ?? raw.name ?? 'Workflow'),
      debugMode: !!(baseOverride?.debugMode ?? false),
      root: !!(baseOverride?.root ?? true),
      disabled: !!(baseOverride?.disabled ?? false),
      configuration: { ...(baseOverride?.configuration ?? {}) },
      additionalInfo: { ...(baseOverride?.additionalInfo ?? {}) },
    },
    metadata: {
      firstNodeIndex: startIndex >= 0 ? startIndex : 0,
      endpoints: endpoiontsRc,
      nodes: nodesRC,
      connections: connectionsRC,
      ruleChainConnections: [],
    },
  };

  return JSON.stringify(ruleChain, null, 2);
}

function buildRuleChainMetaNodes(
  n: any,
  nodesRC: RuleNodeRC[],
  connectionsRC: NodeConnectionRC[]
): void {
  const nodeType = String(n.type);
  const base: RuleNodeRC = {
    id: n.id,
    additionalInfo: n.meta ? { meta: n.meta } : undefined,
    type: nodeType,
    name: n.data?.title ?? nodeType,
    debugMode: false,
    configuration: {
      ...(n.data ?? {}),
    },
  };
  switch (nodeType) {
    case 'endpoint/schedule':
    case 'block-start':
    case 'block-end':
      return;
    case 'group': {
      if (n.data) {
        base.configuration = { nodeIds: n.data?.blockIDs };
        base.type = 'groupAction';
      }
      break;
    }
    case 'for': {
      if (n.data) {
        base.configuration = {
          range: n.data?.note?.content,
          do: n.data?.nodeId?.content,
          mode: n.data?.operationMode?.content,
          extra: {
            blocks: [],
            edges: [],
          },
        };
      }
      if (n.blocks && n.blocks.length > 0) {
        const forBlocks: any[] = [];
        for (const b of n.blocks) {
          forBlocks.push(b);
          const t = String(b.type);
          if (t !== 'block-start' && t !== 'block-end') {
            buildRuleChainMetaNodes(b, nodesRC, connectionsRC);
          }
        }
        base.configuration.extra.blocks = forBlocks;
      }
      if (n.edges && n.edges.length > 0) {
        const forEdges: any = [];
        for (const e of n.edges) {
          const sourceId = e.sourceNodeID ?? '';
          const targetId = e.targetNodeID ?? '';
          forEdges.push(e);
          if (
            !String(sourceId).startsWith('block_start') &&
            !String(targetId).startsWith('block_end')
          ) {
            const connection = buildRuleChainMetaConnection(e);
            if (connection) {
              connectionsRC.push(connection);
            }
          }
        }
        base.configuration.extra.edges = forEdges;
      }
      break;
    }
    case 'restApiCall': {
      const newconfig: Record<string, any> = {};
      if (n.data?.api) {
        newconfig['requestMethod'] = n.data?.api.method;
        if (n.data?.api.url?.content) {
          newconfig['restEndpointUrlPattern'] = n.data?.api.url?.content;
        }
      }
      if (
        n.data?.headersValues &&
        Object.keys(n.data?.headersValues).length > 0 &&
        n.data?.headers &&
        Object.keys(n.data?.headers).length > 0
      ) {
        const headermap: Record<string, any> = {};
        for (const key of Object.keys(n.data.headers.properties)) {
          const v = (n.data.headersValues as any)[key];
          headermap[key] = v?.content;
        }
        newconfig['headers'] = headermap;
      }
      if (
        n.data?.paramsValues &&
        Object.keys(n.data?.paramsValues).length > 0 &&
        n.data?.params &&
        Object.keys(n.data?.params).length > 0
      ) {
        const parammap: Record<string, any> = {};
        for (const key of Object.keys(n.data.params.properties)) {
          const v = (n.data.paramsValues as any)[key];
          parammap[key] = v?.content;
        }
        newconfig['params'] = parammap;
        const urlPattern = newconfig['restEndpointUrlPattern'];
        if (urlPattern && typeof urlPattern === 'string' && Object.keys(parammap).length > 0) {
          const qs = Object.entries(parammap)
            .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v ?? ''))}`)
            .join('&');
          const sep = urlPattern.includes('?') ? '&' : '?';
          newconfig['restEndpointUrlPattern'] = qs ? `${urlPattern}${sep}${qs}` : urlPattern;
        }
      }
      if (n.data?.body && n.data?.body?.bodyType === 'JSON' && n.data?.body?.json?.content) {
        newconfig['body'] = n.data?.body.json.content;
      }
      if (n.data?.timeout) {
        newconfig['readTimeoutMs'] = n.data?.timeout.timeout;
      }
      base.configuration = newconfig;
      break;
    }
    case 'ai/llm': {
      if (
        n.data?.inputs &&
        Object.keys(n.data?.inputs).length > 0 &&
        n.data?.inputsValues &&
        Object.keys(n.data?.inputsValues).length > 0
      ) {
        const configmap: Record<string, any> = {};
        const parammap: Record<string, any> = {};
        for (const key of Object.keys(n.data.inputs.properties)) {
          switch (key) {
            case 'userPrompt':
              configmap['messages'] = [
                {
                  role: 'user',
                  content: (n.data.inputsValues as any)[key]?.content,
                },
              ];
              break;
            case 'temperature':
            case 'responseFormat':
            case 'topP':
            case 'maxTokens':
              parammap[key] = (n.data.inputsValues as any)[key]?.content;
              break;
            default:
              configmap[key] = (n.data.inputsValues as any)[key]?.content;
          }
        }
        configmap['params'] = parammap;
        base.configuration = configmap;
      }
      break;
    }
    case 'switch': {
      if (Array.isArray(n.data?.cases) && n.data.cases.length > 0) {
        const formatValue = (v: any) => {
          const val = v?.content;
          const isNum =
            typeof val === 'number' || (typeof val === 'string' && /^-?\d+(?:\.\d+)?$/.test(val));
          return isNum ? String(val) : JSON.stringify(String(val ?? ''));
        };
        const formatRow = (row: any) => {
          if (!row) return '';
          if (row.content && String(row.content).trim().length > 0) {
            return String(row.content).trim();
          }
          if (row.type === 'expression') {
            const left = row.left?.content ?? '';
            const op = row.operator ?? '';
            const right = formatValue(row.right ?? {});
            if (left && op && right) return `${left} ${op} ${right}`;
          }
          return '';
        };
        const formatGroup = (g: any) => {
          const rows = Array.isArray(g?.rows) ? g.rows : [];
          const exprs = rows.map(formatRow).filter((s: string) => s && s.length > 0);
          const joiner = g?.operator === 'or' ? ' || ' : ' && ';
          if (exprs.length === 0) return '';
          const joined = exprs.join(joiner);
          return exprs.length > 1 ? `(${joined})` : joined;
        };
        const cases = (n.data.cases as any[])
          .map((c: any) => {
            const groups = Array.isArray(c?.groups) ? c.groups : [];
            const groupExprs = groups.map(formatGroup).filter((s: string) => s && s.length > 0);
            const fullExpr = groupExprs.join(' || ');
            return { case: fullExpr, then: String(c.key ?? '') };
          })
          .filter((item) => item.case && item.then);
        base.configuration = { cases };
      }
      break;
    }
    case 'jsTransform':
    case 'log':
    case 'jsFilter': {
      if (n.data?.script) {
        const matchName =
          n.type === 'jsTransform'
            ? 'Transform'
            : n.type === 'log'
            ? 'ToString'
            : n.type === 'jsFilter'
            ? 'Filter'
            : '';
        const scriptText: string = String(n.data?.script?.content ?? '');
        const fnIdx = scriptText.indexOf(`function ${matchName}`);
        const braceStart = fnIdx >= 0 ? scriptText.indexOf('{', fnIdx) : -1;
        if (braceStart >= 0) {
          let i = braceStart + 1;
          let depth = 1;
          let end = scriptText.length;
          let inSingle = false;
          let inDouble = false;
          let inTemplate = false;
          let inLineComment = false;
          let inBlockComment = false;
          for (; i < scriptText.length; i++) {
            const ch = scriptText[i];
            const prev = scriptText[i - 1];
            if (inLineComment) {
              if (ch === '\n') inLineComment = false;
              continue;
            }
            if (inBlockComment) {
              if (ch === '*' && scriptText[i + 1] === '/') {
                inBlockComment = false;
                i++;
              }
              continue;
            }
            if (!inSingle && !inDouble && !inTemplate) {
              if (ch === '/' && scriptText[i + 1] === '/') {
                inLineComment = true;
                i++;
                continue;
              }
              if (ch === '/' && scriptText[i + 1] === '*') {
                inBlockComment = true;
                i++;
                continue;
              }
              if (ch === "'" && prev !== '\\') {
                inSingle = true;
                continue;
              }
              if (ch === '"' && prev !== '\\') {
                inDouble = true;
                continue;
              }
              if (ch === '`' && prev !== '\\') {
                inTemplate = true;
                continue;
              }
            } else {
              if (inSingle && ch === "'" && prev !== '\\') {
                inSingle = false;
                continue;
              }
              if (inDouble && ch === '"' && prev !== '\\') {
                inDouble = false;
                continue;
              }
              if (inTemplate && ch === '`' && prev !== '\\') {
                inTemplate = false;
                continue;
              }
              continue;
            }
            if (ch === '{') depth++;
            else if (ch === '}') {
              depth--;
              if (depth === 0) {
                end = i;
                break;
              }
            }
          }
          const body = scriptText.slice(braceStart + 1, end).trim();
          base.configuration = { jsScript: body };
        }
      }
      break;
    }
    case 'luaTransform': {
      if (n.data?.script) {
        const scriptText: string = String(n.data?.script?.content ?? '');
        const fnIdx = scriptText.indexOf('function Transform');
        if (fnIdx >= 0) {
          const lineEndIdx = scriptText.indexOf('\n', fnIdx);
          const endIdx = scriptText.lastIndexOf('end');
          const body = scriptText
            .slice(
              lineEndIdx >= 0 ? lineEndIdx + 1 : fnIdx,
              endIdx >= 0 ? endIdx : scriptText.length
            )
            .trim();
          base.configuration = { luaScript: body } as any;
        }
      }
      break;
    }
    case 'flow': {
      if (n.data?.inputs && n.data?.inputsValues) {
        const tId = n.data?.inputsValues?.targetId?.content;
        const ext = n.data?.inputsValues?.extend?.content;
        base.configuration = {
          targetId: String(tId ?? ''),
          extend: !!ext,
        } as any;
      }
      break;
    }
    case 'transform/yapi': {
      base.configuration = {
        ...(n.data?.yapiConfig ?? {}),
      };
      break;
    }
    default: {
      // 保持默认逻辑
      if (
        n.data?.inputs &&
        Object.keys(n.data?.inputs).length > 0 &&
        n.data?.inputsValues &&
        Object.keys(n.data?.inputsValues).length > 0
      ) {
        const parammap: Record<string, any> = {};
        for (const key of Object.keys(n.data.inputs.properties)) {
          const v = (n.data.inputsValues as any)[key];
          parammap[key] = v?.content;
        }
        base.configuration = parammap;
      }
      break;
    }
  }

  nodesRC.push(base);
  return;
}

function buildRuleChainMetaConnection(n: any): NodeConnectionRC | null {
  const sourceId = n.sourceNodeID ?? '';
  const targetId = n.targetNodeID ?? '';
  if (String(sourceId).startsWith('block_start') || String(targetId).startsWith('block_end')) {
    return null;
  }
  return {
    fromId: sourceId,
    toId: targetId,
    type: n.sourcePortID ?? 'Success',
    label: n.sourcePortID ?? n.label,
  };
}

export function buildDocumentFromRuleChainJSON(raw: string | RuleChainRC): FlowDocumentJSON {
  const rc: RuleChainRC = typeof raw === 'string' ? (JSON.parse(raw) as any) : (raw as any);
  const spacingX = 440;
  const spacingY = 180;
  const startX = 180;
  const startY = 180;
  const rcNodes: any[] = Array.isArray(rc?.metadata?.nodes) ? (rc as any).metadata.nodes : [];
  const rcConns: any[] = Array.isArray(rc?.metadata?.connections)
    ? (rc as any).metadata.connections
    : [];

  const nestedChildIds = new Set<string>();
  rcNodes.forEach((n: any) => {
    const extra = (n?.configuration as any)?.extra;
    if (extra && Array.isArray(extra.blocks)) {
      for (const b of extra.blocks) nestedChildIds.add(String(b.id));
    }
  });
  const ids = rcNodes.map((n: any) => String(n.id)).filter((id) => !nestedChildIds.has(id));
  const adjacency = new Map<string, string[]>();
  const indegree = new Map<string, number>();
  ids.forEach((id) => {
    adjacency.set(id, []);
    indegree.set(id, 0);
  });
  const topLevelConns = rcConns.filter((c: any) => {
    const fromId = String(c.fromId ?? c.from?.id ?? '');
    const toId = String(c.toId ?? c.to?.id ?? '');
    return !nestedChildIds.has(fromId) && !nestedChildIds.has(toId);
  });
  for (const c of topLevelConns) {
    const fromId = String(c.fromId ?? c.from?.id ?? '');
    const toId = String(c.toId ?? c.to?.id ?? '');
    if (!fromId || !toId) continue;
    if (!adjacency.has(fromId)) adjacency.set(fromId, []);
    adjacency.get(fromId)!.push(toId);
    indegree.set(toId, (indegree.get(toId) ?? 0) + 1);
  }

  let rootIds: string[] = [];
  const firstIdx = (rc as any)?.metadata?.firstNodeIndex;
  if (typeof firstIdx === 'number' && rcNodes[firstIdx]) {
    rootIds = [String(rcNodes[firstIdx].id)];
  } else {
    rootIds = ids.filter((id) => (indegree.get(id) ?? 0) === 0);
    if (rootIds.length === 0 && ids.length > 0) rootIds = [ids[0]];
  }

  const level: Record<string, number> = {};
  ids.forEach((id) => (level[id] = 0));
  const visited = new Set<string>();
  const queue: string[] = [];
  for (const r of rootIds) {
    level[r] = 0;
    queue.push(r);
    visited.add(r);
  }
  while (queue.length) {
    const curr = queue.shift()!;
    const nexts = adjacency.get(curr) ?? [];
    for (const nb of nexts) {
      const nextLevel = level[curr] + 1;
      if (level[nb] < nextLevel) level[nb] = nextLevel;
      if (!visited.has(nb)) {
        visited.add(nb);
        queue.push(nb);
      }
    }
  }
  const maxLevel = Math.max(0, ...Object.values(level));
  ids.forEach((id) => {
    if (!visited.has(id)) level[id] = maxLevel + 1;
  });
  const buckets = new Map<number, string[]>();
  ids.forEach((id) => {
    const lv = level[id];
    if (!buckets.has(lv)) buckets.set(lv, []);
    buckets.get(lv)!.push(id);
  });
  const reverseAdjacency = new Map<string, string[]>();
  ids.forEach((id) => reverseAdjacency.set(id, []));
  for (const [from, tos] of adjacency.entries()) {
    for (const to of tos) {
      if (!reverseAdjacency.has(to)) reverseAdjacency.set(to, []);
      reverseAdjacency.get(to)!.push(from);
    }
  }
  const sortByBarycenter = (
    layerIds: string[],
    neighborPos: Map<string, number>,
    getNeighbors: (id: string) => string[]
  ) => {
    const originalIndex = new Map<string, number>();
    layerIds.forEach((id, i) => originalIndex.set(id, i));
    return [...layerIds]
      .map((id) => {
        const ns = (getNeighbors(id) || []).filter((n) => neighborPos.has(n));
        if (ns.length === 0) {
          return { id, bc: originalIndex.get(id) ?? 0 };
        }
        const bc = ns.reduce((sum, n) => sum + (neighborPos.get(n) ?? 0), 0) / ns.length;
        return { id, bc };
      })
      .sort((a, b) => a.bc - b.bc)
      .map((x) => x.id);
  };
  const layerKeys = Array.from(buckets.keys()).sort((a, b) => a - b);
  for (let i = 1; i < layerKeys.length; i++) {
    const prevLayer = buckets.get(layerKeys[i - 1]) ?? [];
    const currLayer = buckets.get(layerKeys[i]) ?? [];
    const posPrev = new Map<string, number>();
    prevLayer.forEach((id, idx) => posPrev.set(id, idx));
    const reordered = sortByBarycenter(currLayer, posPrev, (id) => reverseAdjacency.get(id) ?? []);
    buckets.set(layerKeys[i], reordered);
  }
  for (let i = layerKeys.length - 2; i >= 0; i--) {
    const nextLayer = buckets.get(layerKeys[i + 1]) ?? [];
    const currLayer = buckets.get(layerKeys[i]) ?? [];
    const posNext = new Map<string, number>();
    nextLayer.forEach((id, idx) => posNext.set(id, idx));
    const reordered = sortByBarycenter(currLayer, posNext, (id) => adjacency.get(id) ?? []);
    buckets.set(layerKeys[i], reordered);
  }

  const nodeById = new Map<string, any>();
  rcNodes.forEach((n) => nodeById.set(String(n.id), n));
  const nodes: FlowNodeJSON[] = ids
    .map((id) => {
      const n = nodeById.get(id) ?? {};
      const lv = level[id] ?? 0;
      const layerNodes = buckets.get(lv) ?? [];
      const idxInLayer = layerNodes.indexOf(id);
      const fallbackX = startX + lv * spacingX;
      const fallbackY = startY + (idxInLayer >= 0 ? idxInLayer : 0) * spacingY;
      const pos = (n?.additionalInfo?.meta?.position as any) || { x: fallbackX, y: fallbackY };
      const t = String(n.type ?? 'default');
      if (t === 'groupAction') return null as any;
      const base: any = {
        id,
        type: t,
        meta: { position: { x: pos.x, y: pos.y } },
        data: { title: n.name ?? String(t) },
      };
      switch (t) {
        case 'start': {
          base.data = {
            title: n.name ?? 'Start',
          };
          break;
        }
        case 'x/redisClient': {
          const cfg = n.configuration ?? {};
          base.data = {
            title: n.name ?? 'Redis 客户端',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['server', 'cmd'],
              properties: {
                server: {
                  type: 'string',
                  extra: {
                    label: '服务器地址',
                    description: '示例：redis://host:port 或 host:port',
                  },
                },
                password: { type: 'string', extra: { label: '密码' } },
                poolSize: {
                  type: 'number',
                  extra: {
                    label: '连接池大小',
                    description: '并发量大时适当增大；留空或 0 使用默认值',
                  },
                },
                db: { type: 'number', extra: { label: '数据库编号', description: '默认为 0' } },
                cmd: {
                  type: 'string',
                  extra: {
                    label: '命令',
                    formComponent: 'prompt-editor',
                    description: '支持使用 ${metadata.key} 或 ${msg.key} 变量进行模板插值',
                  },
                },
                params: {
                  type: 'array',
                  items: { type: 'string' },
                  extra: {
                    label: '参数列表',
                    formComponent: 'array-editor',
                    description:
                      '按顺序传入命令参数；支持使用 ${metadata.key} 或 ${msg.key} 变量进行替换',
                  },
                },
              },
            },
            inputsValues: {
              server: { type: 'constant', content: String((cfg as any).server ?? '') },
              password: { type: 'constant', content: String((cfg as any).password ?? '') },
              poolSize: { type: 'constant', content: Number((cfg as any).poolSize ?? 0) },
              db: { type: 'constant', content: Number((cfg as any).db ?? 0) },
              cmd: { type: 'template', content: String((cfg as any).cmd ?? '') },
              params: {
                type: 'constant',
                content: Array.isArray((cfg as any).params) ? (cfg as any).params : [],
              },
            },
          } as any;
          break;
        }
        case 'restApiCall': {
          const cfg = n.configuration ?? {};
          const fullUrl = String(cfg.restEndpointUrlPattern ?? '');
          let baseUrl = fullUrl;
          const queryValues: Record<string, any> = {};
          const qm = fullUrl.indexOf('?');
          if (qm >= 0) {
            baseUrl = fullUrl.slice(0, qm);
            const qs = fullUrl.slice(qm + 1);
            qs.split('&').forEach((pair) => {
              if (!pair) return;
              const [rawK, rawV] = pair.split('=');
              if (!rawK) return;
              const k = decodeURIComponent(rawK);
              const v = rawV !== undefined ? decodeURIComponent(rawV) : '';
              queryValues[k] = v;
            });
          }
          const headerVals = Object.keys(cfg.headers || {}).reduce((acc: any, k) => {
            acc[k] = { type: 'constant', content: (cfg.headers as any)[k] };
            return acc;
          }, {});
          const paramValsFromCfg = Object.keys(cfg.params || {}).reduce((acc: any, k) => {
            acc[k] = { type: 'constant', content: (cfg.params as any)[k] };
            return acc;
          }, {});
          const paramValsFromUrl = Object.keys(queryValues).reduce((acc: any, k) => {
            acc[k] = { type: 'constant', content: queryValues[k] };
            return acc;
          }, {});
          const mergedParamVals = { ...paramValsFromUrl, ...paramValsFromCfg };

          base.data = {
            title: n.name ?? 'restApiCall',
            positionType: 'middle',
            api: {
              method: cfg.requestMethod ?? 'GET',
              url: baseUrl ? { type: 'template', content: baseUrl } : undefined,
            },
            headers: {},
            headersValues: headerVals,
            params: {},
            paramsValues: mergedParamVals,
            body: {
              bodyType: 'JSON',
              json: cfg.body ? { type: 'template', content: cfg.body } : undefined,
            },
            timeout: { retryTimes: 0, timeout: cfg.readTimeoutMs ?? 0 },
          };
          break;
        }
        case 'ai/llm': {
          const cfg = n.configuration ?? {};
          const msg = Array.isArray(cfg.messages) ? cfg.messages[0]?.content : '';
          const params = cfg.params ?? {};
          base.data = {
            title: n.name ?? 'ai/llm',
            positionType: 'middle',
            inputsValues: {
              model: { type: 'constant', content: String(cfg.model ?? '') },
              key: { type: 'constant', content: String(cfg.key ?? '') },
              url: { type: 'constant', content: String(cfg.url ?? '') },
              systemPrompt: { type: 'template', content: String(cfg.systemPrompt ?? '') },
              userPrompt: { type: 'template', content: String(msg ?? '') },
              temperature: { type: 'constant', content: params.temperature ?? 0.5 },
              topP: { type: 'constant', content: params.topP ?? 0.5 },
              maxTokens: { type: 'constant', content: params.maxTokens ?? 0 },
              responseFormat: { type: 'constant', content: params.responseFormat ?? 'text' },
            },
            inputs: {
              type: 'object',
              required: [
                'model',
                'key',
                'url',
                'temperature',
                'userPrompt',
                'topP',
                'maxTokens',
                'responseFormat',
              ],
              properties: {
                model: { type: 'string', extra: { label: '模型名称' } },
                key: { type: 'string' },
                url: { type: 'string' },
                systemPrompt: {
                  type: 'string',
                  extra: { label: '系统提示词', formComponent: 'prompt-editor' },
                },
                userPrompt: {
                  type: 'string',
                  extra: { label: '用户提示词', formComponent: 'prompt-editor' },
                },
                maxTokens: { type: 'number', extra: { label: '最大输出长度' } },
                responseFormat: {
                  type: 'string',
                  enum: ['text', 'json_object', 'json_schema'],
                  extra: { label: '输出格式', formComponent: 'enum-select' },
                },
                temperature: { type: 'number' },
                topP: { type: 'number' },
              },
            },
            outputs: { type: 'object', properties: {} },
          };
          break;
        }
        case 'jsTransform': {
          const body = String((n.configuration ?? {}).jsScript ?? '');
          base.data = {
            title: n.name ?? 'jsTransform',
            positionType: 'middle',
            script: {
              language: 'javascript',
              content: `// 函数签名不可修改\nasync function Transform(msg, metadata, msgType, dataType) {\n${body}\n}`,
            },
          };
          break;
        }
        case 'dbClient': {
          const cfg = n.configuration ?? {};
          base.data = {
            title: n.name ?? 'dbClient',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['sql', 'getOne', 'poolSize', 'driverName', 'dsn'],
              properties: {
                driverName: {
                  type: 'string',
                  enum: ['mysql', 'postgres'],
                  extra: { label: '数据库驱动名称', formComponent: 'enum-select' },
                },
                dsn: {
                  type: 'string',
                  extra: {
                    description:
                      '数据库连接字符串，支持模板变量。示例：mysql 使用 user:pass@tcp(host:port)/db?charset=utf8mb4；postgres 使用 postgres://user:pass@host:port/db?sslmode=disable',
                  },
                },
                sql: {
                  type: 'string',
                  extra: {
                    label: 'sql',
                    formComponent: 'sql-editor',
                    description:
                      '可以使用 ${metadata.key} 或者 ${msg.key}变量，SQL参数允许使用 ? 占位符',
                  },
                },
                params: {
                  type: 'array',
                  items: {
                    type: 'string',
                  },
                  extra: {
                    label: '参数列表',
                    description:
                      '可以使用 ${metadata.key} 读取元数据中的变量或者使用 ${msg.key} 读取消息负荷中的变量进行替换',
                    formComponent: 'array-editor',
                  },
                },
                getOne: {
                  type: 'boolean',
                  extra: {
                    label: '是否仅返回一条记录',
                    description: '开启后仅返回第一条记录；关闭返回全部记录',
                  },
                },
                poolSize: {
                  type: 'number',
                  extra: {
                    label: '连接池大小',
                    description: '并发量大时适当增大；留空或 0 使用默认值',
                  },
                },
              },
            },
            inputsValues: {
              sql: { type: 'template', content: String((cfg as any).sql ?? '') },
              params: {
                type: 'constant',
                content: Array.isArray((cfg as any).params) ? (cfg as any).params : [],
              },
              getOne: { type: 'constant', content: !!(cfg as any).getOne },
              poolSize: { type: 'constant', content: Number((cfg as any).poolSize ?? 0) },
              driverName: { type: 'constant', content: String((cfg as any).driverName ?? 'mysql') },
              dsn: { type: 'template', content: String((cfg as any).dsn ?? '') },
            },
          } as any;
          break;
        }
        case 'luaTransform': {
          const body = String((n.configuration ?? {}).luaScript ?? '');
          base.data = {
            title: n.name ?? 'luaTransform',
            positionType: 'middle',
            script: {
              language: 'lua',
              content: `-- 函数签名不可修改\nfunction Transform(msg, metadata, msgType, dataType)\n${body}\nend`,
            },
          };
          break;
        }
        case 'log': {
          const body = String((n.configuration ?? {}).jsScript ?? '');
          base.data = {
            title: n.name ?? 'log',
            positionType: 'middle',
            script: {
              language: 'javascript',
              content: `// 函数签名不可修改\nasync function ToString(msg, metadata, msgType, dataType) {\n${body}\n}`,
            },
          };
          break;
        }
        case 'jsFilter': {
          const body = String((n.configuration ?? {}).jsScript ?? '');
          base.data = {
            title: n.name ?? 'jsFilter',
            positionType: 'middle',
            script: {
              language: 'javascript',
              content: `// 函数签名不可修改\nasync function Filter(msg, metadata, msgType, dataType) {\n${body}\n}`,
            },
          };
          break;
        }
        case 'switch': {
          const cfg = n.configuration ?? {};
          const cases = Array.isArray(cfg.cases) ? cfg.cases : [];
          const splitTopLevel = (expr: string, delim: '||' | '&&') => {
            const parts: string[] = [];
            let buf = '';
            let depth = 0;
            let inSingle = false;
            let inDouble = false;
            let inTemplate = false;
            for (let i = 0; i < expr.length; i++) {
              const ch = expr[i];
              const prev = expr[i - 1];
              if (!inSingle && !inDouble && !inTemplate) {
                if (ch === '(') depth++;
                else if (ch === ')') depth = Math.max(0, depth - 1);
                else if (ch === "'" && prev !== '\\') inSingle = true;
                else if (ch === '"' && prev !== '\\') inDouble = true;
                else if (ch === '`' && prev !== '\\') inTemplate = true;
                const isDelim =
                  delim === '||' ? expr.slice(i, i + 2) === '||' : expr.slice(i, i + 2) === '&&';
                if (isDelim && depth === 0) {
                  parts.push(buf.trim());
                  buf = '';
                  i++;
                  continue;
                }
              } else {
                if (inSingle && ch === "'" && prev !== '\\') inSingle = false;
                else if (inDouble && ch === '"' && prev !== '\\') inDouble = false;
                else if (inTemplate && ch === '`' && prev !== '\\') inTemplate = false;
              }
              buf += ch;
            }
            if (buf.trim()) parts.push(buf.trim());
            return parts.filter((p) => p.length > 0);
          };
          const parseRow = (rowExpr: string) => {
            const expr = rowExpr.trim();
            const ops = ['contains', '==', '!=', '>=', '<=', '>', '<'];
            let foundOp = '';
            let left = '';
            let right = '';
            for (const op of ops) {
              const idx = expr.indexOf(op);
              if (idx > 0) {
                foundOp = op;
                left = expr.slice(0, idx).trim();
                right = expr.slice(idx + op.length).trim();
                break;
              }
            }
            if (!foundOp) return { type: 'expression', content: expr };
            const stripQuotes = (s: string) => {
              const t = s.trim();
              if (
                (t.startsWith("'") && t.endsWith("'")) ||
                (t.startsWith('"') && t.endsWith('"'))
              ) {
                return t.slice(1, -1);
              }
              return t;
            };
            return {
              type: 'expression',
              content: '',
              left: { type: 'constant', content: left },
              operator: foundOp,
              right: { type: 'constant', content: stripQuotes(right) },
            } as any;
          };
          base.data = {
            title: n.name ?? 'switch',
            positionType: 'middle',
            cases: cases.map((c: any) => {
              const expr = String(c.case ?? '');
              const groupsExpr = splitTopLevel(expr, '||');
              const groups = groupsExpr.map((ge) => {
                const rowsExpr = splitTopLevel(ge, '&&');
                return { operator: 'and', rows: rowsExpr.map(parseRow) };
              });
              return { key: String(c.then ?? ''), groups };
            }),
            ELSE: true,
          };
          break;
        }
        case 'flow': {
          const cfg = n.configuration ?? {};
          base.data = {
            title: n.name ?? 'flow',
            positionType: 'middle',
            inputsValues: {
              targetId: { type: 'constant', content: String(cfg.targetId ?? '') },
              extend: { type: 'constant', content: !!cfg.extend },
            },
          };
          break;
        }
        case 'transform/multiNodeOutput': {
          const cfg = n.configuration ?? {};
          const arr = Array.isArray((cfg as any).nodeIds) ? (cfg as any).nodeIds : [];
          base.data = {
            title: n.name ?? '获取多节点输出',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['nodeIds'],
              properties: {
                nodeIds: {
                  type: 'array',
                  items: { type: 'string' },
                  extra: { formComponent: 'node-selector-multi', label: '节点列表' },
                },
              },
            },
            inputsValues: {
              nodeIds: { type: 'constant', content: arr },
            },
            outputs: { type: 'object', properties: {} },
          } as any;
          break;
        }
        case 'for': {
          const cfg = n.configuration ?? {};
          base.data = {
            title: n.name ?? 'for',
            positionType: 'middle',
            note: { type: 'constant', content: String(cfg.range ?? '') },
            nodeId: { type: 'constant', content: String(cfg.do ?? '') },
            operationMode: { type: 'constant', content: Number(cfg.mode ?? 0) },
          };
          const extra = (cfg as any).extra ?? {};
          const blocks: any[] = Array.isArray(extra.blocks)
            ? extra.blocks
            : [
                {
                  id: `block_start_${Math.random().toString(36).slice(2, 7)}`,
                  type: 'block-start',
                  meta: { position: { x: 32, y: 0 } },
                  data: { positionType: 'middle' },
                },
                {
                  id: `block_end_${Math.random().toString(36).slice(2, 7)}`,
                  type: 'block-end',
                  meta: { position: { x: 192, y: 0 } },
                  data: { positionType: 'middle' },
                },
              ];
          const bs = blocks.find((b) => String(b.type) === 'block-start');
          const be = blocks.find((b) => String(b.type) === 'block-end');
          let innerEdges: any[] = Array.isArray(extra.edges) ? extra.edges : [];
          if (!innerEdges || innerEdges.length === 0) {
            const targetId = String(cfg.do ?? '') || String(base.data?.nodeId?.content ?? '');
            if (bs && targetId) {
              innerEdges = [
                { sourceNodeID: String(bs.id), targetNodeID: targetId },
                be ? { sourceNodeID: targetId, targetNodeID: String(be.id) } : undefined,
              ].filter(Boolean) as any[];
            }
          }
          base.blocks = blocks;
          base.edges = innerEdges;
          break;
        }
        case 'transform/yapi': {
          const cfg = n.configuration ?? {};
          base.data = {
            title: n.name ?? 'Yapi 接口',
            positionType: 'middle',
            yapiConfig: {
              baseUrl: String(cfg.baseUrl ?? ''),
              userName: String(cfg.userName ?? ''),
              password: String(cfg.password ?? ''),
              interfacePath: String(cfg.interfacePath ?? ''),
              loginType: String(cfg.loginType ?? 'ldap'),
            },
          };
          break;
        }
        default: {
          const cfg = n.configuration ?? {};
          const inputsValues = Object.keys(cfg).reduce((acc: any, k) => {
            acc[k] = { type: 'constant', content: (cfg as any)[k] };
            return acc;
          }, {});
          base.data = { title: n.name ?? t, inputsValues };
          break;
        }
      }
      return base as FlowNodeJSON;
    })
    .filter(Boolean) as FlowNodeJSON[];

  const edges = topLevelConns.map((e: any) => ({
    sourceNodeID: String(e.fromId ?? e.from?.id ?? ''),
    targetNodeID: String(e.toId ?? e.to?.id ?? ''),
    sourcePortID: e.type ?? e.label ?? undefined,
  }));

  const typeById = new Map<string, string>();
  nodes.forEach((n: any) => typeById.set(String(n.id), String(n.type)));
  edges.forEach((e: any) => {
    const srcType = typeById.get(String(e.sourceNodeID));
    const tgtType = typeById.get(String(e.targetNodeID));
    if (e.sourcePortID === 'Success' && srcType === 'for' && tgtType === 'log') {
      delete e.sourcePortID;
    }
  });

  const endpoints: any[] = Array.isArray(rc?.metadata?.endpoints)
    ? (rc as any).metadata.endpoints
    : [];
  for (const ep of endpoints) {
    if (String(ep.type) === 'endpoint/schedule') {
      const cron = ep?.routers?.[0]?.from?.path ?? '';
      const toPath = ep?.routers?.[0]?.to?.path ?? '';
      const pos = (ep?.additionalInfo as any)?.meta?.position;
      const x = typeof pos?.x === 'number' ? pos.x : startX - spacingX;
      const y = typeof pos?.y === 'number' ? pos.y : startY;
      const cronNode: any = {
        id: String(ep.id ?? `cron_${Math.random().toString(36).slice(2, 8)}`),
        type: 'endpoint/schedule',
        meta: { position: { x, y } },
        data: {
          title: ep.name ?? '定时任务',
          positionType: 'header',
          inputsValues: {
            cron: { type: 'constant', content: String(cron ?? '*/10 * * * * *') },
          },
          inputs: {
            type: 'object',
            required: ['cron'],
            properties: {
              cron: {
                type: 'string',
                extra: {
                  label: 'Cron 表达式',
                  description: '支持秒级（六位）Quartz 表达式，例如：*/10 * * * * *',
                  formComponent: 'cron-editor',
                },
              },
            },
          },
        },
      };
      nodes.unshift(cronNode);
      const targetId =
        typeof toPath === 'string' && toPath.includes(':') ? toPath.split(':')[1] : undefined;
      const exists = edges.some(
        (e: any) => e.sourceNodeID === String(cronNode.id) && e.targetNodeID === String(targetId)
      );
      if (targetId && targetId !== String(cronNode.id) && !exists) {
        edges.unshift({
          sourceNodeID: String(cronNode.id),
          sourcePortID: 'Success',
          targetNodeID: String(targetId),
          // 默认端口不写入，保持与示例 b.json 一致
          // sourcePortID: 'Success',
        });
      }
    }
  }

  const globalVariable = {
    type: 'object',
    required: [],
    properties: { userId: { type: 'string' } },
  } as any;

  return { nodes, edges, globalVariable } as any;
}
