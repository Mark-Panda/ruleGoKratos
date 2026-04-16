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
 * @param props.children 展开画布节点或侧栏中的完整表单体
 * @param props.collapsedPreview 仅画布且折叠时展示的摘要（如 Cursor 节点）
 */
export function FormContent(props: {
  children?: React.ReactNode;
  collapsedPreview?: React.ReactNode;
}) {
  const { node, expanded } = useNodeRenderContext();
  const isSidebar = useIsSidebar();
  const registry = node.getNodeRegistry<FlowNodeRegistry>();
  return (
    <FormWrapper>
      <>
        {isSidebar && <FormTitleDescription>{registry.info?.description}</FormTitleDescription>}
        {!isSidebar && <PortHints />}
        {!isSidebar && !expanded && props.collapsedPreview}
        {(expanded || isSidebar) && props.children}
      </>
    </FormWrapper>
  );
}
