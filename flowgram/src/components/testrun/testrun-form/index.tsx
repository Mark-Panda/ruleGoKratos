/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FC } from 'react';

import classNames from 'classnames';
import { DisplaySchemaTag } from '@flowgram.ai/form-materials';
import { Input, Switch, InputNumber } from '@douyinfe/semi-ui';

import { JsonValueEditor } from '../json-value-editor';
import { useFormMeta } from '../hooks/use-form-meta';
import { useFields } from '../hooks/use-fields';
import { useSyncDefault } from '../hooks';

import styles from './index.module.less';

interface TestRunFormProps {
  values: Record<string, unknown>;
  setValues: (values: Record<string, unknown>) => void;
}

export const TestRunForm: FC<TestRunFormProps> = ({ values, setValues }) => {
  const formMeta = useFormMeta();

  const fields = useFields({
    formMeta,
    values,
    setValues,
  });

  useSyncDefault({
    formMeta,
    values,
    setValues,
  });

  const renderField = (field: any) => {
    switch (field.type) {
      case 'boolean':
        return (
          <div className={styles.fieldInput}>
            <Switch checked={field.value} onChange={(checked) => field.onChange(checked)} />
          </div>
        );
      case 'integer':
        return (
          <div className={styles.fieldInput}>
            <InputNumber
              precision={0}
              value={field.value}
              onChange={(value) => field.onChange(value)}
              placeholder="请输入 integer"
            />
          </div>
        );
      case 'number':
        return (
          <div className={styles.fieldInput}>
            <InputNumber
              value={field.value}
              onChange={(value) => field.onChange(value)}
              placeholder="请输入 number"
            />
          </div>
        );
      case 'object':
        return (
          <div className={classNames(styles.fieldInput, styles.codeEditorWrapper)}>
            <JsonValueEditor value={field.value} onChange={(value) => field.onChange(value)} />
          </div>
        );
      case 'array':
        return (
          <div className={classNames(styles.fieldInput, styles.codeEditorWrapper)}>
            <JsonValueEditor value={field.value} onChange={(value) => field.onChange(value)} />
          </div>
        );
      default:
        return (
          <div className={styles.fieldInput}>
            <Input
              value={field.value}
              onChange={(value) => field.onChange(value)}
              placeholder="请输入 text"
            />
          </div>
        );
    }
  };

  return (
    <div className={styles.formContainer}>
      <div className={styles.fieldGroup}>
        <label htmlFor={'msgType'} className={styles.fieldLabel}>
          消息类型<span className={styles.requiredIndicator}>*</span>
        </label>
        <div className={styles.fieldInput}>
          <Input
            value={String(values.msgType ?? '')}
            onChange={(v) => setValues({ ...values, msgType: v })}
            placeholder="请输入消息类型"
          />
        </div>
      </div>

      <div className={styles.fieldGroup}>
        <label htmlFor={'metadata'} className={styles.fieldLabel}>
          元数据
          <span className={styles.fieldTypeIndicator}>
            <DisplaySchemaTag value={{ type: 'string' }} />
          </span>
        </label>
        <div className={styles.fieldInput}>
          <Input
            value={String(values.metadata ?? '')}
            onChange={(v) => setValues({ ...values, metadata: v })}
            placeholder="请输入元数据"
          />
        </div>
      </div>

      <div className={styles.fieldGroup}>
        <label htmlFor={'headers'} className={styles.fieldLabel}>
          请求头
          <span className={styles.fieldTypeIndicator}>
            <DisplaySchemaTag value={{ type: 'object' }} />
          </span>
        </label>
        <div className={classNames(styles.fieldInput, styles.codeEditorWrapper)}>
          <JsonValueEditor
            value={values.headers as any}
            onChange={(val) => setValues({ ...values, headers: val })}
          />
        </div>
      </div>

      <div className={styles.fieldGroup}>
        <label htmlFor={'body'} className={styles.fieldLabel}>
          请求体
          <span className={styles.fieldTypeIndicator}>
            <DisplaySchemaTag value={{ type: 'object' }} />
          </span>
        </label>
        <div className={classNames(styles.fieldInput, styles.codeEditorWrapper)}>
          <JsonValueEditor
            value={values.body as any}
            onChange={(val) => setValues({ ...values, body: val })}
          />
        </div>
      </div>

      {fields.length === 0
        ? null
        : fields.map((field) => (
            <div key={field.name} className={styles.fieldGroup}>
              <label htmlFor={field.name} className={styles.fieldLabel}>
                {field.name}
                {field.required && <span className={styles.requiredIndicator}>*</span>}
                <span className={styles.fieldTypeIndicator}>
                  <DisplaySchemaTag
                    value={{
                      type: field.type,
                      items: field.itemsType
                        ? {
                            type: field.itemsType,
                          }
                        : undefined,
                    }}
                  />
                </span>
              </label>
              {renderField(field)}
            </div>
          ))}
    </div>
  );
};
