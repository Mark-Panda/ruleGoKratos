/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { requestJSON } from './http';

export const getRuleList = async (params: {
  page?: number;
  size?: number;
  keywords?: string;
  root?: boolean;
  /** 对应后端列表查询 disabled；通常为 false 表示只要启用的规则链 */
  disabled?: boolean;
}) => {
  const q: Record<string, string | number | boolean> = {};
  if (params.page !== undefined) q.page = params.page;
  if (params.size !== undefined) q.size = params.size;
  if (params.keywords !== undefined && String(params.keywords).trim() !== '') {
    q.keywords = String(params.keywords).trim();
  }
  if (params.root !== undefined) q.root = params.root;
  if (params.disabled !== undefined) q.disabled = params.disabled;
  return requestJSON<{ items: any[]; total?: number; count?: number }>('/rules', { params: q });
};

export const createRuleBase = async (id: string, body: any) =>
  requestJSON(`/rules/${encodeURIComponent(id)}/base`, { method: 'POST', body });

export const getRuleDetail = async (id: string) => requestJSON(`/rules/${encodeURIComponent(id)}`);

export const updateRule = async (id: string, body: any) =>
  requestJSON(`/rules/${encodeURIComponent(id)}`, { method: 'POST', body });

// 部署（启动）规则链
export const startRuleChain = async (id: string) =>
  requestJSON(`/rules/${encodeURIComponent(id)}/operate/start`, { method: 'POST', body: {} });

// 下线（停止）规则链
export const stopRuleChain = async (id: string) =>
  requestJSON(`/rules/${encodeURIComponent(id)}/operate/stop`, { method: 'POST', body: {} });

// 删除规则链
export const deleteRuleChain = async (id: string) =>
  requestJSON(`/rules/${encodeURIComponent(id)}`, { method: 'DELETE' });
