/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useEffect, useMemo, useRef, useState } from 'react';

import { IFlowValue } from '@flowgram.ai/form-materials';
import { Select } from '@douyinfe/semi-ui';

import { getRuleList } from '../../services/api-rules';

interface RuleSelectProps {
  value?: IFlowValue;
  onChange: (next: IFlowValue) => void;
  readonly?: boolean;
  hasError?: boolean;
}

export const RuleSelect: React.FC<RuleSelectProps> = ({ value, onChange, readonly, hasError }) => {
  const [items, setItems] = useState<Array<{ id: string; name?: string }>>([]);
  const [page, setPage] = useState(1);
  const [size] = useState(5);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [keywords, setKeywords] = useState<string>('');
  const [selectedLabel, setSelectedLabel] = useState<string | undefined>(undefined);
  const debounceTimer = useRef<number | undefined>(undefined);

  const fetchList = async (nextPage: number, kw?: string, append = false) => {
    setLoading(true);
    try {
      const resp = await getRuleList({ page: nextPage, size, keywords: kw, root: false });
      const list = Array.isArray((resp as any)?.items) ? (resp as any).items : [];
      const mapped: Array<{ id: string; name?: string }> = list
        .map((x: any) => ({ id: String(x?.ruleChain?.id ?? ''), name: x?.ruleChain?.name }))
        .filter((x: { id: string; name?: string }) => !!x.id);
      setItems((prev) => (append ? [...prev, ...mapped] : mapped));
      setTotal(Number((resp as any)?.total ?? mapped.length));
      setPage(nextPage);
      const selectedId =
        (value?.type === 'constant' ? (value.content as string) : undefined) ?? undefined;
      if (selectedId) {
        const hit =
          mapped.find((i: { id: string }) => i.id === selectedId) ||
          (append ? items.find((i: { id: string }) => i.id === selectedId) : undefined);
        if (hit) setSelectedLabel(hit.name ? String(hit.name) : hit.id);
      }
    } catch (e) {
      if (!append) setItems([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchList(1, keywords, false);
  }, []);

  const options = useMemo(() => {
    const base = items.map((n) => ({ label: n.name ? String(n.name) : n.id, value: n.id }));
    const canLoadMore = items.length < total;
    return canLoadMore ? [...base, { label: '加载更多…', value: '__LOAD_MORE__' }] : base;
  }, [items, total]);

  const selectedValue =
    (value?.type === 'constant' ? (value.content as string) : undefined) ?? undefined;

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
    const opt = items.find((i: { id: string }) => i.id === String(val));
    setSelectedLabel(opt ? (opt.name ? String(opt.name) : opt.id) : undefined);
    onChange({ type: 'constant', content: String(val) });
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
        onSearch={handleSearch}
        filter
        renderSelectedItem={() => (selectedLabel ? selectedLabel : selectedValue)}
      />
    </div>
  );
};
