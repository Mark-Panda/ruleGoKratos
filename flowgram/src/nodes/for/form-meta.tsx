/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FormRenderProps, FlowNodeJSON, Field, FormMeta } from '@flowgram.ai/free-layout-editor';
import { useService, WorkflowDocument } from '@flowgram.ai/free-layout-editor';
import { SubCanvasRender } from '@flowgram.ai/free-container-plugin';
import {
  BatchOutputs,
  createBatchOutputsFormPlugin,
  DisplayOutputs,
  IFlowValue,
  IFlowRefValue,
  validateFlowValue,
} from '@flowgram.ai/form-materials';
import { Input, Select } from '@douyinfe/semi-ui';

import { defaultFormMeta } from '../default-form-meta';
import { WorkflowNodeType } from '../constants';
import { useIsSidebar, useNodeRenderContext } from '../../hooks';
import { VariablePicker } from '../../form-components/variable-picker';
import { FormHeader, FormContent, FormItem, Feedback } from '../../form-components';

// 使用通用 FlowNodeJSON 即可，无需特定 forFor 字段
type ForNodeJSON = FlowNodeJSON;

export const ForFormRender = ({ form }: FormRenderProps<ForNodeJSON>) => {
  const isSidebar = useIsSidebar();
  const { readonly, node } = useNodeRenderContext();
  const formHeight = 115;

  // 移除 forFor 输入框

  const document = useService(WorkflowDocument);
  const nodeIdLinked = (
    <Field<IFlowValue> name={`nodeId`}>
      {({ field, fieldState }) => {
        const children = document.getAllNodes().filter((n) => n.parent?.id === node.id);
        const blockStart = children.find((n) => n.flowNodeType === WorkflowNodeType.BlockStart);
        const getDisplayName = (n: any) => {
          const json = document.toNodeJSON(n) as any;
          const title = json?.data?.title;
          return title ? String(title) : n.id;
        };
        let targetNodeId = '';
        let targetNodeLabel = '';
        if (blockStart) {
          const outLines = (blockStart as any).lines?.outputLines ?? [];
          const firstBusinessLine = outLines.find((line: any) => {
            const toNode = line?.to;
            return (
              toNode &&
              toNode.parent?.id === node.id &&
              ![WorkflowNodeType.BlockStart, WorkflowNodeType.BlockEnd].includes(
                toNode.flowNodeType as WorkflowNodeType
              )
            );
          });
          const toNode = firstBusinessLine?.to;
          if (toNode) {
            targetNodeId = toNode.id;
            targetNodeLabel = getDisplayName(toNode);
          }
        }
        const currentVal =
          typeof field.value?.content === 'string' ? (field.value?.content as string) : '';
        if (targetNodeId && currentVal !== targetNodeId) {
          field.onChange({ type: 'constant', content: targetNodeId });
        }
        return (
          <FormItem name={'处理节点ID'} type={'string'} required>
            <Input
              value={targetNodeLabel || ''}
              disabled
              placeholder={'请在子画布中将 block-start 连线到处理节点'}
            />
            <Feedback errors={fieldState?.errors} />
          </FormItem>
        );
      }}
    </Field>
  );

  const textInput = (
    <Field<IFlowValue> name={`note`}>
      {({ field, fieldState }) => (
        <FormItem name={'迭代值表达式'} type={'string'} required>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <Input
              value={
                typeof field.value?.content === 'string' ? (field.value?.content as string) : ''
              }
              onChange={(val) => field.onChange({ type: 'constant', content: val })}
              disabled={readonly}
              placeholder={'1..3'}
            />
            <VariablePicker
              size="small"
              disabled={readonly}
              onInsert={(text) => {
                const oldText =
                  typeof field.value?.content === 'string'
                    ? String(field.value?.content as string)
                    : '';
                const nextText = oldText ? `${oldText}${text}` : text;
                field.onChange({ type: 'template', content: nextText } as any);
              }}
            />
          </div>
          <Feedback errors={fieldState?.errors} />
        </FormItem>
      )}
    </Field>
  );

  const modeSelect = (
    <Field<IFlowValue> name={`operationMode`}>
      {({ field, fieldState }) => (
        <FormItem name={'执行模式'} type={'number'} required>
          {(() => {
            const cur = field.value?.content as number | undefined;
            if (typeof cur !== 'number') {
              field.onChange({ type: 'constant', content: 0 });
            }
            return null;
          })()}
          <Select
            value={(field.value?.content as number) ?? undefined}
            onChange={(val) => field.onChange({ type: 'constant', content: val })}
            disabled={readonly}
            optionList={[
              { label: '同步不合并执行结果', value: 0 },
              { label: '同步合并执行结果', value: 1 },
              { label: '同步覆盖执行结果', value: 2 },
              { label: '异步不合并执行结果', value: 3 },
            ]}
            style={{ width: '100%' }}
          />
          <Feedback errors={fieldState?.errors} />
        </FormItem>
      )}
    </Field>
  );

  const forOutputs = (
    <Field<Record<string, IFlowRefValue | undefined> | undefined> name={`forOutputs`}>
      {({ field, fieldState }) => (
        <FormItem name="forOutputs" type="object" vertical>
          <BatchOutputs
            style={{ width: '100%' }}
            value={field.value}
            onChange={(val) => field.onChange(val)}
            readonly={readonly}
            hasError={Object.keys(fieldState?.errors || {}).length > 0}
          />
          <Feedback errors={fieldState?.errors} />
        </FormItem>
      )}
    </Field>
  );

  if (isSidebar) {
    return (
      <>
        <FormHeader />
        <FormContent>
          {/* 已删除 forFor */}
          {textInput}
          {modeSelect}
          {nodeIdLinked}
          {/* {forOutputs} */}
        </FormContent>
      </>
    );
  }
  return (
    <>
      <FormHeader />
      <FormContent>
        {/* 已删除 forFor */}
        {nodeIdLinked}
        {textInput}
        {modeSelect}
        <SubCanvasRender offsetY={0} />
        <DisplayOutputs displayFromScope />
      </FormContent>
    </>
  );
};

export const formMeta: FormMeta = {
  ...defaultFormMeta,
  render: ForFormRender,
  validate: {
    ...(defaultFormMeta as any).validate,
    note: ({ value, context }) =>
      validateFlowValue(value, {
        node: context.node,
        required: true,
        errorMessages: { required: '迭代值表达式为必填' },
      }),
    operationMode: ({ value, context }) =>
      validateFlowValue(value, {
        node: context.node,
        required: true,
        errorMessages: { required: '执行模式为必填' },
      }),
    nodeId: ({ value, context }) =>
      validateFlowValue(value, {
        node: context.node,
        required: true,
        errorMessages: { required: '处理节点ID为必填' },
      }),
  },
  plugins: [createBatchOutputsFormPlugin({ outputKey: 'forOutputs', inferTargetKey: 'outputs' })],
};
