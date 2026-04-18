/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import { DynamicValueInput, PromptEditorWithVariables } from '@flowgram.ai/form-materials';

import { VariablePicker } from '../variable-picker';
import { FormItem } from '../form-item';
import { Feedback } from '../feedback';
import { JsonSchema } from '../../typings';
import { useNodeRenderContext, useIsSidebar } from '../../hooks';
import { SqlTemplateEditor } from './sql-template-editor';
import { RuleSelect } from './rule-select';
import { NodeIdSelect } from './node-id-select';
import { WhileDoTargetSelect } from './while-do-target';
import { NodeIdMultiSelect } from './node-id-multi-select';
import { CronEditor } from './cron-editor';

export type FormInputsProps = {
  /** 返回 true 的字段才会渲染；未设置则渲染全部 */
  propertyFilter?: (propertyKey: string) => boolean;
  /** 渲染顺序；未设置则按 inputs.properties 的键顺序 */
  propertyKeyOrder?: readonly string[];
};

export function FormInputs(props?: FormInputsProps) {
  const { readonly } = useNodeRenderContext();
  const isSidebar = useIsSidebar();
  const propertyFilter = props?.propertyFilter;
  const propertyKeyOrder = props?.propertyKeyOrder;

  return (
    <Field<JsonSchema> name="inputs">
      {({ field: inputsField }) => {
        const required = inputsField.value?.required || [];
        const properties = inputsField.value?.properties;
        if (!properties) {
          return <></>;
        }
        const allKeys = Object.keys(properties);
        const keys =
          propertyKeyOrder && propertyKeyOrder.length > 0
            ? propertyKeyOrder.filter(
                (k) => Boolean(properties[k]) && (!propertyFilter || propertyFilter(k))
              )
            : allKeys.filter((k) => !propertyFilter || propertyFilter(k));
        const content = keys.map((key) => {
          const property = properties[key];

          const enumList = (property as { enum?: unknown }).enum;
          const hasEnum = Array.isArray(enumList) && enumList.length > 0;
          const formComponent =
            property.extra?.formComponent ?? (hasEnum ? 'enum-select' : undefined);
          const displayLabel = property.extra?.label || key;

          const vertical = ['prompt-editor', 'sql-editor', 'while-do-target'].includes(
            formComponent || ''
          );

          return (
            <Field key={key} name={`inputsValues.${key}`} defaultValue={property.default}>
              {({ field, fieldState }) => {
                const isTemplate = (field.value as any)?.type === 'template';
                const renderCore = () => {
                  if (formComponent === 'prompt-editor') {
                    // 在画布视图中只显示截断的文本预览
                    if (!isSidebar) {
                      const content =
                        typeof (field.value as any)?.content === 'string'
                          ? String((field.value as any)?.content)
                          : '';
                      const truncated =
                        content.length > 100 ? content.slice(0, 100) + '...' : content;
                      return (
                        <div
                          style={{
                            padding: '8px',
                            background: '#f5f5f5',
                            borderRadius: '4px',
                            fontSize: '12px',
                            color: '#666',
                            whiteSpace: 'pre-wrap',
                            wordBreak: 'break-word',
                            maxHeight: '60px',
                            overflow: 'hidden',
                          }}
                        >
                          {truncated || '(空)'}
                        </div>
                      );
                    }
                    // 在侧边栏中显示完整的编辑器
                    return (
                      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6 }}>
                        <div style={{ flex: 1 }}>
                          <PromptEditorWithVariables
                            value={field.value}
                            onChange={field.onChange}
                            readonly={readonly}
                            hasError={Object.keys(fieldState?.errors || {}).length > 0}
                          />
                        </div>
                        <VariablePicker
                          size="small"
                          disabled={readonly}
                          onInsert={(text) => {
                            const oldText =
                              typeof (field.value as any)?.content === 'string'
                                ? String((field.value as any)?.content)
                                : '';
                            const nextText = oldText ? `${oldText}${text}` : text;
                            field.onChange({ type: 'template', content: nextText } as any);
                          }}
                        />
                      </div>
                    );
                  }
                  if (formComponent === 'sql-editor') {
                    // 在画布视图中只显示截断的文本预览
                    if (!isSidebar) {
                      const content =
                        typeof (field.value as any)?.content === 'string'
                          ? String((field.value as any)?.content)
                          : '';
                      const truncated =
                        content.length > 100 ? content.slice(0, 100) + '...' : content;
                      return (
                        <div
                          style={{
                            padding: '8px',
                            background: '#f5f5f5',
                            borderRadius: '4px',
                            fontSize: '12px',
                            color: '#666',
                            whiteSpace: 'pre-wrap',
                            wordBreak: 'break-word',
                            maxHeight: '60px',
                            overflow: 'hidden',
                            fontFamily: 'monospace',
                          }}
                        >
                          {truncated || '(空)'}
                        </div>
                      );
                    }
                    // 在侧边栏中显示完整的编辑器
                    return (
                      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6 }}>
                        <div style={{ flex: 1 }}>
                          <SqlTemplateEditor
                            value={field.value}
                            onChange={field.onChange}
                            readonly={readonly}
                            hasError={Object.keys(fieldState?.errors || {}).length > 0}
                          />
                        </div>
                        <VariablePicker
                          size="small"
                          disabled={readonly}
                          onInsert={(text) => {
                            const oldText =
                              typeof (field.value as any)?.content === 'string'
                                ? String((field.value as any)?.content)
                                : '';
                            const nextText = oldText ? `${oldText}${text}` : text;
                            field.onChange({ type: 'template', content: nextText } as any);
                          }}
                        />
                      </div>
                    );
                  }
                  if (isTemplate || !formComponent) {
                    return (
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <div style={{ flex: 1 }}>
                          <DynamicValueInput
                            value={field.value}
                            onChange={field.onChange}
                            readonly={readonly}
                            hasError={Object.keys(fieldState?.errors || {}).length > 0}
                            schema={property}
                          />
                        </div>
                        <VariablePicker
                          size="small"
                          disabled={readonly}
                          onInsert={(text) => {
                            const oldText =
                              typeof (field.value as any)?.content === 'string'
                                ? String((field.value as any)?.content)
                                : '';
                            const nextText = oldText ? `${oldText}${text}` : text;
                            field.onChange({ type: 'template', content: nextText } as any);
                          }}
                        />
                      </div>
                    );
                  }
                  if (formComponent === 'while-do-target') {
                    const extra = (property as { extra?: Record<string, unknown> }).extra;
                    if (!isSidebar) {
                      const c =
                        typeof (field.value as any)?.content === 'string'
                          ? String((field.value as any).content)
                          : '';
                      const preview =
                        c.trim() === ''
                          ? '(未选择)'
                          : c.startsWith('chain:')
                          ? `子规则链：${c.slice('chain:'.length)}`
                          : `节点：${c}`;
                      return (
                        <div
                          style={{
                            padding: '8px',
                            background: '#f5f5f5',
                            borderRadius: '4px',
                            fontSize: '12px',
                            color: '#666',
                            wordBreak: 'break-word',
                          }}
                        >
                          {preview}
                        </div>
                      );
                    }
                    return (
                      <WhileDoTargetSelect
                        value={field.value}
                        onChange={field.onChange}
                        readonly={readonly}
                        hasError={Object.keys(fieldState?.errors || {}).length > 0}
                        excludeSelf={extra?.nodeSelectorExcludeSelf === true}
                        excludeTypes={extra?.nodeSelectorExcludeTypes as string[] | undefined}
                        excludeIds={extra?.nodeSelectorExcludeIds as string[] | undefined}
                        preferSuccessDownstream={extra?.nodeSelectorPreferSuccessDownstream === true}
                      />
                    );
                  }
                  if (formComponent === 'node-selector') {
                    const extra = (property as { extra?: Record<string, unknown> }).extra;
                    return (
                      <NodeIdSelect
                        value={field.value}
                        onChange={field.onChange}
                        readonly={readonly}
                        hasError={Object.keys(fieldState?.errors || {}).length > 0}
                        excludeSelf={extra?.nodeSelectorExcludeSelf === true}
                        excludeTypes={extra?.nodeSelectorExcludeTypes as string[] | undefined}
                        excludeIds={extra?.nodeSelectorExcludeIds as string[] | undefined}
                        preferSuccessDownstream={extra?.nodeSelectorPreferSuccessDownstream === true}
                      />
                    );
                  }
                  if (formComponent === 'node-selector-multi') {
                    return (
                      <NodeIdMultiSelect
                        value={field.value}
                        onChange={field.onChange}
                        readonly={readonly}
                        hasError={Object.keys(fieldState?.errors || {}).length > 0}
                      />
                    );
                  }
                  if (formComponent === 'enum-select') {
                    return (
                      <EnumSelect
                        value={field.value}
                        onChange={field.onChange}
                        readonly={readonly}
                        hasError={Object.keys(fieldState?.errors || {}).length > 0}
                        schema={property}
                      />
                    );
                  }
                  if (formComponent === 'rule-select') {
                    return (
                      <RuleSelect
                        value={field.value}
                        onChange={field.onChange}
                        readonly={readonly}
                        hasError={Object.keys(fieldState?.errors || {}).length > 0}
                      />
                    );
                  }
                  if (formComponent === 'array-editor') {
                    return (
                      <ArrayEditor
                        value={field.value}
                        onChange={field.onChange}
                        readonly={readonly}
                        hasError={Object.keys(fieldState?.errors || {}).length > 0}
                        addButtonLabel={
                          typeof (property as { extra?: { arrayAddLabel?: string } }).extra
                            ?.arrayAddLabel === 'string'
                            ? (property as { extra?: { arrayAddLabel?: string } }).extra
                                ?.arrayAddLabel
                            : undefined
                        }
                      />
                    );
                  }
                  if (formComponent === 'cron-editor') {
                    return (
                      <CronEditor
                        value={field.value}
                        onChange={field.onChange}
                        readonly={readonly}
                        hasError={Object.keys(fieldState?.errors || {}).length > 0}
                      />
                    );
                  }
                  return (
                    <DynamicValueInput
                      value={field.value}
                      onChange={field.onChange}
                      readonly={readonly}
                      hasError={Object.keys(fieldState?.errors || {}).length > 0}
                      schema={property}
                    />
                  );
                };

                return (
                  <FormItem
                    name={displayLabel}
                    vertical={vertical}
                    type={property.type as string}
                    required={required.includes(key)}
                    description={property.extra?.description}
                  >
                    {null}
                    {renderCore()}
                    <Feedback errors={fieldState?.errors} warnings={fieldState?.warnings} />
                  </FormItem>
                );
              }}
            </Field>
          );
        });
        return <>{content}</>;
      }}
    </Field>
  );
}
import { EnumSelect } from './enum-select';
import { ArrayEditor } from './array-editor';
