/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useMemo } from 'react';

import { useService, WorkflowDocument, WorkflowNodeEntity } from '@flowgram.ai/free-layout-editor';
import { IFlowValue } from '@flowgram.ai/form-materials';
import { Select } from '@douyinfe/semi-ui';

import { useNodeRenderContext } from '../../hooks';
import { OutPutPortType } from '../../nodes/constants';

export interface NodeIdSelectProps {
  value?: IFlowValue;
  onChange: (next: IFlowValue) => void;
  readonly?: boolean;
  hasError?: boolean;
  /** 排除当前正在编辑的节点 */
  excludeSelf?: boolean;
  /** 按 flowNodeType 排除（如 start、end） */
  excludeTypes?: string[];
  /** 按 id 排除 */
  excludeIds?: string[];
  /**
   * 为 true 时：若当前节点存在从 Success 口连出的线，则下拉仅展示这些下游节点；
   * 若无任何 Success 出线，则仍展示全表（并应用其它排除规则）。
   */
  preferSuccessDownstream?: boolean;
}

export const NodeIdSelect: React.FC<NodeIdSelectProps> = ({
  value,
  onChange,
  readonly,
  hasError,
  excludeSelf,
  excludeTypes,
  excludeIds,
  preferSuccessDownstream,
}) => {
  const document = useService(WorkflowDocument);
  const { node: currentNode } = useNodeRenderContext();

  const nodes = useMemo<WorkflowNodeEntity[]>(() => {
    try {
      return document.getAllNodes();
    } catch (e) {
      return [];
    }
  }, [document]);

  const successDownstreamIdSet = useMemo(() => {
    if (!preferSuccessDownstream || !currentNode) {
      return null;
    }
    const lines = (currentNode as any).lines?.outputLines ?? [];
    const ids = new Set<string>();
    for (const line of lines) {
      const toNode = line?.to as WorkflowNodeEntity | undefined;
      if (!toNode?.id) continue;
      const portId =
        (line as any).fromPort?.portID ??
        (line as any).sourcePortID ??
        (line as any).sourcePortId;
      if (portId === OutPutPortType.FailurePort) {
        continue;
      }
      if (portId === OutPutPortType.SuccessPort || portId === undefined || portId === '') {
        ids.add(toNode.id);
      }
    }
    return ids.size > 0 ? ids : null;
  }, [preferSuccessDownstream, currentNode]);

  const options = useMemo(() => {
    const typeSet = excludeTypes?.length ? new Set(excludeTypes) : null;
    const idSet = excludeIds?.length ? new Set(excludeIds) : null;

    let list = nodes.filter((n) => {
      if (excludeSelf && currentNode && n.id === currentNode.id) {
        return false;
      }
      if (typeSet?.has(String(n.flowNodeType))) {
        return false;
      }
      if (idSet?.has(n.id)) {
        return false;
      }
      return true;
    });

    if (successDownstreamIdSet) {
      list = list.filter((n) => successDownstreamIdSet.has(n.id));
    }

    return list.map((n) => {
      const json = document.toNodeJSON(n) as any;
      const title = json?.data?.title;
      return {
        label: `${title ? String(title) : n.id} (${n.id})`,
        value: n.id,
        key: n.id,
      };
    });
  }, [
    nodes,
    document,
    excludeSelf,
    excludeTypes,
    excludeIds,
    currentNode,
    successDownstreamIdSet,
  ]);

  const selectedValue =
    (value?.type === 'constant' ? (value.content as string) : undefined) ?? undefined;

  const optionsWithOrphan = useMemo(() => {
    if (!selectedValue || options.some((o) => o.value === selectedValue)) {
      return options;
    }
    return [
      {
        label: `${selectedValue}（当前值，未在列表中时可保留；子链可用 chain:ID）`,
        value: selectedValue,
        key: `orphan_${selectedValue}`,
      },
      ...options,
    ];
  }, [options, selectedValue]);

  const selectedLabel = optionsWithOrphan.find((opt) => opt.value === selectedValue)?.label as
    | string
    | undefined;

  const placeholder = readonly
    ? '只读'
    : preferSuccessDownstream && successDownstreamIdSet && options.length === 0
    ? '暂无符合的 Success 下游节点，请先连线或检查排除类型'
    : '请选择节点';

  return (
    <div style={{ width: '100%' }}>
      <Select
        value={selectedValue}
        onChange={(val) => onChange({ type: 'constant', content: val as string })}
        optionList={optionsWithOrphan}
        placeholder={placeholder}
        disabled={readonly}
        insetLabel={hasError ? '!' : undefined}
        style={{ width: '100%' }}
        filter
        renderSelectedItem={() => (selectedLabel ? selectedLabel : selectedValue)}
      />
    </div>
  );
};
