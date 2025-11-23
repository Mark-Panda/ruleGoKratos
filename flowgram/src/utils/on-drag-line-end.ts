/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import {
  WorkflowNodePanelService,
  WorkflowNodePanelUtils,
} from '@flowgram.ai/free-node-panel-plugin';
import {
  delay,
  FreeLayoutPluginContext,
  onDragLineEndParams,
  WorkflowDragService,
  WorkflowLinesManager,
  WorkflowNodeEntity,
  WorkflowNodeJSON,
} from '@flowgram.ai/free-layout-editor';
import { Toast } from '@douyinfe/semi-ui';

import { getRuleBaseInfo } from '../services/rule-base-info';
import { WorkflowNodeType } from '../nodes';

/**
 * Drag the end of the line to create an add panel (feature optional)
 * 拖拽线条结束需要创建一个添加面板 （功能可选）
 */
export const onDragLineEnd = async (ctx: FreeLayoutPluginContext, params: onDragLineEndParams) => {
  // get services from context - 从上下文获取服务
  const nodePanelService = ctx.get(WorkflowNodePanelService);
  const document = ctx.document;
  const dragService = ctx.get(WorkflowDragService);
  const linesManager = ctx.get(WorkflowLinesManager);

  // get params from drag event - 从拖拽事件获取参数
  const { fromPort, toPort, mousePos, line, originLine } = params;

  // return if invalid line state - 如果线条状态无效则返回
  if (originLine || !line) {
    return;
  }

  // return if target port exists - 如果目标端口存在则返回
  if (toPort || !fromPort) {
    return;
  }

  // get container node for the new node - 获取新节点的容器节点
  const containerNode = fromPort.node.parent;
  const isVertical = fromPort.location === 'bottom';

  // open node selection panel - 打开节点选择面板
  const result = await nodePanelService.singleSelectNodePanel({
    position: isVertical
      ? {
          x: mousePos.x - 165,
          y: mousePos.y + 60,
        }
      : mousePos,
    containerNode,
    panelProps: {
      enableNodePlaceholder: true,
      enableScrollClose: true,
      fromPort,
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
    position: mousePos,
    fromPort,
    toPort,
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
    fromPort,
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
};
