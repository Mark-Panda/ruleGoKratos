/**
 * Agent Playground - 多智能体协作编排主页面
 */

import React, { useEffect, useState, useCallback, useRef, useMemo } from 'react';

import {
  Button,
  ButtonGroup,
  CheckboxGroup,
  Input,
  InputNumber,
  Select,
  TextArea,
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
  IconGallery,
  IconUserGroup,
  IconLayers,
  IconDesktop,
  IconSetting,
  IconArrowRight,
} from '@douyinfe/semi-icons';

function mergeTraceEvents(prev: TraceEvent[], incoming: TraceEvent): TraceEvent[] {
  const m = new Map(prev.map((e) => [e.id, e]));
  m.set(incoming.id, incoming);
  return [...m.values()].sort((a, b) => a.timestamp - b.timestamp);
}
import { buildRuntimeViewModel } from '../utils/runtime-view-model';
import {
  createRequestGuardSnapshot,
  getDisplayedRuntimeState,
  invalidateRequestGuards,
  isRequestGuardCurrent,
  shouldAutoRefreshRun,
} from '../utils/runtime-page-guards';
import { applyRecoveryActionAndRefresh } from '../utils/recovery-actions';
import { WorkflowGraph } from '../components/workflow-graph';
import { TracePanel } from '../components/trace-panel';
import { RunConsole, PreviousRunSnapshot } from '../components/run-console';
import { ModeSelector } from '../components/mode-selector';
import { AgentManager } from '../components/agent-manager';
import { RunWorkspacePanel } from '../components/run-workspace-panel';
import { getApiOrigin } from '../../services/http';
import {
  AgentDefinition,
  AgentPool,
  CollaborationScheme,
  CollaborationMode,
  SchemeConfig,
  TraceEvent,
  RuntimeRunDetail,
  RecoveryAction,
  MODE_NAME_MAP,
  MODE_DESC_MAP,
  buildSchemeBindAgents,
  createDefaultSchemeConfig,
  listSchemes,
  normalizeSchemeConfig,
  createScheme,
  resolveSchemeBindAgentsForSave,
  updateScheme,
  deleteScheme,
  runWorkflow,
  getRun,
  getRunEvents,
  applyRecoveryAction,
  listAgentPools,
} from '../../services/api-playground';

const { Text, Title } = Typography;

