/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FlowNodeRegistry } from '../../typings';
import iconEnd from '../../assets/icon-end.jpg';
import { formMeta } from './form-meta';
import { WorkflowNodeType } from '../constants';

export const EndNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.End,
  meta: {
    deleteDisable: false,
    copyDisable: true,
    nodePanelVisible: true,
    defaultPorts: [{ type: 'input' }],
    size: {
      width: 360,
      height: 211,
    },
  },
  info: {
    icon: iconEnd,
    description:
      'The final node of the workflow, used to return the result information after the workflow is run.',
  },
  /**
   * Render node via formMeta
   */
  formMeta,
  /**
   * Allow adding end node from panel
   */
  onAdd() {
    return {
      id: `${/* id */ Math.random().toString(36).slice(2)}`,
      type: WorkflowNodeType.End,
      data: {
        title: 'End',
        positionType: 'tail',
        inputs: {
          type: 'object',
          properties: {},
        },
      },
    } as any;
  },
};
