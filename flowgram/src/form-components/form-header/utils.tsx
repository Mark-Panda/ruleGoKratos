/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { type FlowNodeEntity } from '@flowgram.ai/free-layout-editor';

import { FlowNodeRegistry } from '../../typings';
import { renderNodePanelIcon } from '../../components/node-panel/node-type-icons';
import { IconWrap } from './styles';

export const getIcon = (node: FlowNodeEntity) => {
  const registry = node.getNodeRegistry<FlowNodeRegistry>();
  return (
    <IconWrap>
      {renderNodePanelIcon(registry, {
        size: 18,
        strokeWidth: 2.6,
      })}
    </IconWrap>
  );
};
