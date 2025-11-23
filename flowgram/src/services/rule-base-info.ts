/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

export interface RuleChainBase {
  id: string;
  name: string;
  debugMode?: boolean;
  root?: boolean;
  disabled?: boolean;
  configuration?: Record<string, any>;
  additionalInfo?: Record<string, any>;
}

let currentRuleBaseInfo: RuleChainBase | undefined;

export const setRuleBaseInfo = (info: RuleChainBase | undefined) => {
  currentRuleBaseInfo = info;
};

export const getRuleBaseInfo = (): RuleChainBase | undefined => currentRuleBaseInfo;
