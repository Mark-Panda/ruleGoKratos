/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useMemo } from 'react';

import { useService, WorkflowDocument, WorkflowNodeEntity } from '@flowgram.ai/free-layout-editor';
import { IFlowValue } from '@flowgram.ai/form-materials';
import { Select } from '@douyinfe/semi-ui';

interface NodeIdSelectProps {
  value?: IFlowValue;
  onChange: (next: IFlowValue) => void;
  readonly?: boolean;
  hasError?: boolean;
}

export const NodeIdSelect: React.FC<NodeIdSelectProps> = ({
  value,
  onChange,
  readonly,
  hasError,
}) => {
  const document = useService(WorkflowDocument);

  const nodes = useMemo<WorkflowNodeEntity[]>(() => {
    try {
      return document.getAllNodes();
    } catch (e) {
      return [];
    }
  }, [document]);

  const options = useMemo(
    () =>
      nodes.map((n) => {
        const json = document.toNodeJSON(n) as any;
        const title = json?.data?.title;
        return {
          label: `${title ? String(title) : n.id} (${n.id})`,
          value: n.id,
          key: n.id,
        };
      }),
    [nodes, document]
  );

  const selectedValue =
    (value?.type === 'constant' ? (value.content as string) : undefined) ?? undefined;
  const selectedLabel = options.find((opt) => opt.value === selectedValue)?.label as
    | string
    | undefined;

  return (
    <div style={{ width: '100%' }}>
      <Select
        value={selectedValue}
        onChange={(val) => onChange({ type: 'constant', content: val })}
        optionList={options}
        placeholder={readonly ? '只读' : '请选择节点'}
        disabled={readonly}
        insetLabel={hasError ? '!' : undefined}
        style={{ width: '100%' }}
        renderSelectedItem={() => (selectedLabel ? selectedLabel : selectedValue)}
      />
    </div>
  );
};
