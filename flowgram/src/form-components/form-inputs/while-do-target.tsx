/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useEffect, useMemo, useState } from 'react';

import { IFlowValue } from '@flowgram.ai/form-materials';
import { Radio, RadioGroup, Space } from '@douyinfe/semi-ui';

import { RuleSelect } from './rule-select';
import { NodeIdSelect, NodeIdSelectProps } from './node-id-select';

export type WhileDoTargetSelectProps = {
  value?: IFlowValue;
  onChange: (next: IFlowValue) => void;
  readonly?: boolean;
  hasError?: boolean;
} & Pick<
  NodeIdSelectProps,
  'excludeSelf' | 'excludeTypes' | 'excludeIds' | 'preferSuccessDownstream'
>;

/** 解析 RuleGo while 的 do：节点 id，或 chain:子规则链ID */
function parseDo(raw: string): { mode: 'node' | 'chain'; nodeId: string; chainId: string } {
  const t = String(raw ?? '').trim();
  if (t.startsWith('chain:')) {
    return { mode: 'chain', nodeId: '', chainId: t.slice('chain:'.length).trim() };
  }
  return { mode: 'node', nodeId: t, chainId: '' };
}

export const WhileDoTargetSelect: React.FC<WhileDoTargetSelectProps> = ({
  value,
  onChange,
  readonly,
  hasError,
  excludeSelf,
  excludeTypes,
  excludeIds,
  preferSuccessDownstream,
}) => {
  const raw = value?.type === 'constant' ? String(value.content ?? '') : '';
  const parsed = useMemo(() => parseDo(raw), [raw]);

  /** 与 Radio 联动：切换为子规则链时常先把 content 清空，仅靠 raw 无法区分「画布空」与「子链未选」，必须用本地 mode */
  const [mode, setMode] = useState<'node' | 'chain'>(() => parsed.mode);

  useEffect(() => {
    const t = raw.trim();
    if (t.startsWith('chain:')) {
      setMode('chain');
    } else if (t !== '') {
      setMode('node');
    }
  }, [raw]);

  const handleModeChange = (next: 'node' | 'chain') => {
    setMode(next);
    const t = raw.trim();
    if (next === 'chain') {
      const cid = t.startsWith('chain:') ? t.slice('chain:'.length).trim() : '';
      onChange({ type: 'constant', content: cid ? `chain:${cid}` : '' });
    } else {
      const nid = t.startsWith('chain:') ? '' : t;
      onChange({ type: 'constant', content: nid });
    }
  };

  return (
    <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
      <RadioGroup
        value={mode}
        onChange={(e) => handleModeChange(e.target.value as 'node' | 'chain')}
        disabled={readonly}
      >
        <Radio value="node">画布上的节点</Radio>
        <Radio value="chain">子规则链</Radio>
      </RadioGroup>
      {mode === 'node' ? (
        <NodeIdSelect
          value={{ type: 'constant', content: parsed.nodeId }}
          onChange={(next) => {
            const c = next.type === 'constant' && next.content != null ? String(next.content) : '';
            onChange({ type: 'constant', content: c });
          }}
          readonly={readonly}
          hasError={hasError}
          excludeSelf={excludeSelf}
          excludeTypes={excludeTypes}
          excludeIds={excludeIds}
          preferSuccessDownstream={preferSuccessDownstream}
        />
      ) : (
        <RuleSelect
          value={{ type: 'constant', content: parsed.chainId }}
          onChange={(next) => {
            const id =
              next.type === 'constant' && next.content != null ? String(next.content).trim() : '';
            onChange({ type: 'constant', content: id ? `chain:${id}` : '' });
          }}
          readonly={readonly}
          hasError={hasError}
        />
      )}
    </Space>
  );
};
