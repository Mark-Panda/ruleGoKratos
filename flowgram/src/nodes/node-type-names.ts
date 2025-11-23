/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType } from './constants';

/**
 * 节点类型中英文名称映射
 * Node type name mapping (English to Chinese)
 */
export const NODE_TYPE_NAMES: Record<string, string> = {
  // 基础节点
  [WorkflowNodeType.Start]: '开始',
  [WorkflowNodeType.End]: '结束',

  // 逻辑控制节点
  [WorkflowNodeType.Condition]: '条件判断',
  [WorkflowNodeType.CaseCondition]: '条件分支',
  [WorkflowNodeType.Loop]: '循环',
  [WorkflowNodeType.For]: '遍历循环',
  [WorkflowNodeType.Fork]: '并发分支',
  [WorkflowNodeType.Join]: '合并分支',
  [WorkflowNodeType.Continue]: '继续',
  [WorkflowNodeType.Break]: '中断',

  // 数据处理节点
  [WorkflowNodeType.Transform]: 'JS数据转换',
  [WorkflowNodeType.JsFilter]: 'JS过滤器',
  [WorkflowNodeType.Variable]: '变量',
  [WorkflowNodeType.FetchNodeOutput]: '获取已完成节点输出',

  // 外部调用节点
  [WorkflowNodeType.HTTP]: 'HTTP请求',
  [WorkflowNodeType.LLM]: 'LLM大模型',
  [WorkflowNodeType.DBClient]: '数据库客户端',
  [WorkflowNodeType.RedisClient]: 'Redis客户端',
  [WorkflowNodeType.Cron]: '定时触发',
  [WorkflowNodeType.Flow]: '子规则链',

  // 辅助节点
  [WorkflowNodeType.LogString]: '日志输出',
  [WorkflowNodeType.Comment]: '注释',
  [WorkflowNodeType.Code]: '代码',

  // 块节点
  [WorkflowNodeType.BlockStart]: '块开始',
  [WorkflowNodeType.BlockEnd]: '块结束',

  // 多输出节点
  [WorkflowNodeType.MultiNodeOutput]: '获取已完成多节点输出',
  [WorkflowNodeType.LuaTransform]: 'Lua脚本转换',
  [WorkflowNodeType.Yapi]: 'Yapi接口',

  // 分组节点
  group: '分组',
};

/**
 * 获取节点类型的中文名称
 * @param nodeType 节点类型
 * @returns 中文名称，如果没有映射则返回原始类型
 */
export function getNodeTypeName(nodeType: string): string {
  return NODE_TYPE_NAMES[nodeType] || nodeType;
}

/**
 * 获取节点类型的显示名称（带图标的可选格式）
 * @param nodeType 节点类型
 * @param showEnglish 是否同时显示英文（默认不显示）
 * @returns 显示名称
 */
export function getNodeDisplayName(nodeType: string, showEnglish: boolean = false): string {
  const chineseName = NODE_TYPE_NAMES[nodeType];
  if (!chineseName) {
    return nodeType;
  }
  return showEnglish ? `${chineseName} (${nodeType})` : chineseName;
}
