/**
 * Agent 管理：维护内置 default Agent 池的成员与托管绑定（协作运行仅使用该池）。
 */

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Typography,
  Table,
  Tag,
  Space,
  Toast,
  Card,
  Switch,
  Input,
  Divider,
  Select,
} from '@douyinfe/semi-ui';

import {
  AgentPool,
  AgentDefinition,
  updateAgentPool,
} from '../../services/api-playground';
import { listManagedAgents, type ManagedAgentItem } from '../../services/api-managed-agents';

const { Text } = Typography;

interface AgentManagerProps {
  pools: AgentPool[];
  onPoolsChange: () => void;
}

export const AgentManager: React.FC<AgentManagerProps> = ({ pools, onPoolsChange }) => {
  /** 协作运行固定使用 default；兼容旧库中仅剩其它 id 的首项 */
  const displayPool = useMemo(() => pools.find(p => p.id === 'default') ?? pools[0], [pools]);

  const [poolName, setPoolName] = useState('');
  const [poolDesc, setPoolDesc] = useState('');

  const [catalogLoading, setCatalogLoading] = useState(false);
  const [managedCatalog, setManagedCatalog] = useState<ManagedAgentItem[]>([]);

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
    if (displayPool) {
      setPoolName(displayPool.name);
      setPoolDesc(displayPool.description);
    } else {
      setPoolName('');
      setPoolDesc('');
    }
  }, [displayPool?.id, displayPool?.name, displayPool?.description]);

  const persistPool = useCallback(async (pool: AgentPool) => {
    await updateAgentPool(pool.id, {
      name: pool.name,
      description: pool.description,
      agents: pool.agents,
    });
    Toast.success('已保存');
    await onPoolsChange();
  }, [onPoolsChange]);

  const handleSavePoolMeta = async () => {
    if (!displayPool) return;
    const name = poolName.trim();
    if (!name) {
      Toast.error('请填写池名称');
      return;
    }
    try {
      await persistPool({
        ...displayPool,
        name,
        description: poolDesc,
      });
    } catch (err) {
      Toast.error(`保存失败: ${err}`);
    }
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
            displayPool &&
            handleBindManagedAgent(
              displayPool,
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
            displayPool && handleToggleAgent(displayPool, record.id, checked)
          }
        />
      ),
      width: 80,
    },
  ];

  return (
    <div>
      <Card title="默认 Agent 池">
        <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 16, lineHeight: 1.65 }}>
          <strong style={{ color: 'var(--semi-color-text-1)' }}>协作运行仅使用本池（id=default）</strong>
          。下方的「设计师」「规划师」等是预置
          <strong style={{ color: 'var(--semi-color-text-1)' }}>角色槽位</strong>
          ，请在「绑定托管 Agent」列为每个槽位选择主站「Agent 配置」；模型与工具以托管为准。
          若下拉为空，请先到顶部菜单 <strong>Agent 管理 → Agent 配置</strong> 新建并启用。
        </Text>

        {!displayPool ? (
          <Text type="warning" size="small">
            暂无默认 Agent 池。请刷新页面；首次请求池列表时服务会自动创建 default。
          </Text>
        ) : (
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
              dataSource={displayPool.agents || []}
              rowKey="id"
              pagination={false}
            />
          </>
        )}
      </Card>
    </div>
  );
};
