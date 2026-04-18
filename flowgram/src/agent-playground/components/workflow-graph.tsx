/**
 * 工作流图组件 - 可视化展示协作流程
 */

import React from 'react';
import {
  Typography,
  Space,
  Tag,
} from '@douyinfe/semi-ui';
import {
  IconBranch,
} from '@douyinfe/semi-icons';

import {
  CollaborationScheme,
  TraceRun,
  TraceEvent,
  CollaborationMode,
  MODE_NAME_MAP,
} from '../../services/api-playground';

/** 优先使用轮询得到的 liveEvents，便于运行中与 getRun.events 对齐 */
function mergeFlowEvents(run?: TraceRun, live?: TraceEvent[]): TraceEvent[] {
  if (live && live.length > 0) {
    return live;
  }
  return run?.events || [];
}

/** 根据 AGENT_ENTER / AGENT_EXIT 栈推导当前正在执行的 Agent（与 Trace 顺序一致） */
function computeActiveAgentId(events: TraceEvent[]): string | undefined {
  const stack: string[] = [];
  for (const e of events) {
    if (e.type === 'AGENT_ENTER_WORKER' && e.agentId) {
      stack.push(e.agentId);
    }
    if (e.type === 'AGENT_EXIT_WORKER' && e.agentId) {
      for (let i = stack.length - 1; i >= 0; i--) {
        if (stack[i] === e.agentId) {
          stack.splice(i, 1);
          break;
        }
      }
    }
  }
  return stack.length ? stack[stack.length - 1] : undefined;
}

function computeDoneAgentIds(events: TraceEvent[]): Set<string> {
  const done = new Set<string>();
  for (const e of events) {
    if (e.type === 'AGENT_EXIT_WORKER' && e.agentId) {
      done.add(e.agentId);
    }
  }
  return done;
}

type StepStatus = 'pending' | 'active' | 'done';

function stepStatusFor(agentId: string, activeId: string | undefined, done: Set<string>, running: boolean): StepStatus {
  if (done.has(agentId)) {
    return 'done';
  }
  if (running && activeId === agentId) {
    return 'active';
  }
  return 'pending';
}

const { Text } = Typography;

interface WorkflowGraphProps {
  scheme: CollaborationScheme;
  currentRun?: TraceRun;
  /** 来自 getRunEvents 轮询，运行过程中比 currentRun.events 更及时 */
  liveEvents?: TraceEvent[];
  /** 嵌入左栏大卡片时：不再重复模式标题，图例收束为一条 */
  variant?: 'default' | 'embedded';
}

export const WorkflowGraph: React.FC<WorkflowGraphProps> = ({
  scheme,
  currentRun,
  liveEvents,
  variant = 'default',
}) => {
  const { mode, bindAgents } = scheme;

  const flowEvents = mergeFlowEvents(currentRun, liveEvents);
  const running = currentRun?.status === 'running';
  const activeAgentId = running ? computeActiveAgentId(flowEvents) : undefined;
  const runPhase: 'idle' | 'running' | 'done' =
    !currentRun ? 'idle' : running ? 'running' : 'done';

  // 渲染不同模式的图
  const renderGraph = () => {
    switch (mode) {
      case 'router_expert':
        return <RouterExpertGraph bindAgents={bindAgents} activeAgentId={activeAgentId} />;
      case 'plan_exec':
        return (
          <PlanExecGraph
            bindAgents={bindAgents}
            activeAgentId={activeAgentId}
            flowEvents={flowEvents}
            running={!!running}
            runPhase={runPhase}
          />
        );
      case 'supervision':
        return <SupervisionGraph bindAgents={bindAgents} activeAgentId={activeAgentId} />;
      case 'peer_handoff':
        return <PeerHandoffGraph bindAgents={bindAgents} activeAgentId={activeAgentId} />;
      default:
        return <Text type="tertiary">未知模式</Text>;
    }
  };

  return (
    <div style={{ padding: '4px 0 0' }}>
      <style>
        {`
        @keyframes pg-node-pulse {
          0%, 100% { box-shadow: 0 0 0 0 rgba(22, 100, 255, 0.22); }
          50% { box-shadow: 0 0 20px 2px rgba(22, 100, 255, 0.32); }
        }
        .pg-node-surface { transition: border-color 0.28s ease, background 0.28s ease, box-shadow 0.28s ease, transform 0.22s ease; }
        .pg-node-active { animation: pg-node-pulse 1.8s ease-in-out infinite; }
        .pg-node-done:hover { transform: translateY(-1px); }
      `}
      </style>
      {variant === 'default' ? (
        <Space style={{ marginBottom: 16 }}>
          <IconBranch />
          <Text strong>{MODE_NAME_MAP[mode]}</Text>
        </Space>
      ) : null}

      {/* 工作流图 */}
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 8,
      }}>
        {renderGraph()}
      </div>

      {/* 图例 */}
      <div
        style={{
          marginTop: variant === 'embedded' ? 12 : 24,
          padding: variant === 'embedded' ? '10px 12px' : 12,
          background: 'var(--semi-color-fill-actual)',
          borderRadius: variant === 'embedded' ? 10 : 8,
        }}
      >
        <Text type="tertiary" style={{ fontSize: 12, lineHeight: 1.6 }}>
          {variant === 'embedded' ? (
            <span>{MODE_NAME_MAP[mode]} · {getModeDescription(mode)}</span>
          ) : (
            <Space>
              <Tag size="small">模式说明</Tag>
              {MODE_NAME_MAP[mode]}: {getModeDescription(mode)}
            </Space>
          )}
        </Text>
      </div>
    </div>
  );
};

