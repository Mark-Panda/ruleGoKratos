/**
 * 封装节点选择面板创建节点后的连线逻辑。
 * 从组件底部出口（Failure）拉出子节点时，优先接到新节点的上部输入（含 portID=input_top）。
 */

import {
  WorkflowLinesManager,
  WorkflowNodeEntity,
  WorkflowNodePortsData,
  WorkflowPortEntity,
} from '@flowgram.ai/free-layout-editor';

function pickTargetInputPort(
  inputPorts: WorkflowPortEntity[],
  fromPort: WorkflowPortEntity
): WorkflowPortEntity | undefined {
  if (!inputPorts.length) return undefined;
  if (fromPort.location === 'bottom') {
    const topByLoc = inputPorts.find((p) => p.location === 'top');
    if (topByLoc) return topByLoc;
    const topById = inputPorts.find((p) => String(p.portID ?? '') === 'input_top');
    if (topById) return topById;
  }
  return inputPorts[0];
}

/** 对齐 WorkflowNodePanelUtils.buildLine，保证底部出口 → 顶部入口 */
export function panelBuildLine(params: {
  fromPort?: WorkflowPortEntity;
  node: WorkflowNodeEntity;
  toPort?: WorkflowPortEntity;
  linesManager: WorkflowLinesManager;
}): void {
  const { fromPort, node, toPort, linesManager } = params;
  const portsData = node.getData(WorkflowNodePortsData);
  if (!portsData) return;

  if (fromPort && portsData.inputPorts?.length > 0) {
    const toTargetPort = pickTargetInputPort(portsData.inputPorts, fromPort);
    if (toTargetPort) {
      linesManager.createLine({
        from: fromPort.node.id,
        fromPort: fromPort.portID,
        to: node.id,
        toPort: toTargetPort.portID,
      });
    }
  }

  if (toPort && portsData.outputPorts?.length > 0) {
    const fromTargetPort = portsData.outputPorts[0];
    linesManager.createLine({
      from: node.id,
      fromPort: fromTargetPort.portID,
      to: toPort.node.id,
      toPort: toPort.portID,
    });
  }
}
