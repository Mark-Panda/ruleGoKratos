import React, { useEffect, useRef, useState } from 'react';

import { Button, Input, Modal, Select, Spin, Table, TextArea, Toast, Typography } from '@douyinfe/semi-ui';

import {
  MCPConfigItem,
  MCPConfigPayload,
  createMCPConfig,
  deleteMCPConfig,
  listMCPConfigs,
  listSkills,
  updateMCPConfig,
  uploadSkill,
} from '../../services/api-agent';

const defaultMCPForm: MCPConfigPayload = {
  name: '',
  server: '',
  endpoint: '',
  headers: {},
  enabled: true,
  description: '',
};

export const AgentSection: React.FC<{ view?: 'skills' | 'mcps' }> = ({ view = 'skills' }) => {
  const [skillRoot, setSkillRoot] = useState('skills');
  const [skills, setSkills] = useState<any[]>([]);
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

  useEffect(() => {
    if (view === 'skills') fetchSkills();
    if (view === 'mcps') fetchMCPs();
  }, [view]);

  const filteredSkills = skills.filter((item) => {
    const kw = skillKeyword.trim().toLowerCase();
    if (!kw) return true;
    return (
      String(item.path || '')
        .toLowerCase()
        .includes(kw) ||
      String(item.name || '')
        .toLowerCase()
        .includes(kw)
    );
  });

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
                placeholder="搜索 skill 文件名或路径"
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
              dataSource={filteredSkills}
              rowKey={(r: any) => String(r.path)}
              pagination={{ pageSize: 10 }}
              columns={[
                { title: '文件名', dataIndex: 'name', width: 220 },
                { title: '相对路径', dataIndex: 'path' },
                {
                  title: '大小',
                  width: 120,
                  render: (_, r: any) => `${Number(r.size || 0)} B`,
                },
                {
                  title: '更新时间',
                  width: 220,
                  render: (_, r: any) => (r.updatedAt ? new Date(r.updatedAt).toLocaleString() : ''),
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
