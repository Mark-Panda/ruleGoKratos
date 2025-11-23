/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { NodePanelResult, WorkflowNodePanelService } from '@flowgram.ai/free-node-panel-plugin';
import {
  Layer,
  injectable,
  inject,
  FreeLayoutPluginContext,
  WorkflowHoverService,
  WorkflowNodeEntity,
  WorkflowNodeJSON,
  WorkflowSelectService,
  WorkflowDocument,
  PositionSchema,
  WorkflowDragService,
} from '@flowgram.ai/free-layout-editor';
import { ContainerUtils } from '@flowgram.ai/free-container-plugin';
import { Toast } from '@douyinfe/semi-ui';

import { getRuleBaseInfo } from '../../services/rule-base-info';
import { WorkflowNodeType } from '../../nodes';

@injectable()
export class ContextMenuLayer extends Layer {
  @inject(FreeLayoutPluginContext) ctx: FreeLayoutPluginContext;

  @inject(WorkflowNodePanelService) nodePanelService: WorkflowNodePanelService;

  @inject(WorkflowHoverService) hoverService: WorkflowHoverService;

  @inject(WorkflowSelectService) selectService: WorkflowSelectService;

  @inject(WorkflowDocument) document: WorkflowDocument;

  @inject(WorkflowDragService) dragService: WorkflowDragService;

  onReady() {
    this.listenPlaygroundEvent('contextmenu', (e) => {
      if (this.config.readonlyOrDisabled) return;
      this.openNodePanel(e);
      e.preventDefault();
      e.stopPropagation();
    });
  }

  openNodePanel(e: MouseEvent) {
    const mousePos = this.getPosFromMouseEvent(e);
    const containerNode = this.getContainerNode(mousePos);
    this.nodePanelService.callNodePanel({
      position: mousePos,
      containerNode,
      panelProps: {},
      // handle node selection from panel - 处理从面板中选择节点
      onSelect: async (panelParams?: NodePanelResult) => {
        if (!panelParams) {
          return;
        }
        const { nodeType, nodeJSON } = panelParams;

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

        // 限制：画布中只能有一个 header 类型的节点（创建前校验）
        const rawBefore = this.ctx.document.toJSON() as any;
        const hasHeader = Array.isArray(rawBefore?.nodes)
          ? rawBefore.nodes.some(
              (n: any) => n?.data?.positionType === 'header' || String(n?.type) === 'start'
            )
          : false;
        if (hasHeader && isHeaderCandidate) {
          Toast.error('画布中只能存在一个 Header 类型的节点');
          return;
        }
        const position = this.dragService.adjustSubNodePosition(nodeType, containerNode, mousePos);
        // create new workflow node based on selected type - 根据选择的类型创建新的工作流节点
        const node: WorkflowNodeEntity = this.ctx.document.createWorkflowNodeByType(
          nodeType,
          position,
          nodeJSON ?? ({} as WorkflowNodeJSON),
          containerNode?.id
        );
        // 二次校验：创建后若出现多个 header，撤销新建
        const rawAfter = this.ctx.document.toJSON() as any;
        const headerCount = Array.isArray(rawAfter?.nodes)
          ? rawAfter.nodes.filter(
              (n: any) => n?.data?.positionType === 'header' || String(n?.type) === 'start'
            ).length
          : 0;
        if (isHeaderCandidate && headerCount > 1) {
          node.dispose();
          Toast.error('画布中只能存在一个 Header 类型的节点');
          return;
        }
        // select the newly created node - 选择新创建的节点
        this.selectService.select(node);
      },
      // handle panel close - 处理面板关闭
      onClose: () => {},
    });
  }

  private getContainerNode(mousePos: PositionSchema): WorkflowNodeEntity | undefined {
    const allNodes = this.document.getAllNodes();
    const containerTransforms = ContainerUtils.getContainerTransforms(allNodes);
    const collisionTransform = ContainerUtils.getCollisionTransform({
      targetPoint: mousePos,
      transforms: containerTransforms,
      document: this.document,
    });
    return collisionTransform?.entity;
  }
}
