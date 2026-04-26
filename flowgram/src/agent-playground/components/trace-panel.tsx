/**
 * Trace Panel 组件
 */

import React, { useEffect, useMemo, useRef, useState } from 'react';

import { Button, ButtonGroup, Card, Empty, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { IconRefresh } from '@douyinfe/semi-icons';

import { RuntimeViewModel } from '../utils/runtime-view-model';
import { canApplyRecoveryAction, recoveryActionButtonLabel } from '../utils/recovery-actions';
import { RecoveryAction, TraceEvent } from '../../services/api-playground';

const { Text } = Typography;

const KEY_TRACE_TYPES = new Set<string>([
  'WORKFLOW_START',
  'WORKFLOW_END',
  'PLAN_SUMMARY',
  'STEP_OUTPUT',
  'TASK_ASSIGNED',
  'AGENT_ENTER_WORKER',
  'AGENT_EXIT_WORKER',
  'TOOL_CALL',
  'TOOL_RESULT',
  'ERROR',
  'HANDOFF',
  'WORKER_DELEGATED',
  'SUPERVISOR_CHECK',
  'SUPERVISOR_INTERVENE',
]);

interface TracePanelProps {
  events: TraceEvent[];
  runtimeViewModel: RuntimeViewModel;
  onRefresh?: () => void;
  onApplyRecovery?: (action: RecoveryAction) => void | Promise<void>;
  applyingRecoveryActionId?: string;
}

export const TracePanel: React.FC<TracePanelProps> = ({
  events,
  runtimeViewModel,
  onRefresh,
  onApplyRecovery,
  applyingRecoveryActionId,
}) => {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [view, setView] = useState<'trace' | 'artifacts' | 'recovery'>('trace');
  const [filter, setFilter] = useState<'key' | 'all'>('all');
  const { run, planNodes, recovery } = runtimeViewModel;

  const displayEvents = useMemo(() => {
    if (filter === 'all') {
      return events;
    }
    return events.filter((event) => KEY_TRACE_TYPES.has(event.type));
  }, [events, filter]);

  useEffect(() => {
    if (view === 'trace' && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [displayEvents.length, view]);

  return (
    <Card
      className="pg-trace-card"
      title={
        <Space spacing="tight">
          <div
            style={{
              width: 32,
              height: 32,
              borderRadius: 8,
              background: 'var(--semi-color-fill-0)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontFamily: 'var(--semi-font-family-monospace)',
              fontWeight: 600,
              fontSize: 14,
            }}
          >
            &gt;_
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Text strong style={{ fontSize: 14 }}>Runtime Context</Text>
            <Tag
              size="small"
              color={
                run.status === 'completed'
                  ? 'green'
                  : run.isWaitingRecovery
                  ? 'orange'
                  : run.status === 'failed'
                  ? 'red'
                  : 'grey'
              }
            >
              {run.label}
            </Tag>
          </div>
        </Space>
      }
      headerExtraContent={
        <Space spacing="tight">
          <ButtonGroup size="small">
            <Button
              type={view === 'trace' ? 'primary' : 'tertiary'}
              theme={view === 'trace' ? 'solid' : 'borderless'}
              onClick={() => setView('trace')}
            >
              Trace
            </Button>
            <Button
              type={view === 'artifacts' ? 'primary' : 'tertiary'}
              theme={view === 'artifacts' ? 'solid' : 'borderless'}
              onClick={() => setView('artifacts')}
            >
              Artifacts
            </Button>
            <Button
              type={view === 'recovery' ? 'primary' : 'tertiary'}
              theme={view === 'recovery' ? 'solid' : 'borderless'}
              onClick={() => setView('recovery')}
            >
              Recovery
            </Button>
          </ButtonGroup>
          {onRefresh && run.runId ? (
            <Button
              size="small"
              type="tertiary"
              icon={<IconRefresh />}
              onClick={() => void onRefresh()}
            >
              刷新
            </Button>
          ) : null}
        </Space>
      }
      style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
        borderRadius: 14,
        boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)',
      }}
      bodyStyle={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
        overflow: 'hidden',
      }}
    >
      {run.runId ? (
        <div
          style={{
            padding: '8px 12px',
            background: 'var(--semi-color-fill-0)',
            borderRadius: 10,
            marginBottom: 12,
            fontSize: 12,
            display: 'flex',
            flexWrap: 'wrap',
            gap: '4px 12px',
          }}
        >
          <Text type="tertiary">Run ID: {run.runId}</Text>
          <Text type="tertiary">步骤: {planNodes.length}</Text>
          <Text type="tertiary">产物: {runtimeViewModel.artifacts.total}</Text>
          <Text type="tertiary">恢复动作: {recovery.summary.count}</Text>
          <Text type="tertiary">错误事件: {runtimeViewModel.trace.errorCount}</Text>
        </div>
      ) : null}

      {view === 'trace' ? (
        <>
          <div style={{ marginBottom: 10 }}>
            <ButtonGroup size="small">
              <Button
                type={filter === 'key' ? 'primary' : 'tertiary'}
                theme={filter === 'key' ? 'solid' : 'borderless'}
                onClick={() => setFilter('key')}
              >
                仅关键
              </Button>
              <Button
                type={filter === 'all' ? 'primary' : 'tertiary'}
                theme={filter === 'all' ? 'solid' : 'borderless'}
                onClick={() => setFilter('all')}
              >
                全部事件
              </Button>
            </ButtonGroup>
          </div>
          <div ref={scrollRef} style={{ flex: 1, overflow: 'auto', minHeight: 120 }}>
            {displayEvents.length === 0 ? (
              <div style={{ padding: '40px 16px', textAlign: 'center' }}>
                <div
                  style={{
                    width: 56,
                    height: 56,
                    borderRadius: 16,
                    background: 'var(--semi-color-fill-0)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    margin: '0 auto 12px',
                    fontSize: 24,
                  }}
                >
                  📋
                </div>
                <Empty
                  description={
                    events.length === 0
                      ? '运行一次方案后会在这里显示事件'
                      : '当前筛选下暂无事件，可切换到「全部事件」'
                  }
                />
              </div>
            ) : (
              <div style={{ padding: '4px 2px 8px' }}>
                {displayEvents.map((event, index) => (
                  <div
                    key={event.id || index}
                    style={{
                      display: 'flex',
                      gap: 10,
                      padding: '8px 12px',
                      borderRadius: 10,
                      marginBottom: 4,
                      background: 'var(--semi-color-bg-1)',
                      fontSize: 12,
                      borderLeft: `3px solid ${leftBorderColor(event.type)}`,
                      transition: 'background 0.15s ease',
                    }}
                  >
                    <Text type="tertiary" style={{ fontSize: 10, minWidth: 64, flexShrink: 0, paddingTop: 1 }}>
                      {formatTime(event.timestamp)}
                    </Text>
                    <span style={{ fontSize: 14, flexShrink: 0, lineHeight: '18px' }}>
                      {getEventIcon(event.type)}
                    </span>
                    {event.agentId ? (
                      <Tag size="small" style={{ borderRadius: 4, flexShrink: 0 }}>
                        {event.agentId}
                      </Tag>
                    ) : null}
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <Text style={{ wordBreak: 'break-all', display: 'block', lineHeight: 1.5 }}>
                        {event.message}
                      </Text>
                      {renderSubAgentConcurrency(event)}
                    </div>
                    {event.metadata && Object.keys(event.metadata).length > 0 ? (
                      <details style={{ fontSize: 10, flexShrink: 0 }}>
                        <summary
                          style={{
                            cursor: 'pointer',
                            color: 'var(--semi-color-tertiary)',
                            borderRadius: 4,
                            padding: '2px 6px',
                            transition: 'background 0.15s ease',
                          }}
                        >
                          meta
                        </summary>
                        <pre
                          style={{
                            fontSize: 10,
                            background: 'var(--semi-color-fill-actual)',
                            padding: 8,
                            borderRadius: 8,
                            marginTop: 4,
                            overflow: 'auto',
                            maxWidth: 240,
                            border: '1px solid var(--semi-color-border)',
                          }}
                        >
                          {JSON.stringify(event.metadata, null, 2)}
                        </pre>
                      </details>
                    ) : null}
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      ) : null}

      {view === 'artifacts' ? (
        <div style={{ flex: 1, overflow: 'auto', minHeight: 120 }}>
          {planNodes.every((node) => node.artifacts.length === 0) ? (
            <div style={{ padding: '40px 16px', textAlign: 'center' }}>
              <div
                style={{
                  width: 56,
                  height: 56,
                  borderRadius: 16,
                  background: 'var(--semi-color-fill-0)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  margin: '0 auto 12px',
                  fontSize: 24,
                }}
              >
                📦
              </div>
              <Empty description="当前运行还没有结构化产物。" />
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {planNodes.map((node) => (
                <div
                  key={node.stepId}
                  style={{
                    padding: '12px 14px',
                    borderRadius: 12,
                    border: '1px solid rgba(28,31,35,0.08)',
                    background: 'var(--semi-color-bg-0)',
                    transition: 'box-shadow 0.2s ease',
                  }}
                >
                  <Space wrap style={{ marginBottom: 8 }}>
                    <Text strong>{node.name}</Text>
                    <Tag size="small" style={{ borderRadius: 4 }}>{node.kind}</Tag>
                    <Tag size="small" color={node.artifacts.length ? 'green' : 'grey'} style={{ borderRadius: 4 }}>
                      产物 {node.artifacts.length}
                    </Tag>
                  </Space>
                  {node.artifacts.length > 0 ? (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                      {node.artifacts.map((artifact) => (
                        <div
                          key={artifact.artifactId}
                          style={{
                            padding: '10px 12px',
                            borderRadius: 10,
                            background: 'var(--semi-color-fill-0)',
                            border: '1px solid rgba(28,31,35,0.04)',
                          }}
                        >
                          <Space wrap>
                            <Tag size="small" color="green" style={{ borderRadius: 4 }}>
                              {artifact.type}
                            </Tag>
                            <Text size="small">{artifact.artifactId}</Text>
                          </Space>
                          <Text
                            type="tertiary"
                            size="small"
                            style={{ display: 'block', marginTop: 6, lineHeight: 1.6 }}
                          >
                            {artifact.summary}
                          </Text>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <Text type="tertiary" size="small">
                      该步骤暂无产物。
                    </Text>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      ) : null}

      {view === 'recovery' ? (
        <div style={{ flex: 1, overflow: 'auto', minHeight: 120 }}>
          {recovery.actions.length === 0 && !runtimeViewModel.failedStep ? (
            <div style={{ padding: '40px 16px', textAlign: 'center' }}>
              <div
                style={{
                  width: 56,
                  height: 56,
                  borderRadius: 16,
                  background: 'var(--semi-color-fill-0)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  margin: '0 auto 12px',
                  fontSize: 24,
                }}
              >
                🛡️
              </div>
              <Empty description="当前运行没有恢复上下文。" />
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {runtimeViewModel.failedStep ? (
                <div
                  style={{
                    padding: '12px 14px',
                    borderRadius: 12,
                    border: '1px solid var(--semi-color-danger)',
                    background: 'var(--semi-color-danger-light-default)',
                  }}
                >
                  <Text strong style={{ display: 'block', marginBottom: 6 }}>
                    失败摘要
                  </Text>
                  <Text size="small" style={{ display: 'block', lineHeight: 1.6 }}>
                    {runtimeViewModel.failedStep.name} ({runtimeViewModel.failedStep.stepId})
                  </Text>
                  {run.failureSummary ? (
                    <Text
                      type="danger"
                      size="small"
                      style={{ display: 'block', marginTop: 6, lineHeight: 1.6 }}
                    >
                      {run.failureSummary}
                    </Text>
                  ) : null}
                </div>
              ) : null}

              {recovery.actions.map((action) => (
                <div
                  key={action.id}
                  style={{
                    padding: '12px 14px',
                    borderRadius: 12,
                    border: '1px solid var(--semi-color-warning)',
                    background: 'var(--semi-color-warning-light-default)',
                  }}
                >
                  <Space wrap>
                    <Tag color="orange">{action.type}</Tag>
                    <Text size="small">步骤: {action.stepId}</Text>
                  </Space>
                  <Text
                    type="tertiary"
                    size="small"
                    style={{ display: 'block', marginTop: 6, lineHeight: 1.6 }}
                  >
                    {action.reason}
                  </Text>
                  {onApplyRecovery ? (
                    <div style={{ marginTop: 10 }}>
                      <Button
                        size="small"
                        type="primary"
                        theme="solid"
                        loading={applyingRecoveryActionId === action.id}
                        disabled={
                          !canApplyRecoveryAction(action, run.status, applyingRecoveryActionId)
                        }
                        onClick={() => void onApplyRecovery(action)}
                      >
                        {recoveryActionButtonLabel(action)}
                      </Button>
                    </div>
                  ) : null}
                </div>
              ))}
            </div>
          )}
        </div>
      ) : null}
    </Card>
  );
};

function getEventIcon(type: string): string {
  const iconMap: Record<string, string> = {
    WORKFLOW_START: '🚀',
    PLAN_SUMMARY: '📝',
    STEP_OUTPUT: '📄',
    TASK_ASSIGNED: '📋',
    AGENT_ENTER_WORKER: '▶️',
    AGENT_EXIT_WORKER: '⏹️',
    WORKER_DELEGATED: '🔀',
    THINKING: '🤔',
    TOOL_CALL: '🔧',
    TOOL_RESULT: '📦',
    HANDOFF: '🔄',
    SUPERVISOR_CHECK: '👁️',
    SUPERVISOR_INTERVENE: '⚠️',
    ERROR: '❌',
    WORKFLOW_END: '🏁',
  };
  return iconMap[type] || '📌';
}

function leftBorderColor(type: string): string {
  const colorMap: Record<string, string> = {
    WORKFLOW_START: 'var(--semi-color-info)',
    PLAN_SUMMARY: 'var(--semi-color-info)',
    STEP_OUTPUT: 'var(--semi-color-success)',
    TASK_ASSIGNED: 'var(--semi-color-info)',
    AGENT_ENTER_WORKER: 'var(--semi-color-success)',
    AGENT_EXIT_WORKER: 'var(--semi-color-border)',
    WORKER_DELEGATED: 'var(--semi-color-warning)',
    THINKING: 'var(--semi-color-warning)',
    TOOL_CALL: 'var(--semi-color-primary)',
    TOOL_RESULT: 'var(--semi-color-primary)',
    HANDOFF: 'var(--semi-color-warning)',
    SUPERVISOR_CHECK: 'var(--semi-color-info)',
    SUPERVISOR_INTERVENE: 'var(--semi-color-danger)',
    ERROR: 'var(--semi-color-danger)',
    WORKFLOW_END: 'var(--semi-color-success)',
  };
  return colorMap[type] || 'var(--semi-color-border)';
}

function formatTime(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function renderSubAgentConcurrency(event: TraceEvent): React.ReactNode {
  if (event.type !== 'TOOL_RESULT') return null;
  const toolName = event.metadata?.toolName;
  if (toolName !== 'run_sub_agent') return null;
  const taskCount = event.metadata?.subAgentTaskCount;
  const conc = event.metadata?.subAgentEffectiveConcurrency;
  const reason = event.metadata?.subAgentConcurrencyReason;
  if (!taskCount || !conc || !reason) return null;
  return (
    <Space spacing={4} wrap>
      <Tag size="small" color="cyan">
        tasks {taskCount}
      </Tag>
      <Tag size="small" color="blue">
        conc {conc}
      </Tag>
      <Tag size="small" color="grey">
        {reason}
      </Tag>
    </Space>
  );
}
