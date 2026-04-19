/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React from 'react';

import { FlowNodeRegistry } from '@flowgram.ai/free-layout-editor';

import { useIsSidebar, useNodeRenderContext } from '../../hooks';
import { FormTitleDescription, FormWrapper } from './styles';
import { PortHints } from '../port-hints';

/**
 * @param props.children 侧栏中的完整表单体；无 collapsedPreview 且画布展开时同内容只读展示
 * @param props.collapsedPreview 画布上展示的摘要（完整表单仅在侧栏编辑）
 */
export function FormContent(props: {
  children?: React.ReactNode;
  collapsedPreview?: React.ReactNode;
}) {
  const { node, expanded } = useNodeRenderContext();
  const isSidebar = useIsSidebar();
  const registry = node.getNodeRegistry<FlowNodeRegistry>();
  const showCanvasExpandedBody = !isSidebar && !props.collapsedPreview && expanded;
  return (
    <FormWrapper>
      <>
        {isSidebar && <FormTitleDescription>{registry.info?.description}</FormTitleDescription>}
        {!isSidebar && <PortHints />}
        {!isSidebar && props.collapsedPreview}
        {isSidebar && props.children}
        {showCanvasExpandedBody && props.children}
      </>
    </FormWrapper>
  );
}
