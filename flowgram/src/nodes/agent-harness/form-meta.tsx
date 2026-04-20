/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useEffect, useMemo, useState } from 'react';

import { Checkbox, Divider, Select, Spin, Typography } from '@douyinfe/semi-ui';
import { DisplayOutputs } from '@flowgram.ai/form-materials';
import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';

import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs } from '../../form-components';
import type { FormInputsProps } from '../../form-components';
import { defaultFormMeta } from '../default-form-meta';
import {
  listLlmConfigs,
  listMCPConfigs,
  listSkills,
  type LlmConfigItem,
  type MCPConfigItem,
  type SkillItem,
} from '../../services/api-agent';
import { groupSkillPackages } from '../../utils/skill-packages';

function truncOneLine(s: string, max: number) {
  const one = s.replace(/\s+/g, ' ').trim();
  if (!one) return '（空）';
  if (one.length <= max) return one;
  return `${one.slice(0, max)}…`;
}

function flowStr(v: unknown): string {
  if (v == null || typeof v !== 'object' || !('content' in (v as object))) return '';
  const c = (v as { content?: unknown }).content;
  return c == null ? '' : String(c);
}

function flowBool(v: unknown): boolean {
  if (v == null || typeof v !== 'object' || !('content' in (v as object))) return false;
  return (v as { content?: unknown }).content === true;
}

function flowNum(v: unknown): number {
  if (v == null || typeof v !== 'object' || !('content' in (v as object))) return 0;
  const c = (v as { content?: unknown }).content;
  if (typeof c === 'number' && Number.isFinite(c)) return c;
  const n = Number(c);
  return Number.isFinite(n) ? n : 0;
}

function flowStringList(v: unknown): string[] {
  if (v == null || typeof v !== 'object' || !('content' in (v as object))) return [];
  const c = (v as { content?: unknown }).content;
  if (Array.isArray(c)) {
    return c.map((x) => String(x ?? '')).filter((s) => s.trim() !== '');
  }
  if (typeof c === 'string' && c.trim() !== '') {
    return c
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
  }
  return [];
}

/** 不渲染白名单字段（由下方勾选区维护）；LLM 配置/模型条目由专用下拉维护。 */
const AGENT_FORM_KEYS_NO_ALLOWLIST: readonly string[] = [
  'model',
  'userPrompt',
  'systemPrompt',
  'enableSkillTool',
  'enableMcpTool',
  'enableWorkspaceTools',
  'maxIterations',
  'maxToolCalls',
  'toolTimeoutSecs',
];

const agentFormInputsProps: FormInputsProps = {
  propertyFilter: (k) =>
    !['skillAllowlist', 'mcpAllowlist', 'llmConfigId', 'llmModelEntryId'].includes(k),
  propertyKeyOrder: AGENT_FORM_KEYS_NO_ALLOWLIST,
};

function toggleString(list: string[], token: string, on: boolean): string[] {
  const set = new Set(list);
  if (on) {
    set.add(token);
  } else {
    set.delete(token);
  }
  return Array.from(set);
}

function packageAllowlistState(selected: string[], keys: string[]): 'all' | 'none' | 'some' {
  let hit = 0;
  for (const k of keys) {
    if (selected.includes(k)) hit += 1;
  }
  if (hit === 0) return 'none';
  if (hit === keys.length) return 'all';
  return 'some';
}

function setAllowlistForPackageKeys(selected: string[], keys: string[], on: boolean): string[] {
  const set = new Set(selected);
  if (on) {
    for (const k of keys) set.add(k);
  } else {
    for (const k of keys) set.delete(k);
  }
  return Array.from(set).sort();
}

