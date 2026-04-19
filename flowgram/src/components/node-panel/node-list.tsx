/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { FC, useMemo, useState } from 'react';

import { Input } from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import styled from 'styled-components';
import { NodePanelRenderProps } from '@flowgram.ai/free-node-panel-plugin';
import {
  useClientContext,
  WorkflowNodeEntity,
  WorkflowPortEntity,
} from '@flowgram.ai/free-layout-editor';

import { canContainNode } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import { nodeRegistries, getNodeTypeName } from '../../nodes';
import {
  PANEL_CATEGORY_LABELS,
  inferPanelCategoryKey,
  panelCategorySortKey,
} from './panel-categories';

const NodeWrap = styled.div`
  width: 100%;
  height: 32px;
  border-radius: 5px;
  display: flex;
  align-items: center;
  cursor: pointer;
  font-size: 19px;
  padding: 0 15px;
  &:hover {
    background-color: hsl(252deg 62% 55% / 9%);
    color: hsl(252 62% 54.9%);
  }
`;

const NodeLabel = styled.div`
  font-size: 13px;
  margin-left: 10px;
  white-space: nowrap;
`;

const SectionHead = styled.div`
  font-size: 11px;
  font-weight: 600;
  color: var(--semi-color-text-2);
  padding: 10px 15px 6px;
  letter-spacing: 0.02em;
`;

const SearchWrap = styled.div`
  padding: 8px 10px 6px;
  border-bottom: 1px solid var(--semi-color-border);
`;

interface NodeProps {
  label: string;
  icon: JSX.Element;
  onClick: React.MouseEventHandler<HTMLDivElement>;
  disabled: boolean;
}

function Node(props: NodeProps) {
  return (
    <NodeWrap
      data-testid={`demo-free-node-list-${props.label}`}
      onClick={props.disabled ? undefined : props.onClick}
      style={props.disabled ? { opacity: 0.3 } : {}}
    >
      <div style={{ fontSize: 14 }}>{props.icon}</div>
      <NodeLabel>{props.label}</NodeLabel>
    </NodeWrap>
  );
}

const NodesWrap = styled.div`
  max-height: 500px;
  overflow: auto;
  &::-webkit-scrollbar {
    display: none;
  }
`;

interface NodeListProps {
  onSelect: NodePanelRenderProps['onSelect'];
  fromPort?: WorkflowPortEntity;
  containerNode?: WorkflowNodeEntity;
}

function filterRegistries(
  list: FlowNodeRegistry[],
  containerNode: WorkflowNodeEntity | undefined
): FlowNodeRegistry[] {
  return list
    .filter((register) => register.meta.nodePanelVisible !== false)
    .filter((register) => {
      if (register.meta.onlyInContainerTypes?.length) {
        return register.meta.onlyInContainerTypes.includes(
          containerNode?.flowNodeType as any
        );
      }
      if (register.meta.onlyInContainer) {
        return register.meta.onlyInContainer === containerNode?.flowNodeType;
      }
      if (containerNode && !canContainNode(register.type, containerNode.flowNodeType)) {
        return false;
      }
      return true;
    });
}

export const NodeList: FC<NodeListProps> = (props) => {
  const { onSelect, containerNode, fromPort: _fromPort } = props;
  const context = useClientContext();
  const [query, setQuery] = useState('');

  const visibleRegistries = useMemo(
    () => filterRegistries(nodeRegistries, containerNode),
    [containerNode]
  );

  const grouped = useMemo(() => {
    const q = query.trim().toLowerCase();
    const filtered = q
      ? visibleRegistries.filter((r) => {
          const label = getNodeTypeName(r.type as string).toLowerCase();
          const typeStr = String(r.type).toLowerCase();
          return label.includes(q) || typeStr.includes(q);
        })
      : visibleRegistries;

    const buckets = new Map<string, FlowNodeRegistry[]>();
    for (const r of filtered) {
      const cat =
        (r.meta.panelCategory && String(r.meta.panelCategory).trim()) ||
        inferPanelCategoryKey(String(r.type));
      const arr = buckets.get(cat);
      if (arr) arr.push(r);
      else buckets.set(cat, [r]);
    }

    const keys = [...buckets.keys()].sort((a, b) => {
      const d = panelCategorySortKey(a) - panelCategorySortKey(b);
      if (d !== 0) return d;
      return a.localeCompare(b);
    });

    return keys.map((key) => ({
      key,
      label: PANEL_CATEGORY_LABELS[key] ?? key,
      items: buckets.get(key) ?? [],
    }));
  }, [visibleRegistries, query]);

  const handleClick = (e: React.MouseEvent, registry: FlowNodeRegistry) => {
    const json = registry.onAdd?.(context);
    onSelect({
      nodeType: registry.type as string,
      selectEvent: e,
      nodeJSON: json,
    });
  };

  return (
    <NodesWrap style={{ width: 228 }}>
      <SearchWrap>
        <Input
          size="small"
          prefix={<IconSearch />}
          placeholder="搜索节点"
          value={query}
          onChange={setQuery}
          showClear
        />
      </SearchWrap>
      {grouped.length === 0 ? (
        <SectionHead style={{ paddingTop: 16 }}>无匹配节点</SectionHead>
      ) : (
        grouped.map((section) => (
          <div key={section.key}>
            <SectionHead>{section.label}</SectionHead>
            {section.items.map((registry) => (
              <Node
                key={registry.type}
                disabled={!(registry.canAdd?.(context) ?? true)}
                icon={
                  <img
                    style={{ width: 10, height: 10, borderRadius: 4 }}
                    src={registry.info?.icon}
                    alt=""
                  />
                }
                label={getNodeTypeName(registry.type as string)}
                onClick={(e) => handleClick(e, registry)}
              />
            ))}
          </div>
        ))
      )}
    </NodesWrap>
  );
};
