/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useEffect, useMemo, useState } from 'react';

import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';
import { Checkbox, Divider, Select, Spin, Typography } from '@douyinfe/semi-ui';

import { defaultFormMeta } from '../default-form-meta';
import { FlowNodeJSON } from '../../typings';
import { listWorkspaces, type WorkspaceItem } from '../../services/api-workspaces';
import {
  listLlmConfigs,
  listMCPConfigs,
  type LlmConfigItem,
  type MCPConfigItem,
} from '../../services/api-agent';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';
import type { FormInputsProps } from '../../form-components';

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

function buildWorkspacePromptPreview(workspace: WorkspaceItem): string {
  const workspaceId = String(workspace.id || '').trim();
  const workspaceName = String(workspace.name || '').trim() || workspaceId;
  const rootDir = String(workspace.rootDir || '').trim() || `/app/code_workspace/${workspaceId}`;
  const repos = Array.isArray(workspace.repositories)
    ? workspace.repositories
        .map((r) => {
          const url = String(r?.url || '').trim();
          if (!url) return '';
          const dir = String(r?.dir || '').trim();
          return dir ? `- ${url}（目录: ${dir}）` : `- ${url}`;
        })
        .filter(Boolean)
    : [];
  const reposText = repos.length > 0 ? repos.join('\n') : '（未配置仓库）';
  return [
    '【工作区使用模式（自动注入）】',
    `你当前绑定的工作区为「${workspaceName}」（id=${workspaceId}）。`,
    '请遵循以下强制约束：',
    `1. 仅允许在该工作区目录及其子目录内进行文件读写与命令执行：${rootDir}`,
    '2. 仅允许在以下仓库范围内完成任务：',
    reposText,
    '3. 严禁访问、读取、修改工作区外的任何路径或未列出的仓库。',
  ].join('\n');
}

/** 不渲染白名单字段（由下方勾选区维护）；LLM 配置/模型条目由专用下拉维护。 */
const AGENT_FORM_KEYS_NO_ALLOWLIST: readonly string[] = [
  'userPrompt',
  'systemPrompt',
  'workspaceId',
  'enableSkillTool',
  'enableMcpTool',
  'enableSubAgentTool',
  'maxIterations',
  'maxToolCalls',
  'toolTimeoutSecs',
];

const agentFormInputsProps: FormInputsProps = {
  propertyFilter: (k) =>
    ![
      'model',
      'skillAllowlist',
      'mcpAllowlist',
      'llmConfigId',
      'llmModelEntryId',
      'workspaceId',
      'enableWorkspaceTools',
    ].includes(k),
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
        须选择后台维护的 LLM 配置及其下一条已启用的模型条目，运行时由此解析凭证与模型名（与主站 Chat
        一致）。
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
                      <Typography.Text
                        type="secondary"
                        size="small"
                        style={{ display: 'block', marginBottom: 4 }}
                      >
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
                      <Typography.Text
                        type="secondary"
                        size="small"
                        style={{ display: 'block', marginBottom: 4 }}
                      >
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

function AgentHarnessWorkspacePick() {
  const [workspaces, setWorkspaces] = useState<WorkspaceItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const rows = await listWorkspaces();
        if (!cancelled) {
          setWorkspaces(Array.isArray(rows) ? rows : []);
        }
      } catch {
        if (!cancelled) setWorkspaces([]);
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
        工作区（可选）
      </Typography.Text>
      <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 10 }}>
        选择后会在运行时自动注入“工作区使用模式”到系统提示词；Workspace 工具固定开启且不可关闭。
      </Typography.Paragraph>
      {loading ? (
        <Spin size="small" />
      ) : (
        <Field name="inputsValues.workspaceId">
          {({ field }) => {
            const current = flowStr(field.value);
            const selected = workspaces.find((w) => String(w.id || '') === current);
            const preview = selected ? buildWorkspacePromptPreview(selected) : '';
            return (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <Select
                  style={{ width: '100%' }}
                  placeholder="不绑定工作区"
                  value={current || undefined}
                  onChange={(v) => {
                    field.onChange({ type: 'constant', content: String(v || '') } as any);
                  }}
                >
                  <Select.Option value="">不绑定工作区</Select.Option>
                  {workspaces.map((w) => (
                    <Select.Option key={w.id} value={w.id}>
                      {w.name}（{w.id}）
                    </Select.Option>
                  ))}
                </Select>
                {preview ? (
                  <div
                    style={{
                      border: '1px solid var(--semi-color-border)',
                      borderRadius: 8,
                      padding: 10,
                      background: 'var(--semi-color-fill-0)',
                    }}
                  >
                    <Typography.Text
                      strong
                      size="small"
                      style={{ display: 'block', marginBottom: 6 }}
                    >
                      自动注入提示词预览
                    </Typography.Text>
                    <Typography.Paragraph
                      style={{
                        margin: 0,
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontSize: 12,
                        lineHeight: '18px',
                      }}
                    >
                      {preview}
                    </Typography.Paragraph>
                  </div>
                ) : null}
              </div>
            );
          }}
        </Field>
      )}
    </div>
  );
}

