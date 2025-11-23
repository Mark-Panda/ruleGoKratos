/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useLayoutEffect } from 'react';

import { Field, FieldArray } from '@flowgram.ai/free-layout-editor';
import { ConditionRowValueType } from '@flowgram.ai/form-materials';
import { Button, Input, Select, Tag } from '@douyinfe/semi-ui';
import { IconPlus, IconCrossCircleStroked } from '@douyinfe/semi-icons';

import { alphaNanoid } from '../../../utils';
import { useNodeRenderContext } from '../../../hooks';
import { VariablePicker } from '../../../form-components/variable-picker';
import { FormItem } from '../../../form-components';
import { Feedback } from '../../../form-components';
import {
  CaseWrapper,
  RowsWrapper,
  GroupBracket,
  GroupLabel,
  CaseHeader,
  CaseTitle,
  ConditionPort,
  ConditionRow,
  OperatorSelectWrapper,
} from './styles';

interface CaseGroupValue {
  operator: 'and' | 'or';
  rows: ConditionRowValueType[];
}
interface CaseItemValue {
  key: string;
  groups: CaseGroupValue[];
}

export function CaseInputs() {
  const { node, readonly } = useNodeRenderContext();

  useLayoutEffect(() => {
    window.requestAnimationFrame(() => {
      node.ports.updateDynamicPorts();
    });
  }, [node]);

  return (
    <FieldArray name="cases">
      {({ field }) => (
        <>
          {field.map((child, index) => (
            <Field<CaseItemValue> key={child.name} name={child.name}>
              {({ field: childField, fieldState: childState }) => {
                const label = index === 0 ? 'IF' : 'ELSE IF';
                const groups: CaseGroupValue[] = (childField.value?.groups ??
                  []) as CaseGroupValue[];
                const ensureGroup = (g?: CaseGroupValue): CaseGroupValue => ({
                  operator: g?.operator ?? 'and',
                  rows: (g?.rows ?? []) as ConditionRowValueType[],
                });
                const setGroups = (next: CaseGroupValue[]) =>
                  childField.onChange({ ...childField.value, groups: next });
                const appendGroup = () =>
                  setGroups([
                    ...groups,
                    {
                      operator: 'and',
                      rows: [{ type: 'expression', content: '' } as ConditionRowValueType],
                    },
                  ]);
                const appendRowToGroup = (gi: number) => {
                  const safe = groups.map(ensureGroup);
                  const target = safe[gi];
                  const nextRows: ConditionRowValueType[] = [
                    ...target.rows,
                    { type: 'expression', content: '' } as ConditionRowValueType,
                  ];
                  const nextGroups = safe.map((g, idx) =>
                    idx === gi ? { ...g, rows: nextRows } : g
                  );
                  setGroups(nextGroups);
                };
                const deleteCase = () => field.delete(index);
                return (
                  <CaseWrapper>
                    <CaseHeader>
                      <CaseTitle>{label}</CaseTitle>
                      {!readonly && (
                        <>
                          <Button size="small" onClick={() => appendGroup()}>
                            添加 OR 条件
                          </Button>
                          <Button
                            theme="borderless"
                            icon={<IconCrossCircleStroked />}
                            onClick={deleteCase}
                          />
                        </>
                      )}
                    </CaseHeader>
                    {groups.map((g, gi) => {
                      const safe = ensureGroup(g);
                      const rows = safe.rows;
                      const displayRows = rows.map(
                        (r) => r ?? ({ type: 'expression', content: '' } as ConditionRowValueType)
                      );
                      const deleteRow = (giToDel: number, rToDel: number) => {
                        const nextGroups = groups.map(ensureGroup);
                        const target = nextGroups[giToDel];
                        const nextRows = [...(target.rows ?? [])];
                        nextRows.splice(rToDel, 1);
                        if (nextRows.length === 0) {
                          nextRows.push({
                            type: 'expression',
                            content: '',
                          } as ConditionRowValueType);
                        }
                        nextGroups[giToDel] = { ...target, rows: nextRows } as any;
                        setGroups(nextGroups);
                      };
                      return (
                        <div key={`group-${gi}`}>
                          {gi > 0 && (
                            <div style={{ margin: '6px 0' }}>
                              <Tag type="light" color="amber" size="small">
                                OR
                              </Tag>
                            </div>
                          )}
                          <RowsWrapper>
                            <GroupBracket />
                            <GroupLabel>{(safe.operator ?? 'and').toUpperCase()}</GroupLabel>
                            {displayRows.map((row, rIndex) => (
                              <FormItem
                                key={rIndex}
                                name={`row_${gi}_${rIndex}`}
                                type="string"
                                required={true}
                                labelWidth={50}
                              >
                                <ConditionRow>
                                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                                    <Input
                                      disabled={readonly}
                                      placeholder="左值：手动输入"
                                      value={
                                        typeof (row as any)?.left?.content === 'string'
                                          ? (row as any).left.content
                                          : ''
                                      }
                                      onChange={(val) => {
                                        const nextGroups = groups.map(ensureGroup);
                                        const nextRows = [...nextGroups[gi].rows];
                                        const base = (row ??
                                          ({
                                            type: 'expression',
                                            content: '',
                                          } as ConditionRowValueType)) as any;
                                        const nextRow: any = {
                                          ...base,
                                          left: { type: 'constant', content: String(val ?? '') },
                                          operator: base.operator ?? '==',
                                          right: base.right ?? { type: 'constant', content: '' },
                                        };
                                        nextRows[rIndex] = nextRow as ConditionRowValueType;
                                        nextGroups[gi] = { ...nextGroups[gi], rows: nextRows };
                                        setGroups(nextGroups);
                                      }}
                                    />
                                    <VariablePicker
                                      size="small"
                                      disabled={readonly}
                                      onInsert={(text) => {
                                        const nextGroups = groups.map(ensureGroup);
                                        const nextRows = [...nextGroups[gi].rows];
                                        const base = (row ??
                                          ({
                                            type: 'expression',
                                            content: '',
                                          } as ConditionRowValueType)) as any;
                                        const oldText =
                                          typeof base?.left?.content === 'string'
                                            ? String(base.left.content)
                                            : '';
                                        const nextText = oldText ? `${oldText}${text}` : text;
                                        const nextRow: any = {
                                          ...base,
                                          left: { type: 'template', content: nextText },
                                          operator: base.operator ?? '==',
                                          right: base.right ?? { type: 'constant', content: '' },
                                        };
                                        nextRows[rIndex] = nextRow as ConditionRowValueType;
                                        nextGroups[gi] = { ...nextGroups[gi], rows: nextRows };
                                        setGroups(nextGroups);
                                      }}
                                    />
                                  </div>
                                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                                    <Input
                                      disabled={readonly}
                                      placeholder="右值：手动输入"
                                      value={
                                        typeof (row as any)?.right?.content === 'string'
                                          ? (row as any).right.content
                                          : ''
                                      }
                                      onChange={(val) => {
                                        const nextGroups = groups.map(ensureGroup);
                                        const nextRows = [...nextGroups[gi].rows];
                                        const base = (row ??
                                          ({
                                            type: 'expression',
                                            content: '',
                                          } as ConditionRowValueType)) as any;
                                        const nextRow: any = {
                                          ...base,
                                          right: { type: 'constant', content: String(val ?? '') },
                                          operator: base.operator ?? '==',
                                          left: base.left ?? { type: 'constant', content: '' },
                                        };
                                        nextRows[rIndex] = nextRow as ConditionRowValueType;
                                        nextGroups[gi] = { ...nextGroups[gi], rows: nextRows };
                                        setGroups(nextGroups);
                                      }}
                                    />
                                    <VariablePicker
                                      size="small"
                                      disabled={readonly}
                                      onInsert={(text) => {
                                        const nextGroups = groups.map(ensureGroup);
                                        const nextRows = [...nextGroups[gi].rows];
                                        const base = (row ??
                                          ({
                                            type: 'expression',
                                            content: '',
                                          } as ConditionRowValueType)) as any;
                                        const oldText =
                                          typeof base?.right?.content === 'string'
                                            ? String(base.right.content)
                                            : '';
                                        const nextText = oldText ? `${oldText}${text}` : text;
                                        const nextRow: any = {
                                          ...base,
                                          right: { type: 'template', content: nextText },
                                          operator: base.operator ?? '==',
                                          left: base.left ?? { type: 'constant', content: '' },
                                        };
                                        nextRows[rIndex] = nextRow as ConditionRowValueType;
                                        nextGroups[gi] = { ...nextGroups[gi], rows: nextRows };
                                        setGroups(nextGroups);
                                      }}
                                    />
                                  </div>
                                  <OperatorSelectWrapper>
                                    <Select
                                      disabled={readonly}
                                      value={(row as any)?.operator ?? '=='}
                                      onChange={(op) => {
                                        const nextGroups = groups.map(ensureGroup);
                                        const nextRows = [...nextGroups[gi].rows];
                                        const base = (row ??
                                          ({
                                            type: 'expression',
                                            content: '',
                                          } as ConditionRowValueType)) as any;
                                        const nextRow: any = {
                                          ...base,
                                          operator: op,
                                          left: base.left ?? { type: 'constant', content: '' },
                                          right: base.right ?? { type: 'constant', content: '' },
                                        };
                                        nextRows[rIndex] = nextRow as ConditionRowValueType;
                                        nextGroups[gi] = { ...nextGroups[gi], rows: nextRows };
                                        setGroups(nextGroups);
                                      }}
                                      optionList={[
                                        { label: '等于', value: '==' },
                                        { label: '不等于', value: '!=' },
                                        { label: '大于', value: '>' },
                                        { label: '大于等于', value: '>=' },
                                        { label: '小于', value: '<' },
                                        { label: '小于等于', value: '<=' },
                                        { label: '包含', value: 'contains' },
                                      ]}
                                    />
                                    {!readonly && (
                                      <Button
                                        theme="borderless"
                                        icon={<IconCrossCircleStroked />}
                                        size="small"
                                        onClick={() => deleteRow(gi, rIndex)}
                                        style={{ marginTop: 6 }}
                                      />
                                    )}
                                  </OperatorSelectWrapper>
                                </ConditionRow>
                              </FormItem>
                            ))}
                            {!readonly && (
                              <Button size="small" onClick={() => appendRowToGroup(gi)}>
                                + 添加 AND 条件
                              </Button>
                            )}
                          </RowsWrapper>
                        </div>
                      );
                    })}
                    <Feedback errors={childState?.errors} invalid={childState?.invalid} />
                    <ConditionPort
                      data-port-id={childField.value?.key ?? `case_${index}`}
                      data-port-type="output"
                    />
                  </CaseWrapper>
                );
              }}
            </Field>
          ))}
          <FormItem name="ELSE" type="boolean" required={true} labelWidth={100}>
            <div style={{ color: 'rgba(6, 7, 9, 0.65)', fontSize: 12, marginBottom: 4 }}>
              ELSE 用于定义当 if 条件不满足时执行的逻辑。
            </div>
            <ConditionPort data-port-id="Default" data-port-type="output" />
          </FormItem>
          {!readonly && (
            <div>
              <Button
                theme="borderless"
                icon={<IconPlus />}
                onClick={() =>
                  field.append({
                    key: `case_${alphaNanoid(6)}`,
                    groups: [
                      {
                        operator: 'and',
                        rows: [{ type: 'expression', content: '' } as ConditionRowValueType],
                      },
                    ],
                  })
                }
              >
                + ELIF
              </Button>
            </div>
          )}
        </>
      )}
    </FieldArray>
  );
}
