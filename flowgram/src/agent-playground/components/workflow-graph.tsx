import React from 'react';

import { Space, Tag, Typography } from '@douyinfe/semi-ui';
import { IconBranch } from '@douyinfe/semi-icons';

import { RuntimePlanNode, RuntimeViewModel } from '../utils/runtime-view-model';
import {
  CollaborationMode,
  CollaborationScheme,
  MODE_NAME_MAP,
} from '../../services/api-playground';

const { Text } = Typography;

interface WorkflowGraphProps {
  scheme: CollaborationScheme;
  runtimeViewModel?: RuntimeViewModel;
  variant?: 'default' | 'embedded';
}

type FallbackNode = Pick<RuntimePlanNode, 'stepId' | 'kind' | 'name' | 'agentBinding'>;

const STATUS_META: Record<
  RuntimePlanNode['status'],
  { label: string; color: string; background: string }
> = {
  pending: {
    label: '待执行',
    color: 'var(--semi-color-border)',
    background: 'var(--semi-color-fill-0)',
  },
  active: {
    label: '执行中',
    color: 'var(--semi-color-primary)',
    background: 'var(--semi-color-primary-light-default)',
  },
  completed: {
    label: '已完成',
    color: 'var(--semi-color-success)',
    background: 'var(--semi-color-success-light-default)',
  },
  failed: {
    label: '失败',
    color: 'var(--semi-color-danger)',
    background: 'var(--semi-color-danger-light-default)',
  },
  skipped: {
    label: '已跳过',
    color: 'var(--semi-color-warning)',
    background: 'var(--semi-color-warning-light-default)',
  },
};

const KIND_ICON_MAP: Record<RuntimePlanNode['kind'], string> = {
  route: '🔀',
  agent: '👤',
  parallel: '⛓️',
  review: '👁️',
  handoff: '🔄',
  finalize: '📤',
};

const KIND_LABEL_MAP: Record<RuntimePlanNode['kind'], string> = {
  route: 'Route',
  agent: 'Agent',
  parallel: 'Parallel',
  review: 'Review',
  handoff: 'Handoff',
  finalize: 'Finalize',
};

export const WorkflowGraph: React.FC<WorkflowGraphProps> = ({
  scheme,
  runtimeViewModel,
  variant = 'default',
}) => {
  const runtimeNodes = runtimeViewModel?.planNodes ?? [];
  const nodes =
    runtimeNodes.length > 0 ? runtimeNodes : buildFallbackNodes(scheme.mode, scheme.bindAgents);

  return (
    <div style={{ padding: '4px 0 0' }}>
      {variant === 'default' ? (
        <Space style={{ marginBottom: 16 }}>
          <IconBranch />
          <Text strong>{MODE_NAME_MAP[scheme.mode]}</Text>
        </Space>
      ) : null}

      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {nodes.map((node, index) => (
          <React.Fragment key={node.stepId}>
            <PlanNodeCard node={node} runtimeEnabled={runtimeNodes.length > 0} />
            {index < nodes.length - 1 ? <ArrowDown /> : null}
          </React.Fragment>
        ))}
      </div>

      <div
        style={{
          marginTop: variant === 'embedded' ? 12 : 24,
          padding: variant === 'embedded' ? '10px 12px' : 12,
          background: 'var(--semi-color-fill-actual)',
          borderRadius: variant === 'embedded' ? 10 : 8,
        }}
      >
        <Text type="tertiary" style={{ fontSize: 12, lineHeight: 1.6 }}>
          {runtimeNodes.length > 0 ? (
            <>
              {MODE_NAME_MAP[scheme.mode]} · 共 {runtimeNodes.length} 个运行步骤，已产出{' '}
              {runtimeViewModel?.artifacts.total ?? 0} 个 产物，恢复动作{' '}
              {runtimeViewModel?.recovery.summary.count ?? 0} 个。
            </>
          ) : (
            <>
              {MODE_NAME_MAP[scheme.mode]} · {getModeDescription(scheme.mode)}
            </>
          )}
        </Text>
      </div>
    </div>
  );
};