function splitCommaSeparated(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function joinCommaSeparated(values?: string[]): string {
  return values?.join(', ') || '';
}

function coerceInteger(value: number | string | undefined, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
}

function extractLatestRoundUserOnly(storedUserInput: string): string {
  const marker = '【本轮说明 / Bug / 追问】\n';
  const idx = storedUserInput.indexOf(marker);
  if (idx >= 0) {
    return storedUserInput.slice(idx + marker.length).trim();
  }
  return storedUserInput.trim();
}

function buildPlaygroundFollowUpUserPayload(
  snapshot: PreviousRunSnapshot,
  roundText: string
): string {
  const tail =
    snapshot.finalOutput.length > 4500
      ? `${snapshot.finalOutput.slice(-4500)}\n…（产出仅保留末尾 4500 字符）`
      : snapshot.finalOutput;
  return (
    `【Playground 关联上一轮运行】\n` +
    `上一轮 runId: ${snapshot.runId}\n` +
    `上一轮状态: ${snapshot.runStatus}\n` +
    `上一轮用户任务（末轮正文）:\n${snapshot.userInput || '（无）'}\n\n` +
    `上一轮产出摘录:\n${tail || '（无）'}\n\n` +
    `（说明：服务端每轮 run 使用独立 workspace 子目录 playground/run_<runId>/，新一轮默认无法读取上一轮目录中的文件；以下为你补充的追问。）\n\n` +
    `---\n【本轮说明 / Bug / 追问】\n${roundText}`
  );
}

type SchemeFormState = {
  name: string;
  description: string;
  mode: CollaborationMode;
  originalMode?: CollaborationMode;
  enableFinalizer: boolean;
  config: SchemeConfig;
};

function buildSchemeFormState(mode: CollaborationMode = 'router_expert'): SchemeFormState {
  return {
    name: '',
    description: '',
    mode,
    originalMode: undefined,
    enableFinalizer: false,
    config: createDefaultSchemeConfig(mode),
  };
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

const OVERVIEW_CARD_HOVER_STYLE: React.CSSProperties = {
  cursor: 'pointer',
  transition: 'box-shadow 0.25s ease, transform 0.15s ease',
};

export const AgentPlaygroundPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const [schemes, setSchemes] = useState<CollaborationScheme[]>([]);
  const [pools, setPools] = useState<AgentPool[]>([]);
  const defaultAgentPool = useMemo(
    () => pools.find((p) => p.id === 'default') ?? pools[0],
    [pools]
  );

  const poolAgentsSelectable = useMemo(() => {
    const list = defaultAgentPool?.agents || [];
    return list.filter((a) => a && a.enabled !== false);
  }, [defaultAgentPool]);

  const poolAgentIdOptions = useMemo(
    () =>
      poolAgentsSelectable.map((a) => ({
        value: a.id,
        label: `${a.name} (${a.id})`,
      })),
    [poolAgentsSelectable]
  );

  const [selectedScheme, setSelectedScheme] = useState<CollaborationScheme | undefined>();
  const [currentRunDetail, setCurrentRunDetail] = useState<RuntimeRunDetail | undefined>();
  const [currentRunSchemeId, setCurrentRunSchemeId] = useState<string | undefined>();
  const [events, setEvents] = useState<TraceEvent[]>([]);
  const [activeTab, setActiveTab] = useState<
    'overview' | 'agents' | 'schemes' | 'run' | 'settings'
  >('overview');
  const [lang, setLang] = useState<PlaygroundLang>('zh');
  const [running, setRunning] = useState(false);
  const [applyingRecoveryActionId, setApplyingRecoveryActionId] = useState<string | undefined>();
  const [previousRunSnapshot, setPreviousRunSnapshot] = useState<PreviousRunSnapshot | null>(null);
  const [attachPreviousRunContext, setAttachPreviousRunContext] = useState(true);

  const [showSchemeModal, setShowSchemeModal] = useState(false);
  const [schemeModalMode, setSchemeModalMode] = useState<'create' | 'edit'>('create');
  const [editingScheme, setEditingScheme] = useState<CollaborationScheme | undefined>();
  const [schemeForm, setSchemeForm] = useState<SchemeFormState>(() => buildSchemeFormState());

  const eventSourceRef = useRef<EventSource | null>(null);
  const sseFallbackPollRef = useRef<number | null>(null);
  const runDetailPollRef = useRef<number | null>(null);
  const activeRunIdRef = useRef<string | undefined>();
  const runDetailRequestVersionRef = useRef(0);
  const eventsRequestVersionRef = useRef(0);
  const completionToastRunIdRef = useRef<string | null>(null);

  const applyRunDetailIfCurrent = useCallback((runRes: RuntimeRunDetail, guardVersion: number) => {
    const snapshot = createRequestGuardSnapshot(runRes.run.runId, guardVersion);
    if (
      !isRequestGuardCurrent(snapshot, activeRunIdRef.current, runDetailRequestVersionRef.current)
    ) {
      return false;
    }
    setCurrentRunDetail(runRes);
    setCurrentRunSchemeId(runRes.run.schemeId);
    setRunning(shouldAutoRefreshRun(runRes.run.status));
    return true;
  }, []);

  const applyEventsIfCurrent = useCallback(
    (runId: string, nextEvents: TraceEvent[], guardVersion: number) => {
      const snapshot = createRequestGuardSnapshot(runId, guardVersion);
      if (
        !isRequestGuardCurrent(snapshot, activeRunIdRef.current, eventsRequestVersionRef.current)
      ) {
        return false;
      }
      setEvents(nextEvents);
      return true;
    },
    []
  );

  const issueRunDetailGuard = useCallback(() => {
    const nextVersion = runDetailRequestVersionRef.current + 1;
    runDetailRequestVersionRef.current = nextVersion;
    return nextVersion;
  }, []);

  const issueEventsGuard = useCallback(() => {
    const nextVersion = eventsRequestVersionRef.current + 1;
    eventsRequestVersionRef.current = nextVersion;
    return nextVersion;
  }, []);

  const invalidateActiveRunRequests = useCallback(() => {
    const invalidated = invalidateRequestGuards({
      runDetailVersion: runDetailRequestVersionRef.current,
      eventsVersion: eventsRequestVersionRef.current,
    });
    activeRunIdRef.current = invalidated.activeRunId;
    runDetailRequestVersionRef.current = invalidated.runDetailVersion;
    eventsRequestVersionRef.current = invalidated.eventsVersion;

    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (sseFallbackPollRef.current != null) {
      clearInterval(sseFallbackPollRef.current);
      sseFallbackPollRef.current = null;
    }
    if (runDetailPollRef.current != null) {
      clearInterval(runDetailPollRef.current);
      runDetailPollRef.current = null;
    }
  }, []);

  const displayedRuntimeState = useMemo(
    () =>
      getDisplayedRuntimeState({
        selectedSchemeId: selectedScheme?.id,
        currentRunSchemeId,
        currentRunDetail,
        events,
        running,
      }),
    [selectedScheme?.id, currentRunSchemeId, currentRunDetail, events, running]
  );

  const runtimeViewModel = useMemo(
    () =>
      buildRuntimeViewModel({
        run: displayedRuntimeState.currentRunDetail?.run,
        steps: displayedRuntimeState.currentRunDetail?.steps,
        artifacts: displayedRuntimeState.currentRunDetail?.artifacts,
        recoveryActions: displayedRuntimeState.currentRunDetail?.recoveryActions,
        events: displayedRuntimeState.events,
      }),
    [displayedRuntimeState]
  );

  const refreshRunState = useCallback(
    async (runId: string) => {
      const runDetailGuardVersion = issueRunDetailGuard();
      const eventsGuardVersion = issueEventsGuard();
      const [runRes, eventsRes] = await Promise.all([
        getRun(runId),
        getRunEvents(runId).catch(() => ({ events: [] as TraceEvent[] })),
      ]);
      applyRunDetailIfCurrent(runRes, runDetailGuardVersion);
      applyEventsIfCurrent(runId, eventsRes.events || [], eventsGuardVersion);
      return runRes;
    },
    [applyEventsIfCurrent, applyRunDetailIfCurrent, issueEventsGuard, issueRunDetailGuard]
  );

  useEffect(() => {
    if (!currentRunDetail?.run.runId) return;
    const { runId, status, finalOutput } = currentRunDetail.run;
    if (status !== 'completed' && status !== 'failed' && status !== 'waiting_recovery') return;
    if (completionToastRunIdRef.current === runId) return;
    completionToastRunIdRef.current = runId;

    if (status === 'completed') {
      const raw = (finalOutput || '').trim();
      const preview =
        raw.length > 0 ? [...raw].slice(0, 220).join('') + (raw.length > 220 ? '…' : '') : '';
      Toast.success({
        content:
          preview.length > 0
            ? `运行完成\n${preview}`
            : '运行完成。\n请在中间栏查看「最终结果」，或右侧 Trace 了解步骤详情。',
        duration: 6,
      });
    } else if (status === 'waiting_recovery') {
      Toast.warning({
        content:
          runtimeViewModel.run.failureSummary || runtimeViewModel.failedStep
            ? `运行进入恢复等待。\n失败步骤：${
                runtimeViewModel.failedStep?.name ||
                runtimeViewModel.failedStep?.stepId ||
                '未知步骤'
              }`
            : '运行进入恢复等待。\n请在中间栏和右侧 Recovery 视图查看建议动作。',
        duration: 6,
      });
    } else {
      Toast.error({
        content: '运行失败。\n请在右侧 Trace 查看错误相关事件。',
        duration: 6,
      });
    }
  }, [
    currentRunDetail?.run.runId,
    currentRunDetail?.run.status,
    currentRunDetail?.run.finalOutput,
    runtimeViewModel,
  ]);

  useEffect(() => {
    const d = currentRunDetail;
    if (!d?.run?.runId) {
      return;
    }
    const st = d.run.status;
    if (st !== 'completed' && st !== 'failed' && st !== 'waiting_recovery') {
      return;
    }
    setPreviousRunSnapshot({
      runId: d.run.runId,
      userInput: extractLatestRoundUserOnly(d.run.userInput || ''),
      finalOutput: (d.run.finalOutput || '').trim(),
      runStatus: st,
    });
  }, [currentRunDetail]);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const [poolsRes, schemesRes] = await Promise.all([listAgentPools(), listSchemes()]);
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
    if (!currentRunDetail?.run.runId || !shouldAutoRefreshRun(currentRunDetail.run.status)) {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (sseFallbackPollRef.current != null) {
        clearInterval(sseFallbackPollRef.current);
        sseFallbackPollRef.current = null;
      }
      if (runDetailPollRef.current != null) {
        clearInterval(runDetailPollRef.current);
        runDetailPollRef.current = null;
      }
      return undefined;
    }

    const runId = currentRunDetail.run.runId;

    const clearFallback = () => {
      if (sseFallbackPollRef.current != null) {
        clearInterval(sseFallbackPollRef.current);
        sseFallbackPollRef.current = null;
      }
    };

    const clearRunDetailPoll = () => {
      if (runDetailPollRef.current != null) {
        clearInterval(runDetailPollRef.current);
        runDetailPollRef.current = null;
      }
    };

    const refreshRunDetail = async () => {
      try {
        const guardVersion = issueRunDetailGuard();
        const runRes = await getRun(runId);
        const applied = applyRunDetailIfCurrent(runRes, guardVersion);
        if (applied && !shouldAutoRefreshRun(runRes.run.status)) {
          setRunning(false);
          clearFallback();
          clearRunDetailPoll();
        }
      } catch {
        /* ignore */
      }
    };

    clearRunDetailPoll();
    runDetailPollRef.current = window.setInterval(() => {
      void refreshRunDetail();
    }, 900) as unknown as number;

    const startFallbackPolling = () => {
      clearFallback();
      sseFallbackPollRef.current = window.setInterval(() => {
        void (async () => {
          try {
            const guardVersion = issueEventsGuard();
            const evRes = await getRunEvents(runId);
            applyEventsIfCurrent(runId, evRes.events || [], guardVersion);
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
    const streamUrl = `${getApiOrigin()}/api/v1/playground/run/${encodeURIComponent(
      runId
    )}/events/stream${token ? `?token=${encodeURIComponent(token)}` : ''}`;

    const es = new EventSource(streamUrl);
    eventSourceRef.current = es;

    es.onmessage = async (event) => {
      try {
        const raw = JSON.parse(event.data) as TraceEvent;
        if (activeRunIdRef.current !== runId) {
          return;
        }
        setEvents((prev) => mergeTraceEvents(prev, raw));
        if (raw.type === 'WORKFLOW_END') {
          es.close();
          eventSourceRef.current = null;
          clearFallback();
          clearRunDetailPoll();
          try {
            const guardVersion = issueRunDetailGuard();
            const runRes = await getRun(runId);
            applyRunDetailIfCurrent(runRes, guardVersion);
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
      clearRunDetailPoll();
    };
  }, [
    applyEventsIfCurrent,
    applyRunDetailIfCurrent,
    currentRunDetail?.run.runId,
    currentRunDetail?.run.status,
    issueEventsGuard,
    issueRunDetailGuard,
  ]);

  const schemeFormBindAgents = useMemo(() => {
    if (schemeModalMode === 'edit' && editingScheme?.bindAgents?.length) {
      return resolveSchemeBindAgentsForSave({
        mode: schemeForm.mode,
        originalMode: schemeForm.originalMode,
        existingBindAgents: editingScheme.bindAgents,
        pool: defaultAgentPool,
      });
    }
    return buildSchemeBindAgents(schemeForm.mode, defaultAgentPool);
  }, [editingScheme, defaultAgentPool, schemeForm.mode, schemeModalMode]);

  const schemeBindAgentHint = useMemo(() => {
    if (!schemeFormBindAgents.length) {
      return '暂无可参考的绑定 Agent，可直接填写 Agent ID。';
    }
    return `当前候选 Agent：${schemeFormBindAgents.map((agent) => agent.agentId).join(', ')}`;
  }, [schemeFormBindAgents]);

  const updateSchemeConfig = useCallback((updater: (config: SchemeConfig) => SchemeConfig) => {
    setSchemeForm((prev) => ({
      ...prev,
      config: updater(prev.config),
    }));
  }, []);

  const renderModeConfigEditor = () => {
    switch (schemeForm.mode) {
      case 'router_expert': {
        const routerConfig = schemeForm.config.modeConfig?.routerConfig;
        return (
          <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                兜底 Agent
              </Text>
              {poolAgentIdOptions.length > 0 ? (
                <Select
                  style={{ width: '100%' }}
                  showClear
                  filter
                  optionList={poolAgentIdOptions}
                  value={routerConfig?.fallbackAgent || undefined}
                  placeholder="从 default 池内选择"
                  onChange={(v) =>
                    updateSchemeConfig((config) => {
                      const normalized = normalizeSchemeConfig('router_expert', config);
                      const id = typeof v === 'string' ? v : '';
                      return {
                        ...normalized,
                        modeConfig: {
                          routerConfig: {
                            ...normalized.modeConfig?.routerConfig,
                            fallbackAgent: id,
                          },
                        },
                      };
                    })
                  }
                />
              ) : (
                <Input
                  style={{ width: '100%' }}
                  value={routerConfig?.fallbackAgent ?? ''}
                  onChange={(value) =>
                    updateSchemeConfig((config) => {
                      const normalized = normalizeSchemeConfig('router_expert', config);
                      return {
                        ...normalized,
                        modeConfig: {
                          routerConfig: {
                            ...normalized.modeConfig?.routerConfig,
                            fallbackAgent: value,
                          },
                        },
                      };
                    })
                  }
                  placeholder="暂无池数据，可直接输入 agentId，例如：engineer"
                />
              )}
              <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 6 }}>
                {schemeBindAgentHint}。单选下拉与池内 id 一致，避免手输错误。
              </Text>
            </div>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                路由提示词
              </Text>
              <TextArea
                rows={3}
                style={{ width: '100%' }}
                value={routerConfig?.routingPrompt ?? ''}
                onChange={(value: string) =>
                  updateSchemeConfig((config) => {
                    const normalized = normalizeSchemeConfig('router_expert', config);
                    return {
                      ...normalized,
                      modeConfig: {
                        routerConfig: {
                          ...normalized.modeConfig?.routerConfig,
                          routingPrompt: value,
                        },
                      },
                    };
                  })
                }
                placeholder="可选，用于补充路由时的偏好或判断依据"
              />
            </div>
          </Space>
        );
      }
      case 'plan_exec': {
        const planExecConfig = schemeForm.config.modeConfig?.planExecConfig;
        return (
          <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                规划 Agent
              </Text>
              {poolAgentIdOptions.length > 0 ? (
                <Select
                  style={{ width: '100%' }}
                  showClear
                  filter
                  optionList={poolAgentIdOptions}
                  value={planExecConfig?.plannerAgent || undefined}
                  placeholder="从 default 池内选择"
                  onChange={(v) =>
                    updateSchemeConfig((config) => {
                      const normalized = normalizeSchemeConfig('plan_exec', config);
                      const id = typeof v === 'string' ? v : '';
                      return {
                        ...normalized,
                        modeConfig: {
                          planExecConfig: {
                            ...normalized.modeConfig?.planExecConfig,
                            plannerAgent: id,
                          },
                        },
                      };
                    })
                  }
                />
              ) : (
                <Input
                  style={{ width: '100%' }}
                  value={planExecConfig?.plannerAgent ?? ''}
                  onChange={(value) =>
                    updateSchemeConfig((config) => {
                      const normalized = normalizeSchemeConfig('plan_exec', config);
                      return {
                        ...normalized,
                        modeConfig: {
                          planExecConfig: {
                            ...normalized.modeConfig?.planExecConfig,
                            plannerAgent: value,
                          },
                        },
                      };
                    })
                  }
                  placeholder="例如：planner"
                />
              )}
              <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 6 }}>
                {schemeBindAgentHint}
              </Text>
            </div>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                执行顺序
              </Text>
              {poolAgentIdOptions.length > 0 ? (
                <Select
                  multiple
                  style={{ width: '100%' }}
                  filter
                  maxTagCount={3}
                  showRestTagsPopover
                  optionList={poolAgentIdOptions}
                  value={planExecConfig?.executionOrder || []}
                  placeholder="多选：按点击选择顺序作为执行顺序"
                  onChange={(v) => {
                    const arr = Array.isArray(v) ? v.map((x) => String(x)).filter(Boolean) : [];
                    updateSchemeConfig((config) => {
                      const normalized = normalizeSchemeConfig('plan_exec', config);
                      return {
                        ...normalized,
                        modeConfig: {
                          planExecConfig: {
                            ...normalized.modeConfig?.planExecConfig,
                            executionOrder: arr,
                          },
                        },
                      };
                    });
                  }}
                />
              ) : null}
              <Input
                style={{ width: '100%', marginTop: poolAgentIdOptions.length > 0 ? 8 : 0 }}
                value={joinCommaSeparated(planExecConfig?.executionOrder)}
                onChange={(value) =>
                  updateSchemeConfig((config) => {
                    const normalized = normalizeSchemeConfig('plan_exec', config);
                    return {
                      ...normalized,
                      modeConfig: {
                        planExecConfig: {
                          ...normalized.modeConfig?.planExecConfig,
                          executionOrder: splitCommaSeparated(value),
                        },
                      },
                    };
                  })
                }
                placeholder={
                  poolAgentIdOptions.length > 0
                    ? '或手动输入逗号分隔顺序（与上方多选二选一即可）'
                    : '按逗号分隔，例如：designer, engineer'
                }
              />
            </div>
          </Space>
        );
      }
      case 'supervision': {
        const supervisionConfig = schemeForm.config.modeConfig?.supervisionConfig;
        return (
          <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                监督 Agent
              </Text>
              {poolAgentIdOptions.length > 0 ? (
                <Select
                  style={{ width: '100%' }}
                  showClear
                  filter
                  optionList={poolAgentIdOptions}
                  value={supervisionConfig?.supervisorAgent || undefined}
                  placeholder="从 default 池内选择"
                  onChange={(v) =>
                    updateSchemeConfig((config) => {
                      const normalized = normalizeSchemeConfig('supervision', config);
                      const id = typeof v === 'string' ? v : '';
                      return {
                        ...normalized,
                        modeConfig: {
                          supervisionConfig: {
                            ...normalized.modeConfig?.supervisionConfig,
                            supervisorAgent: id,
                          },
                        },
                      };
                    })
                  }
                />
              ) : (
                <Input
                  style={{ width: '100%' }}
                  value={supervisionConfig?.supervisorAgent ?? ''}
                  onChange={(value) =>
                    updateSchemeConfig((config) => {
                      const normalized = normalizeSchemeConfig('supervision', config);
                      return {
                        ...normalized,
                        modeConfig: {
                          supervisionConfig: {
                            ...normalized.modeConfig?.supervisionConfig,
                            supervisorAgent: value,
                          },
                        },
                      };
                    })
                  }
                  placeholder="例如：supervisor"
                />
              )}
              <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 6 }}>
                {schemeBindAgentHint}
              </Text>
            </div>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                Worker Agents
              </Text>
              {poolAgentsSelectable.length > 0 ? (
                <>
                  <CheckboxGroup
                    direction="vertical"
                    value={supervisionConfig?.workerAgents || []}
                    onChange={(list) => {
                      const ids = (Array.isArray(list) ? list : [])
                        .map((x) => String(x))
                        .filter(Boolean);
                      updateSchemeConfig((config) => {
                        const normalized = normalizeSchemeConfig('supervision', config);
                        return {
                          ...normalized,
                          modeConfig: {
                            supervisionConfig: {
                              ...normalized.modeConfig?.supervisionConfig,
                              workerAgents: ids,
                            },
                          },
                        };
                      });
                    }}
                    options={poolAgentsSelectable.map((a: AgentDefinition) => ({
                      value: a.id,
                      label: `${a.name} (${a.id})`,
                    }))}
                  />
                  <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 8 }}>
                    勾选参与并行协作的池内 Agent（unordered，与监督逻辑一致即可）。
                  </Text>
                </>
              ) : null}
              <Input
                style={{ width: '100%', marginTop: poolAgentsSelectable.length > 0 ? 10 : 0 }}
                value={joinCommaSeparated(supervisionConfig?.workerAgents)}
                onChange={(value) =>
                  updateSchemeConfig((config) => {
                    const normalized = normalizeSchemeConfig('supervision', config);
                    return {
                      ...normalized,
                      modeConfig: {
                        supervisionConfig: {
                          ...normalized.modeConfig?.supervisionConfig,
                          workerAgents: splitCommaSeparated(value),
                        },
                      },
                    };
                  })
                }
                placeholder={
                  poolAgentsSelectable.length > 0
                    ? '或手动输入逗号分隔（可与勾选同步）'
                    : '按逗号分隔，例如：designer, engineer'
                }
              />
            </div>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                检查间隔（秒）
              </Text>
              <InputNumber
                min={1}
                precision={0}
                style={{ width: '100%' }}
                value={supervisionConfig?.checkInterval ?? 15}
                onChange={(value) =>
                  updateSchemeConfig((config) => {
                    const normalized = normalizeSchemeConfig('supervision', config);
                    return {
                      ...normalized,
                      modeConfig: {
                        supervisionConfig: {
                          ...normalized.modeConfig?.supervisionConfig,
                          checkInterval: coerceInteger(
                            value,
                            normalized.modeConfig?.supervisionConfig?.checkInterval ?? 15
                          ),
                        },
                      },
                    };
                  })
                }
                placeholder="默认 15"
              />
            </div>
          </Space>
        );
      }
      case 'peer_handoff': {
        const peerHandoffConfig = schemeForm.config.modeConfig?.peerHandoffConfig;
        return (
          <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                入口 Agent
              </Text>
              {poolAgentIdOptions.length > 0 ? (
                <Select
                  style={{ width: '100%' }}
                  showClear
                  filter
                  optionList={poolAgentIdOptions}
                  value={peerHandoffConfig?.entryAgent || undefined}
                  placeholder="从 default 池内选择"
                  onChange={(v) =>
                    updateSchemeConfig((config) => {
                      const normalized = normalizeSchemeConfig('peer_handoff', config);
                      const id = typeof v === 'string' ? v : '';
                      return {
                        ...normalized,
                        modeConfig: {
                          peerHandoffConfig: {
                            ...normalized.modeConfig?.peerHandoffConfig,
                            entryAgent: id,
                          },
                        },
                      };
                    })
                  }
                />
              ) : (
                <Input
                  style={{ width: '100%' }}
                  value={peerHandoffConfig?.entryAgent ?? ''}
                  onChange={(value) =>
                    updateSchemeConfig((config) => {
                      const normalized = normalizeSchemeConfig('peer_handoff', config);
                      return {
                        ...normalized,
                        modeConfig: {
                          peerHandoffConfig: {
                            ...normalized.modeConfig?.peerHandoffConfig,
                            entryAgent: value,
                          },
                        },
                      };
                    })
                  }
                  placeholder="例如：designer"
                />
              )}
              <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 6 }}>
                {schemeBindAgentHint}
              </Text>
            </div>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                Mesh Agents
              </Text>
              {poolAgentsSelectable.length > 0 ? (
                <>
                  <CheckboxGroup
                    direction="vertical"
                    value={peerHandoffConfig?.meshAgents || []}
                    onChange={(list) => {
                      const ids = (Array.isArray(list) ? list : [])
                        .map((x) => String(x))
                        .filter(Boolean);
                      updateSchemeConfig((config) => {
                        const normalized = normalizeSchemeConfig('peer_handoff', config);
                        return {
                          ...normalized,
                          modeConfig: {
                            peerHandoffConfig: {
                              ...normalized.modeConfig?.peerHandoffConfig,
                              meshAgents: ids,
                            },
                          },
                        };
                      });
                    }}
                    options={poolAgentsSelectable.map((a: AgentDefinition) => ({
                      value: a.id,
                      label: `${a.name} (${a.id})`,
                    }))}
                  />
                  <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 8 }}>
                    勾选可参与交接的同伴；与方案绑定的候选范围一致。
                  </Text>
                </>
              ) : null}
              <Input
                style={{ width: '100%', marginTop: poolAgentsSelectable.length > 0 ? 10 : 0 }}
                value={joinCommaSeparated(peerHandoffConfig?.meshAgents)}
                onChange={(value) =>
                  updateSchemeConfig((config) => {
                    const normalized = normalizeSchemeConfig('peer_handoff', config);
                    return {
                      ...normalized,
                      modeConfig: {
                        peerHandoffConfig: {
                          ...normalized.modeConfig?.peerHandoffConfig,
                          meshAgents: splitCommaSeparated(value),
                        },
                      },
                    };
                  })
                }
                placeholder={
                  poolAgentsSelectable.length > 0
                    ? '或手动输入逗号分隔（可与勾选同步）'
                    : '按逗号分隔，例如：designer, pm, engineer'
                }
              />
            </div>
            <div style={{ width: '100%' }}>
              <Text strong style={{ display: 'block', marginBottom: 8 }}>
                交接规则
              </Text>
              <TextArea
                rows={3}
                style={{ width: '100%' }}
                value={peerHandoffConfig?.handoffRules ?? ''}
                onChange={(value: string) =>
                  updateSchemeConfig((config) => {
                    const normalized = normalizeSchemeConfig('peer_handoff', config);
                    return {
                      ...normalized,
                      modeConfig: {
                        peerHandoffConfig: {
                          ...normalized.modeConfig?.peerHandoffConfig,
                          handoffRules: value,
                        },
                      },
                    };
                  })
                }
                placeholder="描述交接条件、终止条件或优先级"
              />
            </div>
          </Space>
        );
      }
    }
  };

  const openCreateSchemeModal = () => {
    setEditingScheme(undefined);
    setSchemeForm(buildSchemeFormState());
    setSchemeModalMode('create');
    setShowSchemeModal(true);
  };

  const openEditSchemeModal = (scheme: CollaborationScheme) => {
    setEditingScheme(scheme);
    setSchemeForm({
      name: scheme.name,
      description: scheme.description,
      mode: scheme.mode,
      originalMode: scheme.mode,
      enableFinalizer: scheme.enableFinalizer,
      config: normalizeSchemeConfig(scheme.mode, scheme.config),
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
      const config = normalizeSchemeConfig(schemeForm.mode, schemeForm.config);
      if (schemeModalMode === 'create') {
        const pool = defaultAgentPool;
        const bindAgents = buildSchemeBindAgents(schemeForm.mode, pool);
        if (schemeForm.mode === 'plan_exec' && !bindAgents.some((b) => b.agentId === 'planner')) {
          Toast.warning(
            '当前 Agent 池中缺少「规划师」(planner)，规划执行将无法启动。请在「智能体」页确认默认池含规划师或使用协调人提供的完整池。'
          );
        }
        await createScheme({
          name: schemeForm.name,
          description: schemeForm.description,
          mode: schemeForm.mode,
          bindAgents,
          config,
          enableFinalizer: schemeForm.enableFinalizer,
        });
        Toast.success('方案创建成功');
      } else if (editingScheme) {
        const bindAgents = resolveSchemeBindAgentsForSave({
          mode: schemeForm.mode,
          originalMode: schemeForm.originalMode,
          existingBindAgents: editingScheme.bindAgents,
          pool: defaultAgentPool,
        });
        await updateScheme(editingScheme.id, {
          name: schemeForm.name,
          description: schemeForm.description,
          mode: schemeForm.mode,
          bindAgents,
          config,
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

  const handleRunWorkflow = async (scheme: CollaborationScheme, userInput: string) => {
    if (!userInput.trim()) {
      Toast.error('请输入任务描述');
      return;
    }
    let payload = userInput.trim();
    if (attachPreviousRunContext && previousRunSnapshot?.runId) {
      payload = buildPlaygroundFollowUpUserPayload(previousRunSnapshot, userInput.trim());
    }
    invalidateActiveRunRequests();
    setRunning(true);
    setCurrentRunDetail(undefined);
    setCurrentRunSchemeId(scheme.id);
    setEvents([]);
    setActiveTab('run');

    try {
      const res = await runWorkflow({ schemeId: scheme.id, userInput: payload });
      const runId = res.runId;
      activeRunIdRef.current = runId;
      await refreshRunState(runId);
    } catch (err) {
      Toast.error(`运行失败: ${err}`);
      setRunning(false);
      activeRunIdRef.current = undefined;
    }
  };

  const handleApplyRecovery = useCallback(
    async (action: RecoveryAction) => {
      const runId = runtimeViewModel.run.runId;
      if (!runId) {
        return;
      }
      setApplyingRecoveryActionId(action.id);
      setRunning(true);
      activeRunIdRef.current = runId;
      try {
        const detail = await applyRecoveryActionAndRefresh({
          runId,
          action,
          submit: applyRecoveryAction,
          refresh: refreshRunState,
        });
        if (detail.run.status === 'waiting_recovery') {
          Toast.warning('恢复动作已提交，运行状态正在刷新');
        } else {
          Toast.success('已开始执行恢复动作');
        }
      } catch (err) {
        Toast.error(`恢复失败: ${err}`);
        setRunning(false);
      } finally {
        setApplyingRecoveryActionId(undefined);
      }
    },
    [refreshRunState, runtimeViewModel.run.runId]
  );

  const schemeColumns = [
    {
      title: '序号',
      dataIndex: 'index',
      render: (_: any, __: any, index: number) => (
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
      title: '方案名称',
      dataIndex: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '协作模式',
      dataIndex: 'mode',
      render: (mode: CollaborationMode) => (
        <Tag color="blue" style={{ borderRadius: 6 }}>{MODE_NAME_MAP[mode] || mode}</Tag>
      ),
    },
    {
      title: '绑定Agent',
      dataIndex: 'bindAgents',
      render: (agents: any[]) => (
        <Space>
          {(agents || []).map((a, i) => (
            <Tag key={i} type="ghost" style={{ borderRadius: 6 }}>
              {a.role || a.agentId}
            </Tag>
          ))}
        </Space>
      ),
    },
    {
      title: 'Finalizer',
      dataIndex: 'enableFinalizer',
      render: (enabled: boolean) =>
        enabled ? (
          <Tag color="green" style={{ borderRadius: 6 }}>ON</Tag>
        ) : (
          <Tag color="grey" style={{ borderRadius: 6 }}>OFF</Tag>
        ),
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
            style={{ borderRadius: 6 }}
          >
            选择
          </Button>
          <Button
            size="small"
            icon={<IconEdit />}
            onClick={() => openEditSchemeModal(record)}
            style={{ borderRadius: 6 }}
          />
          <Button
            size="small"
            icon={<IconDelete />}
            onClick={() =>
              Modal.confirm({
                title: '确认删除',
                content: `确定要删除方案「${record.name}」吗？`,
                onOk: () => handleDeleteScheme(record),
              })
            }
            style={{ borderRadius: 6 }}
          />
        </Space>
      ),
      width: 200,
    },
  ];

  const totalAgents = defaultAgentPool?.agents?.length ?? 0;
  const enabledAgents = defaultAgentPool?.agents?.filter((a) => a.enabled !== false).length ?? 0;
  const ui = PLAYGROUND_UI[lang];

  const graphFooterLine = useMemo(() => {
    if (
      runtimeViewModel.run.status === 'running' ||
      runtimeViewModel.run.status === 'ready' ||
      runtimeViewModel.run.status === 'pending'
    ) {
      return `> ${runtimeViewModel.run.label} · 当前步骤 ${
        runtimeViewModel.activeStep?.name || '等待调度'
      }…`;
    }
    if (runtimeViewModel.run.status === 'completed') {
      return `> 已完成 · 产物 ${runtimeViewModel.artifacts.total} 个`;
    }
    if (runtimeViewModel.run.status === 'waiting_recovery') {
      return `> 等待恢复 · ${runtimeViewModel.failedStep?.name || '存在失败步骤'}`;
    }
    if (runtimeViewModel.run.status === 'failed') {
      return '> 运行失败';
    }
    if (selectedScheme) {
      return '> 等待交互…';
    }
    return '> 等待选择方案…';
  }, [runtimeViewModel, selectedScheme]);

  const runAreaMinH = 'min(72vh, 760px)';

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
        .pg-playground-shell .pg-overview-card:hover {
          transform: translateY(-2px);
          box-shadow: 0 6px 24px rgba(28, 31, 35, 0.10) !important;
        }
        .pg-playground-shell .pg-scheme-table .semi-table {
          border-radius: 10px;
          overflow: hidden;
        }
        @keyframes pg-ready-pulse {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.65; transform: scale(0.92); }
        }
        @keyframes pg-fade-in {
          from { opacity: 0; transform: translateY(8px); }
          to { opacity: 1; transform: translateY(0); }
        }
        .pg-playground-shell .pg-fade-in {
          animation: pg-fade-in 0.3s ease-out;
        }
      `}
      </style>

      {/* 顶栏 */}
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
          <div
            style={{
              width: 40,
              height: 40,
              borderRadius: 12,
              background: 'linear-gradient(135deg, var(--semi-color-primary), var(--semi-color-primary-light-default))',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#fff',
              fontSize: 20,
              boxShadow: '0 2px 8px rgba(22, 100, 255, 0.2)',
            }}
          >
            <IconDesktop />
          </div>
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
        onChange={(k) => setActiveTab(k as typeof activeTab)}
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
            style={{
              width: '100%',
              minHeight: 220,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          />
        ) : error ? (
          <Card
            bodyStyle={{ textAlign: 'center', padding: '40px 24px' }}
            style={{ borderRadius: 14, boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)' }}
          >
            <div
              style={{
                width: 56,
                height: 56,
                borderRadius: 16,
                background: 'var(--semi-color-danger-light-default)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                margin: '0 auto 12px',
                fontSize: 24,
              }}
            >
              ⚠️
            </div>
            <Title heading={6} style={{ marginBottom: 8 }}>
              加载失败
            </Title>
            <Text type="danger" style={{ display: 'block', marginBottom: 16 }}>
              {error}
            </Text>
            <Button type="primary" icon={<IconRefresh />} onClick={() => void loadData()}>
              重试
            </Button>
          </Card>
        ) : (
          <>
            {/* 总览页 */}
            {activeTab === 'overview' && (
              <div className="pg-fade-in">
                <Row gutter={[16, 16]}>
                  <Col xs={24} md={8}>
                    <div
                      className="pg-overview-card"
                      role="button"
                      tabIndex={0}
                      onClick={() => setActiveTab('agents')}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          setActiveTab('agents');
                        }
                      }}
                      style={OVERVIEW_CARD_HOVER_STYLE}
                    >
                      <Card
                        style={{ borderRadius: 14, boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)' }}
                        bodyStyle={{ paddingTop: 8 }}
                      >
                        <div
                          style={{
                            width: 44,
                            height: 44,
                            borderRadius: 12,
                            background: 'linear-gradient(135deg, rgba(22, 100, 255, 0.10), rgba(22, 100, 255, 0.04))',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            marginBottom: 16,
                            fontSize: 22,
                          }}
                        >
                          🤖
                        </div>
                        <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 4 }}>
                          <Text strong style={{ fontSize: 28, lineHeight: 1 }}>
                            {totalAgents}
                          </Text>
                          <Text type="tertiary" size="small">
                            个 Agent
                          </Text>
                        </div>
                        <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 12 }}>
                          已启用 {enabledAgents} 个 · default 池
                        </Text>
                        <Divider margin="12px" />
                        <Space wrap>
                          {(defaultAgentPool?.agents || []).slice(0, 6).map((a, i) => (
                            <Tag key={`${a.id}-${i}`} style={{ borderRadius: 6 }}>{a.name}</Tag>
                          ))}
                          {totalAgents === 0 ? (
                            <Text type="tertiary" size="small">
                              暂无 Agent，请在默认池中绑定托管配置
                            </Text>
                          ) : null}
                        </Space>
                        <div style={{ marginTop: 12, display: 'flex', alignItems: 'center', gap: 4 }}>
                          <Text type="primary" size="small" style={{ cursor: 'pointer' }}>
                            管理智能体
                          </Text>
                          <IconArrowRight size="small" style={{ color: 'var(--semi-color-primary)' }} />
                        </div>
                      </Card>
                    </div>
                  </Col>
                  <Col xs={24} md={8}>
                    <div
                      className="pg-overview-card"
                      role="button"
                      tabIndex={0}
                      onClick={() => setActiveTab('schemes')}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          setActiveTab('schemes');
                        }
                      }}
                      style={OVERVIEW_CARD_HOVER_STYLE}
                    >
                      <Card
                        style={{ borderRadius: 14, boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)' }}
                        bodyStyle={{ paddingTop: 8 }}
                      >
                        <div
                          style={{
                            width: 44,
                            height: 44,
                            borderRadius: 12,
                            background: 'linear-gradient(135deg, rgba(19, 194, 194, 0.10), rgba(19, 194, 194, 0.04))',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            marginBottom: 16,
                            fontSize: 22,
                          }}
                        >
                          📋
                        </div>
                        <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 4 }}>
                          <Text strong style={{ fontSize: 28, lineHeight: 1 }}>
                            {schemes.length}
                          </Text>
                          <Text type="tertiary" size="small">
                            个方案
                          </Text>
                        </div>
                        <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 12 }}>
                          已就绪协作编排
                        </Text>
                        <Divider margin="12px" />
                        <Space vertical align="start">
                          {schemes.slice(0, 5).map((s) => (
                            <Space key={s.id}>
                              <Tag color="blue" style={{ borderRadius: 6 }}>{MODE_NAME_MAP[s.mode]}</Tag>
                              <span
                                title={s.name}
                                style={{
                                  maxWidth: 200,
                                  overflow: 'hidden',
                                  textOverflow: 'ellipsis',
                                  whiteSpace: 'nowrap',
                                  display: 'inline-block',
                                  verticalAlign: 'bottom',
                                }}
                              >
                                {s.name}
                              </span>
                            </Space>
                          ))}
                          {schemes.length === 0 ? (
                            <Text type="tertiary" size="small">
                              暂无方案
                            </Text>
                          ) : null}
                        </Space>
                        <div style={{ marginTop: 12, display: 'flex', alignItems: 'center', gap: 4 }}>
                          <Text type="primary" size="small" style={{ cursor: 'pointer' }}>
                            管理方案
                          </Text>
                          <IconArrowRight size="small" style={{ color: 'var(--semi-color-primary)' }} />
                        </div>
                      </Card>
                    </div>
                  </Col>
                  <Col xs={24} md={8}>
                    <Card
                      style={{ borderRadius: 14, boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)' }}
                      bodyStyle={{ paddingTop: 8 }}
                    >
                      <div
                        style={{
                          width: 44,
                          height: 44,
                          borderRadius: 12,
                          background: 'linear-gradient(135deg, rgba(250, 173, 20, 0.10), rgba(250, 173, 20, 0.04))',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          marginBottom: 16,
                          fontSize: 22,
                        }}
                      >
                        🧩
                      </div>
                      <Text strong style={{ display: 'block', marginBottom: 12, fontSize: 14 }}>
                        协作模式速览
                      </Text>
                      <Space vertical align="start" style={{ rowGap: 10, width: '100%' }}>
                        {(Object.keys(MODE_NAME_MAP) as CollaborationMode[]).map((mode) => (
                          <div key={mode} style={{ width: '100%' }}>
                            <Tag color="cyan" style={{ marginBottom: 2, borderRadius: 6 }}>
                              {MODE_NAME_MAP[mode]}
                            </Tag>
                            <Text
                              type="tertiary"
                              size="small"
                              style={{ display: 'block', lineHeight: 1.5, fontSize: 11 }}
                            >
                              {MODE_DESC_MAP[mode]}
                            </Text>
                          </div>
                        ))}
                      </Space>
                    </Card>
                  </Col>
                </Row>

                {/* Quick Actions */}
                <Card
                  style={{
                    marginTop: 16,
                    borderRadius: 14,
                    boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)',
                  }}
                  bodyStyle={{ padding: '16px 20px' }}
                >
                  <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 12, fontWeight: 500 }}>
                    快捷操作
                  </Text>
                  <Space wrap spacing="loose">
                    <Button
                      type="primary"
                      theme="solid"
                      icon={<IconPlus />}
                      onClick={openCreateSchemeModal}
                      style={{ borderRadius: 8 }}
                    >
                      新建方案
                    </Button>
                    <Button
                      icon={<IconPlay />}
                      onClick={() => setActiveTab('run')}
                      style={{ borderRadius: 8 }}
                    >
                      进入运行
                    </Button>
                    <Button
                      icon={<IconUserGroup />}
                      onClick={() => setActiveTab('agents')}
                      style={{ borderRadius: 8 }}
                    >
                      管理 Agent
                    </Button>
                  </Space>
                </Card>
              </div>
            )}

            {/* Agent 管理页 */}
            {activeTab === 'agents' && (
              <div className="pg-fade-in">
                <AgentManager pools={pools} onPoolsChange={loadData} />
              </div>
            )}

            {/* 方案管理页 */}
            {activeTab === 'schemes' && (
              <div className="pg-fade-in">
                <Card
                  title={
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                      <div
                        style={{
                          width: 32,
                          height: 32,
                          borderRadius: 8,
                          background: 'linear-gradient(135deg, rgba(19, 194, 194, 0.10), rgba(19, 194, 194, 0.04))',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: 16,
                        }}
                      >
                        📋
                      </div>
                      <Text strong style={{ fontSize: 14 }}>协作方案</Text>
                    </div>
                  }
                  headerExtraContent={
                    <Button
                      type="primary"
                      theme="solid"
                      icon={<IconPlus />}
                      onClick={openCreateSchemeModal}
                      style={{ borderRadius: 8 }}
                    >
                      新建方案
                    </Button>
                  }
                  style={{ borderRadius: 14, boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)' }}
                >
                  <Table
                    className="pg-scheme-table"
                    size="small"
                    columns={schemeColumns}
                    dataSource={schemes}
                    rowKey="id"
                    pagination={{ pageSize: 10, showSizeChanger: true }}
                  />
                </Card>
              </div>
            )}

            {/* 设置页 */}
            {activeTab === 'settings' && (
              <div className="pg-fade-in">
                <Row gutter={[16, 16]}>
                  <Col xs={24} md={12}>
                    <Card
                      style={{ borderRadius: 14, boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)' }}
                      title={
                        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                          <div
                            style={{
                              width: 32,
                              height: 32,
                              borderRadius: 8,
                              background: 'var(--semi-color-fill-0)',
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                              fontSize: 16,
                            }}
                          >
                            🌐
                          </div>
                          <Text strong style={{ fontSize: 14 }}>显示设置</Text>
                        </div>
                      }
                    >
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
                        <div>
                          <Text
                            type="tertiary"
                            size="small"
                            style={{ display: 'block', marginBottom: 8, fontWeight: 500 }}
                          >
                            界面语言
                          </Text>
                          <ButtonGroup>
                            <Button
                              type={lang === 'zh' ? 'primary' : 'tertiary'}
                              onClick={() => setLang('zh')}
                              style={{ borderRadius: '8px 0 0 8px' }}
                            >
                              中文
                            </Button>
                            <Button
                              type={lang === 'en' ? 'primary' : 'tertiary'}
                              onClick={() => setLang('en')}
                              style={{ borderRadius: '0 8px 8px 0' }}
                            >
                              English
                            </Button>
                          </ButtonGroup>
                          <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 6 }}>
                            控制标签页名称及常用文案的语言
                          </Text>
                        </div>
                      </div>
                    </Card>
                  </Col>
                  <Col xs={24} md={12}>
                    <Card
                      style={{ borderRadius: 14, boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)' }}
                      title={
                        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                          <div
                            style={{
                              width: 32,
                              height: 32,
                              borderRadius: 8,
                              background: 'var(--semi-color-fill-0)',
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                              fontSize: 16,
                            }}
                          >
                            🔧
                          </div>
                          <Text strong style={{ fontSize: 14 }}>运行设置</Text>
                        </div>
                      }
                    >
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
                        <div>
                          <Text type="tertiary" size="small" style={{ display: 'block', lineHeight: 1.65 }}>
                            默认模型、超时与 Trace 采样等将后续在此集中配置；
                            Agent 与模型绑定请使用后台「托管 Agent」相关页面。
                          </Text>
                        </div>
                        <div
                          style={{
                            padding: '12px 16px',
                            background: 'var(--semi-color-fill-0)',
                            borderRadius: 10,
                            border: '1px dashed var(--semi-color-border)',
                            textAlign: 'center',
                          }}
                        >
                          <Text type="tertiary" size="small">
                            更多运行配置即将推出…
                          </Text>
                        </div>
                      </div>
                    </Card>
                  </Col>
                </Row>
              </div>
            )}

            {/* 运行页 */}
            {activeTab === 'run' && (
              <div className="pg-fade-in">
                <Row gutter={[16, 16]} style={{ alignItems: 'stretch', height: runAreaMinH }}>
                  <Col
                    xs={24}
                    xl={8}
                    style={{ display: 'flex', flexDirection: 'column', height: '100%' }}
                  >
                    <Card
                      className="pg-graph-card"
                      style={{
                        flex: 1,
                        display: 'flex',
                        flexDirection: 'column',
                        height: '100%',
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
                        overflow: 'auto',
                      }}
                    >
                      <div style={{ padding: '16px 16px 8px' }}>
                        <Text
                          type="tertiary"
                          size="small"
                          style={{ display: 'block', marginBottom: 8, fontWeight: 500 }}
                        >
                          计划
                        </Text>
                        <Select
                          placeholder={
                            schemes.length ? '请选择要运行的方案' : '暂无方案，请先到「协作编排」新建'
                          }
                          style={{ width: '100%' }}
                          value={selectedScheme?.id}
                          disabled={!schemes.length}
                          onChange={(id) => {
                            const s = schemes.find((x) => x.id === id);
                            setSelectedScheme(s);
                          }}
                        >
                          {schemes.map((s) => (
                            <Select.Option key={s.id} value={s.id}>
                              <Space>
                                <Badge dot type={s.enabled ? 'success' : 'danger'} />
                                <span>{s.name}</span>
                                <Tag size="small" style={{ borderRadius: 4 }}>{MODE_NAME_MAP[s.mode]}</Tag>
                              </Space>
                            </Select.Option>
                          ))}
                        </Select>
                        {selectedScheme ? (
                          <>
                            <Divider margin="12px" />
                            <Text
                              type="tertiary"
                              size="small"
                              style={{ display: 'block', marginBottom: 8 }}
                            >
                              {MODE_DESC_MAP[selectedScheme.mode]}
                            </Text>
                            <Text
                              type="tertiary"
                              size="small"
                              style={{ display: 'block', marginBottom: 6 }}
                            >
                              绑定 Agent
                            </Text>
                            <Space wrap>
                              {(selectedScheme.bindAgents || []).map((a, i) => (
                                <Tag key={i} style={{ borderRadius: 6 }}>{a.role || a.agentId}</Tag>
                              ))}
                            </Space>
                          </>
                        ) : (
                          <Text
                            type="tertiary"
                            size="small"
                            style={{ marginTop: 12, display: 'block' }}
                          >
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
                          Execution Plan
                        </Text>
                        <Tag size="small" color="grey">
                          RUNTIME
                        </Tag>
                      </div>

                      <div
                        style={{ flex: 1, overflow: 'auto', padding: '0 16px 12px', minHeight: 200 }}
                      >
                        {selectedScheme ? (
                          <WorkflowGraph
                            variant="embedded"
                            scheme={selectedScheme}
                            runtimeViewModel={runtimeViewModel}
                          />
                        ) : (
                          <div
                            style={{
                              height: '100%',
                              minHeight: 160,
                              display: 'flex',
                              flexDirection: 'column',
                              alignItems: 'center',
                              justifyContent: 'center',
                              borderRadius: 12,
                              border: `1px dashed var(--semi-color-border)`,
                              color: 'var(--semi-color-tertiary)',
                              fontSize: 13,
                              gap: 8,
                              background: 'var(--semi-color-bg-0)',
                            }}
                          >
                            <div
                              style={{
                                width: 48,
                                height: 48,
                                borderRadius: 14,
                                background: 'var(--semi-color-fill-0)',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                fontSize: 20,
                              }}
                            >
                              📊
                            </div>
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
                          borderRadius: '0 0 14px 14px',
                        }}
                      >
                        {graphFooterLine}
                      </div>
                    </Card>
                  </Col>

                  <Col
                    xs={24}
                    xl={8}
                    style={{ display: 'flex', flexDirection: 'column', height: '100%' }}
                  >
                    <RunConsole
                      scheme={selectedScheme}
                      onRun={(input) => selectedScheme && handleRunWorkflow(selectedScheme, input)}
                      onClear={() => {
                        invalidateActiveRunRequests();
                        setCurrentRunDetail(undefined);
                        setCurrentRunSchemeId(undefined);
                        setEvents([]);
                        setRunning(false);
                      }}
                      onApplyRecovery={handleApplyRecovery}
                      applyingRecoveryActionId={applyingRecoveryActionId}
                      running={displayedRuntimeState.running}
                      runtimeViewModel={runtimeViewModel}
                      attachPreviousRunContext={attachPreviousRunContext}
                      onAttachPreviousRunContextChange={setAttachPreviousRunContext}
                      previousRunSnapshot={previousRunSnapshot}
                    />
                  </Col>

                  <Col
                    xs={24}
                    xl={8}
                    style={{ display: 'flex', flexDirection: 'column', height: '100%' }}
                  >
                    <TracePanel
                      events={displayedRuntimeState.events}
                      runtimeViewModel={runtimeViewModel}
                      onRefresh={async () => {
                        if (runtimeViewModel.run.runId) {
                          await refreshRunState(runtimeViewModel.run.runId);
                        }
                      }}
                      onApplyRecovery={handleApplyRecovery}
                      applyingRecoveryActionId={applyingRecoveryActionId}
                    />
                  </Col>
                </Row>

                {/* 工作区文件面板 - 运行完成后展示 */}
                {currentRunDetail?.run?.status === 'completed' &&
                  currentRunDetail.run.workspacePath && (
                    <RunWorkspacePanel
                      runId={currentRunDetail.run.runId}
                      workspacePath={currentRunDetail.run.workspacePath}
                    />
                  )}
              </div>
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
        width={640}
      >
        <Space vertical align="start" spacing="loose" style={{ width: '100%' }}>
          <div style={{ width: '100%' }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              方案名称 <Text type="danger">*</Text>
            </Text>
            <Input
              autoFocus
              style={{ width: '100%', borderRadius: 8 }}
              value={schemeForm.name}
              onChange={(v) => setSchemeForm({ ...schemeForm, name: v })}
              placeholder="例如：需求评审流水线"
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              描述
            </Text>
            <Input
              style={{ width: '100%', borderRadius: 8 }}
              value={schemeForm.description}
              onChange={(v) => setSchemeForm({ ...schemeForm, description: v })}
              placeholder="可选，便于区分多个方案"
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              协作模式
            </Text>
            <ModeSelector
              value={schemeForm.mode}
              onChange={(mode) =>
                setSchemeForm((prev) => ({
                  ...prev,
                  mode,
                  config: normalizeSchemeConfig(mode, prev.config),
                }))
              }
            />
          </div>
          <Divider margin="4px" />
          <div style={{ width: '100%' }}>
            <Text strong style={{ display: 'block', marginBottom: 12 }}>
              基础配置
            </Text>
            <Row gutter={[12, 12]}>
              <Col span={8}>
                <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 8 }}>
                  Max Iterations
                </Text>
                <InputNumber
                  min={1}
                  precision={0}
                  style={{ width: '100%', borderRadius: 8 }}
                  value={schemeForm.config.maxIterations}
                  onChange={(value) =>
                    updateSchemeConfig((config) => ({
                      ...config,
                      maxIterations: coerceInteger(value, config.maxIterations),
                    }))
                  }
                />
              </Col>
              <Col span={8}>
                <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 8 }}>
                  Max Tool Calls
                </Text>
                <InputNumber
                  min={1}
                  precision={0}
                  style={{ width: '100%', borderRadius: 8 }}
                  value={schemeForm.config.maxToolCalls}
                  onChange={(value) =>
                    updateSchemeConfig((config) => ({
                      ...config,
                      maxToolCalls: coerceInteger(value, config.maxToolCalls),
                    }))
                  }
                />
              </Col>
              <Col span={8}>
                <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 8 }}>
                  Timeout Seconds
                </Text>
                <InputNumber
                  min={1}
                  precision={0}
                  style={{ width: '100%', borderRadius: 8 }}
                  value={schemeForm.config.timeoutSeconds}
                  onChange={(value) =>
                    updateSchemeConfig((config) => ({
                      ...config,
                      timeoutSeconds: coerceInteger(value, config.timeoutSeconds),
                    }))
                  }
                />
              </Col>
            </Row>
            <div style={{ width: '100%', marginTop: 12 }}>
              <Text type="tertiary" size="small" style={{ display: 'block', marginBottom: 8 }}>
                Finalizer Prompt
              </Text>
              <TextArea
                rows={3}
                style={{ width: '100%', borderRadius: 8 }}
                value={schemeForm.config.finalizerPrompt ?? ''}
                onChange={(value: string) =>
                  updateSchemeConfig((config) => ({
                    ...config,
                    finalizerPrompt: value,
                  }))
                }
                placeholder="可选，启用 Finalizer 时用于约束最终整理风格"
              />
            </div>
          </div>
          <Divider margin="4px" />
          <div style={{ width: '100%' }}>
            <Text strong style={{ display: 'block', marginBottom: 12 }}>
              {MODE_NAME_MAP[schemeForm.mode]} 专属配置
            </Text>
            {renderModeConfigEditor()}
          </div>
          <Space>
            <Switch
              checked={schemeForm.enableFinalizer}
              onChange={(checked) => setSchemeForm({ ...schemeForm, enableFinalizer: checked })}
            />
            <Text type="tertiary" size="small">
              启用 Finalizer（输出前由模型做最终整理）
            </Text>
          </Space>
        </Space>
      </Modal>
    </div>
  );
};
