/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import styled from 'styled-components';
import { IconInfoCircle } from '@douyinfe/semi-icons';

export const NodeWrapperStyle = styled.div`
  align-items: flex-start;
  background-color: #fff;
  border: 1px solid rgba(6, 7, 9, 0.15);
  border-radius: 8px;
  box-shadow: 0 2px 6px 0 rgba(0, 0, 0, 0.04), 0 4px 12px 0 rgba(0, 0, 0, 0.02);
  display: flex;
  flex-direction: column;
  /* 勿用 center：容器节点 bounds 被子画布撑高时，表单体被竖直居中，底部 Failure 等端口仍贴在 bounds 底边，圆点会落在白底下方 */
  justify-content: flex-start;
  position: relative;
  width: 360px;
  /* 与 free-container-plugin useSyncNodeRenderSize 注入的节点高度一致，避免「卡片视觉高度 < 引擎 hit 框」 */
  height: 100%;
  box-sizing: border-box;

  &.selected {
    border: 1px solid #4e40e5;
  }
`;

export const ErrorIcon = () => (
  <IconInfoCircle
    style={{
      position: 'absolute',
      color: 'red',
      left: -6,
      top: -6,
      zIndex: 1,
      background: 'white',
      borderRadius: 8,
    }}
  />
);
