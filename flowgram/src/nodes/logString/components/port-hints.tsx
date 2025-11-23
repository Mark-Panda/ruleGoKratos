/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import styled from 'styled-components';

import { useIsSidebar } from '../../../hooks';

const Hint = styled.div`
  position: absolute;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 20px;
  padding: 0 8px;
  border-radius: 10px;
  border: 1px solid rgba(6, 7, 9, 0.15);
  background: #fff;
  color: rgba(6, 7, 9, 0.65);
  font-size: 12px;
  pointer-events: none;
`;

const RightHint = styled(Hint)`
  right: -8px;
  top: 50%;
  transform: translate(100%, -50%);
`;

const BottomHint = styled(Hint)`
  left: 50%;
  bottom: -8px;
  transform: translate(-50%, 100%);
`;

export function LogStringPortHints() {
  const isSidebar = useIsSidebar();
  if (isSidebar) return null;
  return (
    <>
      <RightHint>Success</RightHint>
      <BottomHint>Failure</BottomHint>
    </>
  );
}
