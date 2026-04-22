/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { Field } from '@flowgram.ai/free-layout-editor';
import {
  DynamicValueInput,
  PromptEditorWithVariables,
  type IFlowValue,
} from '@flowgram.ai/form-materials';
import { Button, Switch, TextArea, Toast } from '@douyinfe/semi-ui';

import { VariablePicker } from '../variable-picker';
import { FormItem } from '../form-item';
import { Feedback } from '../feedback';
import { JsonSchema } from '../../typings';
import { useEffectiveReadonly, useIsSidebar, useNodeRenderContext } from '../../hooks';
import {
  CANVAS_TWO_LINE_BOX_STYLE,
  normalizeCanvasPreviewText,
  summarizeFlowValue,
  truncateCanvasText,
} from '../../utils/canvas-node-preview';
import { tryFormatJsonPretty } from '../../utils/format-json-pretty';
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
  const readonly = useEffectiveReadonly();
  const isSidebar = useIsSidebar();
  const { readonly: playgroundReadonly } = useNodeRenderContext();
  const propertyFilter = props?.propertyFilter;
  const propertyKeyOrder = props?.propertyKeyOrder;

  if (!isSidebar) {
    return null;
  }

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
                const switchRow = property.type === 'boolean' && !isTemplate;
                const jsonFormat =
                  (property as { extra?: { jsonFormat?: boolean } }).extra?.jsonFormat === true ||
                  /（JSON）|\(JSON\)/i.test(displayLabel);
                const formatJsonField = () => {
                  const raw = summarizeFlowValue(field.value as IFlowValue | undefined);
                  const r = tryFormatJsonPretty(raw);
                  if (!r.ok) {
                    Toast.warning({
                      content: `无法格式化：${r.error}。含变量时请保证整体为合法 JSON。`,
                    });
                    return;
                  }
                  const t = (field.value as IFlowValue | undefined)?.type;
                  const nextType = t === 'constant' ? 'constant' : 'template';
                  field.onChange({ type: nextType, content: r.text } as IFlowValue);
                  Toast.success({ content: 'JSON 已格式化' });
                };
                const renderCore = () => {
                  if (property.type === 'boolean') {
                    if (isTemplate) {
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
                    const constantVal = (field.value as { type?: string; content?: unknown }) ?? {};
                    const defVal = property.default as { type?: string; content?: unknown } | undefined;
                    const on =
                      constantVal.type === 'constant'
                        ? Boolean(constantVal.content)
                        : defVal?.type === 'constant'
                          ? Boolean(defVal.content)
                          : false;
                    if (!isSidebar) {
                      return (
                        <div
                          style={{
                            padding: '6px 10px',
                            borderRadius: 4,
                            fontSize: 12,
                            color: '#4e5969',
                            background: '#f7f8fa',
                            display: 'inline-block',
                          }}
                        >
                          {on ? '开' : '关'}
                        </div>
                      );
                    }
                    return (
                      <Switch
                        checked={on}
                        disabled={readonly}
                        onChange={(c) => field.onChange({ type: 'constant', content: c })}
                      />
                    );
                  }
                  if (formComponent === 'prompt-editor') {
                    // 在画布视图中只显示截断的文本预览
                    if (!isSidebar) {
                      const content =
                        typeof (field.value as any)?.content === 'string'
                          ? String((field.value as any)?.content)
                          : '';
                      const shown =
                        content.trim() === ''
                          ? '(空)'
                          : truncateCanvasText(normalizeCanvasPreviewText(content), 220);
                      return (
                        <div style={CANVAS_TWO_LINE_BOX_STYLE}>
                          {shown}
                        </div>
                      );
                    }
                    // 在侧边栏中显示完整的编辑器
                    return (
                      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6 }}>
                        <div style={{ flex: 1, minWidth: 0 }}>
                          {!readonly && jsonFormat ? (
                            <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 4 }}>
                              <Button size="small" type="tertiary" onClick={formatJsonField}>
                                格式化 JSON
                              </Button>
                            </div>
                          ) : null}
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
                      const shown =
                        content.trim() === ''
                          ? '(空)'
                          : truncateCanvasText(normalizeCanvasPreviewText(content), 220);
                      return (
                        <div
                          style={{ ...CANVAS_TWO_LINE_BOX_STYLE, fontFamily: 'monospace' }}
                        >
                          {shown}
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
                    if (!isSidebar) {
                      if (isTemplate) {
                        return (
                          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6 }}>
                            <div style={{ flex: 1, minWidth: 0 }}>
                              {jsonFormat && !playgroundReadonly ? (
                                <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 4 }}>
                                  <Button size="small" type="tertiary" onClick={formatJsonField}>
                                    格式化 JSON
                                  </Button>
                                </div>
                              ) : null}
                              <TextArea
                                value={summarizeFlowValue(field.value as IFlowValue | undefined)}
                                onChange={(v) =>
                                  field.onChange({
                                    type: 'template',
                                    content: String(v ?? ''),
                                  } as any)
                                }
                                disabled={playgroundReadonly}
                                autosize={{ minRows: 2, maxRows: 2 }}
                              />
                            </div>
                            <VariablePicker
                              size="small"
                              disabled={playgroundReadonly}
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
                      const pv = summarizeFlowValue(field.value as IFlowValue | undefined);
                      const shown =
                        pv.trim() === ''
                          ? '(空)'
                          : truncateCanvasText(normalizeCanvasPreviewText(pv), 220);
                      return (
                        <div style={CANVAS_TWO_LINE_BOX_STYLE}>
                          {shown}
                        </div>
                      );
                    }
                    return (
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        <div style={{ flex: 1, minWidth: 0 }}>
                          {!readonly && jsonFormat ? (
                            <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 4 }}>
                              <Button size="small" type="tertiary" onClick={formatJsonField}>
                                格式化 JSON
                              </Button>
                            </div>
                          ) : null}
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
                        <div style={CANVAS_TWO_LINE_BOX_STYLE}>
                          {truncateCanvasText(normalizeCanvasPreviewText(preview), 220)}
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
                        <div style={CANVAS_TWO_LINE_BOX_STYLE}>
                          {truncateCanvasText(normalizeCanvasPreviewText(preview), 220)}
                        </div>
                      );
                    }
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
                    if (!isSidebar) {
                      const selectedValues =
                        (field.value as IFlowValue | undefined)?.type === 'constant' &&
                        Array.isArray((field.value as IFlowValue)?.content)
                          ? ((field.value as IFlowValue).content as string[])
                          : [];
                      const text =
                        selectedValues.length > 0 ? selectedValues.join(', ') : '(未选择)';
                      return (
                        <div style={CANVAS_TWO_LINE_BOX_STYLE}>
                          {truncateCanvasText(normalizeCanvasPreviewText(text), 220)}
                        </div>
                      );
                    }
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
                    if (!isSidebar) {
                      const arr = Array.isArray((field.value as IFlowValue | undefined)?.content)
                        ? (((field.value as IFlowValue).content as unknown[]) ?? []).map((x) =>
                            String(x ?? '')
                          )
                        : [];
                      const text = arr.length === 0 ? '(空)' : arr.join(', ');
                      return (
                        <div style={CANVAS_TWO_LINE_BOX_STYLE}>
                          {truncateCanvasText(normalizeCanvasPreviewText(text), 220)}
                        </div>
                      );
                    }
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
                    if (!isSidebar) {
                      const cron =
                        typeof (field.value as IFlowValue | undefined)?.content === 'string'
                          ? String((field.value as IFlowValue).content)
                          : '*/10 * * * * *';
                      return (
                        <div
                          style={{
                            ...CANVAS_TWO_LINE_BOX_STYLE,
                            fontFamily: 'monospace',
                          }}
                        >
                          {truncateCanvasText(normalizeCanvasPreviewText(cron), 220)}
                        </div>
                      );
                    }
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
                  <>
                    <FormItem
                      name={displayLabel}
                      vertical={vertical}
                      switchRow={switchRow}
                      type={property.type as string}
                      required={required.includes(key)}
                      description={property.extra?.description}
                    >
                      {switchRow ? (
                        renderCore()
                      ) : (
                        <>
                          {null}
                          {renderCore()}
                          <Feedback errors={fieldState?.errors} warnings={fieldState?.warnings} />
                        </>
                      )}
                    </FormItem>
                    {switchRow ? (
                      <Feedback errors={fieldState?.errors} warnings={fieldState?.warnings} />
                    ) : null}
                  </>
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
