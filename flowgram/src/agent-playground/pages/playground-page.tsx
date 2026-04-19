/**
 * Agent Playground - 多智能体协作编排主页面
 */

import React, { useEffect, useState, useCallback, useRef, useMemo } from 'react';
import {
  Button,
  ButtonGroup,
  Input,
  Select,
  Typography,
  Tag,
  Modal,
  Toast,
  Table,
  Row,
  Col,
  Switch,
  Space,
  Divider,
  Spin,
  Card,
  Badge,
  Tabs,
} from '@douyinfe/semi-ui';
import {
  IconPlus,
  IconPlay,
  IconRefresh,
  IconDelete,
  IconEdit,
  IconTick,
  IconClose,
  IconGallery,
  IconUserGroup,
  IconLayers,
  IconDesktop,
  IconSetting,
} from '@douyinfe/semi-icons';

import {
  AgentPool,
  CollaborationScheme,
  CollaborationMode,
  TraceRun,
  TraceEvent,
  MODE_NAME_MAP,
  MODE_DESC_MAP,
  listSchemes,
  createScheme,
  updateScheme,
  deleteScheme,
  runWorkflow,
  getRun,
  getRunEvents,
  listAgentPools,
  AgentBinding,
} from '../../services/api-playground';
import { getApiOrigin } from '../../services/http';

function mergeTraceEvents(prev: TraceEvent[], incoming: TraceEvent): TraceEvent[] {
  const m = new Map(prev.map(e => [e.id, e]));
  m.set(incoming.id, incoming);
  return [...m.values()].sort((a, b) => a.timestamp - b.timestamp);
}
import { AgentManager } from '../components/agent-manager';
import { ModeSelector } from '../components/mode-selector';
import { WorkflowGraph } from '../components/workflow-graph';
import { TracePanel } from '../components/trace-panel';
import { RunConsole } from '../components/run-console';

const { Text, Title } = Typography;

/** 规划执行推荐成员顺序（按池内是否存在过滤，与设计器模板一致） */
const PLAN_EXEC_BIND_TEMPLATE: AgentBinding[] = [
  { agentId: 'planner', role: '规划师' },
  { agentId: 'designer', role: '设计师' },
  { agentId: 'pm', role: '产品经理' },
  { agentId: 'engineer', role: '工程师' },
];

function buildCreateBindAgents(mode: CollaborationMode, pool: AgentPool | undefined): AgentBinding[] {
  const defs = pool?.agents || [];
  const byId = new Map(defs.map(a => [a.id, a]));
  if (mode === 'plan_exec') {
    const list: AgentBinding[] = [];
    for (const t of PLAN_EXEC_BIND_TEMPLATE) {
      const def = byId.get(t.agentId);
      if (def) {
        list.push({ agentId: def.id, role: def.name });
      }
    }
    if (list.length > 0) {
      return list;
    }
  }
  return defs.map(a => ({ agentId: a.id, role: a.name }));
}

type PlaygroundLang = 'zh' | 'en';

const PLAYGROUND_UI: Record<
  PlaygroundLang,
  {
    subtitle: string;
    overview: string;
    agents: string;
    schemes: string;
    run: string;
    settings: string;
    refresh: string;
    systemReady: string;
    mvp: string;
  }
> = {
  zh: {
    subtitle: '多智能体编排与执行追踪',
    overview: '总览',
    agents: '智能体',
    schemes: '协作编排',
    run: '运行',
    settings: '设置',
    refresh: '刷新数据',
    systemReady: '系统就绪',
    mvp: 'MVP',
  },
  en: {
    subtitle: 'Multi-agent orchestration & trace',
    overview: 'Overview',
    agents: 'Agents',
    schemes: 'Orchestration',
    run: 'Run',
    settings: 'Settings',
    refresh: 'Refresh',
    systemReady: 'Ready',
    mvp: 'MVP',
  },
};

