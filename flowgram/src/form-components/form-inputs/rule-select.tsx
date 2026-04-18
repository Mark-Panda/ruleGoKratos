/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useEffect, useMemo, useRef, useState } from 'react';

import { IFlowValue } from '@flowgram.ai/form-materials';
import { Select } from '@douyinfe/semi-ui';

import { getRuleList } from '../../services/api-rules';

function mapRuleListRow(x: unknown): { id: string; name?: string } | null {
  const raw = x as Record<string, unknown> | null | undefined;
  const rc = (raw?.ruleChain ?? raw?.rule_chain) as Record<string, unknown> | undefined;
  if (!rc || typeof rc !== 'object') return null;
  const id =
    (rc.id as string | undefined) ??
    (rc.ruleChainId as string | undefined) ??
    (rc.rule_chain_id as string | undefined) ??
    (rc.rule_chainId as string | undefined) ??
    '';
  if (!id) return null;
  const name = rc.name as string | undefined;
  return {
    id: String(id),
    name: name != null && String(name).trim() !== '' ? String(name) : undefined,
  };
}

interface RuleSelectProps {
  value?: IFlowValue;
  onChange: (next: IFlowValue) => void;
  readonly?: boolean;
  hasError?: boolean;
}

export const RuleSelect: React.FC<RuleSelectProps> = ({ value, onChange, readonly, hasError }) => {
  const [items, setItems] = useState<Array<{ id: string; name?: string }>>([]);
  const [page, setPage] = useState(1);
  const [size] = useState(50);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [keywords, setKeywords] = useState<string>('');
  const [selectedLabel, setSelectedLabel] = useState<string | undefined>(undefined);
  const debounceTimer = useRef<number | undefined>(undefined);

  const fetchList = async (nextPage: number, kw?: string, append = false) => {
    setLoading(true);
    try {
      const resp = await getRuleList({
        page: nextPage,
        size,
        keywords: kw?.trim() || undefined,
        root: false,
        disabled: false,
      });
      const list = Array.isArray((resp as any)?.items) ? (resp as any).items : [];
      const mapped = list
        .map((row: unknown) => mapRuleListRow(row))
        .filter(
          (x: { id: string; name?: string } | null): x is { id: string; name?: string } => x != null
        );
      setItems((prev) => (append ? [...prev, ...mapped] : mapped));
      setTotal(Number((resp as any)?.total ?? mapped.length));
      setPage(nextPage);
    } catch (e) {
      if (!append) setItems([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchList(1, keywords, false);
  }, []);

  const selectedValue =
    (value?.type === 'constant' ? (value.content as string) : undefined) ?? undefined;

  const optionsWithOrphan = useMemo(() => {
    const base = items.map((n) => ({ label: n.name ? String(n.name) : n.id, value: n.id }));
    if (!selectedValue || base.some((o) => o.value === selectedValue)) {
      return base;
    }
    return [
      {
        label: `${selectedValue}（当前值，未在已加载列表中）`,
        value: selectedValue,
      },
      ...base,
    ];
  }, [items, selectedValue]);

  useEffect(() => {
    const id =
      value?.type === 'constant' && value.content != null && String(value.content).trim() !== ''
        ? String(value.content)
        : '';
    if (!id) {
      setSelectedLabel(undefined);
      return;
    }
    const hit = items.find((i) => i.id === id);
    setSelectedLabel(hit ? (hit.name ? String(hit.name) : hit.id) : id);
  }, [value, items]);

  const options = useMemo(() => {
    const base = optionsWithOrphan;
    const canLoadMore = items.length < total;
    return canLoadMore ? [...base, { label: '加载更多…', value: '__LOAD_MORE__' }] : base;
  }, [optionsWithOrphan, items.length, total]);

  const handleSearch = (kw: string) => {
    setKeywords(kw);
    window.clearTimeout(debounceTimer.current);
    debounceTimer.current = window.setTimeout(() => {
      fetchList(1, kw, false);
    }, 300);
  };

  const handleChange = async (val: any) => {
    if (val === '__LOAD_MORE__') {
      const nextPage = page + 1;
      await fetchList(nextPage, keywords, true);
      return;
    }
    const id = String(val);
    const hit = items.find((i: { id: string }) => i.id === id);
    setSelectedLabel(hit ? (hit.name ? String(hit.name) : hit.id) : id);
    onChange({ type: 'constant', content: id });
  };

  return (
    <div style={{ width: '100%' }}>
      <Select
        value={selectedValue}
        onChange={handleChange}
        optionList={options}
        placeholder={readonly ? '只读' : '请选择子规则链'}
        disabled={readonly}
        insetLabel={hasError ? '!' : undefined}
        style={{ width: '100%' }}
        showClear
        loading={loading}
        remote
        onSearch={handleSearch}
        filter
        onDropdownVisibleChange={(open) => {
          if (open) fetchList(1, keywords, false);
        }}
        renderSelectedItem={() => (selectedLabel ? selectedLabel : selectedValue)}
      />
    </div>
  );
};
