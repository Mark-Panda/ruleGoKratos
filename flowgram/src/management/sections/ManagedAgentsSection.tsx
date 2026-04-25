/**
 * Agent 配置：系统提示、SKILL、MCP、模型（按 LLM 站点选择全部或指定模型）
 */

import React, { useCallback, useEffect, useMemo, useState } from 'react';

import {
  Button,
  Card,
  CheckboxGroup,
  Divider,
  Input,
  Modal,
  Radio,
  RadioGroup,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  TextArea,
  Toast,
  Typography,
  Switch,
} from '@douyinfe/semi-ui';
import { IconDelete, IconPlus, IconRefresh, IconEdit } from '@douyinfe/semi-icons';

import { listWorkspaces, type WorkspaceItem } from '../../services/api-workspaces';
import {
  listManagedAgents,
  createManagedAgent,
  updateManagedAgent,
  deleteManagedAgent,
  type ManagedAgentItem,
  type ManagedAgentPayload,
} from '../../services/api-managed-agents';
import {
  listLlmConfigs,
  type LlmConfigItem,
} from '../../services/api-agent';

const { Text, Title } = Typography;

function emptyPayload(): ManagedAgentPayload {
  return {
    name: '',
    description: '',
    systemPrompt: '',
    workspaceId: '',
    skillPackageIds: [],
    mcpIds: [],
    llmConfigId: 0,
    modelScope: 'all',
    modelEntryIds: [],
    enabled: true,
  };
}

