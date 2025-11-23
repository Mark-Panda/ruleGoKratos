/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { createPortal } from 'react-dom';

import { useClientContext } from '@flowgram.ai/free-layout-editor';
import { Space } from '@douyinfe/semi-ui';

import { TestRunButton } from '../testrun/testrun-button';
import { VariablePanelToggle } from './variable-panel-toggle';
import { SaveButton } from './save-button';
import { ExportImport } from './export-import';

// 用于标识顶部工具栏容器的ID
const TOP_TOOLBAR_CONTAINER_ID = 'top-toolbar-portal-container';

/**
 * 顶部工具栏组件 - 用于在 rule-detail 页面顶部显示操作按钮
 * 这个组件会在 FreeLayoutEditorProvider 内部渲染，可以访问编辑器上下文
 * 使用 Portal 将内容渲染到页面顶部的指定容器中
 */
export const TopToolbar: React.FC = () => {
  const { playground } = useClientContext();
  const disabled = playground?.config?.readonly ?? false;

  // 查找目标容器
  const container = document.getElementById(TOP_TOOLBAR_CONTAINER_ID);

  // 如果容器不存在，不渲染任何内容
  if (!container) {
    return null;
  }

  // 使用 Portal 将工具栏渲染到顶部导航栏的右侧
  return createPortal(
    <Space spacing={12}>
      <VariablePanelToggle />
      <ExportImport disabled={disabled} />
      <TestRunButton disabled={disabled} />
      <SaveButton disabled={disabled} />
    </Space>,
    container
  );
};