// ========== 不同协作模式的图示 ==========

interface GraphProps {
  bindAgents?: { agentId: string; role: string }[];
  activeAgentId?: string;
}

const RouterExpertGraph: React.FC<GraphProps> = ({ bindAgents, activeAgentId }) => (
  <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
    {/* 用户输入 */}
    <NodeBox icon="📥" label="用户输入" active={false} />

    {/* 箭头 */}
    <ArrowDown />

    {/* 路由节点 */}
    <NodeBox icon="🔀" label="路由专家" subLabel="LLM决策" active={activeAgentId === 'router'} />

    {/* 箭头 */}
    <ArrowDown />

    {/* Agent 节点 */}
    <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', justifyContent: 'center' }}>
      {(bindAgents || []).map((a, i) => (
        <NodeBox
          key={i}
          icon="👤"
          label={a.role || a.agentId}
          active={activeAgentId === a.agentId}
          style={{ minWidth: 80 }}
        />
      ))}
    </div>

    {/* 箭头 */}
    <ArrowDown />

    {/* 输出 */}
    <NodeBox icon="📤" label="输出结果" active={false} />
  </div>
);

interface PlanExecGraphProps extends GraphProps {
  flowEvents: TraceEvent[];
  running: boolean;
  runPhase: 'idle' | 'running' | 'done';
}

const PlanExecGraph: React.FC<PlanExecGraphProps> = ({
  bindAgents,
  activeAgentId,
  flowEvents,
  running,
  runPhase,
}) => {
  const doneIds = computeDoneAgentIds(flowEvents);
  const plannerDone = doneIds.has('planner');
  const plannerStatus: StepStatus = plannerDone
    ? 'done'
    : running && activeAgentId === 'planner'
      ? 'active'
      : 'pending';

  const execAgents = (bindAgents || []).filter(a => a.agentId !== 'planner');

  const outputStatus: StepStatus =
    runPhase === 'idle' ? 'pending' : runPhase === 'running' ? 'pending' : 'done';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
      <NodeBox icon="📥" label="用户输入" stepStatus="pending" />

      <ArrowDown />

      <NodeBox icon="📋" label="规划师" subLabel="任务拆解" stepStatus={plannerStatus} />

      <ArrowDown />

      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', justifyContent: 'center', alignItems: 'flex-start' }}>
        {execAgents.map((a, i, arr) => {
          const st = stepStatusFor(a.agentId, activeAgentId, doneIds, running);
          return (
            <React.Fragment key={a.agentId}>
              <NodeBox
                icon="👤"
                label={a.role || a.agentId}
                stepStatus={st}
                style={{ minWidth: 88 }}
              />
              {i < arr.length - 1 && <ArrowRight />}
            </React.Fragment>
          );
        })}
      </div>

      <ArrowDown />

      <NodeBox icon="📤" label="输出结果" stepStatus={outputStatus} />
    </div>
  );
};

