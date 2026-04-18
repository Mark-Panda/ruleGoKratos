import React, { useEffect, useMemo, useRef, useState } from 'react';

import { Button, Input, Modal, Select, Spin, Table, TextArea, Toast, Typography } from '@douyinfe/semi-ui';

import {
  MCPConfigItem,
  MCPConfigPayload,
  LlmConfigItem,
  LlmConfigPayload,
  LlmModelEntryItem,
  LlmModelEntryPayload,
  createMCPConfig,
  createLlmConfig,
  createLlmModelEntry,
  deleteMCPConfig,
  deleteLlmConfig,
  deleteLlmModelEntry,
  listMCPConfigs,
  listLlmConfigs,
  listSkills,
  type SkillItem,
  updateMCPConfig,
  updateLlmConfig,
  updateLlmModelEntry,
  uploadSkill,
} from '../../services/api-agent';
import { groupSkillPackages } from '../../utils/skill-packages';

const defaultMCPForm: MCPConfigPayload = {
  name: '',
  server: '',
  endpoint: '',
  headers: {},
  enabled: true,
  description: '',
};

const defaultLlmConfigForm: LlmConfigPayload = {
  name: '',
  provider: 'openai',
  baseUrl: '',
  apiKey: '',
  enabled: true,
  description: '',
  models: [{ modelName: '', description: '', enabled: true }],
};

const defaultEntryForm: LlmModelEntryPayload = {
  modelName: '',
  description: '',
  enabled: true,
};

