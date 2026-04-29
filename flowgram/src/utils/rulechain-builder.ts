/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { customAlphabet } from 'nanoid';
import type { WorkflowDocument } from '@flowgram.ai/free-layout-editor';

import type { FlowDocumentJSON, FlowNodeJSON } from '../typings/node';
import type { FlowValueLike, NodeMappingSpec } from './dsl-mapping/types';
import {
  buildForFlowFromDsl,
  buildScheduleEndpointsFromDocument,
  buildScheduleEndpointFlowNode,
  buildStartFlowData,
  emitForToRuleChain,
  emitGroupToRuleChain,
  shouldSkipRuleChainMetaNode,
} from './dsl-mapping/structure-engine';
import { getNodeMappingSpec } from './dsl-mapping/specs';
import {
  mapDslToNodeInputsValues,
  mapNodeToDslConfig,
  type InputsValuesMap,
} from './dsl-mapping/engine';

const alphaNanoid = (size: number) =>
  customAlphabet('0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ', size)();

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

function pickTargetPortIDForConnectionType(connType: unknown): string | undefined {
  const t = String(connType ?? '').trim().toLowerCase();
  if (t === 'failure' || t === 'false' || t === 'else') return 'input_top';
  return undefined;
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

/**
 * 从画布 restApiCall 节点 data 组装 spec 引擎所需的 inputsValues。
 * params / headers：优先按 schema `*.properties` 的键对齐；若导入后缺失 `properties`，则从
 * `paramsValues` / `headersValues` 的键兜底（避免再导出时丢 query 表单项）。
 */
function buildRestApiCallInputsValues(n: any): Record<string, FlowValueLike> {
  const urlContent = n.data?.api?.url?.content ?? '';
  const method = n.data?.api?.method ?? 'GET';

  const parammap: Record<string, unknown> = {};
  const pv = n.data?.paramsValues;
  if (pv && typeof pv === 'object' && Object.keys(pv).length > 0) {
    let paramKeys: string[] = [];
    const props = n.data?.params?.properties;
    if (props && typeof props === 'object') {
      paramKeys = Object.keys(props);
    }
    if (paramKeys.length === 0) {
      paramKeys = Object.keys(pv);
    }
    for (const key of paramKeys) {
      const cell = (pv as any)[key];
      if (cell !== undefined) {
        parammap[key] = cell?.content;
      }
    }
  }

  const headermap: Record<string, unknown> = {};
  const hv = n.data?.headersValues;
  if (hv && typeof hv === 'object' && Object.keys(hv).length > 0) {
    let headerKeys: string[] = [];
    const hprops = n.data?.headers?.properties;
    if (hprops && typeof hprops === 'object') {
      headerKeys = Object.keys(hprops);
    }
    if (headerKeys.length === 0) {
      headerKeys = Object.keys(hv);
    }
    for (const key of headerKeys) {
      const cell = (hv as any)[key];
      if (cell !== undefined) {
        headermap[key] = cell?.content;
      }
    }
  }

  const out: Record<string, FlowValueLike> = {
    url: { content: urlContent },
    requestMethod: { content: method },
  };
  if (Object.keys(parammap).length > 0) {
    out.params = { content: parammap };
  }
  if (Object.keys(headermap).length > 0) {
    out.headers = { content: headermap };
  }
  if (n.data?.body?.bodyType === 'JSON' && n.data?.body?.json?.content !== undefined) {
    out.body = { content: String(n.data.body.json.content ?? '') };
  }
  if (n.data?.timeout && n.data.timeout.timeout !== undefined) {
    out.readTimeoutMs = { content: n.data.timeout.timeout };
  }
  return out;
}

function buildYapiInputsValues(n: any): Record<string, FlowValueLike> {
  const cfg = n.data?.yapiConfig ?? {};
  return {
    baseUrl: { content: String(cfg.baseUrl ?? '') },
    userName: { content: String(cfg.userName ?? '') },
    password: { content: String(cfg.password ?? '') },
    interfacePath: { content: String(cfg.interfacePath ?? '') },
    loginType: { content: String(cfg.loginType ?? 'ldap') },
  };
}

function extractJsFunctionBody(scriptText: string, fnName: string): string {
  const fnIdx = scriptText.indexOf(`function ${fnName}`);
  const braceStart = fnIdx >= 0 ? scriptText.indexOf('{', fnIdx) : -1;
  if (braceStart < 0) return '';
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
  return scriptText.slice(braceStart + 1, end).trim();
}

function extractLuaTransformBody(scriptText: string): string {
  const fnIdx = scriptText.indexOf('function Transform');
  if (fnIdx < 0) return '';
  const lineEndIdx = scriptText.indexOf('\n', fnIdx);
  const endIdx = scriptText.lastIndexOf('end');
  return scriptText
    .slice(lineEndIdx >= 0 ? lineEndIdx + 1 : fnIdx, endIdx >= 0 ? endIdx : scriptText.length)
    .trim();
}

function buildScriptInputsValues(n: any): Record<string, FlowValueLike> {
  const scriptText = String(n.data?.script?.content ?? '');
  const matchName =
    n.type === 'jsTransform'
      ? 'Transform'
      : n.type === 'log'
      ? 'ToString'
      : n.type === 'jsFilter'
      ? 'Filter'
      : '';
  const scriptBody = matchName ? extractJsFunctionBody(scriptText, matchName) : '';
  return { scriptBody: { content: scriptBody } };
}

function buildLuaInputsValues(n: any): Record<string, FlowValueLike> {
  const scriptText = String(n.data?.script?.content ?? '');
  return { scriptBody: { content: extractLuaTransformBody(scriptText) } };
}

function inputsValuesMapToFlowData(
  iv: InputsValuesMap,
  spec: NodeMappingSpec
): Record<string, any> {
  const out: Record<string, any> = {};
  for (const f of spec.fields) {
    const cell = iv[f.inputKey];
    const flowType = f.valueType === 'template' ? 'template' : 'constant';
    out[f.inputKey] = { type: flowType, content: cell?.content };
  }
  return out;
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
  const endpoiontsRc: EndpointDsl[] = buildScheduleEndpointsFromDocument(
    flattened,
    baseOverride,
    alphaNanoid
  );

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
  if (shouldSkipRuleChainMetaNode(nodeType)) {
    return;
  }
  switch (nodeType) {
    case 'group': {
      emitGroupToRuleChain(n, base);
      break;
    }
    case 'for': {
      emitForToRuleChain(
        n,
        base,
        nodesRC,
        connectionsRC,
        (child) => buildRuleChainMetaNodes(child, nodesRC, connectionsRC),
        buildRuleChainMetaConnection
      );
      break;
    }
    case 'restApiCall': {
      const specRest = getNodeMappingSpec('restApiCall');
      if (!specRest) break;
      const synthetic = {
        data: { inputsValues: buildRestApiCallInputsValues(n) },
      };
      base.configuration = mapNodeToDslConfig(synthetic, specRest) as Record<string, any>;
      break;
    }
    case 'ai/agentHarness': {
      const specAh = getNodeMappingSpec('ai/agentHarness');
      if (!specAh) break;
      base.configuration = mapNodeToDslConfig(n, specAh) as Record<string, any>;
      break;
    }
    case 'switch': {
      const specSwitch = getNodeMappingSpec('switch');
      if (!specSwitch) break;
      const synthetic = {
        data: { inputsValues: { cases: { content: n.data?.cases ?? [] } } },
      };
      base.configuration = mapNodeToDslConfig(synthetic, specSwitch) as Record<string, any>;
      break;
    }
    case 'inclusive': {
      const specInc = getNodeMappingSpec('inclusive');
      if (!specInc) break;
      const synthetic = {
        data: { inputsValues: { cases: { content: n.data?.cases ?? [] } } },
      };
      base.configuration = mapNodeToDslConfig(synthetic, specInc) as Record<string, any>;
      break;
    }
    case 'end':
    case 'break': {
      base.configuration = {};
      break;
    }
    case 'while':
    case 'exec':
    case 'x/fileRead':
    case 'x/fileWrite':
    case 'x/fileDelete':
    case 'x/fileList':
    case 'x/jsonExtract':
    case 'ci/gitClone':
    case 'ci/gitCommit':
    case 'ci/gitPush':
    case 'opensearch/search':
    case 'volcTls/searchLogs': {
      const specExtra = getNodeMappingSpec(nodeType);
      if (!specExtra) break;
      base.configuration = mapNodeToDslConfig(n, specExtra) as Record<string, any>;
      break;
    }
    case 'dbClient': {
      const specDb = getNodeMappingSpec('dbClient');
      if (!specDb) break;
      base.configuration = mapNodeToDslConfig(n, specDb) as Record<string, any>;
      break;
    }
    case 'x/redisClient': {
      const specRedis = getNodeMappingSpec('x/redisClient');
      if (!specRedis) break;
      base.configuration = mapNodeToDslConfig(n, specRedis) as Record<string, any>;
      break;
    }
    case 'x/cursorCli': {
      const specCursorCli = getNodeMappingSpec('x/cursorCli');
      if (!specCursorCli) break;
      base.configuration = mapNodeToDslConfig(n, specCursorCli) as Record<string, any>;
      break;
    }
    case 'x/cursorCliAuth': {
      const specCursorCliAuth = getNodeMappingSpec('x/cursorCliAuth');
      if (!specCursorCliAuth) break;
      base.configuration = mapNodeToDslConfig(n, specCursorCliAuth) as Record<string, any>;
      break;
    }
    case 'x/cursorAcp': {
      const specAcp = getNodeMappingSpec('x/cursorAcp');
      if (!specAcp) break;
      base.configuration = mapNodeToDslConfig(n, specAcp) as Record<string, any>;
      break;
    }
    case 'x/feishuWebhook': {
      const specFs = getNodeMappingSpec('x/feishuWebhook');
      if (!specFs) break;
      base.configuration = mapNodeToDslConfig(n, specFs) as Record<string, any>;
      break;
    }
    case 'x/feishuCliAuth': {
      const specFsAuth = getNodeMappingSpec('x/feishuCliAuth');
      if (!specFsAuth) break;
      base.configuration = mapNodeToDslConfig(n, specFsAuth) as Record<string, any>;
      break;
    }
    case 'x/workspaceSync': {
      const specWs = getNodeMappingSpec('x/workspaceSync');
      if (!specWs) break;
      base.configuration = mapNodeToDslConfig(n, specWs) as Record<string, any>;
      break;
    }
    case 'transform/multiNodeOutput': {
      const specMulti = getNodeMappingSpec('transform/multiNodeOutput');
      if (!specMulti) break;
      base.configuration = mapNodeToDslConfig(n, specMulti) as Record<string, any>;
      break;
    }
    case 'jsTransform':
    case 'log':
    case 'jsFilter': {
      const specJs = getNodeMappingSpec(nodeType);
      if (!specJs) break;
      const synthetic = { data: { inputsValues: buildScriptInputsValues(n) } };
      base.configuration = mapNodeToDslConfig(synthetic, specJs) as Record<string, any>;
      break;
    }
    case 'luaTransform': {
      const specLua = getNodeMappingSpec('luaTransform');
      if (!specLua) break;
      const synthetic = { data: { inputsValues: buildLuaInputsValues(n) } };
      base.configuration = mapNodeToDslConfig(synthetic, specLua) as Record<string, any>;
      break;
    }
    case 'flow': {
      const specFlow = getNodeMappingSpec('flow');
      if (!specFlow) break;
      base.configuration = mapNodeToDslConfig(n, specFlow) as Record<string, any>;
      break;
    }
    case 'transform/yapi': {
      const specYapi = getNodeMappingSpec('transform/yapi');
      if (!specYapi) break;
      const synthetic = { data: { inputsValues: buildYapiInputsValues(n) } };
      base.configuration = mapNodeToDslConfig(synthetic, specYapi) as Record<string, any>;
      break;
    }
    default: {
      // 保持默认逻辑
      const props = n.data?.inputs?.properties;
      if (
        n.data?.inputs &&
        Object.keys(n.data.inputs).length > 0 &&
        props &&
        typeof props === 'object' &&
        !Array.isArray(props) &&
        n.data?.inputsValues &&
        Object.keys(n.data.inputsValues).length > 0
      ) {
        const parammap: Record<string, any> = {};
        for (const key of Object.keys(props)) {
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
      const pos = (n?.additionalInfo?.meta?.position as any) || {
        x: fallbackX,
        y: fallbackY,
      };
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
          base.data = buildStartFlowData(n);
          break;
        }
        case 'x/redisClient': {
          const cfg = n.configuration ?? {};
          const specRedis = getNodeMappingSpec('x/redisClient');
          const ivMapRedis = specRedis
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specRedis)
            : ({} as InputsValuesMap);
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
                db: {
                  type: 'number',
                  extra: { label: '数据库编号', description: '默认为 0' },
                },
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
              server: {
                type: 'constant',
                content: String(ivMapRedis.server?.content ?? (cfg as any).server ?? ''),
              },
              password: {
                type: 'constant',
                content: String(ivMapRedis.password?.content ?? (cfg as any).password ?? ''),
              },
              poolSize: {
                type: 'constant',
                content: Number(ivMapRedis.poolSize?.content ?? (cfg as any).poolSize ?? 0),
              },
              db: {
                type: 'constant',
                content: Number(ivMapRedis.db?.content ?? (cfg as any).db ?? 0),
              },
              cmd: {
                type: 'template',
                content: String(ivMapRedis.cmd?.content ?? (cfg as any).cmd ?? ''),
              },
              params: {
                type: 'constant',
                content: Array.isArray(ivMapRedis.params?.content)
                  ? ivMapRedis.params?.content
                  : Array.isArray((cfg as any).params)
                  ? (cfg as any).params
                  : [],
              },
            },
          } as any;
          break;
        }
        case 'opensearch/search': {
          const cfg = n.configuration ?? {};
          const specOs = getNodeMappingSpec('opensearch/search');
          const ivMapOs = specOs
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specOs)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'OpenSearch 检索',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['endpoint', 'index'],
              properties: {
                endpoint: {
                  type: 'string',
                  extra: {
                    label: 'Endpoint',
                    formComponent: 'prompt-editor',
                    description: 'OpenSearch 地址，例如 https://opensearch:9200',
                  },
                },
                index: {
                  type: 'string',
                  extra: {
                    label: '索引',
                    formComponent: 'prompt-editor',
                    description: '支持单索引、多索引（逗号分隔）和通配符',
                  },
                },
                username: { type: 'string', extra: { label: '用户名（可选）' } },
                password: { type: 'string', extra: { label: '密码（可选）' } },
                insecureSkipVerify: {
                  type: 'boolean',
                  extra: { label: '跳过 TLS 证书校验' },
                },
                timeoutSec: { type: 'number', extra: { label: '超时（秒）' } },
                searchType: {
                  type: 'string',
                  enum: ['query_then_fetch', 'dfs_query_then_fetch'],
                  extra: { label: 'search_type', formComponent: 'enum-select' },
                },
                ignoreUnavailable: {
                  type: 'boolean',
                  extra: { label: '忽略不可用索引' },
                },
                defaultSearchBody: {
                  type: 'string',
                  extra: {
                    label: '默认查询体（JSON）',
                    formComponent: 'prompt-editor',
                    jsonFormat: true,
                  },
                },
              },
            },
            inputsValues: specOs
              ? inputsValuesMapToFlowData(ivMapOs, specOs)
              : {
                  endpoint: { type: 'template', content: String((cfg as any).endpoint ?? '') },
                  index: { type: 'template', content: String((cfg as any).index ?? '') },
                },
          } as any;
          break;
        }
        case 'volcTls/searchLogs': {
          const cfg = n.configuration ?? {};
          const specTls = getNodeMappingSpec('volcTls/searchLogs');
          const ivMapTls = specTls
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specTls)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? '火山 TLS 检索',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['region', 'accessKeyId', 'secretAccessKey', 'topicId'],
              properties: {
                endpoint: { type: 'string', extra: { label: 'Endpoint（可选）' } },
                region: { type: 'string', extra: { label: 'Region（必填）' } },
                accessKeyId: {
                  type: 'string',
                  extra: { label: 'AccessKeyId', formComponent: 'prompt-editor' },
                },
                secretAccessKey: {
                  type: 'string',
                  extra: { label: 'SecretAccessKey', formComponent: 'prompt-editor' },
                },
                sessionToken: {
                  type: 'string',
                  extra: { label: 'SessionToken（可选）', formComponent: 'prompt-editor' },
                },
                topicId: {
                  type: 'string',
                  extra: { label: 'TopicId（默认）', formComponent: 'prompt-editor' },
                },
                defaultQuery: {
                  type: 'string',
                  extra: { label: '默认查询语句', formComponent: 'prompt-editor' },
                },
                limit: { type: 'number', extra: { label: 'Limit（1-500）' } },
                useApiV3: { type: 'boolean', extra: { label: '使用 SearchLogsV2' } },
                timeoutSec: { type: 'number', extra: { label: '超时（秒）' } },
                timeRangePreset: {
                  type: 'string',
                  enum: [
                    'last_15m',
                    'last_30m',
                    'last_1h',
                    'last_6h',
                    'last_24h',
                    'last_7d',
                    'today_local',
                    'custom',
                  ],
                  extra: { label: '默认时间窗', formComponent: 'enum-select' },
                },
                defaultStartTimeMs: { type: 'number', extra: { label: '自定义开始时间（毫秒）' } },
                defaultEndTimeMs: { type: 'number', extra: { label: '自定义结束时间（毫秒）' } },
                defaultSort: {
                  type: 'string',
                  enum: ['desc', 'asc'],
                  extra: { label: '默认排序', formComponent: 'enum-select' },
                },
                highLight: { type: 'boolean', extra: { label: '默认开启高亮' } },
              },
            },
            inputsValues: specTls
              ? inputsValuesMapToFlowData(ivMapTls, specTls)
              : {
                  region: { type: 'constant', content: String((cfg as any).region ?? '') },
                },
          } as any;
          break;
        }
        case 'x/cursorCli': {
          const cfg = n.configuration ?? {};
          const specCc = getNodeMappingSpec('x/cursorCli');
          const ivMapCc = specCc
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specCc)
            : ({} as InputsValuesMap);
          const agentPathFallback =
            (cfg as any).agentPath != null && String((cfg as any).agentPath).trim() !== ''
              ? String((cfg as any).agentPath)
              : String((cfg as any).cursorPath ?? 'agent');
          base.data = {
            title: n.name ?? 'Cursor CLI (agent)',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['agentPath', 'args', 'log', 'replaceData', 'timeoutMs'],
              properties: {
                printMode: {
                  type: 'boolean',
                  extra: {
                    label: '打印模式（-p）',
                    description: '开启后插入 -p；任务说明、模型用下方独立字段',
                  },
                },
                prompt: {
                  type: 'string',
                  extra: {
                    label: '任务说明（-p 后）',
                    formComponent: 'prompt-editor',
                    description: '等价 agent -p "…" 中带引号的那段；支持 ${msg.xxx}',
                  },
                },
                outputFormat: {
                  type: 'string',
                  enum: ['text', 'json', 'stream-json'],
                  default: { type: 'constant', content: 'text' } as any,
                  extra: {
                    label: '输出格式（--output-format）',
                    formComponent: 'enum-select',
                    description:
                      '仅在打印模式开启时插入；text / json / stream-json，与官方文档一致；勿写入「额外参数」',
                  },
                },
                model: {
                  type: 'string',
                  extra: {
                    label: '模型（--model）',
                    formComponent: 'prompt-editor',
                    description: '非空追加 --model <值>；单独配置',
                  },
                },
                agentPath: {
                  type: 'string',
                  extra: {
                    label: 'agent 可执行文件',
                    description:
                      '官方 CLI 为 agent（默认 ~/.local/bin/agent 或 PATH）；后端允许 basename 为 agent 或 cursor',
                  },
                },
                workspacePath: {
                  type: 'string',
                  extra: {
                    label: '工作区路径（--workspace）',
                    formComponent: 'prompt-editor',
                    description:
                      '代码仓库根目录，作为 Agent 代码上下文；与 workDir（进程 cwd）不同',
                  },
                },
                worktree: {
                  type: 'boolean',
                  extra: {
                    label: 'Git Worktree（--worktree）',
                    description:
                      '开启后注入 --worktree，让 Agent 在新建的 Git worktree 中运行，而非直接编辑当前 checkout',
                  },
                },
                workDir: {
                  type: 'string',
                  extra: {
                    label: '进程工作目录（cwd）',
                    formComponent: 'prompt-editor',
                    description: '子进程 cmd.Dir；留空用 metadata.workDir',
                  },
                },
                args: {
                  type: 'array',
                  items: { type: 'string' },
                  extra: {
                    label: '额外命令行参数',
                    formComponent: 'array-editor',
                    description:
                      '在 -p、任务说明、--output-format、--model 之后追加；勿重复写 --output-format',
                  },
                },
                log: {
                  type: 'boolean',
                  extra: { label: '输出到调试日志' },
                },
                replaceData: {
                  type: 'boolean',
                  extra: { label: '用 stdout 替换消息体' },
                },
                timeoutMs: {
                  type: 'number',
                  extra: { label: '超时(毫秒)' },
                },
              },
            },
            inputsValues: specCc
              ? inputsValuesMapToFlowData(ivMapCc, specCc)
              : {
                  printMode: { type: 'constant', content: !!(cfg as any).printMode },
                  prompt: { type: 'template', content: String((cfg as any).prompt ?? '') },
                  outputFormat: {
                    type: 'constant',
                    content: String((cfg as any).outputFormat ?? 'text'),
                  },
                  model: { type: 'template', content: String((cfg as any).model ?? '') },
                  agentPath: {
                    type: 'constant',
                    content: agentPathFallback,
                  },
                  worktree: { type: 'constant', content: !!(cfg as any).worktree },
                  workspacePath: {
                    type: 'template',
                    content: String((cfg as any).workspacePath ?? ''),
                  },
                  workDir: { type: 'template', content: String((cfg as any).workDir ?? '') },
                  args: {
                    type: 'constant',
                    content: Array.isArray((cfg as any).args) ? (cfg as any).args : [],
                  },
                  log: { type: 'constant', content: !!(cfg as any).log },
                  replaceData: { type: 'constant', content: (cfg as any).replaceData !== false },
                  timeoutMs: { type: 'constant', content: Number((cfg as any).timeoutMs ?? 0) },
                },
          } as any;
          break;
        }
        case 'x/cursorCliAuth': {
          const cfg = n.configuration ?? {};
          const specCcAuth = getNodeMappingSpec('x/cursorCliAuth');
          const ivMapCcAuth = specCcAuth
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specCcAuth)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'Cursor CLI 授权检查',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['agentPath', 'workspacePath', 'timeoutMs'],
              properties: {
                agentPath: { type: 'string', extra: { label: 'agent 可执行文件' } },
                workspacePath: {
                  type: 'string',
                  extra: { label: '工作区路径（--workspace）', formComponent: 'prompt-editor' },
                },
                worktree: { type: 'boolean', extra: { label: 'Git Worktree（--worktree）' } },
                force: { type: 'boolean', extra: { label: '强制允许命令（--force）' } },
                workDir: {
                  type: 'string',
                  extra: { label: '进程工作目录（cwd）', formComponent: 'prompt-editor' },
                },
                timeoutMs: { type: 'number', extra: { label: '超时(毫秒)' } },
                replaceData: { type: 'boolean', extra: { label: '用状态 JSON 替换消息体' } },
              },
            },
            inputsValues: specCcAuth
              ? inputsValuesMapToFlowData(ivMapCcAuth, specCcAuth)
              : {
                  agentPath: { type: 'constant', content: String((cfg as any).agentPath ?? 'agent') },
                  workspacePath: {
                    type: 'template',
                    content: String((cfg as any).workspacePath ?? '$HOME'),
                  },
                  worktree: { type: 'constant', content: !!(cfg as any).worktree },
                  force: { type: 'constant', content: (cfg as any).force !== false },
                  workDir: { type: 'template', content: String((cfg as any).workDir ?? '') },
                  timeoutMs: { type: 'constant', content: Number((cfg as any).timeoutMs ?? 15000) },
                  replaceData: { type: 'constant', content: (cfg as any).replaceData !== false },
                },
          } as any;
          break;
        }
        case 'x/cursorAcp': {
          const cfg = n.configuration ?? {};
          const specAcp = getNodeMappingSpec('x/cursorAcp');
          const ivMapAcp = specAcp
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specAcp)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'Cursor ACP (agent acp)',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['agentPath', 'args', 'stdinLines', 'timeoutMs'],
              properties: {
                stdinLines: {
                  type: 'array',
                  items: { type: 'string' },
                  extra: {
                    label: 'stdin JSON-RPC（用户指令写在对应行的 JSON 里）',
                    formComponent: 'array-editor',
                    description:
                      '每行一条 JSON-RPC。任务/说明放在 session/prompt（等）那一行的 params 里；支持 ${msg.*}。须至少一行。',
                  },
                },
                workspacePath: {
                  type: 'string',
                  extra: {
                    label: '工作区路径（--workspace）',
                    formComponent: 'prompt-editor',
                    description: '非空时插入 --workspace，指定仓库根（代码上下文）',
                  },
                },
                worktree: {
                  type: 'boolean',
                  extra: {
                    label: 'Git Worktree（--worktree）',
                    description:
                      '开启后注入 --worktree，让 Agent 在新建的 Git worktree 中运行，而非直接编辑当前 checkout',
                  },
                },
                workDir: {
                  type: 'string',
                  extra: {
                    label: '进程工作目录（cwd）',
                    formComponent: 'prompt-editor',
                    description: '子进程 cmd.Dir；留空用 metadata.workDir',
                  },
                },
                agentPath: {
                  type: 'string',
                  extra: {
                    label: 'agent 可执行文件',
                    description: '默认可写 ~/.local/bin/agent 或留空使用 PATH 中的 agent',
                  },
                },
                args: {
                  type: 'array',
                  items: { type: 'string' },
                  extra: {
                    label: 'argv',
                    formComponent: 'array-editor',
                    description: '首项须为 acp，如 ["acp"]；--workspace 用下方专用字段',
                  },
                },
                log: { type: 'boolean', extra: { label: '输出到调试日志' } },
                replaceData: {
                  type: 'boolean',
                  extra: { label: '用 stdout 替换消息体' },
                },
                timeoutMs: {
                  type: 'number',
                  extra: { label: '超时(毫秒)', description: '默认 120000' },
                },
              },
            },
            inputsValues: specAcp
              ? inputsValuesMapToFlowData(ivMapAcp, specAcp)
              : {
                  stdinLines: {
                    type: 'constant',
                    content: Array.isArray((cfg as any).stdinLines) ? (cfg as any).stdinLines : [],
                  },
                  worktree: { type: 'constant', content: !!(cfg as any).worktree },
                  workspacePath: {
                    type: 'template',
                    content: String((cfg as any).workspacePath ?? ''),
                  },
                  workDir: { type: 'template', content: String((cfg as any).workDir ?? '') },
                  agentPath: {
                    type: 'constant',
                    content: String((cfg as any).agentPath ?? 'agent'),
                  },
                  args: {
                    type: 'constant',
                    content: Array.isArray((cfg as any).args) ? (cfg as any).args : ['acp'],
                  },
                  log: { type: 'constant', content: !!(cfg as any).log },
                  replaceData: { type: 'constant', content: (cfg as any).replaceData !== false },
                  timeoutMs: {
                    type: 'constant',
                    content: Number((cfg as any).timeoutMs ?? 120000),
                  },
                },
          } as any;
          break;
        }
        case 'x/feishuWebhook': {
          const cfg = n.configuration ?? {};
          const specFs = getNodeMappingSpec('x/feishuWebhook');
          const ivMapFs = specFs
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specFs)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? '飞书 Webhook',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['msgType', 'webhookUrl', 'timeoutMs'],
              properties: {
                msgType: {
                  type: 'string',
                  enum: ['text', 'post', 'interactive', 'raw'],
                  default: { type: 'constant', content: 'text' } as any,
                  extra: {
                    label: '消息类型',
                    formComponent: 'enum-select',
                    description: 'text / post / interactive / raw',
                  },
                },
                webhookUrl: {
                  type: 'string',
                  extra: {
                    label: 'Webhook URL',
                    formComponent: 'prompt-editor',
                    description: 'https；建议 ${metadata.xxx} 注入',
                  },
                },
                text: {
                  type: 'string',
                  extra: {
                    label: '纯文本（text）',
                    formComponent: 'prompt-editor',
                    description: 'msg_type=text',
                  },
                },
                postTitle: {
                  type: 'string',
                  extra: {
                    label: '富文本标题（post）',
                    formComponent: 'prompt-editor',
                  },
                },
                postBody: {
                  type: 'string',
                  extra: {
                    label: '富文本正文（post）',
                    formComponent: 'prompt-editor',
                  },
                },
                postLang: {
                  type: 'string',
                  enum: ['zh_cn', 'en_us', 'ja_jp'],
                  default: { type: 'constant', content: 'zh_cn' } as any,
                  extra: {
                    label: '富文本语言（post）',
                    formComponent: 'enum-select',
                  },
                },
                postSplitByLine: {
                  type: 'boolean',
                  extra: { label: '正文按换行拆段（post）' },
                },
                postAtAllBefore: {
                  type: 'boolean',
                  extra: { label: '正文前 @所有人（post）' },
                },
                postAtAllAfter: {
                  type: 'boolean',
                  extra: { label: '正文后 @所有人（post）' },
                },
                postMentionUserIds: {
                  type: 'array',
                  items: { type: 'string' },
                  extra: {
                    label: '@成员 id 列表（post）',
                    formComponent: 'array-editor',
                  },
                },
                interactivePreset: {
                  type: 'string',
                  enum: ['card_json', 'notice_card'],
                  default: { type: 'constant', content: 'card_json' } as any,
                  extra: {
                    label: '卡片方式（interactive）',
                    formComponent: 'enum-select',
                  },
                },
                cardNoticeTitle: {
                  type: 'string',
                  extra: {
                    label: '通知卡片标题',
                    formComponent: 'prompt-editor',
                  },
                },
                cardNoticeMarkdown: {
                  type: 'string',
                  extra: {
                    label: '通知卡片 Markdown',
                    formComponent: 'prompt-editor',
                  },
                },
                cardJson: {
                  type: 'string',
                  extra: {
                    label: '卡片 JSON（自定义）',
                    formComponent: 'prompt-editor',
                    jsonFormat: true,
                  },
                },
                rawJson: {
                  type: 'string',
                  extra: {
                    label: '自定义整包（raw）',
                    formComponent: 'prompt-editor',
                    jsonFormat: true,
                  },
                },
                timeoutMs: {
                  type: 'number',
                  extra: { label: '超时(毫秒)', description: '默认 15000' },
                },
                replaceData: {
                  type: 'boolean',
                  extra: {
                    label: '用响应体替换消息体',
                    description: '成功时把接口返回 JSON 写入下游 msg 数据',
                  },
                },
              },
            },
            inputsValues: specFs
              ? inputsValuesMapToFlowData(ivMapFs, specFs)
              : {
                  msgType: {
                    type: 'constant',
                    content: String((cfg as any).msgType ?? 'text'),
                  },
                  webhookUrl: {
                    type: 'template',
                    content: String((cfg as any).webhookUrl ?? ''),
                  },
                  text: { type: 'template', content: String((cfg as any).text ?? '') },
                  postTitle: { type: 'template', content: String((cfg as any).postTitle ?? '') },
                  postBody: { type: 'template', content: String((cfg as any).postBody ?? '') },
                  postLang: {
                    type: 'constant',
                    content: String((cfg as any).postLang ?? 'zh_cn'),
                  },
                  postSplitByLine: {
                    type: 'constant',
                    content: (cfg as any).postSplitByLine === true,
                  },
                  postAtAllBefore: {
                    type: 'constant',
                    content: (cfg as any).postAtAllBefore === true,
                  },
                  postAtAllAfter: {
                    type: 'constant',
                    content: (cfg as any).postAtAllAfter === true,
                  },
                  postMentionUserIds: {
                    type: 'constant',
                    content: Array.isArray((cfg as any).postMentionUserIds)
                      ? (cfg as any).postMentionUserIds
                      : [],
                  },
                  interactivePreset: {
                    type: 'constant',
                    content: String((cfg as any).interactivePreset ?? 'card_json'),
                  },
                  cardNoticeTitle: {
                    type: 'template',
                    content: String((cfg as any).cardNoticeTitle ?? ''),
                  },
                  cardNoticeMarkdown: {
                    type: 'template',
                    content: String((cfg as any).cardNoticeMarkdown ?? ''),
                  },
                  cardJson: { type: 'template', content: String((cfg as any).cardJson ?? '') },
                  rawJson: { type: 'template', content: String((cfg as any).rawJson ?? '') },
                  timeoutMs: {
                    type: 'constant',
                    content: Number((cfg as any).timeoutMs ?? 15000),
                  },
                  replaceData: {
                    type: 'constant',
                    content: (cfg as any).replaceData === true,
                  },
                },
          } as any;
          break;
        }
        case 'x/feishuCliAuth': {
          const cfg = n.configuration ?? {};
          const specFsAuth = getNodeMappingSpec('x/feishuCliAuth');
          const ivMapFsAuth = specFsAuth
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specFsAuth)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? '飞书 CLI 授权检查',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['cliPath', 'timeoutMs'],
              properties: {
                cliPath: { type: 'string', extra: { label: 'CLI 可执行文件' } },
                args: {
                  type: 'array',
                  items: { type: 'string' },
                  extra: { label: '命令参数', formComponent: 'array-editor' },
                },
                workDir: {
                  type: 'string',
                  extra: { label: '进程工作目录（cwd）', formComponent: 'prompt-editor' },
                },
                timeoutMs: { type: 'number', extra: { label: '超时(毫秒)' } },
                replaceData: { type: 'boolean', extra: { label: '用状态 JSON 替换消息体' } },
              },
            },
            inputsValues: specFsAuth
              ? inputsValuesMapToFlowData(ivMapFsAuth, specFsAuth)
              : {
                  cliPath: { type: 'constant', content: String((cfg as any).cliPath ?? 'lark-cli') },
                  args: {
                    type: 'constant',
                    content: Array.isArray((cfg as any).args)
                      ? (cfg as any).args
                      : ['auth', 'status'],
                  },
                  workDir: { type: 'template', content: String((cfg as any).workDir ?? '') },
                  timeoutMs: { type: 'constant', content: Number((cfg as any).timeoutMs ?? 15000) },
                  replaceData: { type: 'constant', content: (cfg as any).replaceData !== false },
                },
          } as any;
          break;
        }
        case 'x/workspaceSync': {
          const cfg = n.configuration ?? {};
          const specWs = getNodeMappingSpec('x/workspaceSync');
          const ivMapWs = specWs
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specWs)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? '工作区刷新',
            positionType: 'middle',
            inputs: {
              type: 'object',
              required: ['workspaceId'],
              properties: {
                workspaceId: { type: 'string', extra: { label: '工作区 ID' } },
                replaceData: { type: 'boolean', extra: { label: '用结果 JSON 替换消息体' } },
              },
            },
            inputsValues: specWs
              ? inputsValuesMapToFlowData(ivMapWs, specWs)
              : {
                  workspaceId: {
                    type: 'constant',
                    content: String((cfg as any).workspaceId ?? ''),
                  },
                  replaceData: { type: 'constant', content: (cfg as any).replaceData !== false },
                },
          } as any;
          break;
        }
        case 'restApiCall': {
          const cfg = n.configuration ?? {};
          const specRest = getNodeMappingSpec('restApiCall');
          if (!specRest) break;
          const iv = mapDslToNodeInputsValues(cfg as Record<string, unknown>, specRest);
          const hObj = (iv.headers?.content ?? {}) as Record<string, unknown>;
          const headerVals = Object.keys(hObj).reduce((acc: any, k) => {
            acc[k] = { type: 'constant', content: hObj[k] };
            return acc;
          }, {});
          const pObj = (iv.params?.content ?? {}) as Record<string, unknown>;
          const mergedParamVals = Object.keys(pObj).reduce((acc: any, k) => {
            acc[k] = { type: 'constant', content: pObj[k] };
            return acc;
          }, {});
          const urlStr =
            iv.url?.content != null && String(iv.url.content).length > 0
              ? String(iv.url.content)
              : '';
          base.data = {
            title: n.name ?? 'restApiCall',
            positionType: 'middle',
            api: {
              method: (iv.requestMethod?.content as string) ?? 'GET',
              url: urlStr ? { type: 'template', content: urlStr } : undefined,
            },
            headers: {},
            headersValues: headerVals,
            params: {},
            paramsValues: mergedParamVals,
            body: {
              bodyType: 'JSON',
              json: iv.body?.content ? { type: 'template', content: iv.body.content } : undefined,
            },
            timeout: {
              retryTimes: 0,
              timeout: Number(iv.readTimeoutMs?.content ?? 0),
            },
          };
          break;
        }
        case 'ai/agentHarness': {
          const cfg = n.configuration ?? {};
          const specAh = getNodeMappingSpec('ai/agentHarness');
          const inputsSchemaAh = {
            type: 'object',
            required: [
              'model',
              'userPrompt',
              'systemPrompt',
              'enableSkillTool',
              'enableWorkspaceTools',
              'skillAllowlist',
              'maxIterations',
              'maxToolCalls',
              'toolTimeoutSecs',
              'gitWorktreeMode',
            ],
            properties: {
              model: {
                type: 'string',
                extra: {
                  label: '模型名称',
                  formComponent: 'prompt-editor',
                  description: '留空则用配置默认模型；支持 ${} 模板',
                },
              },
              userPrompt: {
                type: 'string',
                extra: { label: '用户提示词', formComponent: 'prompt-editor' },
              },
              systemPrompt: {
                type: 'string',
                extra: { label: '系统提示词', formComponent: 'prompt-editor' },
              },
              enableSkillTool: {
                type: 'boolean',
                extra: {
                  label: '启用 Skill',
                  description: '允许模型调用 Eino 官方 Skill 工具',
                },
              },
              enableWorkspaceTools: {
                type: 'boolean',
                extra: {
                  label: '启用 Workspace 工具',
                  description: '读/写文件与 shell（与 Chat Agent 一致）',
                },
              },
              skillAllowlist: {
                type: 'array',
                items: { type: 'string' },
                extra: {
                  label: 'Skill 白名单',
                  description: 'string[]；空=不限制',
                },
              },
              maxIterations: {
                type: 'number',
                extra: {
                  label: '最大迭代轮次',
                  description: '0 表示使用服务默认',
                },
              },
              maxToolCalls: {
                type: 'number',
                extra: {
                  label: '最大工具调用次数',
                  description: '0 表示使用服务默认',
                },
              },
              toolTimeoutSecs: {
                type: 'number',
                extra: {
                  label: '单次工具超时(秒)',
                  description: '0 表示使用服务默认',
                },
              },
              gitWorktreeMode: {
                type: 'boolean',
                extra: {
                  label: '启用 Git Worktree 模式',
                  description:
                    '启用后，模型在操作 git 仓库时必须通过 git worktree 创建隔离工作树，禁止直接在仓库主分支上执行修改性操作',
                },
              },
            },
          };
          if (!specAh) break;
          const ivMapAh = mapDslToNodeInputsValues(cfg as Record<string, unknown>, specAh);
          base.data = {
            title: n.name ?? 'ai/agentHarness',
            positionType: 'middle',
            inputsValues: inputsValuesMapToFlowData(ivMapAh, specAh),
            inputs: inputsSchemaAh,
            outputs: { type: 'object', properties: {} },
          };
          break;
        }
        case 'jsTransform': {
          const cfg = n.configuration ?? {};
          const specJs = getNodeMappingSpec('jsTransform');
          const ivMap = specJs
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specJs)
            : ({} as InputsValuesMap);
          const body = String(ivMap.scriptBody?.content ?? (cfg as any).jsScript ?? '');
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
          const specDb = getNodeMappingSpec('dbClient');
          const ivMapDb = specDb
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specDb)
            : ({} as InputsValuesMap);
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
                  extra: {
                    label: '数据库驱动名称',
                    formComponent: 'enum-select',
                  },
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
              sql: {
                type: 'template',
                content: String(ivMapDb.sql?.content ?? (cfg as any).sql ?? ''),
              },
              params: {
                type: 'constant',
                content: Array.isArray(ivMapDb.params?.content)
                  ? ivMapDb.params?.content
                  : Array.isArray((cfg as any).params)
                  ? (cfg as any).params
                  : [],
              },
              getOne: {
                type: 'constant',
                content: !!(ivMapDb.getOne?.content ?? (cfg as any).getOne),
              },
              poolSize: {
                type: 'constant',
                content: Number(ivMapDb.poolSize?.content ?? (cfg as any).poolSize ?? 0),
              },
              driverName: {
                type: 'constant',
                content: String(ivMapDb.driverName?.content ?? (cfg as any).driverName ?? 'mysql'),
              },
              dsn: {
                type: 'template',
                content: String(ivMapDb.dsn?.content ?? (cfg as any).dsn ?? ''),
              },
            },
          } as any;
          break;
        }
        case 'luaTransform': {
          const cfg = n.configuration ?? {};
          const specLua = getNodeMappingSpec('luaTransform');
          const ivMap = specLua
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specLua)
            : ({} as InputsValuesMap);
          const body = String(ivMap.scriptBody?.content ?? (cfg as any).luaScript ?? '');
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
          const cfg = n.configuration ?? {};
          const specLog = getNodeMappingSpec('log');
          const ivMap = specLog
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specLog)
            : ({} as InputsValuesMap);
          const body = String(ivMap.scriptBody?.content ?? (cfg as any).jsScript ?? '');
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
          const cfg = n.configuration ?? {};
          const specJsFilter = getNodeMappingSpec('jsFilter');
          const ivMap = specJsFilter
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specJsFilter)
            : ({} as InputsValuesMap);
          const body = String(ivMap.scriptBody?.content ?? (cfg as any).jsScript ?? '');
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
          const specSwitch = getNodeMappingSpec('switch');
          const ivMapSwitch = specSwitch
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specSwitch)
            : ({} as InputsValuesMap);
          const cases = Array.isArray(ivMapSwitch.cases?.content)
            ? (ivMapSwitch.cases?.content as any[])
            : [];
          base.data = {
            title: n.name ?? 'switch',
            positionType: 'middle',
            cases,
            ELSE: true,
          };
          break;
        }
        case 'inclusive': {
          const cfg = n.configuration ?? {};
          const specIncl = getNodeMappingSpec('inclusive');
          const ivMapIncl = specIncl
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specIncl)
            : ({} as InputsValuesMap);
          const cases = Array.isArray(ivMapIncl.cases?.content)
            ? (ivMapIncl.cases?.content as any[])
            : [];
          base.data = {
            title: n.name ?? '包容分支',
            positionType: 'middle',
            cases,
            ELSE: true,
          };
          break;
        }
        case 'end': {
          base.data = {
            title: n.name ?? 'End',
            positionType: 'tail',
            inputs: { type: 'object', properties: {} },
            inputsValues: {},
          };
          break;
        }
        case 'break': {
          base.data = {
            title: n.name ?? '中断',
            positionType: 'middle',
          };
          break;
        }
        case 'while': {
          const cfg = n.configuration ?? {};
          const specW = getNodeMappingSpec('while');
          const ivMapW = specW
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specW)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'While',
            positionType: 'middle',
            inputsValues: {
              condition: {
                type: 'template',
                content: String(ivMapW.condition?.content ?? (cfg as any).condition ?? ''),
              },
              do: {
                type: 'constant',
                content: String(ivMapW.do?.content ?? (cfg as any).do ?? ''),
              },
            },
            inputs: {
              type: 'object',
              required: ['condition', 'do'],
              properties: {
                condition: {
                  type: 'string',
                  extra: {
                    label: '循环条件',
                    formComponent: 'prompt-editor',
                  },
                },
                do: {
                  type: 'string',
                  extra: { label: '执行目标节点 ID' },
                },
              },
            },
          };
          break;
        }
        case 'exec': {
          const cfg = n.configuration ?? {};
          const specEx = getNodeMappingSpec('exec');
          const ivMapEx = specEx
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specEx)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'Exec',
            positionType: 'middle',
            inputsValues: {
              cmd: {
                type: 'template',
                content: String(ivMapEx.cmd?.content ?? (cfg as any).cmd ?? ''),
              },
              args: {
                type: 'constant',
                content: Array.isArray(ivMapEx.args?.content)
                  ? ivMapEx.args?.content
                  : Array.isArray((cfg as any).args)
                  ? (cfg as any).args
                  : [],
              },
              log: {
                type: 'constant',
                content: !!(ivMapEx.log?.content ?? (cfg as any).log),
              },
              replaceData: {
                type: 'constant',
                content: !!(ivMapEx.replaceData?.content ?? (cfg as any).replaceData),
              },
            },
            inputs: {
              type: 'object',
              required: ['cmd'],
              properties: {
                cmd: {
                  type: 'string',
                  extra: { label: '命令', formComponent: 'prompt-editor' },
                },
                args: {
                  type: 'array',
                  items: { type: 'string' },
                  extra: { label: '参数', formComponent: 'array-editor' },
                },
                log: { type: 'boolean', extra: { label: '记录 stdout' } },
                replaceData: {
                  type: 'boolean',
                  extra: { label: '替换 msg.Data' },
                },
              },
            },
          };
          break;
        }
        case 'x/fileRead': {
          const cfg = n.configuration ?? {};
          const specFr = getNodeMappingSpec('x/fileRead');
          const ivMapFr = specFr
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specFr)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? '读取文件',
            positionType: 'middle',
            inputsValues: {
              path: {
                type: 'template',
                content: String(ivMapFr.path?.content ?? (cfg as any).path ?? ''),
              },
              dataType: {
                type: 'constant',
                content: String(ivMapFr.dataType?.content ?? (cfg as any).dataType ?? 'text'),
              },
              recursive: {
                type: 'constant',
                content: !!(ivMapFr.recursive?.content ?? (cfg as any).recursive),
              },
            },
            inputs: {
              type: 'object',
              required: ['path', 'dataType'],
              properties: {
                path: {
                  type: 'string',
                  extra: { label: '路径或 Glob', formComponent: 'prompt-editor' },
                },
                dataType: {
                  type: 'string',
                  enum: ['text', 'base64'],
                  extra: { label: '数据类型', formComponent: 'enum-select' },
                },
                recursive: { type: 'boolean', extra: { label: '递归' } },
              },
            },
          };
          break;
        }
        case 'x/fileWrite': {
          const cfg = n.configuration ?? {};
          const specFw = getNodeMappingSpec('x/fileWrite');
          const ivMapFw = specFw
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specFw)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? '写入文件',
            positionType: 'middle',
            inputsValues: {
              path: {
                type: 'template',
                content: String(ivMapFw.path?.content ?? (cfg as any).path ?? ''),
              },
              content: {
                type: 'template',
                content: String(ivMapFw.content?.content ?? (cfg as any).content ?? '${data}'),
              },
              append: {
                type: 'constant',
                content: !!(ivMapFw.append?.content ?? (cfg as any).append),
              },
            },
            inputs: {
              type: 'object',
              required: ['path'],
              properties: {
                path: {
                  type: 'string',
                  extra: { label: '文件路径', formComponent: 'prompt-editor' },
                },
                content: {
                  type: 'string',
                  extra: { label: '写入内容', formComponent: 'prompt-editor' },
                },
                append: { type: 'boolean', extra: { label: '追加' } },
              },
            },
          };
          break;
        }
        case 'x/fileDelete': {
          const cfg = n.configuration ?? {};
          const specFd = getNodeMappingSpec('x/fileDelete');
          const ivMapFd = specFd
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specFd)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? '删除文件',
            positionType: 'middle',
            inputsValues: {
              path: {
                type: 'template',
                content: String(ivMapFd.path?.content ?? (cfg as any).path ?? ''),
              },
            },
            inputs: {
              type: 'object',
              required: ['path'],
              properties: {
                path: {
                  type: 'string',
                  extra: { label: '路径或 Glob', formComponent: 'prompt-editor' },
                },
              },
            },
          };
          break;
        }
        case 'x/fileList': {
          const cfg = n.configuration ?? {};
          const specFl = getNodeMappingSpec('x/fileList');
          const ivMapFl = specFl
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specFl)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? '文件列表',
            positionType: 'middle',
            inputsValues: {
              path: {
                type: 'template',
                content: String(ivMapFl.path?.content ?? (cfg as any).path ?? ''),
              },
              recursive: {
                type: 'constant',
                content: !!(ivMapFl.recursive?.content ?? (cfg as any).recursive),
              },
            },
            inputs: {
              type: 'object',
              required: ['path'],
              properties: {
                path: {
                  type: 'string',
                  extra: { label: '路径模式', formComponent: 'prompt-editor' },
                },
                recursive: { type: 'boolean', extra: { label: '递归' } },
              },
            },
          };
          break;
        }
        case 'x/jsonExtract': {
          const cfg = n.configuration ?? {};
          const specJe = getNodeMappingSpec('x/jsonExtract');
          const ivMapJe = specJe
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specJe)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'JSON 提取',
            positionType: 'middle',
            inputsValues: specJe
              ? inputsValuesMapToFlowData(ivMapJe, specJe)
              : {
                  source: { type: 'template', content: String((cfg as any).source ?? '') },
                },
            inputs: {
              type: 'object',
              required: ['source'],
              properties: {
                source: {
                  type: 'string',
                  extra: {
                    label: '输入文本',
                    formComponent: 'prompt-editor',
                    description: '包含 JSON 的文本内容，通常来自 Agent 输出的 markdown 代码块',
                  },
                },
              },
            },
            outputs: {
              type: 'object',
              properties: {
                result: { type: 'string' },
                extractedJson: { type: 'object' },
                error: { type: 'string' },
                success: { type: 'boolean' },
              },
            },
          };
          break;
        }
        case 'ci/gitClone': {
          const cfg = n.configuration ?? {};
          const specGc = getNodeMappingSpec('ci/gitClone');
          const ivMapGc = specGc
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specGc)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'Git拉取',
            positionType: 'middle',
            inputsValues: {
              repository: {
                type: 'template',
                content: String(ivMapGc.repository?.content ?? (cfg as any).repository ?? ''),
              },
              directory: {
                type: 'template',
                content: String(ivMapGc.directory?.content ?? (cfg as any).directory ?? ''),
              },
              reference: {
                type: 'constant',
                content: String(
                  ivMapGc.reference?.content ?? (cfg as any).reference ?? 'refs/heads/main'
                ),
              },
              authType: {
                type: 'constant',
                content: String(ivMapGc.authType?.content ?? (cfg as any).authType ?? 'token'),
              },
              authUser: {
                type: 'constant',
                content: String(ivMapGc.authUser?.content ?? (cfg as any).authUser ?? ''),
              },
              authPassword: {
                type: 'template',
                content: String(ivMapGc.authPassword?.content ?? (cfg as any).authPassword ?? ''),
              },
              authPemFile: {
                type: 'constant',
                content: String(ivMapGc.authPemFile?.content ?? (cfg as any).authPemFile ?? ''),
              },
              proxyUrl: {
                type: 'constant',
                content: String(ivMapGc.proxyUrl?.content ?? (cfg as any).proxyUrl ?? ''),
              },
              proxyUsername: {
                type: 'constant',
                content: String(ivMapGc.proxyUsername?.content ?? (cfg as any).proxyUsername ?? ''),
              },
              proxyPassword: {
                type: 'constant',
                content: String(ivMapGc.proxyPassword?.content ?? (cfg as any).proxyPassword ?? ''),
              },
            },
            inputs: {
              type: 'object',
              properties: {
                repository: {
                  type: 'string',
                  extra: { label: '仓库 URL', formComponent: 'prompt-editor' },
                },
                directory: { type: 'string', extra: { label: '本地目录' } },
                reference: { type: 'string', extra: { label: '分支/引用' } },
                authType: {
                  type: 'string',
                  enum: ['ssh', 'password', 'token'],
                  extra: { formComponent: 'enum-select', label: '认证类型' },
                },
                authUser: { type: 'string', extra: { label: '用户名' } },
                authPassword: {
                  type: 'string',
                  extra: { label: '密码/token', formComponent: 'prompt-editor' },
                },
                authPemFile: { type: 'string', extra: { label: 'SSH 密钥路径' } },
                proxyUrl: { type: 'string', extra: { label: '代理地址' } },
                proxyUsername: { type: 'string', extra: { label: '代理用户' } },
                proxyPassword: { type: 'string', extra: { label: '代理密码' } },
              },
            },
          };
          break;
        }
        case 'ci/gitCommit': {
          const cfg = n.configuration ?? {};
          const specGcm = getNodeMappingSpec('ci/gitCommit');
          const ivMapGcm = specGcm
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specGcm)
            : ({} as InputsValuesMap);
          let sig: Record<string, unknown> = {};
          if (
            ivMapGcm.signature?.content &&
            typeof ivMapGcm.signature.content === 'object' &&
            !Array.isArray(ivMapGcm.signature.content)
          ) {
            sig = ivMapGcm.signature.content as Record<string, unknown>;
          } else if ((cfg as any).signature && typeof (cfg as any).signature === 'object') {
            sig = (cfg as any).signature as Record<string, unknown>;
          }
          base.data = {
            title: n.name ?? 'Git提交',
            positionType: 'middle',
            inputsValues: {
              directory: {
                type: 'template',
                content: String(ivMapGcm.directory?.content ?? (cfg as any).directory ?? ''),
              },
              pattern: {
                type: 'constant',
                content: String(ivMapGcm.pattern?.content ?? (cfg as any).pattern ?? ''),
              },
              message: {
                type: 'template',
                content: String(ivMapGcm.message?.content ?? (cfg as any).message ?? ''),
              },
              signature: {
                type: 'json',
                content: {
                  authorName: String(sig.authorName ?? ''),
                  authorEmail: String(sig.authorEmail ?? ''),
                },
              },
            },
            inputs: {
              type: 'object',
              properties: {
                directory: {
                  type: 'string',
                  extra: { label: '本地仓库目录', formComponent: 'prompt-editor' },
                },
                pattern: { type: 'string', extra: { label: '添加文件 Glob' } },
                message: {
                  type: 'string',
                  extra: { label: '提交说明', formComponent: 'prompt-editor' },
                },
                signature: { type: 'object', extra: { label: '作者' } },
              },
            },
          };
          break;
        }
        case 'ci/gitPush': {
          const cfg = n.configuration ?? {};
          const specGp = getNodeMappingSpec('ci/gitPush');
          const ivMapGp = specGp
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specGp)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'Git推送',
            positionType: 'middle',
            inputsValues: {
              repository: {
                type: 'template',
                content: String(ivMapGp.repository?.content ?? (cfg as any).repository ?? ''),
              },
              directory: {
                type: 'template',
                content: String(ivMapGp.directory?.content ?? (cfg as any).directory ?? ''),
              },
              refSpecs: {
                type: 'constant',
                content: String(
                  ivMapGp.refSpecs?.content ??
                    (cfg as any).refSpecs ??
                    'refs/heads/main:refs/heads/main'
                ),
              },
              authType: {
                type: 'constant',
                content: String(ivMapGp.authType?.content ?? (cfg as any).authType ?? 'token'),
              },
              authUser: {
                type: 'constant',
                content: String(ivMapGp.authUser?.content ?? (cfg as any).authUser ?? ''),
              },
              authPassword: {
                type: 'template',
                content: String(ivMapGp.authPassword?.content ?? (cfg as any).authPassword ?? ''),
              },
              authPemFile: {
                type: 'constant',
                content: String(ivMapGp.authPemFile?.content ?? (cfg as any).authPemFile ?? ''),
              },
              proxyUrl: {
                type: 'constant',
                content: String(ivMapGp.proxyUrl?.content ?? (cfg as any).proxyUrl ?? ''),
              },
              proxyUsername: {
                type: 'constant',
                content: String(ivMapGp.proxyUsername?.content ?? (cfg as any).proxyUsername ?? ''),
              },
              proxyPassword: {
                type: 'constant',
                content: String(ivMapGp.proxyPassword?.content ?? (cfg as any).proxyPassword ?? ''),
              },
            },
            inputs: {
              type: 'object',
              properties: {
                repository: {
                  type: 'string',
                  extra: { label: '远程仓库 URL', formComponent: 'prompt-editor' },
                },
                directory: { type: 'string', extra: { label: '本地目录' } },
                refSpecs: { type: 'string', extra: { label: 'ref 映射' } },
                authType: {
                  type: 'string',
                  enum: ['ssh', 'password', 'token'],
                  extra: { formComponent: 'enum-select', label: '认证类型' },
                },
                authUser: { type: 'string', extra: { label: '用户名' } },
                authPassword: {
                  type: 'string',
                  extra: { label: '密码/token', formComponent: 'prompt-editor' },
                },
                authPemFile: { type: 'string', extra: { label: 'SSH 密钥路径' } },
                proxyUrl: { type: 'string', extra: { label: '代理地址' } },
                proxyUsername: { type: 'string', extra: { label: '代理用户' } },
                proxyPassword: { type: 'string', extra: { label: '代理密码' } },
              },
            },
          };
          break;
        }
        case 'flow': {
          const cfg = n.configuration ?? {};
          const specFlow = getNodeMappingSpec('flow');
          const ivMapFlow = specFlow
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specFlow)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'flow',
            positionType: 'middle',
            inputsValues: {
              targetId: {
                type: 'constant',
                content: String(ivMapFlow.targetId?.content ?? cfg.targetId ?? ''),
              },
              extend: {
                type: 'constant',
                content: !!(ivMapFlow.extend?.content ?? cfg.extend),
              },
            },
          };
          break;
        }
        case 'transform/multiNodeOutput': {
          const cfg = n.configuration ?? {};
          const specMulti = getNodeMappingSpec('transform/multiNodeOutput');
          const ivMapMulti = specMulti
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specMulti)
            : ({} as InputsValuesMap);
          const arr = Array.isArray(ivMapMulti.nodeIds?.content)
            ? ivMapMulti.nodeIds?.content
            : Array.isArray((cfg as any).nodeIds)
            ? (cfg as any).nodeIds
            : [];
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
                  extra: {
                    formComponent: 'node-selector-multi',
                    label: '节点列表',
                  },
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
          const { data, blocks, edges } = buildForFlowFromDsl(n);
          base.data = data;
          base.blocks = blocks;
          base.edges = edges;
          break;
        }
        case 'transform/yapi': {
          const cfg = n.configuration ?? {};
          const specYapi = getNodeMappingSpec('transform/yapi');
          const ivMapYapi = specYapi
            ? mapDslToNodeInputsValues(cfg as Record<string, unknown>, specYapi)
            : ({} as InputsValuesMap);
          base.data = {
            title: n.name ?? 'Yapi 接口',
            positionType: 'middle',
            yapiConfig: {
              baseUrl: String(ivMapYapi.baseUrl?.content ?? cfg.baseUrl ?? ''),
              userName: String(ivMapYapi.userName?.content ?? cfg.userName ?? ''),
              password: String(ivMapYapi.password?.content ?? cfg.password ?? ''),
              interfacePath: String(ivMapYapi.interfacePath?.content ?? cfg.interfacePath ?? ''),
              loginType: String(ivMapYapi.loginType?.content ?? cfg.loginType ?? 'ldap'),
            },
          };
          break;
        }
        case 'x/taskBoard': {
          const cfg = n.configuration ?? {};
          base.data = {
            title: n.name ?? 'TaskBoard',
            positionType: 'middle',
            inputsValues: {
              action: { type: 'constant', content: String(cfg.action ?? 'save') },
              name: { type: 'template', content: String(cfg.name ?? '') },
              priority: { type: 'constant', content: Number(cfg.priority ?? 0) },
              taskType: { type: 'constant', content: Number(cfg.taskType ?? 0) },
              handlerUserId: { type: 'template', content: String(cfg.handlerUserId ?? '') },
              description: { type: 'template', content: String(cfg.description ?? '') },
              taskId: { type: 'constant', content: Number(cfg.taskId ?? 0) },
              status: { type: 'constant', content: Number(cfg.status ?? 0) },
            },
            inputs: {
              type: 'object',
              required: ['action'],
              properties: {
                action: {
                  type: 'string',
                  enum: ['create', 'get', 'update', 'delete'],
                  default: { type: 'constant', content: 'create' } as any,
                  extra: {
                    label: '操作类型',
                    formComponent: 'enum-select',
                    options: [
                      { label: '创建任务', value: 'create' },
                      { label: '获取任务', value: 'get' },
                      { label: '更新任务', value: 'update' },
                      { label: '删除任务', value: 'delete' },
                    ],
                    description:
                      'create: 创建任务 | get: 获取任务 | update: 更新任务 | delete: 删除任务',
                  },
                },
                name: {
                  type: 'string',
                  extra: { label: '任务名称', formComponent: 'prompt-editor' },
                },
                priority: { type: 'number', extra: { label: '优先级' } },
                taskType: { type: 'number', extra: { label: '任务类型' } },
                handlerUserId: { type: 'string', extra: { label: '处理人用户ID' } },
                description: { type: 'string', extra: { label: '任务描述' } },
                taskId: { type: 'number', extra: { label: '任务ID' } },
                status: { type: 'number', extra: { label: '状态' } },
              },
            },
            outputs: {
              type: 'object',
              properties: {
                success: { type: 'boolean' },
                task: { type: 'object' },
                message: { type: 'string' },
              },
            },
          };
          break;
        }
        case 'x/serviceManagement': {
          const cfg = n.configuration ?? {};
          base.data = {
            title: n.name ?? 'ServiceManagement',
            positionType: 'middle',
            inputsValues: {
              action: { type: 'constant', content: String(cfg.action ?? 'save') },
              name: { type: 'template', content: String(cfg.name ?? '') },
              status: { type: 'constant', content: Number(cfg.status ?? 0) },
              volcLogServiceId: { type: 'template', content: String(cfg.volcLogServiceId ?? '') },
              gitRepoUrl: { type: 'template', content: String(cfg.gitRepoUrl ?? '') },
              description: { type: 'template', content: String(cfg.description ?? '') },
              serviceId: { type: 'constant', content: Number(cfg.serviceId ?? 0) },
            },
            inputs: {
              type: 'object',
              required: ['action'],
              properties: {
                action: {
                  type: 'string',
                  enum: ['create', 'get', 'update', 'delete'],
                  default: { type: 'constant', content: 'create' } as any,
                  extra: {
                    label: '操作类型',
                    formComponent: 'enum-select',
                    options: [
                      { label: '创建服务', value: 'create' },
                      { label: '获取服务', value: 'get' },
                      { label: '更新服务', value: 'update' },
                      { label: '删除服务', value: 'delete' },
                    ],
                    description:
                      'create: 创建服务 | get: 获取服务 | update: 更新服务 | delete: 删除服务',
                  },
                },
                name: {
                  type: 'string',
                  extra: { label: '服务名称', formComponent: 'prompt-editor' },
                },
                status: { type: 'number', extra: { label: '服务状态' } },
                volcLogServiceId: { type: 'string', extra: { label: '火山引擎日志服务ID' } },
                gitRepoUrl: { type: 'string', extra: { label: 'Git仓库URL' } },
                description: { type: 'string', extra: { label: '服务描述' } },
                serviceId: { type: 'number', extra: { label: '服务ID' } },
              },
            },
            outputs: {
              type: 'object',
              properties: {
                success: { type: 'boolean' },
                service: { type: 'object' },
                message: { type: 'string' },
              },
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

  const edges = topLevelConns.map((e: any) => {
    const sourcePortID = e.type ?? e.label ?? undefined;
    const edge: Record<string, any> = {
      sourceNodeID: String(e.fromId ?? e.from?.id ?? ''),
      targetNodeID: String(e.toId ?? e.to?.id ?? ''),
      sourcePortID,
    };
    const targetPortID = pickTargetPortIDForConnectionType(sourcePortID);
    if (targetPortID) {
      edge.targetPortID = targetPortID;
    }
    return edge;
  });

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
      const { cronNode, targetId } = buildScheduleEndpointFlowNode(ep, {
        fallbackX: startX - spacingX,
        fallbackY: startY,
      });
      nodes.unshift(cronNode);
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