const SupervisionGraph: React.FC<GraphProps> = ({ bindAgents, activeAgentId }) => (
  <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
    {/* 用户输入 */}
    <NodeBox icon="📥" label="用户输入" active={false} />

    {/* 箭头 */}
    <ArrowDown />

    {/* 监督者 */}
    <NodeBox icon="👁️" label="监督者" active={activeAgentId === 'supervisor'} />

    {/* 并行分配 */}
    <ArrowDown />
    <div style={{
      padding: '8px 16px',
      border: '1px dashed var(--semi-color-border)',
      borderRadius: 8,
      textAlign: 'center',
    }}>
      <Text type="tertiary" style={{ fontSize: 12 }}>并行分配</Text>
    </div>

    {/* Workers */}
    <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', justifyContent: 'center' }}>
      {(bindAgents || []).filter(a => a.agentId !== 'supervisor').map((a, i) => (
        <NodeBox
          key={i}
          icon="👤"
          label={a.role || a.agentId}
          active={activeAgentId === a.agentId}
          style={{ minWidth: 80 }}
        />
      ))}
    </div>

    {/* 监控回环 */}
    <div style={{
      padding: '8px 16px',
      border: '1px dashed var(--semi-color-warning)',
      borderRadius: 8,
    }}>
      <Text type="warning" style={{ fontSize: 12 }}>👁️ 监控 + 干预</Text>
    </div>

    {/* 输出 */}
    <NodeBox icon="📤" label="输出结果" active={false} />
  </div>
);

const PeerHandoffGraph: React.FC<GraphProps> = ({ bindAgents, activeAgentId }) => (
  <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
    {/* 用户输入 */}
    <NodeBox icon="📥" label="用户输入" active={false} />

    {/* 入口 Agent */}
    <NodeBox icon="👤" label={bindAgents?.[0]?.role || '入口Agent'} active={activeAgentId === bindAgents?.[0]?.agentId} />

    {/* Peer Mesh */}
    <div style={{
      padding: '12px 24px',
      border: '2px solid var(--semi-color-primary)',
      borderRadius: 12,
      background: 'var(--semi-color-fill-actual)',
    }}>
      <Text strong>🔗 Peer Mesh</Text>
      <div style={{ marginTop: 8, display: 'flex', gap: 8, justifyContent: 'center', flexWrap: 'wrap' }}>
        {(bindAgents || []).map((a, i) => (
          <NodeBox
            key={i}
            icon="👤"
            label={a.role || a.agentId}
            active={activeAgentId === a.agentId}
            style={{ minWidth: 60 }}
            small
          />
        ))}
      </div>
    </div>

    {/* 交接箭头 */}
    <Text type="tertiary" style={{ fontSize: 12 }}>🔄 自主协商 → 交接</Text>

    {/* 输出 */}
    <NodeBox icon="📤" label="输出结果" active={false} />
  </div>
);

// ========== 辅助组件 ==========

interface NodeBoxProps {
  icon: string;
  label: string;
  subLabel?: string;
  /** 兼容旧用法：仅高亮边框 */
  active?: boolean;
  /** 规划执行：pending / active / done */
  stepStatus?: StepStatus;
  style?: React.CSSProperties;
  small?: boolean;
}

