/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useCallback } from 'react';

import { DisplaySchemaTag } from '@flowgram.ai/form-materials';
import { Typography, Tooltip } from '@douyinfe/semi-ui';

import './index.css';

const { Text } = Typography;

interface FormItemProps {
  children: React.ReactNode;
  name: string;
  type?: string;
  required?: boolean;
  description?: string;
  labelWidth?: number;
  labelStyle?: React.CSSProperties;
  vertical?: boolean;
  /** 布尔开关行：标签占满左侧可用宽度，控件右对齐，避免窄标签列裁切文案 */
  switchRow?: boolean;
  style?: React.CSSProperties;
}
export function FormItem({
  children,
  name,
  required,
  description,
  type,
  labelWidth,
  labelStyle,
  vertical,
  switchRow,
  style,
}: FormItemProps): JSX.Element {
  const renderTitle = useCallback(
    (showTooltip?: boolean) => (
      <div style={{ width: '0', display: 'flex', flex: '1' }}>
        <Text style={{ width: '100%' }} ellipsis={{ showTooltip: !!showTooltip }}>
          {name}
          {required && <span style={{ color: '#f93920', paddingLeft: '2px' }}>*</span>}
        </Text>
      </div>
    ),
    []
  );

  if (switchRow && !vertical) {
    return (
      <div
        style={{
          fontSize: 12,
          marginBottom: 6,
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          gap: 4,
          ...style,
        }}
      >
        <div
          style={{
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            gap: 10,
          }}
        >
          <div
            style={{
              flex: 1,
              minWidth: 0,
              display: 'flex',
              alignItems: 'center',
              columnGap: 4,
              color: 'var(--semi-color-text-0)',
            }}
          >
            {type && <DisplaySchemaTag value={{ type }} />}
            {description ? (
              <Tooltip content={description}>{renderTitle()}</Tooltip>
            ) : (
              renderTitle(true)
            )}
          </div>
          <div style={{ flexShrink: 0, display: 'flex', alignItems: 'center' }}>{children}</div>
        </div>
      </div>
    );
  }

  return (
    <div
      style={{
        fontSize: 12,
        marginBottom: 6,
        width: '100%',
        position: 'relative',
        display: 'flex',
        gap: 8,
        ...(vertical
          ? { flexDirection: 'column' }
          : {
              justifyContent: 'center',
              alignItems: 'center',
            }),
        ...style,
      }}
    >
      <div
        style={{
          justifyContent: 'center',
          alignItems: 'center',
          color: 'var(--semi-color-text-0)',
          width: labelWidth || 118,
          minWidth: labelWidth || 118,
          maxWidth: labelWidth || 118,
          position: 'relative',
          display: 'flex',
          columnGap: 4,
          flexShrink: 0,
          ...labelStyle,
        }}
      >
        {type && <DisplaySchemaTag value={{ type }} />}
        {description ? <Tooltip content={description}>{renderTitle()}</Tooltip> : renderTitle(true)}
      </div>

      <div
        style={{
          flexGrow: 1,
          minWidth: 0,
        }}
      >
        {children}
      </div>
    </div>
  );
}
