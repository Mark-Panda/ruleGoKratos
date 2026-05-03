/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useEffect, useState } from 'react';

import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';
import { Divider, Select, Spin, Typography } from '@douyinfe/semi-ui';

import { defaultFormMeta } from '../default-form-meta';
import { FlowNodeJSON } from '../../typings';
import { listWorkspaces, type WorkspaceItem } from '../../services/api-workspaces';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';
import type { FormInputsProps } from '../../form-components';

const workspaceSyncFormInputsProps: FormInputsProps = {
  propertyFilter: (k) => k !== 'workspaceId',
};

function WorkspaceSyncPicker() {
  const [workspaces, setWorkspaces] = useState<WorkspaceItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const rows = await listWorkspaces();
        if (!cancelled) {
          setWorkspaces(Array.isArray(rows) ? rows : []);
        }
      } catch {
        if (!cancelled) setWorkspaces([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div style={{ marginBottom: 12 }}>
      <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
        工作区
      </Typography.Text>
      <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 10 }}>
        从「工作区管理」列表选择；运行时将同步该工作区下全部 Git
        仓库（与页面上的「同步仓库」一致）。
      </Typography.Paragraph>
      {loading ? (
        <Spin size="small" />
      ) : (
        <Field name="inputsValues.workspaceId">
          {({ field }) => {
            const raw = field.value as { content?: unknown } | undefined;
            const current = raw?.content == null ? '' : String(raw.content);
            return (
              <Select
                style={{ width: '100%' }}
                placeholder="请选择工作区"
                value={current || undefined}
                filter
                showClear
                onChange={(v) => {
                  field.onChange({ type: 'constant', content: v == null ? '' : String(v) } as any);
                }}
              >
                {workspaces.map((w) => (
                  <Select.Option key={w.id} value={w.id}>
                    {(w.name || '').trim() || w.id}
                  </Select.Option>
                ))}
              </Select>
            );
          }}
        </Field>
      )}
    </div>
  );
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent>
      <WorkspaceSyncPicker />
      <FormInputs {...workspaceSyncFormInputsProps} />
      <Divider />
      <OutputsPeek />
    </FormContent>
  </>
);

export const workspaceSyncFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
  validate: {
    title: ({ value }) => (value ? undefined : 'Title is required'),
    'inputsValues.*': ({ value, name }) => {
      const valuePropertyKey = name.replace(/^inputsValues\./, '');
      if (valuePropertyKey !== 'workspaceId') return undefined;
      const content = value?.content;
      const s = content == null ? '' : String(content).trim();
      if (!s) {
        return '请选择工作区';
      }
      return undefined;
    },
  },
};