const NodeBox: React.FC<NodeBoxProps> = ({ icon, label, subLabel, active, stepStatus, style, small }) => {
  const st: StepStatus =
    stepStatus ?? (active ? 'active' : 'pending');
  const border =
    st === 'active'
      ? 'var(--semi-color-primary)'
      : st === 'done'
        ? 'var(--semi-color-success)'
        : 'var(--semi-color-border)';
  const bg =
    st === 'active'
      ? 'var(--semi-color-primary-emphasis)'
      : st === 'done'
        ? 'var(--semi-color-success-light-active)'
        : 'var(--semi-color-bg-1)';
  const strongColor = st === 'active' ? 'white' : undefined;
  const badge =
    st === 'active' ? '● 进行中' : st === 'done' ? '✓ 完成' : '○ 待定';

  const pulseClass =
    st === 'active'
      ? 'pg-node-surface pg-node-active'
      : `pg-node-surface${st === 'done' ? ' pg-node-done' : ''}`;

  return (
    <div className={pulseClass}
      style={{
      padding: small ? '4px 12px' : '8px 14px',
      border: `2px solid ${border}`,
      borderRadius: 10,
      background: bg,
      textAlign: 'center',
      minWidth: 80,
      ...style,
    }}>
      <div style={{ fontSize: small ? 14 : 20 }}>{icon}</div>
      <Text strong style={{ fontSize: small ? 10 : 12, color: strongColor }}>
        {label}
      </Text>
      {subLabel && (
        <Text type="tertiary" style={{ fontSize: 10, display: 'block' }}>
          {subLabel}
        </Text>
      )}
      <div style={{
        fontSize: 10,
        marginTop: 3,
        letterSpacing: 0.3,
        color: st === 'active' ? 'rgba(255,255,255,0.92)' : 'var(--semi-color-tertiary)',
      }}>
        {badge}
      </div>
    </div>
  );
};

/** 纵向虚线箭头（示意拓扑连线） */
const ArrowDown: React.FC = () => {
  const uid = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const markerId = `wf_mk_d_${uid}`;
  return (
    <div
      className="pg-wf-edge pg-wf-edge-v"
      style={{
        display: 'flex',
        justifyContent: 'center',
        lineHeight: 0,
        color: 'var(--semi-color-tertiary)',
      }}
    >
      <svg width={28} height={42} viewBox="0 0 28 42" fill="none" aria-hidden>
        <defs>
          <marker
            id={markerId}
            markerUnits="userSpaceOnUse"
            markerWidth={9}
            markerHeight={9}
            refX={8}
            refY={4.5}
            orient="auto"
          >
            <path d="M0 0 L9 4.5 L0 9 Z" fill="currentColor" />
          </marker>
        </defs>
        <line
          x1={14}
          y1={3}
          x2={14}
          y2={29}
          stroke="currentColor"
          strokeWidth={1.35}
          strokeDasharray="5 6"
          strokeLinecap="round"
          markerEnd={`url(#${markerId})`}
        />
      </svg>
    </div>
  );
};

/** 横向虚线箭头；可选文字标注在箭头右侧 */
const ArrowRight: React.FC<{ label?: string }> = ({ label }) => {
  const uid = React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
  const markerId = `wf_mk_r_${uid}`;
  return (
    <div
      className="pg-wf-edge pg-wf-edge-h"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: label ? 6 : 0,
        color: 'var(--semi-color-tertiary)',
        flexShrink: 0,
        padding: '0 2px',
      }}
    >
      <svg width={40} height={22} viewBox="0 0 40 22" fill="none" aria-hidden>
        <defs>
          <marker
            id={markerId}
            markerUnits="userSpaceOnUse"
            markerWidth={9}
            markerHeight={9}
            refX={8}
            refY={4.5}
            orient="auto"
          >
            <path d="M0 0 L9 4.5 L0 9 Z" fill="currentColor" />
          </marker>
        </defs>
        <line
          x1={2}
          y1={11}
          x2={31}
          y2={11}
          stroke="currentColor"
          strokeWidth={1.35}
          strokeDasharray="5 6"
          strokeLinecap="round"
          markerEnd={`url(#${markerId})`}
        />
      </svg>
      {label ? (
        <span style={{ fontSize: 11, opacity: 0.88, whiteSpace: 'nowrap' }}>
          {label}
        </span>
      ) : null}
    </div>
  );
};

// 模式说明
function getModeDescription(mode: CollaborationMode): string {
  const descriptions: Record<CollaborationMode, string> = {
    router_expert: '用户输入 → LLM 分析意图 → 路由到最合适的单个 Agent → 返回结果',
    plan_exec: '用户输入 → 规划师拆解任务 → 按顺序分配给子 Agent → 串联执行',
    supervision: '用户输入 → 监督者并行分配 → Workers 执行 → 监督者实时监控干预',
    peer_handoff: '用户输入 → 入口 Agent 处理 → 自主协商交接 → Peer Mesh 协作 → 完成',
  };
  return descriptions[mode] || '';
}