export const ManagedAgentsSection: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [rows, setRows] = useState<ManagedAgentItem[]>([]);
  const [llmConfigs, setLlmConfigs] = useState<LlmConfigItem[]>([]);
  const [workspaces, setWorkspaces] = useState<WorkspaceItem[]>([]);

  const [modalOpen, setModalOpen] = useState(false);
  const [modalMode, setModalMode] = useState<'create' | 'edit'>('create');
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<ManagedAgentPayload>(emptyPayload());

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const [agents, llm, ws] = await Promise.all([
        listManagedAgents(),
        listLlmConfigs(),
        listWorkspaces(),
      ]);
      setRows(agents);
      setLlmConfigs(Array.isArray(llm) ? llm : []);
      setWorkspaces(Array.isArray(ws) ? ws : []);
    } catch (e) {
      Toast.error(`加载失败：${e}`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  const enabledLlmConfigs = useMemo(() => llmConfigs.filter((c) => c.enabled), [llmConfigs]);

  const selectedConfigModels = useMemo(() => {
    const c = enabledLlmConfigs.find((x) => x.id === form.llmConfigId);
    return (c?.models || []).filter((m) => m.enabled);
  }, [enabledLlmConfigs, form.llmConfigId]);

  const configName = useCallback(
    (id: number) => enabledLlmConfigs.find((c) => c.id === id)?.name || `#${id}`,
    [enabledLlmConfigs]
  );

  const openCreate = () => {
    setModalMode('create');
    setEditingId(null);
    setForm(emptyPayload());
    setModalOpen(true);
  };

  const openEdit = (r: ManagedAgentItem) => {
    setModalMode('edit');
    setEditingId(r.id);
    setForm({
      name: r.name,
      description: r.description || '',
      systemPrompt: r.systemPrompt || '',
      workspaceId: r.workspaceId || '',
      skillPackageIds: [],
      mcpIds: [],
      llmConfigId: r.llmConfigId,
      modelScope: r.modelScope === 'explicit' ? 'explicit' : 'all',
      modelEntryIds: [...(r.modelEntryIds || [])],
      enabled: r.enabled !== false,
    });
    setModalOpen(true);
  };

  const submit = async () => {
    if (!form.name?.trim()) {
      Toast.warning('请填写名称');
      return;
    }
    if (!form.llmConfigId) {
      Toast.warning('请选择 LLM 配置（站点）');
      return;
    }
    const payload: ManagedAgentPayload = {
      ...form,
      name: form.name.trim(),
      description: (form.description || '').trim(),
      skillPackageIds: [],
      mcpIds: [],
      modelEntryIds: form.modelScope === 'explicit' ? form.modelEntryIds || [] : [],
    };
    if (
      payload.modelScope === 'explicit' &&
      (!payload.modelEntryIds || payload.modelEntryIds.length === 0)
    ) {
      Toast.warning('指定模型时请至少勾选一条启用中的模型');
      return;
    }
    try {
      if (modalMode === 'create') {
        await createManagedAgent(payload);
        Toast.success('已创建');
      } else if (editingId != null) {
        await updateManagedAgent(editingId, payload);
        Toast.success('已更新');
      }
      setModalOpen(false);
      void loadAll();
    } catch (e) {
      Toast.error(`保存失败：${e}`);
    }
  };

  const onDelete = (r: ManagedAgentItem) => {
    Modal.confirm({
      title: '删除 Agent 配置',
      content: `确定删除「${r.name}」吗？`,
      onOk: async () => {
        try {
          await deleteManagedAgent(r.id);
          Toast.success('已删除');
          void loadAll();
        } catch (e) {
          Toast.error(`删除失败：${e}`);
        }
      },
    });
  };

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      render: (t: string, r: ManagedAgentItem) => (
        <Space>
          <Text strong>{t}</Text>
          {!r.enabled && <Tag color="grey">停用</Tag>}
        </Space>
      ),
    },
    {
      title: 'LLM 站点',
      key: 'llm',
      render: (_: unknown, r: ManagedAgentItem) => <Text>{configName(r.llmConfigId)}</Text>,
    },
    {
      title: '模型范围',
      key: 'scope',
      render: (_: unknown, r: ManagedAgentItem) =>
        r.modelScope === 'explicit' ? (
          <Tag color="blue">指定 {r.modelEntryIds?.length || 0} 个模型</Tag>
        ) : (
          <Tag color="green">站点下全部启用模型</Tag>
        ),
    },
    {
      title: '工作区',
      key: 'workspace',
      width: 220,
      ellipsis: true,
      render: (_: unknown, r: ManagedAgentItem) => {
        const wid = (r.workspaceId || '').trim();
        if (!wid) return '—';
        const matched = workspaces.find((w) => (w.id || '').trim() === wid);
        return matched ? `${matched.name}（${matched.id}）` : wid;
      },
    },
    {
      title: '操作',
      key: 'op',
      width: 200,
      render: (_: unknown, r: ManagedAgentItem) => (
        <Space>
          <Button type="tertiary" icon={<IconEdit />} size="small" onClick={() => openEdit(r)}>
            编辑
          </Button>
          <Button type="danger" icon={<IconDelete />} size="small" onClick={() => onDelete(r)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Card>
        <Space vertical align="start" style={{ width: '100%' }}>
          <Title heading={5} style={{ margin: 0 }}>
            Agent 配置
          </Title>
          <Space style={{ marginTop: 8 }}>
            <Button type="primary" theme="solid" icon={<IconPlus />} onClick={openCreate}>
              新建 Agent
            </Button>
            <Button icon={<IconRefresh />} onClick={() => void loadAll()}>
              刷新
            </Button>
          </Space>
        </Space>
      </Card>

      <Card style={{ marginTop: 16 }}>
        <Spin spinning={loading}>
          <Table columns={columns} dataSource={rows} rowKey="id" pagination={{ pageSize: 10 }} />
        </Spin>
      </Card>

      <Modal
        visible={modalOpen}
        title={modalMode === 'create' ? '新建 Agent 配置' : '编辑 Agent 配置'}
        onCancel={() => setModalOpen(false)}
        onOk={() => void submit()}
        width={720}
        maskClosable={false}
      >
        <Space vertical align="start" style={{ width: '100%' }}>
          <div style={{ width: '100%' }}>
            <Text type="tertiary">名称 *</Text>
            <Input
              style={{ marginTop: 8 }}
              value={form.name}
              onChange={(v) => setForm((f) => ({ ...f, name: v }))}
              placeholder="便于识别的名称"
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text type="tertiary">说明</Text>
            <Input
              style={{ marginTop: 8 }}
              value={form.description || ''}
              onChange={(v) => setForm((f) => ({ ...f, description: v }))}
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text type="tertiary">系统提示词（System prompt）</Text>
            <TextArea
              style={{ marginTop: 8 }}
              rows={6}
              value={form.systemPrompt || ''}
              onChange={(v) => setForm((f) => ({ ...f, systemPrompt: v }))}
              placeholder="定义 Agent 的角色与行为约束"
            />
            <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 8 }}>
              若选择工作区，系统会在运行时自动把“工作区使用模式”提示注入到系统提示词中。
            </Text>
          </div>

          <div style={{ width: '100%' }}>
            <Text type="tertiary">工作区（可选）</Text>
            <Select
              style={{ width: '100%', marginTop: 8 }}
              value={form.workspaceId || ''}
              onChange={(v) => setForm((f) => ({ ...f, workspaceId: String(v || '') }))}
              placeholder="不选择则不注入工作区使用模式"
              filter
            >
              <Select.Option value="">不绑定工作区</Select.Option>
              {workspaces.map((w) => (
                <Select.Option key={w.id} value={w.id} text={`${w.name}（${w.id}）`}>
                  {w.name}（{w.id}）
                </Select.Option>
              ))}
            </Select>
          </div>

          <Divider margin="12px" />

          <Text type="tertiary" size="small">
            MCP 默认加载「MCP 配置」中所有已启用的 server，无需在每个 Agent 中单独勾选。
          </Text>

          <div style={{ width: '100%' }}>
            <Text strong>模型（模型管理）</Text>
            <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 8 }}>
              先选择 LLM 配置站点，再决定使用该站点下全部启用模型或仅指定部分模型条目。
            </Text>
            <Select
              placeholder="选择 LLM 配置"
              style={{ width: '100%', marginBottom: 12 }}
              value={form.llmConfigId || undefined}
              onChange={(v) => {
                const id = typeof v === 'number' ? v : Number(v);
                setForm((f) => ({
                  ...f,
                  llmConfigId: Number.isFinite(id) ? id : 0,
                  modelEntryIds: [],
                }));
              }}
              filter
            >
              {enabledLlmConfigs.map((c) => (
                <Select.Option key={c.id} value={c.id} text={c.name}>
                  {c.name}（{c.provider}）
                </Select.Option>
              ))}
            </Select>

            <RadioGroup
              value={form.modelScope}
              onChange={(e) => {
                const v = e.target.value as 'all' | 'explicit';
                setForm((f) => ({
                  ...f,
                  modelScope: v,
                  modelEntryIds: v === 'all' ? [] : f.modelEntryIds,
                }));
              }}
              aria-label="模型范围"
            >
              <Radio value="all">使用该站点下全部启用模型</Radio>
              <Radio value="explicit">仅使用下列勾选的模型条目</Radio>
            </RadioGroup>

            {form.modelScope === 'explicit' && form.llmConfigId ? (
              <div
                style={{
                  marginTop: 12,
                  maxHeight: 200,
                  overflow: 'auto',
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 8,
                  padding: 12,
                }}
              >
                {selectedConfigModels.length === 0 ? (
                  <Text type="warning">当前站点下没有启用的模型条目，请先在模型管理中添加。</Text>
                ) : (
                  <CheckboxGroup
                    direction="vertical"
                    value={form.modelEntryIds || []}
                    onChange={(v) => setForm((f) => ({ ...f, modelEntryIds: v as number[] }))}
                    options={selectedConfigModels.map((m) => ({
                      label: `${m.modelName}${m.description ? ` — ${m.description}` : ''}`,
                      value: m.id,
                    }))}
                  />
                )}
              </div>
            ) : null}
          </div>

          <Space style={{ marginTop: 8 }}>
            <Text type="tertiary">启用</Text>
            <Switch
              checked={form.enabled !== false}
              onChange={(checked) => setForm((f) => ({ ...f, enabled: checked }))}
            />
          </Space>
        </Space>
      </Modal>
    </div>
  );
};
