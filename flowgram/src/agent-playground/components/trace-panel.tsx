/**
 * Trace 面板组件 - 实时展示运行日志
 */

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Typography,
  Space,
  Tag,
  Card,
  Empty,
  Button,
  ButtonGroup,
} from '@douyinfe/semi-ui';
import { IconRefresh } from '@douyinfe/semi-icons';

import {
  TraceRun,
  TraceEvent,
} from '../../services/api-playground';

const { Text } = Typography;

/** 仅关键：保留里程碑与异常，折叠细粒度思考/工具流 */
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
  run?: TraceRun;
  onRefresh?: () => void;
}

export const TracePanel: React.FC<TracePanelProps> = ({ events, run, onRefresh }) => {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [filter, setFilter] = useState<'key' | 'all'>('all');

  const displayEvents = useMemo(() => {
    if (filter === 'all') {
      return events;
    }
    return events.filter(e => KEY_TRACE_TYPES.has(e.type));
  }, [events, filter]);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [displayEvents.length]);

  const getEventIcon = (type: string): string => {
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
  };

  const leftBorderColor = (type: string): string => {
    const colorMap: Record<string, string> = {
      WORKFLOW_START: 'blue',
      PLAN_SUMMARY: 'cyan',
      STEP_OUTPUT: 'green',
      TASK_ASSIGNED: 'cyan',
      AGENT_ENTER_WORKER: 'green',
      AGENT_EXIT_WORKER: 'grey',
      WORKER_DELEGATED: 'amber',
      THINKING: 'yellow',
      TOOL_CALL: 'purple',
      TOOL_RESULT: 'violet',
      HANDOFF: 'orange',
      SUPERVISOR_CHECK: 'blue',
      SUPERVISOR_INTERVENE: 'red',
      ERROR: 'red',
      WORKFLOW_END: 'green',
    };
    const key = colorMap[type] || 'grey';
    const css: Record<string, string> = {
      blue: 'var(--semi-color-info)',
      cyan: 'var(--semi-color-info)',
      green: 'var(--semi-color-success)',
      grey: 'var(--semi-color-border)',
      amber: 'var(--semi-color-warning)',
      yellow: 'var(--semi-color-warning)',
      purple: 'var(--semi-color-primary)',
      violet: 'var(--semi-color-primary)',
      orange: 'var(--semi-color-warning)',
      red: 'var(--semi-color-danger)',
    };
    return css[key] || 'var(--semi-color-border)';
  };

  const formatTime = (timestamp: number): string => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  };

  const realtime = run?.status === 'running';

  return (
    <Card
      className="pg-trace-card"
      title={
        <Space spacing="tight">
          <span
            style={{
              fontFamily: 'var(--semi-font-family-monospace)',
              fontWeight: 600,
              letterSpacing: '-0.02em',
            }}
          >
            {'>_ Trace'}
          </span>
          {realtime ? (
            <Tag color="green" size="small">
              实时
            </Tag>
          ) : null}
          {run?.status === 'completed' ? (
            <Tag color="grey" size="small">
              已完成
            </Tag>
          ) : null}
          {run?.status === 'failed' ? (
            <Tag color="red" size="small">
              失败
            </Tag>
          ) : null}
        </Space>
      }
      headerExtraContent={
        <Space spacing="tight">
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
          {onRefresh && run?.runId ? (
            <Button size="small" type="tertiary" icon={<IconRefresh />} onClick={() => void onRefresh?.()}>
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
      bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, overflow: 'hidden' }}
    >
      {run ? (
        <div
          style={{
            padding: '8px 12px',
            background: 'var(--semi-color-fill-0)',
            borderRadius: 10,
            marginBottom: 12,
            fontSize: 12,
          }}
        >
          <Space wrap>
            <Text type="tertiary">Run ID: {run.runId}</Text>
            {run.totalMs > 0 ? <Text type="tertiary">耗时: {run.totalMs}ms</Text> : null}
            <Text type="tertiary">
              事件: {displayEvents.length}
              {filter === 'key' && events.length !== displayEvents.length ? ` / 共 ${events.length}` : ''}
            </Text>
          </Space>
        </div>
      ) : null}

      <div ref={scrollRef} style={{ flex: 1, overflow: 'auto', minHeight: 120 }}>
        {displayEvents.length === 0 ? (
          <Empty
            description={
              events.length === 0
                ? '运行一次方案后会在这里显示事件'
                : '当前筛选下暂无事件，可切换到「全部事件」'
            }
          />
        ) : (
          <div style={{ padding: '4px 2px 8px' }}>
            {displayEvents.map((event, index) => (
              <div
                key={event.id || index}
                style={{
                  display: 'flex',
                  gap: 8,
                  padding: '8px 10px',
                  borderRadius: 10,
                  marginBottom: 6,
                  background: 'var(--semi-color-bg-1)',
                  fontSize: 12,
                  borderLeft: `3px solid ${leftBorderColor(event.type)}`,
                  transition: 'background 0.2s ease',
                }}
              >
                <Text type="tertiary" style={{ fontSize: 10, minWidth: 70, flexShrink: 0 }}>
                  {formatTime(event.timestamp)}
                </Text>

                <span style={{ fontSize: 12, flexShrink: 0 }}>{getEventIcon(event.type)}</span>

                {event.agentId ? (
                  <Tag size="small" style={{ fontSize: 10, flexShrink: 0 }}>
                    {event.agentId}
                  </Tag>
                ) : null}

                <Text style={{ flex: 1, wordBreak: 'break-all' }}>{event.message}</Text>

                {event.metadata && Object.keys(event.metadata).length > 0 ? (
                  <details style={{ fontSize: 10, flexShrink: 0 }}>
                    <summary style={{ cursor: 'pointer', color: 'var(--semi-color-tertiary)' }}>...</summary>
                    <pre
                      style={{
                        fontSize: 10,
                        background: 'var(--semi-color-fill-actual)',
                        padding: 6,
                        borderRadius: 6,
                        marginTop: 4,
                        overflow: 'auto',
                        maxWidth: 220,
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

      <div
        style={{
          padding: '10px 12px',
          borderTop: '1px solid var(--semi-color-border)',
          marginTop: 'auto',
          fontSize: 10,
        }}
      >
        <Text type="tertiary">类型速览：</Text>
        <Space wrap style={{ marginTop: 6 }}>
          {['WORKFLOW_START', 'TASK_ASSIGNED', 'THINKING', 'TOOL_CALL', 'HANDOFF', 'WORKFLOW_END'].map(type => (
            <Tag key={type} size="small" style={{ fontSize: 9 }}>
              {getEventIcon(type)} {type}
            </Tag>
          ))}
        </Space>
      </div>
    </Card>
  );
};
