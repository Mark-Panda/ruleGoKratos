/**
 * 协作模式选择器组件
 */

import React from 'react';

import { Typography, Radio, Space } from '@douyinfe/semi-ui';
import { IconBranch, IconTick } from '@douyinfe/semi-icons';

import { CollaborationMode, MODE_NAME_MAP, MODE_DESC_MAP } from '../../services/api-playground';

const { Text } = Typography;

interface ModeSelectorProps {
  value: CollaborationMode;
  onChange: (mode: CollaborationMode) => void;
  style?: React.CSSProperties;
  showDescription?: boolean;
}

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
        {modes.map((mode) => (
          <div
            key={mode}
            style={{
              width: '100%',
              padding: '12px 16px',
              cursor: 'pointer',
              border:
                value === mode
                  ? '2px solid var(--semi-color-primary)'
                  : '1px solid var(--semi-color-border)',
              background:
                value === mode ? 'var(--semi-color-fill-actual)' : 'var(--semi-color-bg-1)',
              borderRadius: 8,
              marginBottom: 8,
            }}
            onClick={() => onChange(mode)}
          >
            <Space>
              {value === mode ? (
                <IconTick style={{ color: 'var(--semi-color-primary)' }} />
              ) : (
                <IconBranch style={{ color: 'var(--semi-color-tertiary)' }} />
              )}
              <Radio value={mode}>
                <Text strong>{MODE_NAME_MAP[mode]}</Text>
              </Radio>
            </Space>
            {showDescription && (
              <div style={{ marginLeft: 28, marginTop: 4 }}>
                <Text type="tertiary" style={{ fontSize: 12 }}>
                  {MODE_DESC_MAP[mode]}
                </Text>
              </div>
            )}
          </div>
        ))}
      </Space>
    </Radio.Group>
  );
};
