/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useState, useEffect } from 'react';

import { usePanelManager } from '@flowgram.ai/panel-manager-plugin';
import { useClientContext, CommandService } from '@flowgram.ai/free-layout-editor';
import { Button } from '@douyinfe/semi-ui';
import { IconClose, IconSmallTriangleDown, IconSmallTriangleLeft } from '@douyinfe/semi-icons';

import { toggleLoopExpanded } from '../../utils';
import { FlowCommandId } from '../../shortcuts';
import { useIsSidebar, useNodeRenderContext } from '../../hooks';
import { nodeFormPanelFactory } from '../../components/sidebar';
import { NodeMenu } from '../../components/node-menu';
import { getIcon } from './utils';
import { TitleInput } from './title-input';
import { Header, Operators } from './styles';

export function FormHeader() {
  const { node, expanded, toggleExpand, readonly } = useNodeRenderContext();
  const [titleEdit, updateTitleEdit] = useState<boolean>(false);
  const ctx = useClientContext();
  const isSidebar = useIsSidebar();
  const panelManager = usePanelManager();
  const handleExpand = (e: React.MouseEvent) => {
    toggleExpand();
    e.stopPropagation(); // Disable clicking prevents the sidebar from opening
  };
  const handleDelete = () => {
    ctx.get<CommandService>(CommandService).executeCommand(FlowCommandId.DELETE, [node]);
  };
  const handleClose = () => {
    panelManager.close(nodeFormPanelFactory.key);
  };
  useEffect(() => {
    // 折叠 loop 子节点
    if (node.flowNodeType === 'loop') {
      toggleLoopExpanded(node, expanded);
    }
  }, [expanded]);

  return (
    <Header>
      {getIcon(node)}
      <TitleInput readonly={readonly} updateTitleEdit={updateTitleEdit} titleEdit={titleEdit} />
      {node.renderData.expandable && !isSidebar && (
        <Button
          type="primary"
          icon={expanded ? <IconSmallTriangleDown /> : <IconSmallTriangleLeft />}
          size="small"
          theme="borderless"
          onClick={handleExpand}
        />
      )}
      {readonly ? undefined : (
        <Operators>
          <NodeMenu node={node} deleteNode={handleDelete} updateTitleEdit={updateTitleEdit} />
        </Operators>
      )}
      {isSidebar && (
        <Button
          type="primary"
          icon={<IconClose />}
          size="small"
          theme="borderless"
          onClick={handleClose}
        />
      )}
    </Header>
  );
}