function AgentHarnessToolAllowlists() {
  const [mcps, setMcps] = useState<MCPConfigItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const mcpRows = await listMCPConfigs();
        if (cancelled) return;
        setMcps(Array.isArray(mcpRows) ? mcpRows : []);
      } catch {
        if (!cancelled) {
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
        MCP 白名单（勾选生效）
      </Typography.Text>
      <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 10 }}>
        Skill 默认可使用系统、Agent、工作流三个目录下的全部技能，无需在节点里勾选。MCP
        每项对应「该 server 下全部工具」（server:*）。
      </Typography.Paragraph>
      {loading ? (
        <Spin size="small" />
      ) : (
        <>
          <Field name="inputsValues.enableMcpTool">
            {({ field: en }) => {
              const mcpOn = flowBool(en.value);
              return (
                <div>
                  <Typography.Text
                    type="secondary"
                    size="small"
                    style={{ display: 'block', marginBottom: 6 }}
                  >
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
                                  <Typography.Text
                                    type="tertiary"
                                    size="small"
                                    style={{ marginLeft: 6 }}
                                  >
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

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent>
      <Typography.Paragraph type="tertiary" size="small" style={{ margin: '0 10px 10px' }}>
        上方须选择模型管理中的 LLM
        配置与模型条目（运行时解析密钥与模型名）；可选绑定工作区并自动注入约束提示。generate_uuid 与
        workspace 工具由服务端固定启用；可开启 run_sub_agent 让 Agent 自动分派子任务（支持批量并发
        sub_tasks_json，未指定并发时自动估算）。
      </Typography.Paragraph>
      <AgentHarnessManagedModelPick />
      <AgentHarnessWorkspacePick />
      <div
        style={{
          margin: '0 10px 12px',
          padding: 10,
          borderRadius: 8,
          border: '1px solid var(--semi-color-border)',
          background: 'var(--semi-color-fill-0)',
        }}
      >
        <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
          多模态附件输入约定
        </Typography.Text>
        <Typography.Paragraph type="tertiary" size="small" style={{ margin: 0 }}>
          `ai/agentHarness` 会从进入节点的 `msg.data.attachments` 或 `metadata.attachments`
          中读取附件。统一字段为 `filename / mimeType / text /
          contentBase64`；图片、视频、音频优先走原生多模态，普通文件当前会按统一结构接入，并在模型适配层不支持时自动降级为文本附件块。
        </Typography.Paragraph>
        <Typography.Paragraph
          style={{
            margin: '8px 0 0',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            fontSize: 12,
            lineHeight: '18px',
          }}
        >
          {`msg.data:
{
  "query": "请分析附件",
  "attachments": [
    { "filename": "screen.png", "mimeType": "image/png", "contentBase64": "..." }
  ]
}

metadata.attachments:
[
  { "filename": "spec.pdf", "mimeType": "application/pdf", "contentBase64": "..." }
]`}
        </Typography.Paragraph>
      </div>
      <FormInputs {...agentFormInputsProps} />
      <Divider />
      <AgentHarnessToolAllowlists />
      <Divider />
      <OutputsPeek />
    </FormContent>
  </>
);

export const agentHarnessFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