function AgentHarnessManagedModelPick() {
  const [llmConfigs, setLlmConfigs] = useState<LlmConfigItem[]>([]);
  const [loading, setLoading] = useState(true);

  const enabledConfigs = useMemo(() => llmConfigs.filter((c) => c.enabled), [llmConfigs]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const rows = await listLlmConfigs();
        if (cancelled) return;
        setLlmConfigs(Array.isArray(rows) ? rows : []);
      } catch {
        if (!cancelled) setLlmConfigs([]);
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
        模型管理（必选）
      </Typography.Text>
      <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 10 }}>
        须选择后台维护的 LLM 配置及其下一条已启用的模型条目，运行时由此解析凭证与模型名（与主站 Chat 一致）。
      </Typography.Paragraph>
      {loading ? (
        <Spin size="small" />
      ) : (
        <Field name="inputsValues.llmConfigId">
          {({ field: cfgField }) => (
            <Field name="inputsValues.llmModelEntryId">
              {({ field: entryField }) => {
                const cfgId = flowNum(cfgField.value);
                const entryId = flowNum(entryField.value);
                const cfg = enabledConfigs.find((c) => c.id === cfgId);
                const models = (cfg?.models || []).filter((m) => m.enabled);
                return (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                    <div>
                      <Typography.Text type="secondary" size="small" style={{ display: 'block', marginBottom: 4 }}>
                        LLM 配置
                      </Typography.Text>
                      <Select
                        style={{ width: '100%' }}
                        placeholder="请选择 LLM 配置"
                        value={cfgId > 0 ? cfgId : undefined}
                        onChange={(v) => {
                          const id = v == null || v === '' ? 0 : Number(v);
                          const nextId = Number.isFinite(id) ? id : 0;
                          cfgField.onChange({ type: 'constant', content: nextId } as any);
                          const nextCfg = enabledConfigs.find((c) => c.id === nextId);
                          const ents = (nextCfg?.models || []).filter((m) => m.enabled);
                          const cur = flowNum(entryField.value);
                          if (!ents.some((e) => e.id === cur)) {
                            entryField.onChange({ type: 'constant', content: 0 } as any);
                          }
                        }}
                      >
                        {enabledConfigs.map((c) => (
                          <Select.Option key={c.id} value={c.id}>
                            {c.name || `#${c.id}`}
                          </Select.Option>
                        ))}
                      </Select>
                    </div>
                    <div>
                      <Typography.Text type="secondary" size="small" style={{ display: 'block', marginBottom: 4 }}>
                        模型条目
                      </Typography.Text>
                      <Select
                        style={{ width: '100%' }}
                        placeholder={cfgId > 0 ? '请选择模型' : '请先选择 LLM 配置'}
                        disabled={cfgId <= 0 || models.length === 0}
                        value={entryId > 0 ? entryId : undefined}
                        onChange={(v) => {
                          const id = v == null || v === '' ? 0 : Number(v);
                          const nextId = Number.isFinite(id) ? id : 0;
                          entryField.onChange({ type: 'constant', content: nextId } as any);
                        }}
                      >
                        {models.map((m) => (
                          <Select.Option key={m.id} value={m.id}>
                            {m.modelName ? `${m.modelName}` : `#${m.id}`}
                          </Select.Option>
                        ))}
                      </Select>
                    </div>
                    {enabledConfigs.length === 0 ? (
                      <Typography.Text type="warning" size="small">
                        暂无已启用的 LLM 配置，请在「管理 → Agent → 模型」中添加。
                      </Typography.Text>
                    ) : cfgId > 0 && models.length === 0 ? (
                      <Typography.Text type="warning" size="small">
                        该配置下没有已启用的模型条目，请在模型管理中为该配置添加模型。
                      </Typography.Text>
                    ) : null}
                  </div>
                );
              }}
            </Field>
          )}
        </Field>
      )}
    </div>
  );
}

