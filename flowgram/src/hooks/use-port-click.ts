/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useCallback } from 'react';

import {
  WorkflowNodePanelService,
  WorkflowNodePanelUtils,
} from '@flowgram.ai/free-node-panel-plugin';
import {
  delay,
  usePlayground,
  useService,
  WorkflowDocument,
  WorkflowDragService,
  WorkflowLinesManager,
  WorkflowNodeEntity,
  WorkflowNodeJSON,
  WorkflowPortEntity,
} from '@flowgram.ai/free-layout-editor';
import { Toast } from '@douyinfe/semi-ui';

import { getRuleBaseInfo } from '../services/rule-base-info';
import { WorkflowNodeType } from '../nodes';

/**
 * click port to trigger node select panel
 * 点击端口后唤起节点选择面板
 */
export const usePortClick = () => {
  const playground = usePlayground();
  const nodePanelService = useService(WorkflowNodePanelService);
  const document = useService(WorkflowDocument);
  const dragService = useService(WorkflowDragService);
  const linesManager = useService(WorkflowLinesManager);

  const onPortClick = useCallback(async (e: React.MouseEvent, port: WorkflowPortEntity) => {
    if (port.portType === 'input') return;
    const mousePos = playground.config.getPosFromMouseEvent(e);
    const containerNode = port.node.parent;
    // open node selection panel - 打开节点选择面板
    const result = await nodePanelService.singleSelectNodePanel({
      position: mousePos,
      containerNode,
      panelProps: {
        enableScrollClose: true,
        fromPort: port,
      },
    });

    // return if no node selected - 如果没有选择节点则返回
    if (!result) {
      return;
    }

    // get selected node type and data - 获取选择的节点类型和数据
    const { nodeType, nodeJSON } = result;
    if (
      containerNode?.flowNodeType === WorkflowNodeType.For &&
      (nodeJSON?.data?.positionType === 'header' ||
        nodeType === WorkflowNodeType.Start ||
        nodeType === WorkflowNodeType.Cron)
    ) {
      Toast.error('For 子画布不允许连接 Header 类型的节点');
      return;
    }

    // 获取当前规则链信息
    const ruleBaseInfo = getRuleBaseInfo();
    const isChildRuleChain = ruleBaseInfo?.root === false;

    // 限制：子规则链中不允许添加除 start 之外的 header 类型节点
    const isHeaderCandidate =
      nodeJSON?.data?.positionType === 'header' || nodeType === WorkflowNodeType.Start;
    const isStartNode = nodeType === WorkflowNodeType.Start;

    if (isChildRuleChain && isHeaderCandidate && !isStartNode) {
      Toast.error('子规则链中不允许添加 Header 类型的节点（start 节点除外）');
      return;
    }

    // 画布只允许一个 Header 类型节点（创建前校验）
    const rawBefore = document.toJSON();
    const headerExists = Array.isArray(rawBefore.nodes)
      ? rawBefore.nodes.some(
          (n: any) => n?.data?.positionType === 'header' || String(n?.type) === 'start'
        )
      : false;
    if (headerExists && isHeaderCandidate) {
      Toast.error('画布中只能存在一个 Header 类型的节点');
      return;
    }

    // calculate position for the new node - 计算新节点的位置
    const nodePosition = WorkflowNodePanelUtils.adjustNodePosition({
      nodeType,
      position:
        port.location === 'bottom'
          ? {
              x: mousePos.x,
              y: mousePos.y + 100,
            }
          : {
              x: mousePos.x + 100,
              y: mousePos.y,
            },
      fromPort: port,
      containerNode,
      document,
      dragService,
    });

    // create new workflow node - 创建新的工作流节点
    const node: WorkflowNodeEntity = document.createWorkflowNodeByType(
      nodeType,
      nodePosition,
      nodeJSON ?? ({} as WorkflowNodeJSON),
      containerNode?.id
    );

    // wait for node render - 等待节点渲染
    await delay(20);

    // build connection line - 构建连接线
    WorkflowNodePanelUtils.buildLine({
      fromPort: port,
      node,
      linesManager,
    });

    // 二次校验：避免并发情况下出现多个 Header
    const rawAfter = document.toJSON();
    const headerCount = Array.isArray(rawAfter.nodes)
      ? rawAfter.nodes.filter(
          (n: any) => n?.data?.positionType === 'header' || String(n?.type) === 'start'
        ).length
      : 0;
    if (isHeaderCandidate && headerCount > 1) {
      node.dispose();
      Toast.error('画布中只能存在一个 Header 类型的节点');
      return;
    }
  }, []);

  return onPortClick;
};
