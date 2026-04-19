import React, { useRef, useState } from 'react';
import { IconClear, IconCopy, IconSend, IconUndo } from '@douyinfe/semi-icons';
import { Button, Card, Checkbox, Space, Spin, Tag, TextArea, Toast, Typography } from '@douyinfe/semi-ui';

import { CollaborationScheme, MODE_NAME_MAP, RecoveryAction } from '../../services/api-playground';
import { RuntimeViewModel } from '../utils/runtime-view-model';
import { canApplyRecoveryAction, recoveryActionButtonLabel } from '../utils/recovery-actions';

const { Text } = Typography;

/** 上一轮已结束的运行摘要，用于下一轮输入时附带上下文（避免新 run 丢失 Bug 语义） */
export interface PreviousRunSnapshot {
  runId: string;
  userInput: string;
  finalOutput: string;
  runStatus: string;
}

interface RunConsoleProps {
  scheme?: CollaborationScheme;
  onRun: (input: string) => void;
  onClear?: () => void;
  onApplyRecovery?: (action: RecoveryAction) => void | Promise<void>;
  applyingRecoveryActionId?: string;
  running: boolean;
  runtimeViewModel: RuntimeViewModel;
  /** 存在上一轮快照时，是否在发送时自动拼入完整上下文 */
  attachPreviousRunContext?: boolean;
  onAttachPreviousRunContextChange?: (value: boolean) => void;
  previousRunSnapshot?: PreviousRunSnapshot | null;
}

const STATUS_COLOR_MAP: Record<RuntimeViewModel['run']['status'], 'blue' | 'green' | 'red' | 'orange' | 'grey'> = {
  idle: 'grey',
  pending: 'blue',
  ready: 'blue',
  running: 'blue',
  waiting_recovery: 'orange',
  completed: 'green',
  failed: 'red',
  cancelled: 'grey',
};