function AgentHarnessToolAllowlists() {
  const [skills, setSkills] = useState<SkillItem[]>([]);
  const [mcps, setMcps] = useState<MCPConfigItem[]>([]);
  const [loading, setLoading] = useState(true);

  const skillPackages = useMemo(() => groupSkillPackages(skills), [skills]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [sk, mcpRows] = await Promise.all([listSkills(), listMCPConfigs()]);
        if (cancelled) return;
        setSkills(Array.isArray(sk.items) ? sk.items : []);
        setMcps(Array.isArray(mcpRows) ? mcpRows : []);
      } catch {
        if (!cancelled) {
          setSkills([]);
          setMcps([]);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div style={{ marginTop: 8, marginBottom: 8 }}>
      <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
        Skill / MCP 白名单（勾选生效）
      </Typography.Text>
      <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 10 }}>
        Skill 按「套装」勾选（同一目录下多个技能文件会一并加入白名单）；不勾选任何项表示不限制。MCP 每项对应「该 server 下全部工具」（server:*）。
      </Typography.Paragraph>
      {loading ? (
        <Spin size="small" />
      ) : (
        <>
          <Field name="inputsValues.enableSkillTool">
            {({ field: en }) => {
              const skillOn = flowBool(en.value);
              return (
                <div style={{ marginBottom: 12 }}>
                  <Typography.Text type="secondary" size="small" style={{ display: 'block', marginBottom: 6 }}>
                    Skill（需开启「启用 run_skill」）
                  </Typography.Text>
                  {!skillOn ? (
                    <Typography.Text type="warning" size="small">
                      当前已关闭 Skill 工具，勾选不会生效。
                    </Typography.Text>
                  ) : skillPackages.length === 0 ? (
                    <Typography.Text type="tertiary" size="small">
                      暂无已注册 Skill，请在后端管理页上传。
                    </Typography.Text>
                  ) : (
                    <Field name="inputsValues.skillAllowlist">
                      {({ field }) => {
                        const selected = flowStringList(field.value);
                        return (
                          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                            {skillPackages.map((pkg) => {
                              const st = packageAllowlistState(selected, pkg.keys);
                              const checked = st === 'all';
                              const indeterminate = st === 'some';
                              return (
                                <Checkbox
                                  key={pkg.id}
                                  checked={checked}
                                  indeterminate={indeterminate}
                                  disabled={!skillOn}
                                  onChange={(e) => {
                                    const on = !!(e.target.checked ?? false);
                                    field.onChange({
                                      type: 'constant',
                                      content: setAllowlistForPackageKeys(selected, pkg.keys, on),
                                    } as any);
                                  }}
                                >
                                  {pkg.id}
                                </Checkbox>
                              );
                            })}
                          </div>
                        );
                      }}
                    </Field>
                  )}
                </div>
              );
            }}
          </Field>

          <Field name="inputsValues.enableMcpTool">
            {({ field: en }) => {
              const mcpOn = flowBool(en.value);
              return (
                <div>
                  <Typography.Text type="secondary" size="small" style={{ display: 'block', marginBottom: 6 }}>
                    MCP（需开启「启用 call_mcp_tool」）
                  </Typography.Text>
                  {!mcpOn ? (
                    <Typography.Text type="warning" size="small">
                      当前已关闭 MCP 工具，勾选不会生效。
                    </Typography.Text>
                  ) : mcps.length === 0 ? (
                    <Typography.Text type="tertiary" size="small">
                      暂无 MCP 配置，请在后端管理页添加。
                    </Typography.Text>
                  ) : (
                    <Field name="inputsValues.mcpAllowlist">
                      {({ field }) => {
                        const selected = flowStringList(field.value);
                        return (
                          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                            {mcps.map((m) => {
                              const srv = String(m.server || '').trim();
                              if (!srv) return null;
                              const token = `${srv}:*`;
                              const label = m.name ? `${m.name}（${srv}）` : srv;
                              return (
                                <Checkbox
                                  key={`${m.id}-${srv}`}
                                  checked={selected.includes(token)}
                                  disabled={!mcpOn}
                                  onChange={(e) => {
                                    const on = !!(e.target.checked ?? false);
                                    field.onChange({
                                      type: 'constant',
                                      content: toggleString(selected, token, on),
                                    } as any);
                                  }}
                                >
                                  {label}
                                  <Typography.Text type="tertiary" size="small" style={{ marginLeft: 6 }}>
                                    允许 {srv}:*
                                  </Typography.Text>
                                </Checkbox>
                              );
                            })}
                          </div>
                        );
                      }}
                    </Field>
                  )}
                </div>
              );
            }}
          </Field>
        </>
      )}
    </div>
  );
}

