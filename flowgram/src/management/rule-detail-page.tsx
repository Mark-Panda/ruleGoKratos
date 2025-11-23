/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useEffect, useState } from 'react';

import { Spin, Typography } from '@douyinfe/semi-ui';

import { setRuleBaseInfo } from '../services/rule-base-info';
import { getRuleDetail } from '../services/api-rules';
import { RuleDetail, RuleDetailData } from './rule-detail';

export const RuleDetailPage: React.FC<{ id: string; tab?: 'workflow' | 'design' }> = ({
  id,
  tab,
}) => {
  const [data, setData] = useState<RuleDetailData | undefined>();
  const [error, setError] = useState<string | undefined>();
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    let mounted = true;
    (async () => {
      setLoading(true);
      setError(undefined);
      try {
        const json = await getRuleDetail(id);
        if (mounted) {
          setData(json as RuleDetailData);
          try {
            setRuleBaseInfo((json as any)?.ruleChain);
          } catch {}
        }
      } catch (e) {
        if (mounted) setError(String((e as Error)?.message ?? e));
      } finally {
        if (mounted) setLoading(false);
      }
    })();
    return () => {
      mounted = false;
    };
  }, [id]);

  if (loading)
    return (
      <Spin
        tip="加载中..."
        style={{
          width: '100%',
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      />
    );
  if (error || !data)
    return (
      <div style={{ padding: 24 }}>
        <Typography.Text type="danger">加载失败：{error ?? '未知错误'}</Typography.Text>
      </div>
    );

  return (
    <RuleDetail
      data={data}
      onBack={() => {
        window.location.hash = '#/admin';
      }}
      initialTab={tab ?? 'workflow'}
    />
  );
};
