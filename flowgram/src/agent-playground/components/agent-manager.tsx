/**
 * Agent 管理：Agent 池的新建、删除与池信息维护；池内 Agent 只读展示（不可单独编辑/删除）
 */

import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Typography,
  Table,
  Tag,
  Space,
  Toast,
  Card,
  Switch,
  Modal,
  Input,
  Divider,
  Select,
  Spin,
  CheckboxGroup,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconDelete,
} from '@douyinfe/semi-icons';

import {
  AgentPool,
  AgentDefinition,
  createAgentPool,
  updateAgentPool,
  deleteAgentPool,
} from '../../services/api-playground';
import { listManagedAgents, type ManagedAgentItem } from '../../services/api-managed-agents';

const { Text } = Typography;

/** 根据主站「Agent 配置」勾选结果生成池内 Agent（托管关联） */
function agentsFromManagedSelection(ids: number[], catalog: ManagedAgentItem[]): AgentDefinition[] {
  return ids.map((mid, i) => {
    const m = catalog.find(x => x.id === mid);
    return {
      id: '',
      name: (m?.name || '').trim() || `Agent-${mid}`,
      role: '',
      desc: (m?.description || '').trim(),
      model: '',
      tools: [],
      enabled: m?.enabled !== false,
      priority: i,
      managedAgentId: mid,
    };
  });
}

interface AgentManagerProps {
  pools: AgentPool[];
  onPoolsChange: () => void;
}

