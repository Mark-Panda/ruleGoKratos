/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import styled from 'styled-components';

export const CaseWrapper = styled.div`
  margin-bottom: 8px;
  padding: 8px 12px 12px 32px;
  border-radius: 6px;
  background: #fff;
  position: relative;
`;

// 仅包裹 IF/ELSE IF 的条件行区域，使分组标记严格处于组内
export const RowsWrapper = styled.div`
  position: relative;
  padding-left: 24px; // 为左侧括号留空
`;

export const GroupBracket = styled.div`
  position: absolute;
  left: 0px;
  top: 0px;
  bottom: 0px;
  width: 14px;
  border-left: 2px solid rgba(6, 7, 9, 0.15);
`;

export const GroupLabel = styled.div`
  position: absolute;
  left: 10px;
  top: -10px;
  transform: translateX(-50%);
  background: #fff;
  color: rgba(6, 7, 9, 0.65);
  font-size: 12px;
  border: 1px solid rgba(6, 7, 9, 0.15);
  border-radius: 8px;
  padding: 0 8px;
  height: 20px;
  line-height: 20px;
`;

export const OrChip = styled.div`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 11px;
  border: 1px solid rgba(6, 7, 9, 0.15);
  background: #fff;
  color: rgba(6, 7, 9, 0.65);
  font-size: 12px;
  margin: 6px 0;
`;

export const CaseHeader = styled.div`
  display: flex;
  align-items: center;
  column-gap: 8px;
  margin-bottom: 6px;
  color: rgba(6, 7, 9, 0.65);
`;

export const CaseTitle = styled.span`
  font-weight: 600;
`;

export const ConditionPort = styled.div`
  position: absolute;
  right: -12px;
  top: 50%;
`;

// 条件行：上下布局的两个输入框，右侧中间放操作符选择
export const ConditionRow = styled.div`
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-right: 132px; // 为右侧操作符选择预留空间
`;

export const OperatorSelectWrapper = styled.div`
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 120px;
`;
