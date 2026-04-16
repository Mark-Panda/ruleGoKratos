/**
 * 结构类节点：子图、块、入口、定时 endpoint 等与「纯字段 spec」分离的映射逻辑。
 * 由 rulechain-builder 调用，避免与 field-mapping engine 混在同一套 spec 里。
 */

/** 与 rulechain-builder 中 RuleNodeRC 对齐的最小形状 */
export interface RuleNodeRC {
  id: string;
  additionalInfo?: Record<string, any>;
  type: string;
  name: string;
  debugMode: boolean;
  configuration: Record<string, any>;
}

export interface NodeConnectionRC {
  fromId: string;
  toId: string;
  type: string;
  label?: string;
}

export type EndpointDsl = RuleNodeRC & {
  processors?: string[];
  routers?: any[];
};

/** toDSL：不参与 metadata.nodes 输出的画布节点类型 */
export function shouldSkipRuleChainMetaNode(nodeType: string): boolean {
  return nodeType === 'endpoint/schedule' || nodeType === 'block-start' || nodeType === 'block-end';
}

/** toDSL：从画布扁平节点列表生成 endpoint/schedule 的 metadata.endpoints */
export function buildScheduleEndpointsFromDocument(
  flattened: any[],
  baseOverride: { id?: string } | undefined,
  nanoid: (size: number) => string
): EndpointDsl[] {
  const endpointTypes = new Set(['endpoint/schedule']);
  return flattened
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
      if (nodeType === 'endpoint/schedule' && n.data?.inputs && n.data?.inputsValues) {
        base.routers = [
          {
            id: nanoid(16),
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
      return base;
    });
}

/** toDSL：group -> groupAction + nodeIds */
export function emitGroupToRuleChain(n: any, base: RuleNodeRC): void {
  if (n.data) {
    base.configuration = { nodeIds: n.data?.blockIDs };
    base.type = 'groupAction';
  }
}

/** toDSL：for 子图与内边连接 */
export function emitForToRuleChain(
  n: any,
  base: RuleNodeRC,
  nodesRC: RuleNodeRC[],
  connectionsRC: NodeConnectionRC[],
  visitChild: (child: any) => void,
  buildMetaConnection: (e: any) => NodeConnectionRC | null
): void {
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
        visitChild(b);
      }
    }
    (base.configuration as any).extra.blocks = forBlocks;
  }
  if (n.edges && n.edges.length > 0) {
    const forEdges: any[] = [];
    for (const e of n.edges) {
      const sourceId = e.sourceNodeID ?? '';
      const targetId = e.targetNodeID ?? '';
      forEdges.push(e);
      if (
        !String(sourceId).startsWith('block_start') &&
        !String(targetId).startsWith('block_end')
      ) {
        const connection = buildMetaConnection(e);
        if (connection) {
          connectionsRC.push(connection);
        }
      }
    }
    (base.configuration as any).extra.edges = forEdges;
  }
}

/** fromDSL：start 节点 data */
export function buildStartFlowData(n: any): Record<string, any> {
  return {
    title: n.name ?? 'Start',
  };
}

export interface ScheduleEndpointFlowResult {
  cronNode: any;
  targetId?: string;
}

/** fromDSL：metadata.endpoints 中 endpoint/schedule -> 画布节点 + 目标节点 id（用于补边） */
export function buildScheduleEndpointFlowNode(
  ep: any,
  opts: { fallbackX: number; fallbackY: number }
): ScheduleEndpointFlowResult {
  const { fallbackX, fallbackY } = opts;
  const cron = ep?.routers?.[0]?.from?.path ?? '';
  const toPath = ep?.routers?.[0]?.to?.path ?? '';
  const pos = (ep?.additionalInfo as any)?.meta?.position;
  const x = typeof pos?.x === 'number' ? pos.x : fallbackX;
  const y = typeof pos?.y === 'number' ? pos.y : fallbackY;
  const cronNode: any = {
    id: String(ep.id ?? `cron_${Math.random().toString(36).slice(2, 8)}`),
    type: 'endpoint/schedule',
    meta: { position: { x, y } },
    data: {
      title: ep.name ?? '定时任务',
      positionType: 'header',
      inputsValues: {
        cron: {
          type: 'constant',
          content: String(cron ?? '*/10 * * * * *'),
        },
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
  const targetId =
    typeof toPath === 'string' && toPath.includes(':') ? toPath.split(':')[1] : undefined;
  return { cronNode, targetId };
}

/** fromDSL：for 节点 data / blocks / edges */
export function buildForFlowFromDsl(n: any): {
  data: Record<string, any>;
  blocks: any[];
  edges: any[];
} {
  const cfg = n.configuration ?? {};
  const data: Record<string, any> = {
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
    const targetId = String(cfg.do ?? '') || String(data?.nodeId?.content ?? '');
    if (bs && targetId) {
      innerEdges = [
        { sourceNodeID: String(bs.id), targetNodeID: targetId },
        be ? { sourceNodeID: targetId, targetNodeID: String(be.id) } : undefined,
      ].filter(Boolean) as any[];
    }
  }
  return { data, blocks, edges: innerEdges };
}
