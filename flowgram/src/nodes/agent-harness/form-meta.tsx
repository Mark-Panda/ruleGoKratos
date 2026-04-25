/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useEffect, useMemo, useState } from 'react';

import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';
import { Collapse, Divider, Select, Spin, Typography } from '@douyinfe/semi-ui';

import { defaultFormMeta } from '../default-form-meta';
import { FlowNodeJSON } from '../../typings';
import { listWorkspaces, type WorkspaceItem } from '../../services/api-workspaces';
import { listLlmConfigs, type LlmConfigItem } from '../../services/api-agent';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';
import type { FormInputsProps } from '../../form-components';

function flowStr(v: unknown): string {
  if (v == null || typeof v !== 'object' || !('content' in (v as object))) return '';
  const c = (v as { content?: unknown }).content;
  return c == null ? '' : String(c);
}

function flowNum(v: unknown): number {
  if (v == null || typeof v !== 'object' || !('content' in (v as object))) return 0;
  const c = (v as { content?: unknown }).content;
  if (typeof c === 'number' && Number.isFinite(c)) return c;
  const n = Number(c);
  return Number.isFinite(n) ? n : 0;
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
      'enableMcpTool',
      'llmConfigId',
      'llmModelEntryId',
      'workspaceId',
      'enableWorkspaceTools',
    ].includes(k),
  propertyKeyOrder: AGENT_FORM_KEYS_NO_ALLOWLIST,
};

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

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent>
      <AgentHarnessManagedModelPick />
      <AgentHarnessWorkspacePick />
      <FormInputs {...agentFormInputsProps} />
      <div style={{ marginTop: 6, marginBottom: 4, width: '100%', boxSizing: 'border-box' }}>
        <Collapse defaultActiveKey={[]} style={{ width: '100%' }} keepDOM>
          <Collapse.Panel
            header={<Typography.Text strong>多模态附件输入约定</Typography.Text>}
            itemKey="attachments"
          >
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 10,
                padding: '0 0 2px',
                maxWidth: '100%',
                boxSizing: 'border-box',
              }}
            >
              <Typography.Paragraph type="tertiary" size="small" style={{ margin: 0 }}>
                <code>ai/agentHarness</code> 会读取 <code>msg.data.attachments</code> 或{' '}
                <code>metadata.attachments</code>。字段为 <code>filename</code> /{' '}
                <code>mimeType</code> / <code>text</code> / <code>contentBase64</code>
                ；图/视频/音优先多模态，其他文件在适配层不支持时降级为文本附件块。
              </Typography.Paragraph>
              <div style={{ minWidth: 0 }}>
                <Typography.Text
                  type="tertiary"
                  size="small"
                  style={{ display: 'block', marginBottom: 6, fontSize: 12 }}
                >
                  示例
                </Typography.Text>
                <pre
                  style={{
                    margin: 0,
                    maxWidth: '100%',
                    boxSizing: 'border-box',
                    padding: 10,
                    borderRadius: 6,
                    border: '1px solid var(--semi-color-border)',
                    background: 'var(--semi-color-bg-1)',
                    fontSize: 11,
                    lineHeight: 1.5,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    fontFamily:
                      'var(--semi-font-mono, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace)',
                  }}
                >{`msg.data:
{
  "query": "请分析附件",
  "attachments": [
    { "filename": "screen.png", "mimeType": "image/png", "contentBase64": "..." }
  ]
}

metadata.attachments:
[
  { "filename": "spec.pdf", "mimeType": "application/pdf", "contentBase64": "..." }
]`}</pre>
              </div>
            </div>
          </Collapse.Panel>
        </Collapse>
      </div>
      <Divider />
      <OutputsPeek />
    </FormContent>
  </>
);

export const agentHarnessFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
