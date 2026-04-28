import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';
import { Collapse, Divider, Select, Switch, TextArea, Typography } from '@douyinfe/semi-ui';

import { defaultFormMeta } from '../default-form-meta';
import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';

const parseModeOptions = [
  { value: 'strict', label: 'strict（轻量解析）' },
  { value: 'auto', label: 'auto（默认）' },
  { value: 'aggressive', label: 'aggressive（激进修复）' },
];

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent>
      <Collapse keepDOM defaultActiveKey={[]} style={{ marginBottom: 12 }}>
        <Collapse.Panel itemKey="advanced" header={<Typography.Text strong>高级配置</Typography.Text>}>
          <Field name="inputsValues.parseMode">
            {({ field }) => {
              const current = String(field.value?.content ?? 'auto') || 'auto';
              return (
                <div style={{ marginBottom: 12 }}>
                  <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                    解析强度
                  </Typography.Text>
                  <Select
                    style={{ width: '100%' }}
                    value={current}
                    onChange={(v) => {
                      field.onChange({ type: 'constant', content: String(v || 'auto') } as any);
                    }}
                  >
                    {parseModeOptions.map((opt) => (
                      <Select.Option key={opt.value} value={opt.value}>
                        {opt.label}
                      </Select.Option>
                    ))}
                  </Select>
                </div>
              );
            }}
          </Field>
          <Field name="inputsValues.schemaPaths">
            {({ field }) => {
              const current = String(field.value?.content ?? '');
              return (
                <div style={{ marginBottom: 12 }}>
                  <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                    Schema 路径约束
                  </Typography.Text>
                  <TextArea
                    value={current}
                    rows={4}
                    maxCount={1200}
                    placeholder={`data[].name
data[].spaceName
data[].manager
data[].language
data[].framework`}
                    onChange={(v) => {
                      field.onChange({ type: 'constant', content: String(v || '') } as any);
                    }}
                  />
                  <Typography.Text type="tertiary" size="small" style={{ display: 'block', marginTop: 6 }}>
                    支持逗号/分号/换行分隔，用于候选评分优选与缺失字段补齐。
                  </Typography.Text>
                </div>
              );
            }}
          </Field>
          <Field name="inputsValues.emitReport">
            {({ field }) => {
              const checked = Boolean(field.value?.content ?? false);
              return (
                <div style={{ marginBottom: 4 }}>
                  <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                    输出修复报告
                  </Typography.Text>
                  <Switch
                    checked={checked}
                    checkedText="开启"
                    uncheckedText="关闭"
                    onChange={(v) => {
                      field.onChange({ type: 'constant', content: Boolean(v) } as any);
                    }}
                  />
                  <Typography.Text type="tertiary" size="small" style={{ display: 'block', marginTop: 6 }}>
                    开启后输出 source_strategy、repair_strategies、score、schema_missing 等诊断信息，便于排障。
                  </Typography.Text>
                </div>
              );
            }}
          </Field>
        </Collapse.Panel>
      </Collapse>
      <FormInputs propertyFilter={(k) => k !== 'parseMode' && k !== 'schemaPaths' && k !== 'emitReport'} />
      <Divider />
      <OutputsPeek />
    </FormContent>
  </>
);

export const formMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
