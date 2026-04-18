/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useEffect, useMemo, useState } from 'react';

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

/** 不渲染白名单字段（由下方勾选区维护） */
const AGENT_FORM_KEYS_NO_ALLOWLIST: readonly string[] = [
  'llmConfigId',
  'llmModelEntryId',
  'model',
  'userPrompt',
  'systemPrompt',
  'enableSkillTool',
  'enableMcpTool',
  'enableUUIDTool',
  'enableWorkspaceTools',
  'maxIterations',
  'maxToolCalls',
  'toolTimeoutSecs',
];

const agentFormInputsProps: FormInputsProps = {
  propertyFilter: (k) =>
    !['skillAllowlist', 'mcpAllowlist', 'llmConfigId', 'llmModelEntryId', 'model'].includes(k),
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

function AgentHarnessManagedLLMPanel() {
  const [configs, setConfigs] = useState<LlmConfigItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const rows = await listLlmConfigs();
        if (!cancelled) setConfigs(Array.isArray(rows) ? rows : []);
      } catch {
        if (!cancelled) setConfigs([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const enabledConfigs = useMemo(
    () => configs.filter((c) => c.enabled && Array.isArray(c.models)),
    [configs]
  );

  return (
    <div style={{ marginBottom: 12 }}>
      <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
        模型（来自「Agent 管理 → 模型管理」）
      </Typography.Text>
      <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 10 }}>
        先选启用中的配置，再选该配置下的模型；运行时从服务端读取 Base URL 与 API Key，不再使用环境变量。
      </Typography.Paragraph>
      {loading ? (
        <Spin size="small" />
      ) : enabledConfigs.length === 0 ? (
        <Typography.Text type="warning" size="small">
          暂无启用的 LLM 配置，请先在管理页创建并启用。
        </Typography.Text>
      ) : (
        <Field name="inputsValues.llmConfigId">
          {({ field: cfgField }) => (
            <Field name="inputsValues.llmModelEntryId">
              {({ field: entField }) => (
                <Field name="inputsValues.model">
                  {({ field: modelField }) => {
                    const cid = flowNum(cfgField.value);
                    const selCfg = enabledConfigs.find((c) => c.id === cid);
                    const models = (selCfg?.models || []).filter((m) => m.enabled);
                    return (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                        <Select
                          placeholder="选择 LLM 配置"
                          style={{ width: '100%' }}
                          value={cid ? String(cid) : ''}
                          onChange={(v) => {
                            const next = Number(v);
                            cfgField.onChange({ type: 'constant', content: next } as any);
                            entField.onChange({ type: 'constant', content: 0 } as any);
                            modelField.onChange({ type: 'constant', content: '' } as any);
                          }}
                        >
                          {enabledConfigs.map((c) => (
                            <Select.Option key={c.id} value={String(c.id)}>
                              {c.name}
                            </Select.Option>
                          ))}
                        </Select>
                        <Select
                          placeholder="选择模型"
                          style={{ width: '100%' }}
                          value={flowNum(entField.value) ? String(flowNum(entField.value)) : ''}
                          disabled={!cid}
                          onChange={(v) => {
                            const eid = Number(v);
                            entField.onChange({ type: 'constant', content: eid } as any);
                            const row = models.find((m) => m.id === eid);
                            modelField.onChange({
                              type: 'constant',
                              content: row?.modelName ?? '',
                            } as any);
                          }}
                        >
                          {models.map((m) => (
                            <Select.Option key={m.id} value={String(m.id)}>
                              {m.modelName}
                            </Select.Option>
                          ))}
                        </Select>
                      </div>
                    );
                  }}
                </Field>
              )}
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
      <Field name="inputsValues.llmConfigId">
        {({ field: cf }) => (
          <Field name="inputsValues.llmModelEntryId">
            {({ field: ef }) => (
              <Field name="inputsValues.model">
                {({ field: mf }) => (
                  <div style={{ marginBottom: 6 }}>
                    <span style={{ color: '#86909c' }}>模型 </span>
                    <strong style={{ color: '#1d2129' }}>
                      {flowNum(cf.value) && flowNum(ef.value)
                        ? `#${flowNum(cf.value)} / ${truncOneLine(flowStr(mf.value), 36)}`
                        : '未选择'}
                    </strong>
                  </div>
                )}
              </Field>
            )}
          </Field>
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
        <Field name="inputsValues.enableUUIDTool">
          {({ field }) => <span>UUID {flowBool(field.value) ? '开' : '关'}</span>}
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
      <AgentHarnessManagedLLMPanel />
      <Divider />
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