function AgentHarnessCollapsedPreview() {
  return (
    <div
      style={{
        margin: '0 10px 6px',
        padding: '8px',
        borderRadius: 6,
        background: '#f7f8fa',
        fontSize: 12,
        color: '#1d2129',
        lineHeight: 1.55,
      }}
    >
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 10px', fontSize: 11, color: '#86909c', marginBottom: 6 }}>
        <Field name="inputsValues.llmConfigId">
          {({ field: f }) => (
            <span>
              LLM配置 #{flowNum(f.value) || '—'}
            </span>
          )}
        </Field>
        <Field name="inputsValues.llmModelEntryId">
          {({ field: f }) => (
            <span>
              模型条目 #{flowNum(f.value) || '—'}
            </span>
          )}
        </Field>
      </div>
      <Field name="inputsValues.model">
        {({ field }) => (
          <div style={{ marginBottom: 6 }}>
            <span style={{ color: '#86909c' }}>模型名模板（可选）</span>{' '}
            <strong style={{ color: '#1d2129' }}>
              {truncOneLine(flowStr(field.value), 40) || '（空）'}
            </strong>
          </div>
        )}
      </Field>
      <Field name="inputsValues.userPrompt">
        {({ field }) => (
          <div style={{ color: '#4e5969', wordBreak: 'break-word', marginBottom: 6 }}>
            <span style={{ color: '#86909c' }}>用户提示 </span>
            {truncOneLine(flowStr(field.value), 120)}
          </div>
        )}
      </Field>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 10px', fontSize: 11, color: '#86909c' }}>
        <Field name="inputsValues.enableSkillTool">
          {({ field }) => <span>Skill {flowBool(field.value) ? '开' : '关'}</span>}
        </Field>
        <Field name="inputsValues.enableMcpTool">
          {({ field }) => <span>MCP {flowBool(field.value) ? '开' : '关'}</span>}
        </Field>
        <Field name="inputsValues.enableWorkspaceTools">
          {({ field }) => <span>WS {flowBool(field.value) ? '开' : '关'}</span>}
        </Field>
        <Field name="inputsValues.skillAllowlist">
          {({ field }) => (
            <span>Skill白 {flowStringList(field.value).length || '不限'}</span>
          )}
        </Field>
        <Field name="inputsValues.mcpAllowlist">
          {({ field }) => (
            <span>MCP白 {flowStringList(field.value).length || '不限'}</span>
          )}
        </Field>
        <Field name="inputsValues.maxIterations">
          {({ field }) => <span>迭代 {flowStr(field.value) || '0'}</span>}
        </Field>
      </div>
    </div>
  );
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent collapsedPreview={<AgentHarnessCollapsedPreview />}>
      <Typography.Paragraph type="tertiary" size="small" style={{ margin: '0 10px 10px' }}>
        上方须选择模型管理中的 LLM 配置与模型条目（运行时解析密钥与模型名）；下方可编辑提示词与各工具选项。generate_uuid 由服务端固定启用。
      </Typography.Paragraph>
      <AgentHarnessManagedModelPick />
      <FormInputs {...agentFormInputsProps} />
      <Divider />
      <AgentHarnessToolAllowlists />
      <Divider />
      <DisplayOutputs displayFromScope />
    </FormContent>
  </>
);

export const agentHarnessFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
