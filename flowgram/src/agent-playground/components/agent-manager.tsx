/**
 * Agent 管理：维护内置 default Agent 池的成员与托管绑定（协作运行仅使用该池）。
 */

import React, { useCallback, useEffect, useMemo, useState } from 'react';

import {
  Button,
  Typography,
  Table,
  Tag,
  Toast,
  Card,
  Switch,
  Input,
  Divider,
  Select,
} from '@douyinfe/semi-ui';
import { IconSave } from '@douyinfe/semi-icons';

import { AgentPool, AgentDefinition, updateAgentPool } from '../../services/api-playground';
import { listManagedAgents, type ManagedAgentItem } from '../../services/api-managed-agents';

const { Text } = Typography;

interface AgentManagerProps {
  pools: AgentPool[];
  onPoolsChange: () => void;
}

export const AgentManager: React.FC<AgentManagerProps> = ({ pools, onPoolsChange }) => {
  const displayPool = useMemo(() => pools.find((p) => p.id === 'default') ?? pools[0], [pools]);

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

  const persistPool = useCallback(
    async (pool: AgentPool) => {
      await updateAgentPool(pool.id, {
        name: pool.name,
        description: pool.description,
        agents: pool.agents,
      });
      Toast.success('已保存');
      await onPoolsChange();
    },
    [onPoolsChange]
  );

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
      agents: pool.agents.map((a) => (a.id === agentId ? { ...a, enabled } : a)),
    };
    try {
      await persistPool(updatedPool);
    } catch (err) {
      Toast.error(`更新失败: ${err}`);
    }
  };

  const handleBindManagedAgent = async (
    pool: AgentPool,
    agentId: string,
    raw: string | number | undefined | null
  ) => {
    let mid = 0;
    if (raw !== '' && raw !== undefined && raw !== null) {
      const n = typeof raw === 'string' ? Number(raw) : Number(raw);
      if (Number.isFinite(n) && n > 0) {
        mid = n;
      }
    }
    const updatedPool = {
      ...pool,
      agents: pool.agents.map((a) =>
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
      render: (_: unknown, __: unknown, index: number) => (
        <div
          style={{
            width: 24,
            height: 24,
            borderRadius: 6,
            background: 'var(--semi-color-fill-0)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 12,
            color: 'var(--semi-color-tertiary)',
          }}
        >
          {index + 1}
        </div>
      ),
      width: 60,
    },
    {
      title: 'ID',
      dataIndex: 'id',
      width: 120,
      render: (id: string) => (
        <Text
          type="tertiary"
          style={{ fontFamily: 'var(--semi-font-family-monospace)', fontSize: 12 }}
        >
          {id || '(新建时由服务端生成)'}
        </Text>
      ),
    },
    {
      title: 'Agent 名称',
      dataIndex: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '角色',
      dataIndex: 'role',
      render: (text: string) => (
        <Tag color="cyan" style={{ borderRadius: 6 }}>
          {text}
        </Tag>
      ),
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
            )
          }
        >
          <Select.Option value="">未绑定（协作运行前须选择）</Select.Option>
          {managedCatalog
            .filter((m) => m.enabled !== false)
            .map((m) => (
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
      render: (val: number) => (
        <Tag size="small" style={{ borderRadius: 4 }}>
          {val}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      render: (enabled: boolean, record: AgentDefinition) => (
        <Switch
          checked={enabled}
          onChange={(checked) => displayPool && handleToggleAgent(displayPool, record.id, checked)}
        />
      ),
      width: 80,
    },
  ];

  return (
    <div>
      <Card
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: 8,
                background: 'linear-gradient(135deg, rgba(22, 100, 255, 0.08), rgba(19, 194, 194, 0.08))',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 16,
              }}
            >
              🤖
            </div>
            <div>
              <Text strong style={{ fontSize: 14, display: 'block' }}>默认 Agent 池</Text>
              <Text type="tertiary" size="small">id=default</Text>
            </div>
          </div>
        }
        style={{ borderRadius: 14, boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)' }}
      >
        <div
          style={{
            padding: '12px 16px',
            background: 'var(--semi-color-info-light-default)',
            border: '1px solid rgba(22, 100, 255, 0.12)',
            borderRadius: 10,
            marginBottom: 16,
            lineHeight: 1.65,
          }}
        >
          <Text size="small" style={{ lineHeight: 1.65 }}>
            <strong style={{ color: 'var(--semi-color-text-1)' }}>
              协作运行仅使用本池（id=default）
            </strong>
            。下方的「设计师」「规划师」等是预置
            <strong style={{ color: 'var(--semi-color-text-1)' }}>角色槽位</strong>
            ，请在「绑定托管 Agent」列为每个槽位选择主站「Agent 配置」；模型与工具以托管为准。
            若下拉为空，请先到顶部菜单 <strong>Agent 管理 → Agent 配置</strong> 新建并启用。
          </Text>
        </div>

        {!displayPool ? (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <div
              style={{
                width: 56,
                height: 56,
                borderRadius: 16,
                background: 'var(--semi-color-warning-light-default)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                margin: '0 auto 12px',
                fontSize: 24,
              }}
            >
              ⚠️
            </div>
            <Text type="warning" size="small">
              暂无默认 Agent 池。请刷新页面；首次请求池列表时服务会自动创建 default。
            </Text>
          </div>
        ) : (
          <>
            <Divider margin="12px" align="center">
              池信息
            </Divider>
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 16,
                maxWidth: 520,
                marginBottom: 16,
              }}
            >
              <div>
                <Text
                  type="tertiary"
                  size="small"
                  style={{ display: 'block', marginBottom: 6, fontWeight: 500 }}
                >
                  名称
                </Text>
                <Input
                  value={poolName}
                  onChange={setPoolName}
                  placeholder="池名称"
                  style={{ borderRadius: 8 }}
                />
              </div>
              <div>
                <Text
                  type="tertiary"
                  size="small"
                  style={{ display: 'block', marginBottom: 6, fontWeight: 500 }}
                >
                  描述
                </Text>
                <Input
                  value={poolDesc}
                  onChange={setPoolDesc}
                  placeholder="可选"
                  style={{ borderRadius: 8 }}
                />
              </div>
              <Button
                type="primary"
                theme="solid"
                icon={<IconSave />}
                onClick={() => void handleSavePoolMeta()}
                style={{ alignSelf: 'flex-start', borderRadius: 8 }}
              >
                保存池信息
              </Button>
            </div>

            <Divider margin="12px" align="center">
              Agent 列表
            </Divider>
            <Table
              columns={columns}
              dataSource={displayPool.agents || []}
              rowKey="id"
              pagination={false}
              style={{ borderRadius: 10, overflow: 'hidden' }}
            />
          </>
        )}
      </Card>
    </div>
  );
};