// Playground 主页面组件
export const AgentPlaygroundPage: React.FC = () => {
  // 状态
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const [schemes, setSchemes] = useState<CollaborationScheme[]>([]);
  const [pools, setPools] = useState<AgentPool[]>([]);
  const [selectedScheme, setSelectedScheme] = useState<CollaborationScheme | undefined>();
  const [currentRun, setCurrentRun] = useState<TraceRun | undefined>();
  const [events, setEvents] = useState<TraceEvent[]>([]);
  const [activeTab, setActiveTab] = useState<'overview' | 'agents' | 'schemes' | 'run' | 'settings'>('overview');
  const [lang, setLang] = useState<PlaygroundLang>('zh');
  const [running, setRunning] = useState(false);

  // Scheme CRUD Modal
  const [showSchemeModal, setShowSchemeModal] = useState(false);
  const [schemeModalMode, setSchemeModalMode] = useState<'create' | 'edit'>('create');
  const [editingScheme, setEditingScheme] = useState<CollaborationScheme | undefined>();
  const [schemeForm, setSchemeForm] = useState({
    name: '',
    description: '',
    mode: 'router_expert' as CollaborationMode,
    enableFinalizer: false,
  });

  /** SSE EventSource（实时 Trace）；失败时用 fallback 定时拉取 */
  const eventSourceRef = useRef<EventSource | null>(null);
  const sseFallbackPollRef = useRef<number | null>(null);
  /** 单次运行完成/失败仅弹出一次 Toast（SSE 与轮询路径共用） */
  const completionToastRunIdRef = useRef<string | null>(null);

  // 运行结束后 Toast 告知用户最终结果摘要（与中间栏「最终结果」一致）
  useEffect(() => {
    if (!currentRun?.runId) return;
    const { runId, status } = currentRun;
    if (status !== 'completed' && status !== 'failed') return;
    if (completionToastRunIdRef.current === runId) return;
    completionToastRunIdRef.current = runId;

    if (status === 'completed') {
      const raw = (currentRun.finalOutput || '').trim();
      const preview =
        raw.length > 0
          ? [...raw].slice(0, 220).join('') + (raw.length > 220 ? '…' : '')
          : '';
      Toast.success({
        content:
          preview.length > 0
            ? `运行完成\n${preview}`
            : '运行完成。\n请在中间栏查看「最终结果」，或右侧 Trace 了解步骤详情。',
        duration: 6,
      });
    } else {
      Toast.error({
        content: '运行失败。\n请在右侧 Trace 查看错误相关事件。',
        duration: 6,
      });
    }
  }, [currentRun?.runId, currentRun?.status, currentRun?.finalOutput]);

  // 加载数据
  const loadData = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const [poolsRes, schemesRes] = await Promise.all([
        listAgentPools(),
        listSchemes(),
      ]);
      setPools(poolsRes.pools || []);
      setSchemes(schemesRes.schemes || []);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  useEffect(() => {
    if (!currentRun?.runId || currentRun.status !== 'running') {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (sseFallbackPollRef.current != null) {
        clearInterval(sseFallbackPollRef.current);
        sseFallbackPollRef.current = null;
      }
      return undefined;
    }

    const runId = currentRun.runId;

    const clearFallback = () => {
      if (sseFallbackPollRef.current != null) {
        clearInterval(sseFallbackPollRef.current);
        sseFallbackPollRef.current = null;
      }
    };

    const startFallbackPolling = () => {
      clearFallback();
      sseFallbackPollRef.current = window.setInterval(() => {
        void (async () => {
          try {
            const [runRes, evRes] = await Promise.all([
              getRun(runId),
              getRunEvents(runId),
            ]);
            setCurrentRun(runRes.run);
            setEvents(evRes.events || []);
            if (runRes.run.status !== 'running') {
              setRunning(false);
              clearFallback();
            }
          } catch {
            /* ignore */
          }
        })();
      }, 650) as unknown as number;
    };

    const token =
      typeof window !== 'undefined'
        ? window.localStorage.getItem('AUTH_TOKEN') || window.localStorage.getItem('token') || ''
        : '';
    const streamUrl = `${getApiOrigin()}/api/v1/playground/run/${encodeURIComponent(runId)}/events/stream${
      token ? `?token=${encodeURIComponent(token)}` : ''
    }`;

    const es = new EventSource(streamUrl);
    eventSourceRef.current = es;

    es.onmessage = async event => {
      try {
        const raw = JSON.parse(event.data) as TraceEvent;
        setEvents(prev => mergeTraceEvents(prev, raw));
        if (raw.type === 'WORKFLOW_END') {
          es.close();
          eventSourceRef.current = null;
          clearFallback();
          try {
            const runRes = await getRun(runId);
            setCurrentRun(runRes.run);
            setEvents(runRes.run.events?.length ? runRes.run.events : []);
          } catch {
            /* ignore */
          }
          setRunning(false);
        }
      } catch {
        /* ignore malformed chunk */
      }
    };

    es.onerror = () => {
      es.close();
      eventSourceRef.current = null;
      startFallbackPolling();
    };

    return () => {
      es.close();
      eventSourceRef.current = null;
      clearFallback();
    };
  }, [currentRun?.runId, currentRun?.status]);

  // Scheme Modal 操作
  const openCreateSchemeModal = () => {
    setSchemeForm({ name: '', description: '', mode: 'router_expert', enableFinalizer: false });
    setSchemeModalMode('create');
    setShowSchemeModal(true);
  };

  const openEditSchemeModal = (scheme: CollaborationScheme) => {
    setEditingScheme(scheme);
    setSchemeForm({
      name: scheme.name,
      description: scheme.description,
      mode: scheme.mode,
      enableFinalizer: scheme.enableFinalizer,
    });
    setSchemeModalMode('edit');
    setShowSchemeModal(true);
  };

  const handleSaveScheme = async () => {
    if (!schemeForm.name.trim()) {
      Toast.error('请输入方案名称');
      return;
    }
    try {
      if (schemeModalMode === 'create') {
        const pool = pools[0];
        const bindAgents = buildCreateBindAgents(schemeForm.mode, pool);
        if (schemeForm.mode === 'plan_exec' && !bindAgents.some(b => b.agentId === 'planner')) {
          Toast.warning(
            '当前 Agent 池中缺少「规划师」(planner)，规划执行将无法启动。请在「智能体」页确认默认池含规划师或使用协调人提供的完整池。',
          );
        }
        await createScheme({
          name: schemeForm.name,
          description: schemeForm.description,
          mode: schemeForm.mode,
          bindAgents,
          enableFinalizer: schemeForm.enableFinalizer,
        });
        Toast.success('方案创建成功');
      } else if (editingScheme) {
        await updateScheme(editingScheme.id, {
          name: schemeForm.name,
          description: schemeForm.description,
          mode: schemeForm.mode,
          enableFinalizer: schemeForm.enableFinalizer,
        });
        Toast.success('方案更新成功');
      }
      setShowSchemeModal(false);
      loadData();
    } catch (err) {
      Toast.error(`操作失败: ${err}`);
    }
  };

  const handleDeleteScheme = async (scheme: CollaborationScheme) => {
    try {
      await deleteScheme(scheme.id);
      Toast.success('删除成功');
      if (selectedScheme?.id === scheme.id) {
        setSelectedScheme(undefined);
      }
      loadData();
    } catch (err) {
      Toast.error(`删除失败: ${err}`);
    }
  };

  // 运行工作流
  const handleRunWorkflow = async (scheme: CollaborationScheme, userInput: string) => {
    if (!userInput.trim()) {
      Toast.error('请输入任务描述');
      return;
    }
    setRunning(true);
    setCurrentRun(undefined);
    setEvents([]);
    setActiveTab('run');

    try {
      const res = await runWorkflow({ schemeId: scheme.id, userInput });
      const runId = res.runId;

      const [runRes, eventsRes] = await Promise.all([
        getRun(runId),
        getRunEvents(runId).catch(() => ({ events: [] as TraceEvent[] })),
      ]);
      setCurrentRun(runRes.run);
      setEvents(eventsRes.events || []);
    } catch (err) {
      Toast.error(`运行失败: ${err}`);
      setRunning(false);
    }
  };

  // Scheme 表格列
  const schemeColumns = [
    {
      title: '序号',
      dataIndex: 'index',
      render: (_: any, __: any, index: number) => index + 1,
      width: 60,
    },
    {
      title: '方案名称',
      dataIndex: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '协作模式',
      dataIndex: 'mode',
      render: (mode: CollaborationMode) => (
        <Tag color="blue">{MODE_NAME_MAP[mode] || mode}</Tag>
      ),
    },
    {
      title: '绑定Agent',
      dataIndex: 'bindAgents',
      render: (agents: any[]) => (
        <Space>
          {(agents || []).map((a, i) => (
            <Tag key={i} type="ghost">{a.role || a.agentId}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: 'Finalizer',
      dataIndex: 'enableFinalizer',
      render: (enabled: boolean) => enabled ? <IconTick /> : <IconClose />,
      width: 80,
    },
    {
      title: '操作',
      render: (_: any, record: CollaborationScheme) => (
        <Space>
          <Button
            size="small"
            icon={<IconPlay />}
            onClick={() => setSelectedScheme(record)}
          >
            选择
          </Button>
          <Button
            size="small"
            icon={<IconEdit />}
            onClick={() => openEditSchemeModal(record)}
          />
          <Button
            size="small"
            icon={<IconDelete />}
            onClick={() => Modal.confirm({
              title: '确认删除',
              content: `确定要删除方案「${record.name}」吗？`,
              onOk: () => handleDeleteScheme(record),
            })}
          />
        </Space>
      ),
      width: 200,
    },
  ];

  const totalAgents = pools.reduce((acc, p) => acc + (p.agents?.length || 0), 0);
  const ui = PLAYGROUND_UI[lang];

  const graphFooterLine = useMemo(() => {
    if (running) {
      return '> 运行中 · Trace 实时推送…';
    }
    if (currentRun?.status === 'completed') {
      const ms = currentRun.totalMs > 0 ? `${currentRun.totalMs}ms` : '—';
      return `> 已完成 · ${ms}`;
    }
    if (currentRun?.status === 'failed') {
      return '> 运行失败';
    }
    if (selectedScheme) {
      return '> 等待交互…';
    }
    return '> 等待选择方案…';
  }, [running, currentRun, selectedScheme]);

  /** 运行页三栏最小高度，兼顾笔记本小屏 */
  const runAreaMinH = 'min(72vh, 760px)';

  const cardClickStyle: React.CSSProperties = {
    cursor: 'pointer',
    transition: 'box-shadow 0.2s, transform 0.15s',
  };

  return (
    <div
      className="pg-playground-shell"
      style={{
        boxSizing: 'border-box',
        width: '100%',
        maxWidth: '100%',
        margin: 0,
        padding: '12px clamp(12px, 1.5vw, 20px) 28px',
        height: '100%',
        overflow: 'auto',
        background: 'var(--semi-color-fill-0)',
      }}
    >
      <style>
        {`
        .pg-playground-shell .pg-shell-tabs .semi-tabs-bar {
          border-bottom: none !important;
          padding: 4px 0 12px;
        }
        .pg-playground-shell .pg-shell-tabs .semi-tabs-tab {
          border-radius: 999px !important;
          padding: 8px 14px !important;
          margin-right: 6px !important;
          transition: background 0.2s ease, color 0.2s ease;
        }
        .pg-playground-shell .pg-shell-tabs .semi-tabs-tab:hover {
          background: var(--semi-color-fill-1) !important;
        }
        .pg-playground-shell .pg-shell-tabs .semi-tabs-tab-active {
          background: rgba(28, 31, 35, 0.92) !important;
          color: #fff !important;
          font-weight: 600;
        }
        .pg-playground-shell .pg-shell-tabs .semi-tabs-tab-active:hover {
          background: rgba(28, 31, 35, 0.88) !important;
          color: #fff !important;
        }
        .pg-playground-shell .pg-shell-tabs .semi-tabs-tab-active .semi-icon {
          color: #fff !important;
        }
        .pg-playground-shell .pg-graph-card.semi-card > .semi-card-header {
          border-bottom: none;
        }
        @keyframes pg-ready-pulse {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.65; transform: scale(0.92); }
        }
      `}
      </style>

      {/* 顶栏：品牌 + 标签 + 状态 */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 16,
          flexWrap: 'wrap',
          marginBottom: 4,
        }}
      >
        <Space align="center" spacing="loose">
          <IconDesktop size="large" style={{ color: 'var(--semi-color-primary)' }} />
          <div>
            <Text strong style={{ fontSize: 18, display: 'block', lineHeight: '24px' }}>
              Agent Playground
            </Text>
            <Text type="tertiary" size="small">
              {ui.subtitle}
            </Text>
          </div>
        </Space>
        <Space spacing="tight" style={{ marginLeft: 'auto' }}>
          <ButtonGroup size="small" type="tertiary">
            <Button type={lang === 'zh' ? 'primary' : 'tertiary'} onClick={() => setLang('zh')}>
              中文
            </Button>
            <Button type={lang === 'en' ? 'primary' : 'tertiary'} onClick={() => setLang('en')}>
              EN
            </Button>
          </ButtonGroup>
          <Tag color="grey" size="large">
            {ui.mvp}
          </Tag>
          <Space spacing={4}>
            <span
              aria-hidden
              style={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                background: 'var(--semi-color-success)',
                display: 'inline-block',
                animation: 'pg-ready-pulse 2s ease-in-out infinite',
              }}
            />
            <Text size="small" type="tertiary">
              {ui.systemReady}
            </Text>
          </Space>
        </Space>
      </div>

      <Tabs
        className="pg-shell-tabs"
        type="line"
        activeKey={activeTab}
        onChange={k => setActiveTab(k as typeof activeTab)}
        tabBarExtraContent={
          <Button icon={<IconRefresh />} size="small" onClick={() => void loadData()}>
            {ui.refresh}
          </Button>
        }
        keepDOM={false}
      >
        <Tabs.TabPane
          tab={
            <span>
              <IconGallery style={{ marginRight: 6, verticalAlign: '-2px' }} />
              {ui.overview}
            </span>
          }
          itemKey="overview"
        />
        <Tabs.TabPane
          tab={
            <span>
              <IconUserGroup style={{ marginRight: 6, verticalAlign: '-2px' }} />
              {ui.agents}
            </span>
          }
          itemKey="agents"
        />
        <Tabs.TabPane
          tab={
            <span>
              <IconLayers style={{ marginRight: 6, verticalAlign: '-2px' }} />
              {ui.schemes}
            </span>
          }
          itemKey="schemes"
        />
        <Tabs.TabPane
          tab={
            <span>
              <IconPlay style={{ marginRight: 6, verticalAlign: '-2px' }} />
              {ui.run}
            </span>
          }
          itemKey="run"
        />
        <Tabs.TabPane
          tab={
            <span>
              <IconSetting style={{ marginRight: 6, verticalAlign: '-2px' }} />
              {ui.settings}
            </span>
          }
          itemKey="settings"
        />
      </Tabs>

      <div style={{ marginTop: 16 }}>
      {loading ? (
        <Spin
          tip="加载中..."
          style={{ width: '100%', minHeight: 220, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
        />
      ) : error ? (
        <Card bodyStyle={{ textAlign: 'center', padding: '40px 24px' }}>
          <Title heading={6} style={{ marginBottom: 8 }}>加载失败</Title>
          <Text type="danger" style={{ display: 'block', marginBottom: 16 }}>{error}</Text>
          <Button type="primary" icon={<IconRefresh />} onClick={() => void loadData()}>
            重试
          </Button>
        </Card>
      ) : (
        <>
          {/* 总览页 */}
          {activeTab === 'overview' && (
            <Row gutter={[16, 16]}>
              <Col xs={24} md={8}>
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => setActiveTab('agents')}
                  onKeyDown={e => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      setActiveTab('agents');
                    }
                  }}
                  style={cardClickStyle}
                >
                  <Card title="Agent 池" bodyStyle={{ paddingTop: 8 }}>
                    <Text strong style={{ fontSize: 22 }}>{totalAgents}</Text>
                    <Text type="tertiary" style={{ marginLeft: 8 }}>个 Agent</Text>
                    <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 8 }}>
                      {pools.length} 个池 · 点此进入管理
                    </Text>
                    <Divider margin="12px" />
                    <Space wrap>
                      {pools.flatMap(p => p.agents || []).slice(0, 6).map((a, i) => (
                        <Tag key={`${a.id}-${i}`}>{a.name}</Tag>
                      ))}
                      {totalAgents === 0 ? <Text type="tertiary" size="small">暂无 Agent，请先创建池</Text> : null}
                    </Space>
                  </Card>
                </div>
              </Col>
              <Col xs={24} md={8}>
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => setActiveTab('schemes')}
                  onKeyDown={e => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      setActiveTab('schemes');
                    }
                  }}
                  style={cardClickStyle}
                >
                  <Card title="协作方案" bodyStyle={{ paddingTop: 8 }}>
                    <Text strong style={{ fontSize: 22 }}>{schemes.length}</Text>
                    <Text type="tertiary" style={{ marginLeft: 8 }}>个方案</Text>
                    <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 8 }}>
                      点此进入方案列表
                    </Text>
                    <Divider margin="12px" />
                    <Space vertical align="start">
                      {schemes.slice(0, 5).map(s => (
                        <Space key={s.id}>
                          <Tag color="blue">{MODE_NAME_MAP[s.mode]}</Tag>
                          <span title={s.name} style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'inline-block', verticalAlign: 'bottom' }}>
                            {s.name}
                          </span>
                        </Space>
                      ))}
                      {schemes.length === 0 ? <Text type="tertiary" size="small">暂无方案</Text> : null}
                    </Space>
                  </Card>
                </div>
              </Col>
              <Col xs={24} md={8}>
                <Card title="协作模式速览" bodyStyle={{ paddingTop: 8 }}>
                  <Space vertical align="start" style={{ rowGap: 12 }}>
                    {(Object.keys(MODE_NAME_MAP) as CollaborationMode[]).map(mode => (
                      <div key={mode}>
                        <Tag color="cyan" style={{ marginBottom: 4 }}>{MODE_NAME_MAP[mode]}</Tag>
                        <Text type="tertiary" size="small" style={{ display: 'block', lineHeight: 1.5 }}>
                          {MODE_DESC_MAP[mode]}
                        </Text>
                      </div>
                    ))}
                  </Space>
                </Card>
              </Col>
            </Row>
          )}

          {/* Agent 管理页 */}
          {activeTab === 'agents' && (
            <AgentManager pools={pools} onPoolsChange={loadData} />
          )}

          {/* 方案管理页 */}
          {activeTab === 'schemes' && (
            <Card
              title="协作方案"
              headerExtraContent={
                <Button type="primary" theme="solid" icon={<IconPlus />} onClick={openCreateSchemeModal}>
                  新建方案
                </Button>
              }
            >
              <Table
                size="small"
                columns={schemeColumns}
                dataSource={schemes}
                rowKey="id"
                pagination={{ pageSize: 10, showSizeChanger: true }}
              />
            </Card>
          )}

          {/* 设置占位 */}
          {activeTab === 'settings' && (
            <Card
              style={{
                borderRadius: 14,
                boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)',
              }}
              title="设置"
            >
              <Text type="tertiary" style={{ lineHeight: 1.7 }}>
                语言切换已在顶栏生效（标签与常用文案）。默认模型、超时与 Trace 采样等将后续在此集中配置；Agent 与模型绑定请使用后台「托管 Agent」相关页面。
              </Text>
            </Card>
          )}

          {/* 运行页：三栏等高，内部各自滚动 */}
          {activeTab === 'run' && (
            <Row gutter={[16, 16]} style={{ alignItems: 'stretch', minHeight: runAreaMinH }}>
              <Col xs={24} xl={8} style={{ display: 'flex', flexDirection: 'column', minHeight: runAreaMinH }}>
                <Card
                  className="pg-graph-card"
                  style={{
                    flex: 1,
                    display: 'flex',
                    flexDirection: 'column',
                    minHeight: runAreaMinH,
                    borderRadius: 14,
                    overflow: 'hidden',
                    boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)',
                  }}
                  bodyStyle={{
                    flex: 1,
                    display: 'flex',
                    flexDirection: 'column',
                    padding: 0,
                    minHeight: 0,
                  }}
                >
                  <div style={{ padding: '16px 16px 8px' }}>
                    <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 8 }}>
                      计划
                    </Text>
                    <Select
                      placeholder={
                        schemes.length ? '请选择要运行的方案' : '暂无方案，请先到「协作编排」新建'
                      }
                      style={{ width: '100%' }}
                      value={selectedScheme?.id}
                      disabled={!schemes.length}
                      onChange={id => {
                        const s = schemes.find(x => x.id === id);
                        setSelectedScheme(s);
                      }}
                    >
                      {schemes.map(s => (
                        <Select.Option key={s.id} value={s.id}>
                          <Space>
                            <Badge dot type={s.enabled ? 'success' : 'danger'} />
                            <span>{s.name}</span>
                            <Tag size="small">{MODE_NAME_MAP[s.mode]}</Tag>
                          </Space>
                        </Select.Option>
                      ))}
                    </Select>
                    {selectedScheme ? (
                      <>
                        <Divider margin="12px" />
                        <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 8 }}>
                          {MODE_DESC_MAP[selectedScheme.mode]}
                        </Text>
                        <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 6 }}>
                          绑定 Agent
                        </Text>
                        <Space wrap>
                          {(selectedScheme.bindAgents || []).map((a, i) => (
                            <Tag key={i}>{a.role || a.agentId}</Tag>
                          ))}
                        </Space>
                      </>
                    ) : (
                      <Text type="tertiary" size="small" style={{ marginTop: 12, display: 'block' }}>
                        选择方案后将展示示意图并可在中间发起运行
                      </Text>
                    )}
                  </div>

                  <div
                    style={{
                      padding: '4px 16px 8px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      gap: 12,
                    }}
                  >
                    <Text strong style={{ fontSize: 14 }}>
                      Workflow Graph
                    </Text>
                    <Tag size="small" color="grey">
                      VISUAL
                    </Tag>
                  </div>

                  <div style={{ flex: 1, overflow: 'auto', padding: '0 16px 12px', minHeight: 200 }}>
                    {selectedScheme ? (
                      <WorkflowGraph
                        variant="embedded"
                        scheme={selectedScheme}
                        currentRun={currentRun}
                        liveEvents={events}
                      />
                    ) : (
                      <div
                        style={{
                          height: '100%',
                          minHeight: 160,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          borderRadius: 12,
                          border: `1px dashed var(--semi-color-border)`,
                          color: 'var(--semi-color-tertiary)',
                          fontSize: 13,
                        }}
                      >
                        选择左侧计划后展示拓扑
                      </div>
                    )}
                  </div>

                  <div
                    style={{
                      flexShrink: 0,
                      padding: '10px 16px',
                      fontFamily: 'var(--semi-font-family-monospace)',
                      fontSize: 12,
                      background: 'rgba(28, 31, 35, 0.94)',
                      color: 'rgba(232, 234, 237, 0.92)',
                      letterSpacing: 0.2,
                    }}
                  >
                    {graphFooterLine}
                  </div>
                </Card>
              </Col>

              <Col xs={24} xl={8} style={{ display: 'flex', flexDirection: 'column', minHeight: runAreaMinH }}>
                <RunConsole
                  scheme={selectedScheme}
                  onRun={input => selectedScheme && handleRunWorkflow(selectedScheme, input)}
                  onClear={() => {
                    setCurrentRun(undefined);
                    setEvents([]);
                  }}
                  running={running}
                  currentRun={currentRun}
                />
              </Col>

              <Col xs={24} xl={8} style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
                <TracePanel
                  events={events}
                  run={currentRun}
                  onRefresh={async () => {
                    if (currentRun?.runId) {
                      const res = await getRunEvents(currentRun.runId);
                      setEvents(res.events || []);
                    }
                  }}
                />
              </Col>
            </Row>
          )}
        </>
      )}
      </div>

      {/* Scheme 创建/编辑 Modal */}
      <Modal
        visible={showSchemeModal}
        onCancel={() => setShowSchemeModal(false)}
        onOk={handleSaveScheme}
        title={schemeModalMode === 'create' ? '新建协作方案' : '编辑协作方案'}
        okText="保存"
        maskClosable={false}
        width={560}
      >
        <Space vertical align="start" spacing="loose" style={{ width: '100%' }}>
          <div style={{ width: '100%' }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              方案名称 <Text type="danger">*</Text>
            </Text>
            <Input
              autoFocus
              style={{ width: '100%' }}
              value={schemeForm.name}
              onChange={(v) => setSchemeForm({ ...schemeForm, name: v })}
              placeholder="例如：需求评审流水线"
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>描述</Text>
            <Input
              style={{ width: '100%' }}
              value={schemeForm.description}
              onChange={(v) => setSchemeForm({ ...schemeForm, description: v })}
              placeholder="可选，便于区分多个方案"
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>协作模式</Text>
            <ModeSelector
              value={schemeForm.mode}
              onChange={(mode) => setSchemeForm({ ...schemeForm, mode })}
            />
          </div>
          <Space>
            <Switch
              checked={schemeForm.enableFinalizer}
              onChange={(checked) => setSchemeForm({ ...schemeForm, enableFinalizer: checked })}
            />
            <Text type="tertiary" size="small">启用 Finalizer（输出前由模型做最终整理）</Text>
          </Space>
        </Space>
      </Modal>
    </div>
  );
};