const PlanNodeCard: React.FC<{
  node: RuntimePlanNode | FallbackNode;
  runtimeEnabled: boolean;
}> = ({ node, runtimeEnabled }) => {
  const runtimeNode = 'status' in node ? node : undefined;
  const meta = runtimeNode ? STATUS_META[runtimeNode.status] : STATUS_META.pending;

  return (
    <div
      style={{
        border: `1px solid ${meta.color}`,
        borderRadius: 12,
        padding: '12px 14px',
        background: meta.background,
        boxShadow: runtimeNode?.isCurrent ? '0 0 0 2px rgba(22, 100, 255, 0.12)' : 'none',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          gap: 12,
        }}
      >
        <Space align="start" spacing="tight">
          <span style={{ fontSize: 18, lineHeight: '22px' }}>{KIND_ICON_MAP[node.kind]}</span>
          <div>
            <Text strong style={{ display: 'block', lineHeight: 1.5 }}>
              {node.name}
            </Text>
            <Text type="tertiary" size="small">
              {node.stepId}
            </Text>
          </div>
        </Space>
        <Tag
          color={
            runtimeNode?.status === 'failed'
              ? 'red'
              : runtimeNode?.status === 'completed'
              ? 'green'
              : 'grey'
          }
        >
          {runtimeEnabled ? meta.label : '模式骨架'}
        </Tag>
      </div>

      <Space wrap style={{ marginTop: 10 }}>
        <Tag size="small">{KIND_LABEL_MAP[node.kind]}</Tag>
        {node.agentBinding ? (
          <Tag size="small" color="blue">
            {node.agentBinding}
          </Tag>
        ) : null}
        {runtimeNode ? <Tag size="small">原始状态: {runtimeNode.runtimeStatus}</Tag> : null}
        {runtimeNode?.artifacts.length ? (
          <Tag size="small" color="green">
            产物 {runtimeNode.artifacts.length}
          </Tag>
        ) : null}
        {runtimeNode?.recoveryActions.length ? (
          <Tag size="small" color="orange">
            恢复 {runtimeNode.recoveryActions.length}
          </Tag>
        ) : null}
      </Space>

      {runtimeNode?.failureSummary ? (
        <Text
          type="danger"
          size="small"
          style={{ display: 'block', marginTop: 10, lineHeight: 1.6 }}
        >
          {runtimeNode.failureSummary}
        </Text>
      ) : null}
    </div>
  );
};

const ArrowDown: React.FC = () => (
  <div style={{ display: 'flex', justifyContent: 'center', color: 'var(--semi-color-tertiary)' }}>
    <div
      style={{
        width: 1,
        height: 18,
        borderLeft: '1px dashed currentColor',
        position: 'relative',
      }}
    >
      <span
        style={{
          position: 'absolute',
          bottom: -2,
          left: -4,
          fontSize: 10,
        }}
      >
        ▼
      </span>
    </div>
  </div>
);

function buildFallbackNodes(
  mode: CollaborationMode,
  bindAgents?: { agentId: string; role: string }[]
): FallbackNode[] {
  const roles = bindAgents?.map((agent) => agent.role || agent.agentId) ?? [];
  switch (mode) {
    case 'router_expert':
      return [
        { stepId: 'route', kind: 'route', name: '路由决策' },
        { stepId: 'agent', kind: 'agent', name: roles[0] || '目标 Agent', agentBinding: roles[0] },
        { stepId: 'finalize', kind: 'finalize', name: '结果整理' },
      ];
    case 'plan_exec':
      return [
        { stepId: 'planner', kind: 'agent', name: roles[0] || '规划师', agentBinding: roles[0] },
        {
          stepId: 'execution',
          kind: 'agent',
          name: roles.slice(1).join(' / ') || '顺序执行',
          agentBinding: roles[1],
        },
        { stepId: 'finalize', kind: 'finalize', name: '结果整理' },
      ];
    case 'supervision':
      return [
        { stepId: 'assign', kind: 'review', name: roles[0] || '监督分工', agentBinding: roles[0] },
        {
          stepId: 'parallel',
          kind: 'parallel',
          name: '并行执行',
          agentBinding: roles.slice(1).join(' / '),
        },
        { stepId: 'review', kind: 'review', name: '监督复核', agentBinding: roles[0] },
        { stepId: 'finalize', kind: 'finalize', name: '结果整理' },
      ];
    case 'peer_handoff':
      return [
        { stepId: 'entry', kind: 'agent', name: roles[0] || '入口 Agent', agentBinding: roles[0] },
        { stepId: 'handoff', kind: 'handoff', name: '交接决策' },
        { stepId: 'finalize', kind: 'finalize', name: '结果整理' },
      ];
    default:
      return [];
  }
}

function getModeDescription(mode: CollaborationMode): string {
  const descriptions: Record<CollaborationMode, string> = {
    router_expert: '根据输入先做路由，再由单个 Agent 执行并整理输出。',
    plan_exec: '先拆解任务，再按顺序推进多个 Agent 步骤。',
    supervision: '由监督者分工、并行执行，再统一复核。',
    peer_handoff: '由入口 Agent 启动，并在多 Agent 间自主交接。',
  };
  return descriptions[mode];
}