export const AgentSection: React.FC<{ view?: 'skills' | 'mcps' | 'models' }> = ({
  view = 'skills',
}) => {
  const [skillRoot, setSkillRoot] = useState('skills');
  const [skills, setSkills] = useState<SkillItem[]>([]);
  const [skillLoading, setSkillLoading] = useState(false);
  const [skillKeyword, setSkillKeyword] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [mcpList, setMcpList] = useState<MCPConfigItem[]>([]);
  const [mcpLoading, setMcpLoading] = useState(false);
  const [mcpModalVisible, setMcpModalVisible] = useState(false);
  const [mcpSubmitting, setMcpSubmitting] = useState(false);
  const [mcpEditing, setMcpEditing] = useState<MCPConfigItem | null>(null);
  const [mcpForm, setMcpForm] = useState<MCPConfigPayload>(defaultMCPForm);
  const [headersText, setHeadersText] = useState('{}');

  const [llmConfigList, setLlmConfigList] = useState<LlmConfigItem[]>([]);
  const [llmLoading, setLlmLoading] = useState(false);
  const [llmConfigModalVisible, setLlmConfigModalVisible] = useState(false);
  const [llmConfigSubmitting, setLlmConfigSubmitting] = useState(false);
  const [llmConfigEditing, setLlmConfigEditing] = useState<LlmConfigItem | null>(null);
  const [llmConfigForm, setLlmConfigForm] = useState<LlmConfigPayload>(defaultLlmConfigForm);
  const [entryModalVisible, setEntryModalVisible] = useState(false);
  const [entrySubmitting, setEntrySubmitting] = useState(false);
  const [entryEditing, setEntryEditing] = useState<LlmModelEntryItem | null>(null);
  const [entryConfigId, setEntryConfigId] = useState<number | null>(null);
  const [entryForm, setEntryForm] = useState<LlmModelEntryPayload>(defaultEntryForm);

  const fetchSkills = async () => {
    setSkillLoading(true);
    try {
      const data = await listSkills();
      setSkillRoot(data.root || 'skills');
      setSkills(Array.isArray(data.items) ? data.items : []);
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setSkillLoading(false);
    }
  };

  const fetchMCPs = async () => {
    setMcpLoading(true);
    try {
      const data = await listMCPConfigs();
      setMcpList(data);
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setMcpLoading(false);
    }
  };

  const fetchLlmConfigs = async () => {
    setLlmLoading(true);
    try {
      const data = await listLlmConfigs();
      setLlmConfigList(Array.isArray(data) ? data : []);
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setLlmLoading(false);
    }
  };

  useEffect(() => {
    if (view === 'skills') fetchSkills();
    if (view === 'mcps') fetchMCPs();
    if (view === 'models') fetchLlmConfigs();
  }, [view]);

  const skillPackages = useMemo(() => groupSkillPackages(skills), [skills]);

  const filteredSkillPackages = useMemo(() => {
    const kw = skillKeyword.trim().toLowerCase();
    if (!kw) return skillPackages;
    return skillPackages.filter((pkg) => {
      if (pkg.id.toLowerCase().includes(kw)) return true;
      return pkg.files.some((f) => {
        const p = String(f.path || '').toLowerCase();
        const n = String(f.name || '').toLowerCase();
        return p.includes(kw) || n.includes(kw);
      });
    });
  }, [skillPackages, skillKeyword]);

  const openCreateMCP = () => {
    setMcpEditing(null);
    setMcpForm(defaultMCPForm);
    setHeadersText('{}');
    setMcpModalVisible(true);
  };

  const openEditMCP = (item: MCPConfigItem) => {
    const headers = item.headers || {};
    setMcpEditing(item);
    setMcpForm({
      name: item.name,
      server: item.server,
      endpoint: item.endpoint,
      headers,
      enabled: !!item.enabled,
      description: item.description || '',
    });
    setHeadersText(JSON.stringify(headers, null, 2));
    setMcpModalVisible(true);
  };

  const submitMCP = async () => {
    if (!mcpForm.name.trim() || !mcpForm.server.trim() || !mcpForm.endpoint.trim()) {
      Toast.warning({ content: '请填写 name/server/endpoint' });
      return;
    }
    let parsedHeaders: Record<string, any> = {};
    try {
      parsedHeaders = headersText.trim() ? JSON.parse(headersText) : {};
    } catch {
      Toast.error({ content: 'headers 必须是合法 JSON' });
      return;
    }
    const payload: MCPConfigPayload = { ...mcpForm, headers: parsedHeaders };
    setMcpSubmitting(true);
    try {
      if (mcpEditing) {
        await updateMCPConfig(mcpEditing.id, payload);
        Toast.success({ content: '更新成功' });
      } else {
        await createMCPConfig(payload);
        Toast.success({ content: '创建成功' });
      }
      setMcpModalVisible(false);
      await fetchMCPs();
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setMcpSubmitting(false);
    }
  };

  const openCreateLlmConfig = () => {
    setLlmConfigEditing(null);
    setLlmConfigForm(defaultLlmConfigForm);
    setLlmConfigModalVisible(true);
  };

  const openEditLlmConfig = (item: LlmConfigItem) => {
    setLlmConfigEditing(item);
    setLlmConfigForm({
      name: item.name,
      provider: item.provider || 'openai',
      baseUrl: item.baseUrl || '',
      apiKey: '',
      enabled: !!item.enabled,
      description: item.description || '',
    });
    setLlmConfigModalVisible(true);
  };

  const addLlmModelDraftRow = () => {
    setLlmConfigForm((prev) => ({
      ...prev,
      models: [...(prev.models || []), { modelName: '', description: '', enabled: true }],
    }));
  };

  const removeLlmModelDraftRow = (index: number) => {
    setLlmConfigForm((prev) => {
      const models = [...(prev.models || [])];
      models.splice(index, 1);
      return { ...prev, models };
    });
  };

  const updateLlmModelDraftRow = (index: number, patch: Partial<LlmModelEntryPayload>) => {
    setLlmConfigForm((prev) => {
      const models = [...(prev.models || [])];
      const cur = models[index];
      if (!cur) return prev;
      models[index] = { ...cur, ...patch };
      return { ...prev, models };
    });
  };

  const submitLlmConfig = async () => {
    if (!llmConfigForm.name.trim()) {
      Toast.warning({ content: '请填写配置名称' });
      return;
    }
    const base: LlmConfigPayload = {
      name: llmConfigForm.name.trim(),
      provider: (llmConfigForm.provider || 'openai').trim(),
      baseUrl: llmConfigForm.baseUrl.trim(),
      description: llmConfigForm.description.trim(),
      apiKey: llmConfigForm.apiKey.trim(),
      enabled: llmConfigForm.enabled,
    };
    setLlmConfigSubmitting(true);
    try {
      if (llmConfigEditing) {
        await updateLlmConfig(llmConfigEditing.id, base);
        Toast.success({ content: '更新成功' });
      } else {
        const models = (llmConfigForm.models || [])
          .map((m) => ({
            modelName: m.modelName.trim(),
            description: m.description.trim(),
            enabled: m.enabled,
          }))
          .filter((m) => m.modelName !== '');
        const names = models.map((m) => m.modelName);
        if (names.length !== new Set(names).size) {
          Toast.warning({ content: '模型 ID 列表中存在重复，请检查' });
          return;
        }
        await createLlmConfig({ ...base, models });
        Toast.success({ content: '创建成功' });
      }
      setLlmConfigModalVisible(false);
      await fetchLlmConfigs();
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setLlmConfigSubmitting(false);
    }
  };

  const openCreateEntry = (configId: number) => {
    setEntryEditing(null);
    setEntryConfigId(configId);
    setEntryForm(defaultEntryForm);
    setEntryModalVisible(true);
  };

  const openEditEntry = (_configId: number, row: LlmModelEntryItem) => {
    setEntryEditing(row);
    setEntryConfigId(_configId);
    setEntryForm({
      modelName: row.modelName,
      description: row.description || '',
      enabled: !!row.enabled,
    });
    setEntryModalVisible(true);
  };

  const submitEntry = async () => {
    if (!entryForm.modelName.trim()) {
      Toast.warning({ content: '请填写模型 ID（modelName）' });
      return;
    }
    const payload: LlmModelEntryPayload = {
      modelName: entryForm.modelName.trim(),
      description: entryForm.description.trim(),
      enabled: entryForm.enabled,
    };
    setEntrySubmitting(true);
    try {
      if (entryEditing) {
        await updateLlmModelEntry(entryEditing.id, payload);
        Toast.success({ content: '更新成功' });
      } else if (entryConfigId != null) {
        await createLlmModelEntry(entryConfigId, payload);
        Toast.success({ content: '已添加模型' });
      }
      setEntryModalVisible(false);
      await fetchLlmConfigs();
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setEntrySubmitting(false);
    }
  };

  return (
    <div style={{ padding: 24, display: 'flex', flexDirection: 'column', gap: 16 }}>
      {view === 'skills' && (
        <>
          <div
            style={{
              background: '#fff',
              borderRadius: 12,
              border: '1px solid rgba(6,7,9,0.06)',
              padding: 12,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, flex: 1 }}>
              <Input
                value={skillKeyword}
                onChange={setSkillKeyword}
                placeholder="搜索套装 id、路径或文件名"
                showClear
                style={{ maxWidth: 360 }}
              />
              <Typography.Text type="tertiary">目录：{skillRoot}</Typography.Text>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <input
                ref={fileInputRef}
                type="file"
                style={{ display: 'none' }}
                onChange={async (e) => {
                  const file = e.target.files?.[0];
                  if (!file) return;
                  try {
                    await uploadSkill(file, file.name);
                    Toast.success({ content: '上传成功' });
                    await fetchSkills();
                  } catch (err) {
                    Toast.error({ content: String((err as Error)?.message ?? err) });
                  } finally {
                    e.target.value = '';
                  }
                }}
              />
              <Button onClick={() => fileInputRef.current?.click()} theme="solid" type="primary">
                上传 SKILL
              </Button>
              <Button onClick={() => fetchSkills()}>刷新</Button>
            </div>
          </div>
          <Spin spinning={skillLoading}>
            <Table
              dataSource={filteredSkillPackages}
              rowKey="id"
              pagination={{ pageSize: 10 }}
              columns={[
                { title: '套装', dataIndex: 'id' },
                {
                  title: '文件数',
                  width: 100,
                  render: (_, r) => r.files.length,
                },
              ]}
            />
          </Spin>
        </>
      )}

      {view === 'mcps' && (
        <>
          <div
            style={{
              background: '#fff',
              borderRadius: 12,
              border: '1px solid rgba(6,7,9,0.06)',
              padding: 12,
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 8,
            }}
          >
            <Button onClick={() => fetchMCPs()}>刷新</Button>
            <Button theme="solid" type="primary" onClick={openCreateMCP}>
              新增 MCP
            </Button>
          </div>
          <Spin spinning={mcpLoading}>
            <Table
              dataSource={mcpList}
              rowKey="id"
              pagination={{ pageSize: 10 }}
              columns={[
                { title: '名称', dataIndex: 'name', width: 180 },
                { title: 'Server', dataIndex: 'server', width: 180 },
                { title: 'Endpoint', dataIndex: 'endpoint' },
                {
                  title: '状态',
                  width: 100,
                  render: (_, r) => (r.enabled ? '启用' : '禁用'),
                },
                {
                  title: '操作',
                  width: 180,
                  render: (_, r) => (
                    <div style={{ display: 'flex', gap: 8 }}>
                      <Button size="small" onClick={() => openEditMCP(r)}>
                        编辑
                      </Button>
                      <Button
                        size="small"
                        type="danger"
                        onClick={async () => {
                          try {
                            await deleteMCPConfig(r.id);
                            Toast.success({ content: '删除成功' });
                            fetchMCPs();
                          } catch (e) {
                            Toast.error({ content: String((e as Error)?.message ?? e) });
                          }
                        }}
                      >
                        删除
                      </Button>
                    </div>
                  ),
                },
              ]}
            />
          </Spin>
        </>
      )}

      {view === 'models' && (
        <>
          <div
            style={{
              background: '#fff',
              borderRadius: 12,
              border: '1px solid rgba(6,7,9,0.06)',
              padding: 12,
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 8,
            }}
          >
            <Button onClick={() => fetchLlmConfigs()}>刷新</Button>
            <Button theme="solid" type="primary" onClick={openCreateLlmConfig}>
              新增配置
            </Button>
          </div>
          <Spin spinning={llmLoading}>
            <Table
              dataSource={llmConfigList}
              rowKey="id"
              pagination={{ pageSize: 8 }}
              expandRowByClick
              expandedRowRender={(record) => (
                <div
                  style={{
                    padding: '8px 0 16px 24px',
                    background: 'rgba(6,7,9,0.02)',
                    borderRadius: 8,
                  }}
                >
                  <div
                    style={{
                      marginBottom: 8,
                      display: 'flex',
                      justifyContent: 'flex-end',
                    }}
                  >
                    <Button size="small" theme="solid" onClick={() => openCreateEntry(record.id)}>
                      添加模型
                    </Button>
                  </div>
                  <Table
                    size="small"
                    dataSource={record.models || []}
                    rowKey="id"
                    pagination={false}
                    columns={[
                      { title: '模型 ID', dataIndex: 'modelName', width: 200 },
                      {
                        title: '说明',
                        dataIndex: 'description',
                        ellipsis: true,
                      },
                      {
                        title: '状态',
                        width: 72,
                        render: (_, r) => (r.enabled ? '启用' : '禁用'),
                      },
                      {
                        title: '操作',
                        width: 160,
                        render: (_, r) => (
                          <div style={{ display: 'flex', gap: 8 }}>
                            <Button size="small" onClick={() => openEditEntry(record.id, r)}>
                              编辑
                            </Button>
                            <Button
                              size="small"
                              type="danger"
                              onClick={async () => {
                                try {
                                  await deleteLlmModelEntry(r.id);
                                  Toast.success({ content: '已删除' });
                                  fetchLlmConfigs();
                                } catch (e) {
                                  Toast.error({
                                    content: String((e as Error)?.message ?? e),
                                  });
                                }
                              }}
                            >
                              删除
                            </Button>
                          </div>
                        ),
                      },
                    ]}
                  />
                </div>
              )}
              columns={[
                { title: '配置名称', dataIndex: 'name', width: 160 },
                { title: '厂商', dataIndex: 'provider', width: 100 },
                {
                  title: '模型数',
                  width: 80,
                  render: (_, r) => r.models?.length ?? 0,
                },
                {
                  title: 'Base URL',
                  dataIndex: 'baseUrl',
                  ellipsis: true,
                  render: (v: string) => v || '—',
                },
                {
                  title: 'API Key',
                  width: 90,
                  render: (_, r) => (r.apiKey ? '已配置' : '未配置'),
                },
                {
                  title: '状态',
                  width: 72,
                  render: (_, r) => (r.enabled ? '启用' : '禁用'),
                },
                {
                  title: '操作',
                  width: 180,
                  render: (_, r) => (
                    <div style={{ display: 'flex', gap: 8 }}>
                      <Button size="small" onClick={() => openEditLlmConfig(r)}>
                        编辑
                      </Button>
                      <Button
                        size="small"
                        type="danger"
                        onClick={async () => {
                          try {
                            await deleteLlmConfig(r.id);
                            Toast.success({ content: '删除成功' });
                            fetchLlmConfigs();
                          } catch (e) {
                            Toast.error({ content: String((e as Error)?.message ?? e) });
                          }
                        }}
                      >
                        删除
                      </Button>
                    </div>
                  ),
                },
              ]}
            />
          </Spin>
        </>
      )}

      <Modal
        title={llmConfigEditing ? '编辑 LLM 配置' : '新增 LLM 配置'}
        visible={llmConfigModalVisible}
        onCancel={() => setLlmConfigModalVisible(false)}
        onOk={submitLlmConfig}
        confirmLoading={llmConfigSubmitting}
        style={{ width: llmConfigEditing ? 640 : 720 }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <Input
            value={llmConfigForm.name}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, name: String(v) })}
            placeholder="配置名称（唯一），其下可添加多条模型 ID"
          />
          <Input
            value={llmConfigForm.provider}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, provider: String(v) })}
            placeholder="厂商，如 openai / anthropic / azure / ollama"
          />
          <Input
            value={llmConfigForm.baseUrl}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, baseUrl: String(v) })}
            placeholder="API Base URL（可选）"
          />
          <Input
            mode="password"
            value={llmConfigForm.apiKey}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, apiKey: String(v) })}
            placeholder={
              llmConfigEditing ? 'API Key（留空则不修改已保存的密钥）' : 'API Key（可选）'
            }
          />
          {llmConfigEditing && (
            <Typography.Text type="tertiary" size="small">
              编辑时留空 API Key 将保留原值。模型请在下方表格展开行中增删改。
            </Typography.Text>
          )}
          <Select
            value={llmConfigForm.enabled ? '1' : '0'}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, enabled: v === '1' })}
          >
            <Select.Option value="1">启用</Select.Option>
            <Select.Option value="0">禁用</Select.Option>
          </Select>
          {!llmConfigEditing && (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 10,
                padding: 12,
                background: 'rgba(6,7,9,0.03)',
                borderRadius: 8,
                border: '1px solid rgba(6,7,9,0.06)',
              }}
            >
              <Typography.Text strong size="small">
                模型列表（可选）
              </Typography.Text>
              <Typography.Text type="tertiary" size="small">
                同一套 Key/Base 下可一次添加多条模型 ID；空行会在保存时忽略。
              </Typography.Text>
              {(llmConfigForm.models || []).map((row, idx) => (
                <div
                  key={idx}
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 8,
                    paddingBottom: 10,
                    borderBottom:
                      idx < (llmConfigForm.models || []).length - 1
                        ? '1px dashed rgba(6,7,9,0.08)'
                        : undefined,
                  }}
                >
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
                    <Input
                      style={{ flex: '1 1 200px', minWidth: 160 }}
                      value={row.modelName}
                      onChange={(v) => updateLlmModelDraftRow(idx, { modelName: String(v) })}
                      placeholder="模型 ID，如 gpt-4o"
                    />
                    <Select
                      style={{ width: 108 }}
                      value={row.enabled ? '1' : '0'}
                      onChange={(v) => updateLlmModelDraftRow(idx, { enabled: v === '1' })}
                    >
                      <Select.Option value="1">启用</Select.Option>
                      <Select.Option value="0">禁用</Select.Option>
                    </Select>
                    <Button type="danger" size="small" onClick={() => removeLlmModelDraftRow(idx)}>
                      删除本行
                    </Button>
                  </div>
                  <Input
                    value={row.description}
                    onChange={(v) => updateLlmModelDraftRow(idx, { description: String(v) })}
                    placeholder="该行说明（可选）"
                  />
                </div>
              ))}
              <div>
                <Button size="small" onClick={addLlmModelDraftRow}>
                  添加一行模型
                </Button>
              </div>
            </div>
          )}
          <TextArea
            value={llmConfigForm.description}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, description: String(v) })}
            autosize={{ minRows: 2, maxRows: 6 }}
            placeholder="整项配置描述（可选）"
          />
        </div>
      </Modal>

      <Modal
        title={entryEditing ? '编辑模型 ID' : '添加模型 ID'}
        visible={entryModalVisible}
        onCancel={() => setEntryModalVisible(false)}
        onOk={submitEntry}
        confirmLoading={entrySubmitting}
        style={{ width: 520 }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <Input
            value={entryForm.modelName}
            onChange={(v) => setEntryForm({ ...entryForm, modelName: String(v) })}
            placeholder="模型 ID，如 gpt-4o、gpt-4o-mini"
          />
          <Select
            value={entryForm.enabled ? '1' : '0'}
            onChange={(v) => setEntryForm({ ...entryForm, enabled: v === '1' })}
          >
            <Select.Option value="1">启用</Select.Option>
            <Select.Option value="0">禁用</Select.Option>
          </Select>
          <TextArea
            value={entryForm.description}
            onChange={(v) => setEntryForm({ ...entryForm, description: String(v) })}
            autosize={{ minRows: 2, maxRows: 6 }}
            placeholder="说明（可选）"
          />
        </div>
      </Modal>

      <Modal
        title={mcpEditing ? '编辑 MCP 配置' : '新增 MCP 配置'}
        visible={mcpModalVisible}
        onCancel={() => setMcpModalVisible(false)}
        onOk={submitMCP}
        confirmLoading={mcpSubmitting}
        style={{ width: 820 }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <Input
            value={mcpForm.name}
            onChange={(v) => setMcpForm({ ...mcpForm, name: String(v) })}
            placeholder="名称（唯一）"
          />
          <Input
            value={mcpForm.server}
            onChange={(v) => setMcpForm({ ...mcpForm, server: String(v) })}
            placeholder="Server 名称"
          />
          <Input
            value={mcpForm.endpoint}
            onChange={(v) => setMcpForm({ ...mcpForm, endpoint: String(v) })}
            placeholder="Endpoint（例如 https://example.com/mcp）"
          />
          <Select
            value={mcpForm.enabled ? '1' : '0'}
            onChange={(v) => setMcpForm({ ...mcpForm, enabled: v === '1' })}
          >
            <Select.Option value="1">启用</Select.Option>
            <Select.Option value="0">禁用</Select.Option>
          </Select>
          <TextArea
            value={headersText}
            onChange={(v) => setHeadersText(String(v))}
            autosize={{ minRows: 6, maxRows: 12 }}
            placeholder='Headers JSON，例如 {"Authorization":"Bearer xxx"}'
          />
          <TextArea
            value={mcpForm.description}
            onChange={(v) => setMcpForm({ ...mcpForm, description: String(v) })}
            autosize={{ minRows: 2, maxRows: 6 }}
            placeholder="描述（可选）"
          />
        </div>
      </Modal>
    </div>
  );
};

export default AgentSection;
