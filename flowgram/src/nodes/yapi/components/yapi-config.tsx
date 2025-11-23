/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React from 'react';

import { Field } from '@flowgram.ai/free-layout-editor';
import { Input, Select } from '@douyinfe/semi-ui';

import { useIsSidebar, useNodeRenderContext } from '../../../hooks';
import { FormItem } from '../../../form-components/form-item';

export function YapiConfig() {
  const isSidebar = useIsSidebar();
  const { readonly } = useNodeRenderContext();

  if (!isSidebar) {
    return null;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, padding: 12 }}>
      <Field<string> name="yapiConfig.baseUrl">
        {({ field }) => (
          <FormItem name="Base URL" required vertical>
            <Input
              value={field.value}
              onChange={field.onChange}
              placeholder="请输入 Yapi Base URL"
              disabled={readonly}
            />
          </FormItem>
        )}
      </Field>

      <Field<string> name="yapiConfig.userName">
        {({ field }) => (
          <FormItem name="用户名" required vertical>
            <Input
              value={field.value}
              onChange={field.onChange}
              placeholder="请输入用户名"
              disabled={readonly}
            />
          </FormItem>
        )}
      </Field>

      <Field<string> name="yapiConfig.password">
        {({ field }) => (
          <FormItem name="密码" required vertical>
            <Input
              type="password"
              value={field.value}
              onChange={field.onChange}
              placeholder="请输入密码"
              disabled={readonly}
            />
          </FormItem>
        )}
      </Field>

      <Field<string> name="yapiConfig.interfacePath">
        {({ field }) => (
          <FormItem name="接口路径" required vertical>
            <Input
              value={field.value}
              onChange={field.onChange}
              placeholder="请输入接口路径 (例如 /api/project/list)"
              disabled={readonly}
            />
          </FormItem>
        )}
      </Field>

      <Field<string> name="yapiConfig.loginType">
        {({ field }) => (
          <FormItem name="登录方式" required vertical>
            <Select
              value={field.value}
              onChange={(v) => field.onChange(v as string)}
              disabled={readonly}
              style={{ width: '100%' }}
            >
              <Select.Option value="ldap">LDAP</Select.Option>
              <Select.Option value="normal">普通登录</Select.Option>
            </Select>
          </FormItem>
        )}
      </Field>
    </div>
  );
}
