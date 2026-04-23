/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import {
  WorkflowNodeEntity,
  PositionSchema,
  FlowNodeTransformData,
} from '@flowgram.ai/free-layout-editor';

import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLoop from '../../assets/icon-loop.jpg';
import { formMeta } from './form-meta';
import {
  FOR_SUBCANVAS_DEFAULT_HEIGHT_PX,
  FOR_SUBCANVAS_TOP_FORM_RESERVE_PX,
} from './subcanvas-layout';
import { OutPutPortType, WorkflowNodeType } from '../constants';

let index = 0;
export const ForNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.For,
  info: {
    icon: iconLoop,
    description: '遍历目标节点',
  },
  meta: {
    /**
     * Mark as subcanvas
     * 子画布标记
     */
    isContainer: true,
    /**
     * The subcanvas default size setting
     * 子画布默认大小设置（略小于 loop，画布占位更紧凑）
     */
    size: {
      width: 360,
      height: FOR_SUBCANVAS_DEFAULT_HEIGHT_PX,
    },
    // autoResizeDisable: true,
    /**
     * The subcanvas padding setting
     * 子画布 padding 设置
     */
    padding: (transform) => {
      if (!transform.isContainer) {
        return {
          top: 0,
          bottom: 0,
          left: 0,
          right: 0,
        };
      }
      return {
        // 与 form-meta 中 SubCanvasRender 负 offset 预留一致，见 subcanvas-layout.ts
        top: FOR_SUBCANVAS_TOP_FORM_RESERVE_PX,
        bottom: 56,
        left: 72,
        right: 72,
      };
    },
    /**
     * Controls the node selection status within the subcanvas
     * 控制子画布内的节点选中状态
     */
    selectable(node: WorkflowNodeEntity, mousePos?: PositionSchema): boolean {
      if (!mousePos) {
        return true;
      }
      const transform = node.getData<FlowNodeTransformData>(FlowNodeTransformData);
      // 鼠标开始时所在位置不包括当前节点时才可选中
      return !transform.bounds.contains(mousePos.x, mousePos.y);
    },
    // expandable: false, // disable expanded
    wrapperStyle: {
      minWidth: 'unset',
      width: '100%',
    },
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.For,
      data: {
        title: `For_${++index}`,
        positionType: 'middle',
      },
      blocks: [
        {
          id: `block_start_${alphaNanoid(5)}`,
          type: WorkflowNodeType.BlockStart,
          meta: {
            position: {
              x: 28,
              y: 0,
            },
          },
          data: { positionType: 'middle' },
        },
        {
          id: `block_end_${alphaNanoid(5)}`,
          type: WorkflowNodeType.BlockEnd,
          meta: {
            position: {
              x: 168,
              y: 0,
            },
          },
          data: { positionType: 'middle' },
        },
      ],
    };
  },
  formMeta,
};
