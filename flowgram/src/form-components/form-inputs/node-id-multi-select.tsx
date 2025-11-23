/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useMemo } from 'react';

import { useService, WorkflowDocument, WorkflowNodeEntity } from '@flowgram.ai/free-layout-editor';
import { IFlowValue } from '@flowgram.ai/form-materials';
import { Select } from '@douyinfe/semi-ui';

interface NodeIdMultiSelectProps {
  value?: IFlowValue;
  onChange: (next: IFlowValue) => void;
  readonly?: boolean;
  hasError?: boolean;
}

export const NodeIdMultiSelect: React.FC<NodeIdMultiSelectProps> = ({
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

  const selectedValues =
    (value?.type === 'constant'
      ? Array.isArray(value.content)
        ? (value.content as string[])
        : []
      : []) ?? [];

  return (
    <div style={{ width: '100%' }}>
      <Select
        multiple
        value={selectedValues}
        onChange={(vals) => onChange({ type: 'constant', content: vals })}
        optionList={options}
        placeholder={readonly ? '只读' : '请选择多个节点'}
        disabled={readonly}
        insetLabel={hasError ? '!' : undefined}
        style={{ width: '100%' }}
      />
    </div>
  );
};