export const RunConsole: React.FC<RunConsoleProps> = ({
  scheme,
  onRun,
  onClear,
  onApplyRecovery,
  applyingRecoveryActionId,
  running,
  runtimeViewModel,
  attachPreviousRunContext = true,
  onAttachPreviousRunContextChange,
  previousRunSnapshot,
}) => {
  const [input, setInput] = useState('');
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const { run, activeStep, failedStep, recovery, planNodes } = runtimeViewModel;

  const handleSubmit = () => {
    if (!input.trim() || !scheme) {
      return;
    }
    onRun(input);
    setInput('');
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleClearAll = () => {
    setInput('');
    onClear?.();
  };

  const showIdleHero = !running && run.status === 'idle';

  return (
    <Card
      className="pg-run-console"
      title={
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', width: '100%', gap: 12 }}>
          <Space spacing="tight">
            <span style={{ fontSize: 18 }} role="img" aria-label="run">
              💬
            </span>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Text strong style={{ fontSize: 15, lineHeight: '20px' }}>
                Run / Recovery
              </Text>
              {scheme ? (
                <Text type="tertiary" size="small" style={{ maxWidth: 260 }} ellipsis={{ showTooltip: true }}>
                  {scheme.name} · {MODE_NAME_MAP[scheme.mode]}
                </Text>
              ) : (
                <Text type="tertiary" size="small">
                  请先在左侧选择方案
                </Text>
              )}
            </div>
          </Space>
          <Button
            size="small"
            type="tertiary"
            icon={<IconUndo />}
            disabled={!scheme && run.status === 'idle' && !input}
            onClick={handleClearAll}
          >
            清空
          </Button>
        </div>
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
        paddingBottom: 12,
      }}
    >
      {run.status !== 'idle' ? (
        <div
          style={{
            padding: '12px 14px',
            marginBottom: 12,
            borderRadius: 12,
            border: `1px solid ${
              run.status === 'completed'
                ? 'var(--semi-color-success)'
                : run.status === 'waiting_recovery'
                  ? 'var(--semi-color-warning)'
                  : run.status === 'failed'
                    ? 'var(--semi-color-danger)'
                    : 'rgba(22, 100, 255, 0.35)'
            }`,
            background:
              run.status === 'completed'
                ? 'var(--semi-color-success-light-default)'
                : run.status === 'waiting_recovery'
                  ? 'var(--semi-color-warning-light-default)'
                  : run.status === 'failed'
                    ? 'var(--semi-color-danger-light-default)'
                    : 'var(--semi-color-primary-light-default)',
          }}
        >
          <Space vertical align="start" spacing="tight" style={{ width: '100%' }}>
            <div
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: 12,
              }}
            >
              <Space spacing="tight">
                {running ? <Spin size="small" /> : null}
                <Text strong>{run.label}</Text>
                <Tag color={STATUS_COLOR_MAP[run.status]}>{run.status}</Tag>
              </Space>
              {run.finalOutput ? (
                <Button
                  size="small"
                  type="tertiary"
                  theme="borderless"
                  icon={<IconCopy />}
                  onClick={() => {
                    void navigator.clipboard.writeText(run.finalOutput).then(
                      () => Toast.success({ content: '已复制最终输出', duration: 2 }),
                      () => Toast.warning({ content: '复制失败，请手动复制', duration: 3 }),
                    );
                  }}
                >
                  复制结果
                </Button>
              ) : null}
            </div>

            <Space wrap>
              {activeStep ? <Tag color="blue">当前步骤: {activeStep.name}</Tag> : null}
              {failedStep ? <Tag color="red">失败步骤: {failedStep.name}</Tag> : null}
              <Tag>步骤数: {planNodes.length}</Tag>
              <Tag>恢复动作: {recovery.summary.count}</Tag>
            </Space>

            {run.failureSummary ? (
              <Text
                type={run.isWaitingRecovery || run.status === 'failed' ? 'danger' : 'tertiary'}
                size="small"
                style={{ lineHeight: 1.6 }}
              >
                {run.failureSummary}
              </Text>
            ) : null}
          </Space>
        </div>
      ) : null}

      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          gap: 12,
          overflow: 'auto',
          minHeight: 160,
          padding: showIdleHero ? '24px 16px' : '0 0 8px',
        }}
      >
        {showIdleHero ? (
          <div style={{ textAlign: 'center', maxWidth: 360, margin: 'auto' }}>
            <div style={{ fontSize: 52, lineHeight: 1.2, marginBottom: 12, opacity: 0.88 }} aria-hidden>
              ✈️
            </div>
            <Text strong style={{ fontSize: 16, display: 'block', marginBottom: 8 }}>
              开始一次运行
            </Text>
            <Text type="tertiary" size="small" style={{ lineHeight: 1.65 }}>
              输入任务后，页面会围绕 Plan、Run 和 Recovery 三类信息同步更新。
            </Text>
          </div>
        ) : null}

        {run.finalOutput ? (
          <div
            style={{
              padding: '14px 16px',
              border: '1px solid rgba(28,31,35,0.08)',
              borderRadius: 12,
              background: 'var(--semi-color-bg-0)',
            }}
          >
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              最终输出
            </Text>
            <div
              style={{
                maxHeight: 'min(36vh, 320px)',
                overflow: 'auto',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                lineHeight: 1.65,
                fontSize: 13,
              }}
            >
              {run.finalOutput}
            </div>
          </div>
        ) : null}

        {failedStep || recovery.actions.length > 0 ? (
          <div
            style={{
              padding: '14px 16px',
              border: '1px solid var(--semi-color-warning)',
              borderRadius: 12,
              background: 'var(--semi-color-warning-light-default)',
            }}
          >
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              Recovery
            </Text>
            {failedStep ? (
              <Text size="small" style={{ display: 'block', lineHeight: 1.6 }}>
                失败步骤：{failedStep.name}
                {failedStep.agentBinding ? ` · ${failedStep.agentBinding}` : ''}
              </Text>
            ) : null}
            {run.failureSummary ? (
              <Text type="danger" size="small" style={{ display: 'block', lineHeight: 1.6, marginTop: 6 }}>
                {run.failureSummary}
              </Text>
            ) : null}
            {recovery.actions.length > 0 ? (
              <div style={{ marginTop: 10, display: 'flex', flexDirection: 'column', gap: 8 }}>
                {recovery.actions.map(action => (
                  <div
                    key={action.id}
                    style={{
                      padding: '10px 12px',
                      borderRadius: 10,
                      background: 'rgba(255, 255, 255, 0.72)',
                      border: '1px solid rgba(244, 114, 23, 0.18)',
                    }}
                  >
                    <Space wrap>
                      <Tag color="orange">{action.type}</Tag>
                      <Text size="small">目标步骤: {action.stepId}</Text>
                    </Space>
                    <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 6, lineHeight: 1.6 }}>
                      {action.reason}
                    </Text>
                    <div style={{ marginTop: 10 }}>
                      <Button
                        size="small"
                        type="primary"
                        theme="solid"
                        loading={applyingRecoveryActionId === action.id}
                        disabled={!canApplyRecoveryAction(action, run.status, applyingRecoveryActionId)}
                        onClick={() => void onApplyRecovery?.(action)}
                      >
                        {recoveryActionButtonLabel(action)}
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <Text type="tertiary" size="small">
                暂无可展示的恢复动作。
              </Text>
            )}
          </div>
        ) : null}
      </div>

      <div style={{ flexShrink: 0, marginTop: 'auto' }}>
        {previousRunSnapshot && onAttachPreviousRunContextChange != null ? (
          <div style={{ marginBottom: 8 }}>
            <Checkbox
              checked={attachPreviousRunContext}
              onChange={e => onAttachPreviousRunContextChange(!!(e?.target as HTMLInputElement).checked)}
            >
              <Text size="small">
                附带上一轮运行上下文（runId、原任务、产出摘录），便于反馈 Bug；关闭则仅发送下方正文（新区间可能无法理解上一轮）
              </Text>
            </Checkbox>
          </div>
        ) : null}
        <TextArea
          ref={inputRef as never}
          value={input}
          onChange={value => setInput(value)}
          onKeyDown={handleKeyDown}
          placeholder={scheme ? '输入测试问题…' : '请先在左侧选择协作方案'}
          disabled={!scheme || running}
          autosize={{ minRows: 1, maxRows: 5 }}
          style={{
            resize: 'none',
            fontFamily: 'var(--semi-font-family-regular)',
            borderRadius: 12,
          }}
        />
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 10 }}>
          <Text type="tertiary" size="small">
            Ctrl / ⌘ + Enter 发送
          </Text>
          <Space>
            {scheme && input && !running ? (
              <Button type="tertiary" size="small" icon={<IconClear />} onClick={() => setInput('')}>
                清除文本
              </Button>
            ) : null}
            <Button
              type="primary"
              theme="solid"
              icon={<IconSend />}
              disabled={!scheme || !input.trim() || running}
              loading={running}
              onClick={handleSubmit}
            >
              {running ? '运行中…' : '发送'}
            </Button>
          </Space>
        </div>
      </div>
    </Card>
  );
};
