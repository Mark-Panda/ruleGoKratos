/**
 * 协作模式选择器组件
 */

import React from 'react';

import { Typography, Radio, Space } from '@douyinfe/semi-ui';
import { IconServer, IconList, IconEyeOpened, IconSync } from '@douyinfe/semi-icons';

import { CollaborationMode, MODE_NAME_MAP, MODE_DESC_MAP } from '../../services/api-playground';

const { Text } = Typography;

interface ModeSelectorProps {
  value: CollaborationMode;
  onChange: (mode: CollaborationMode) => void;
  style?: React.CSSProperties;
  showDescription?: boolean;
}

const MODE_ICON_MAP: Record<CollaborationMode, React.ReactNode> = {
  router_expert: <IconServer style={{ fontSize: 18 }} />,
  plan_exec: <IconList style={{ fontSize: 18 }} />,
  supervision: <IconEyeOpened style={{ fontSize: 18 }} />,
  peer_handoff: <IconSync style={{ fontSize: 18 }} />,
};

const MODE_GRADIENT_MAP: Record<CollaborationMode, { start: string; end: string }> = {
  router_expert: { start: 'rgba(22, 100, 255, 0.08)', end: 'rgba(22, 100, 255, 0.02)' },
  plan_exec: { start: 'rgba(19, 194, 194, 0.08)', end: 'rgba(19, 194, 194, 0.02)' },
  supervision: { start: 'rgba(250, 173, 20, 0.08)', end: 'rgba(250, 173, 20, 0.02)' },
  peer_handoff: { start: 'rgba(82, 196, 26, 0.08)', end: 'rgba(82, 196, 26, 0.02)' },
};

const MODE_BORDER_MAP: Record<CollaborationMode, string> = {
  router_expert: 'var(--semi-color-primary)',
  plan_exec: 'var(--semi-color-tertiary)',
  supervision: 'var(--semi-color-warning)',
  peer_handoff: 'var(--semi-color-success)',
};

export const ModeSelector: React.FC<ModeSelectorProps> = ({
  value,
  onChange,
  style,
  showDescription = true,
}) => {
  const modes: CollaborationMode[] = ['router_expert', 'plan_exec', 'supervision', 'peer_handoff'];

  return (
    <Radio.Group value={value} onChange={(e) => onChange(e.target.value)} style={style}>
      <Space vertical align="start" style={{ width: '100%' }}>
        {modes.map((mode) => {
          const isSelected = value === mode;
          const gradient = MODE_GRADIENT_MAP[mode];
          const border = isSelected ? MODE_BORDER_MAP[mode] : '1px solid var(--semi-color-border)';
          return (
            <div
              key={mode}
              style={{
                width: '100%',
                padding: '14px 16px',
                cursor: 'pointer',
                border: isSelected ? `2px solid ${MODE_BORDER_MAP[mode]}` : border,
                background: isSelected
                  ? `linear-gradient(135deg, ${gradient.start}, ${gradient.end})`
                  : 'var(--semi-color-bg-1)',
                borderRadius: 12,
                marginBottom: 8,
                transition: 'all 0.2s ease',
                boxShadow: isSelected ? `0 2px 12px ${gradient.start}` : 'none',
              }}
              onClick={() => onChange(mode)}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <div
                  style={{
                    width: 36,
                    height: 36,
                    borderRadius: 10,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: isSelected
                      ? `linear-gradient(135deg, ${gradient.start}, ${gradient.end})`
                      : 'var(--semi-color-fill-0)',
                    color: isSelected ? MODE_BORDER_MAP[mode] : 'var(--semi-color-tertiary)',
                    flexShrink: 0,
                    transition: 'all 0.2s ease',
                  }}
                >
                  {MODE_ICON_MAP[mode]}
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <Radio value={mode}>
                    <Text strong style={{ fontSize: 14 }}>
                      {MODE_NAME_MAP[mode]}
                    </Text>
                  </Radio>
                </div>
              </div>
              {showDescription && (
                <div style={{ marginLeft: 48, marginTop: 4 }}>
                  <Text type="tertiary" style={{ fontSize: 12, lineHeight: 1.5 }}>
                    {MODE_DESC_MAP[mode]}
                  </Text>
                </div>
              )}
            </div>
          );
        })}
      </Space>
    </Radio.Group>
  );
};
