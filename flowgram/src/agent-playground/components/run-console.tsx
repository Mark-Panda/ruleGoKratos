/**
 * 运行控制台组件
 */

import React, { useRef, useState } from 'react';
import {
  Typography,
  TextArea,
  Button,
  Space,
  Card,
  Spin,
  Toast,
} from '@douyinfe/semi-ui';
import {
  IconSend,
  IconClear,
  IconUndo,
  IconCopy,
} from '@douyinfe/semi-icons';

import {
  CollaborationScheme,
  TraceRun,
  MODE_NAME_MAP,
} from '../../services/api-playground';

const { Text } = Typography;

interface RunConsoleProps {
  scheme?: CollaborationScheme;
  onRun: (input: string) => void;
  /** 清空当前运行轨迹与本地输入（顶栏「清空」） */
  onClear?: () => void;
  running: boolean;
  currentRun?: TraceRun;
}

export const RunConsole: React.FC<RunConsoleProps> = ({
  scheme,
  onRun,
  onClear,
  running,
  currentRun,
}) => {
  const [input, setInput] = useState('');
  const inputRef = useRef<HTMLTextAreaElement>(null);

  const handleSubmit = () => {
    if (!input.trim() || !scheme) return;
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

  const showIdleHero =
    !running &&
    !currentRun;

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
                运行
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
            disabled={!scheme && !currentRun && !input}
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
      bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, overflow: 'hidden', paddingBottom: 12 }}
    >
      <style>
        {`
        @keyframes pg-run-shimmer {
          0% { background-position: 0% 50%; }
          100% { background-position: 200% 50%; }
        }
      `}
      </style>

      {running ? (
        <div
          style={{
            padding: '12px 14px',
            marginBottom: 12,
            borderRadius: 12,
            border: '1px solid rgba(22, 100, 255, 0.35)',
            background:
              'linear-gradient(110deg, var(--semi-color-primary-light-active) 0%, var(--semi-color-fill-0) 40%, var(--semi-color-primary-light-active) 80%)',
            backgroundSize: '200% 100%',
            animation: 'pg-run-shimmer 2.4s ease-in-out infinite',
          }}
        >
          <Space>
            <Spin size="small" />
            <Text strong>运行中</Text>
            <Text type="tertiary" size="small">
              路由、示意图与 Trace 将同步更新
            </Text>
          </Space>
        </div>
      ) : null}

      {currentRun && currentRun.status !== 'running' ? (
        <div
          style={{
            padding: '14px 16px',
            background:
              currentRun.status === 'completed'
                ? 'var(--semi-color-success-light-active)'
                : 'var(--semi-color-danger-light-active)',
            borderRadius: 12,
            marginBottom: 12,
            border: `1px solid ${currentRun.status === 'completed' ? 'var(--semi-color-success)' : 'var(--semi-color-danger)'}`,
          }}
        >
          <Space vertical align="start" style={{ width: '100%' }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: 8,
                width: '100%',
              }}
            >
              <Text strong style={{ fontSize: 15 }}>
                {currentRun.status === 'completed' ? '最终结果' : '运行失败'}
              </Text>
              {currentRun.status === 'completed' &&
              currentRun.finalOutput &&
              currentRun.finalOutput.trim().length > 0 ? (
                <Button
                  size="small"
                  type="tertiary"
                  theme="borderless"
                  icon={<IconCopy />}
                  onClick={() => {
                    void navigator.clipboard.writeText(currentRun.finalOutput || '').then(
                      () => Toast.success({ content: '已复制完整结果', duration: 2 }),
                      () => Toast.warning({ content: '复制失败，请手动选中复制', duration: 3 }),
                    );
                  }}
                >
                  复制全文
                </Button>
              ) : null}
            </div>
            <Text type="tertiary" size="small" style={{ display: 'block', lineHeight: 1.5 }}>
              {currentRun.status === 'completed'
                ? currentRun.finalOutput?.trim()
                  ? '以下为本次协作返回的汇总输出；详细步骤见右侧 Trace。'
                  : '本次运行未写入汇总文本，请右侧 Trace 查看各步骤产出。'
                : '协作提前结束，请在右侧 Trace 查看 ERROR 等事件定位原因。'}
            </Text>
            {currentRun.finalOutput?.trim() ? (
              <div
                style={{
                  width: '100%',
                  maxHeight: 'min(52vh, 440px)',
                  overflow: 'auto',
                  padding: '12px 14px',
                  borderRadius: 10,
                  background: 'var(--semi-color-bg-0)',
                  border: '1px solid rgba(28,31,35,0.08)',
                  fontSize: 13,
                  lineHeight: 1.65,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  fontFamily: 'var(--semi-font-family-regular)',
                  color: 'var(--semi-color-text-0)',
                }}
              >
                {currentRun.finalOutput}
              </div>
            ) : null}
          </Space>
        </div>
      ) : null}

      {/* 主区域：空态或留白 */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: showIdleHero ? 'center' : 'flex-start',
          minHeight: 160,
          padding: showIdleHero ? '24px 16px' : '8px 0',
        }}
      >
        {showIdleHero ? (
          <div style={{ textAlign: 'center', maxWidth: 360 }}>
            <div style={{ fontSize: 52, lineHeight: 1.2, marginBottom: 12, opacity: 0.88 }} aria-hidden>
              ✈️
            </div>
            <Text strong style={{ fontSize: 16, display: 'block', marginBottom: 8 }}>
              开始一次运行
            </Text>
            <Text type="tertiary" size="small" style={{ lineHeight: 1.65 }}>
              发送问题并观察路由、图高亮和 Trace 变化
            </Text>
          </div>
        ) : null}
      </div>

      {/* 底部输入 */}
      <div style={{ flexShrink: 0, marginTop: 'auto' }}>
        <TextArea
          ref={inputRef as any}
          value={input}
          onChange={v => setInput(v)}
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