export const AgentManager: React.FC<AgentManagerProps> = ({ pools, onPoolsChange }) => {
  const [selectedPool, setSelectedPool] = useState<AgentPool | undefined>(() => pools[0]);

  const [poolName, setPoolName] = useState('');
  const [poolDesc, setPoolDesc] = useState('');

  const [catalogLoading, setCatalogLoading] = useState(false);
  const [managedCatalog, setManagedCatalog] = useState<ManagedAgentItem[]>([]);

  const [createPoolModalVisible, setCreatePoolModalVisible] = useState(false);
  const [newPoolName, setNewPoolName] = useState('');
  const [newPoolDesc, setNewPoolDesc] = useState('');
  /** 新建池时预置：勾选的「Agent 配置」id（主站 Agent 管理菜单） */
  const [bootstrapManagedIds, setBootstrapManagedIds] = useState<number[]>([]);

  // 主站「Agent 配置」列表（Agent 管理菜单）
  useEffect(() => {
    let cancelled = false;
    (async () => {
      setCatalogLoading(true);
      try {
        const managed = await listManagedAgents();
        if (!cancelled) setManagedCatalog(Array.isArray(managed) ? managed : []);
      } catch {
        if (!cancelled) setManagedCatalog([]);
      } finally {
        if (!cancelled) setCatalogLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!pools.length) {
      setSelectedPool(undefined);
      setPoolName('');
      setPoolDesc('');
      return;
    }
    setSelectedPool(prev => {
      const still = pools.find(p => p.id === prev?.id);
      return still ?? pools[0];
    });
  }, [pools]);

  useEffect(() => {
    if (selectedPool) {
      setPoolName(selectedPool.name);
      setPoolDesc(selectedPool.description);
    }
  }, [selectedPool?.id, selectedPool?.name, selectedPool?.description]);

  const persistPool = useCallback(async (pool: AgentPool) => {
    await updateAgentPool(pool.id, {
      name: pool.name,
      description: pool.description,
      agents: pool.agents,
    });
    Toast.success('已保存');
    await onPoolsChange();
  }, [onPoolsChange]);

  const openCreatePoolModal = () => {
    setNewPoolName(`Agent池-${Date.now()}`);
    setNewPoolDesc('');
    setBootstrapManagedIds([]);
    setCreatePoolModalVisible(true);
  };

  const submitCreatePool = async () => {
    const name = newPoolName.trim();
    if (!name) {
      Toast.warning({ content: '请填写池名称' });
      return;
    }
    if (bootstrapManagedIds.length === 0) {
      Toast.warning({ content: '请至少勾选一项主站「Agent 配置」' });
      return;
    }
    try {
      const agents = agentsFromManagedSelection(bootstrapManagedIds, managedCatalog);
      const r = await createAgentPool({
        name,
        description: newPoolDesc.trim() || '新建的 Agent 池',
        agents,
      });
      Toast.success({ content: '创建成功' });
      setCreatePoolModalVisible(false);
      await onPoolsChange();
      if (r.pool) {
        setSelectedPool(r.pool);
      }
    } catch (err) {
      Toast.error({ content: `创建失败: ${err}` });
    }
  };

  const handleSavePoolMeta = async () => {
    if (!selectedPool) return;
    const name = poolName.trim();
    if (!name) {
      Toast.error('请填写池名称');
      return;
    }
    try {
      await persistPool({
        ...selectedPool,
        name,
        description: poolDesc,
      });
    } catch (err) {
      Toast.error(`保存失败: ${err}`);
    }
  };

  const handleDeletePool = (pool: AgentPool) => {
    if (pool.id === 'default') {
      Toast.warning('默认 Agent 池不可删除');
      return;
    }
    Modal.confirm({
      title: '删除 Agent 池',
      content: `确定删除「${pool.name}」吗？池内 Agent 定义将一并删除。`,
      onOk: async () => {
        try {
          await deleteAgentPool(pool.id);
          Toast.success('已删除');
          if (selectedPool?.id === pool.id) {
            setSelectedPool(undefined);
          }
          onPoolsChange();
        } catch (err) {
          Toast.error(`删除失败: ${err}`);
        }
      },
    });
  };

  const handleToggleAgent = async (pool: AgentPool, agentId: string, enabled: boolean) => {
    const updatedPool = {
      ...pool,
      agents: pool.agents.map(a => (a.id === agentId ? { ...a, enabled } : a)),
    };
    try {
      await persistPool(updatedPool);
    } catch (err) {
      Toast.error(`更新失败: ${err}`);
    }
  };

  /** 将池内某一成员关联到主站「Agent 配置」，模型/SKILL/MCP 以托管为准 */
  const handleBindManagedAgent = async (pool: AgentPool, agentId: string, raw: string | number | undefined | null) => {
    let mid = 0;
    if (raw !== '' && raw !== undefined && raw !== null) {
      const n = typeof raw === 'string' ? Number(raw) : Number(raw);
      if (Number.isFinite(n) && n > 0) {
        mid = n;
      }
    }
    const updatedPool = {
      ...pool,
      agents: pool.agents.map(a =>
        a.id === agentId ? { ...a, managedAgentId: mid > 0 ? mid : undefined } : a
      ),
    };
    try {
      await persistPool(updatedPool);
    } catch (err) {
      Toast.error(`绑定失败: ${err}`);
    }
  };

  const columns = [
    {
      title: '序号',
      dataIndex: 'index',
      render: (_: unknown, __: unknown, index: number) => index + 1,
      width: 60,
    },
    {
      title: 'ID',
      dataIndex: 'id',
      width: 120,
      render: (id: string) => (
        <Text type="tertiary" style={{ fontFamily: 'var(--semi-font-family-monospace)', fontSize: 12 }}>
          {id || '(新建时由服务端生成)'}
        </Text>
      ),
    },
    {
      title: 'Agent名称',
      dataIndex: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '角色',
      dataIndex: 'role',
      render: (text: string) => <Tag>{text}</Tag>,
    },
    {
      title: '绑定托管 Agent',
      key: 'managed',
      width: 280,
      render: (_: unknown, r: AgentDefinition) => (
        <Select
          placeholder="选择主站 Agent 配置…"
          style={{ width: '100%', maxWidth: 260 }}
          value={r.managedAgentId && r.managedAgentId > 0 ? String(r.managedAgentId) : ''}
          disabled={catalogLoading}
          onChange={(v) =>
            selectedPool &&
            handleBindManagedAgent(
              selectedPool,
              r.id,
              typeof v === 'string' || typeof v === 'number' ? v : undefined
            )}
        >
          <Select.Option value="">未绑定（协作运行前须选择）</Select.Option>
          {managedCatalog
            .filter(m => m.enabled !== false)
            .map(m => (
              <Select.Option key={m.id} value={String(m.id)}>
                {m.name} (#{m.id})
              </Select.Option>
            ))}
        </Select>
      ),
    },
    {
      title: '模型',
      dataIndex: 'model',
      render: (_text: string, r: AgentDefinition) => (
        <Text type="tertiary">{r.managedAgentId ? '（托管）' : '—'}</Text>
      ),
    },
    {
      title: '工具',
      dataIndex: 'tools',
      render: (_tools: string[], r: AgentDefinition) => (
        <Text type="tertiary">{r.managedAgentId ? '（托管）' : '—'}</Text>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      width: 72,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      render: (enabled: boolean, record: AgentDefinition) => (
        <Switch
          checked={enabled}
          onChange={checked =>
            selectedPool && handleToggleAgent(selectedPool, record.id, checked)
          }
        />
      ),
      width: 80,
    },
  ];

  return (
    <div>
      <Card
        title="Agent 池"
        headerExtraContent={
          <Button type="primary" theme="solid" icon={<IconPlus />} onClick={openCreatePoolModal}>
            新建 Agent 池
          </Button>
        }
      >
        <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 16, lineHeight: 1.65 }}>
          <strong style={{ color: 'var(--semi-color-text-1)' }}>默认池</strong>
          里的「设计师」「规划师」等是 Playground 预置
          <strong style={{ color: 'var(--semi-color-text-1)' }}>角色槽位</strong>
          ，不会在主站「Agent 配置」列表里自动出现同名条目。请在下方「绑定托管 Agent」列为每个槽位选择一项已在主站创建的配置，模型与工具以托管为准。
          若下拉为空，请先到顶部菜单 <strong>Agent 管理 → Agent 配置</strong> 新建并启用。
        </Text>
        <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 16, lineHeight: 1.65 }}>
          自定义新建池仍可在「新建 Agent 池」向导中勾选多条主站配置一键生成成员；任意池内也可在本表修改绑定或仅用启用/禁用开关。
        </Text>

        <div style={{ marginBottom: 16, display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 12 }}>
          <Text type="tertiary" size="small" style={{ flexShrink: 0 }}>当前池</Text>
          {pools.length === 0 ? (
            <Text type="warning" size="small">暂无 Agent 池，请先新建</Text>
          ) : (
            <>
              <Select
                placeholder="选择 Agent 池"
                style={{ minWidth: 260, maxWidth: 420 }}
                value={selectedPool?.id}
                onChange={id => {
                  const p = pools.find(x => x.id === id);
                  if (p) setSelectedPool(p);
                }}
              >
                {pools.map(pool => (
                  <Select.Option key={pool.id} value={pool.id}>
                    {pool.name} · {pool.agents?.length || 0} 个 Agent
                  </Select.Option>
                ))}
              </Select>
              {selectedPool && selectedPool.id !== 'default' ? (
                <Button
                  type="danger"
                  theme="borderless"
                  icon={<IconDelete />}
                  onClick={() => handleDeletePool(selectedPool)}
                >
                  删除当前池
                </Button>
              ) : null}
            </>
          )}
        </div>

        {selectedPool ? (
          <>
            <Divider margin="12px" align="center">
              池信息
            </Divider>
            <Space vertical align="start" style={{ width: '100%', marginBottom: 16 }}>
              <div style={{ width: '100%', maxWidth: 480 }}>
                <Text type="tertiary" style={{ display: 'block', marginBottom: 8 }}>
                  名称
                </Text>
                <Input
                  value={poolName}
                  onChange={setPoolName}
                  placeholder="池名称"
                />
              </div>
              <div style={{ width: '100%', maxWidth: 480 }}>
                <Text type="tertiary" style={{ display: 'block', marginBottom: 8 }}>
                  描述
                </Text>
                <Input
                  value={poolDesc}
                  onChange={setPoolDesc}
                  placeholder="可选"
                />
              </div>
              <Button type="primary" onClick={() => void handleSavePoolMeta()}>
                保存池信息
              </Button>
            </Space>

            <Divider margin="12px" align="center">
              Agent 列表
            </Divider>
            <Table
              columns={columns}
              dataSource={selectedPool.agents || []}
              rowKey="id"
              pagination={false}
            />
          </>
        ) : (
          <Text type="tertiary">暂无 Agent 池，请先新建</Text>
        )}
      </Card>

      <Modal
        title="新建 Agent 池"
        visible={createPoolModalVisible}
        onCancel={() => setCreatePoolModalVisible(false)}
        onOk={() => void submitCreatePool()}
        okText="创建"
        okButtonProps={{
          disabled: !newPoolName.trim() || bootstrapManagedIds.length === 0,
        }}
        maskClosable={false}
        width={520}
      >
        <Space vertical align="start" style={{ width: '100%' }}>
          <div style={{ width: '100%' }}>
            <Text type="tertiary" style={{ display: 'block', marginBottom: 8 }}>
              池名称 *
            </Text>
            <Input
              value={newPoolName}
              onChange={setNewPoolName}
              placeholder="例如：产品评审池"
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text type="tertiary" style={{ display: 'block', marginBottom: 8 }}>
              描述
            </Text>
            <Input value={newPoolDesc} onChange={setNewPoolDesc} placeholder="可选" />
          </div>
          <Divider margin="8px" />
          <Text strong style={{ display: 'block' }}>
            关联主站 Agent 配置 *
          </Text>
          <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 8 }}>
            与顶部菜单「Agent 管理 → Agent 配置」中已创建的条目对应；须至少勾选一项，可多选，创建后池内会自动生成带托管关联的
            Agent。
          </Text>
          <Spin spinning={catalogLoading} style={{ width: '100%' }}>
            {managedCatalog.filter(m => m.enabled !== false).length === 0 ? (
              <Text type="warning" size="small">
                暂无可用配置。请先在主站「Agent 管理 → Agent 配置」中新建并启用。
              </Text>
            ) : (
              <div
                style={{
                  maxHeight: 220,
                  overflow: 'auto',
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 8,
                  padding: 12,
                  width: '100%',
                }}
              >
                <CheckboxGroup
                  direction="vertical"
                  value={bootstrapManagedIds.map(String)}
                  onChange={v => {
                    const arr = Array.isArray(v) ? v : [];
                    setBootstrapManagedIds(
                      arr.map(x => Number(x)).filter(n => Number.isFinite(n) && n > 0)
                    );
                  }}
                  options={managedCatalog
                    .filter(m => m.enabled !== false)
                    .map(m => ({
                      label: `${m.name} (#${m.id})`,
                      value: String(m.id),
                    }))}
                />
              </div>
            )}
          </Spin>
        </Space>
      </Modal>
    </div>
  );
};
