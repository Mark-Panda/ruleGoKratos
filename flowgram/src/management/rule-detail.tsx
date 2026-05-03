/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useMemo, useState } from 'react';

import { DataAll, GraphicDesign, Histogram, SettingConfig } from '@icon-park/react';
import {
  Button,
  Input,
  Nav,
  Switch,
  Typography,
  Toast,
  Tag,
  DatePicker,
  Table,
  Pagination,
  Spin,
  Modal,
} from '@douyinfe/semi-ui';

import { runLogChainDisplay, runLogTableRowKey } from '../utils/run-log-display';
import { buildDocumentFromRuleChainJSON } from '../utils/rulechain-builder';
import { emptyRuleChainParamsJson } from '../utils/rule-chain-request-params';
import {
  buildRuleChainConfigurationWithFlowgram,
  parseRuleChainFlowgramFromConfiguration,
} from '../utils/rule-chain-flowgram-dsl';
import { FlowDocumentJSON, FlowNodeJSON } from '../typings';
import { setRuleBaseInfo } from '../services/rule-base-info';
import { requestJSON } from '../services/http';
import { createRuleBase, getRuleDetail } from '../services/api-rules';
import { WorkflowNodeType } from '../nodes';
import { Editor } from '../editor';
import { RULE_CHAIN_DEPLOY_STATUS_EVENT } from '../constants/deploy-status-event';
import { RuleChainSkillAction } from '../components/rule-chain-skill-action';
import { RuleChainRequestParamsEditor } from '../components/rule-chain-request-params-editor';

const menuIconProps = {
  theme: 'outline' as const,
  size: 14,
  strokeWidth: 2.2,
};

export interface RuleDetailData {
  ruleChain: {
    id: string;
    name: string;
    debugMode?: boolean;
    root?: boolean;
    disabled?: boolean;
    /** 规则链 configuration（含 flowgram.io 入参/出参说明等），与 RuleGo DSL 中 ruleChain.configuration 对齐 */
    configuration?: Record<string, unknown>;
    additionalInfo?: {
      description?: string;
      createTime?: string;
      updateTime?: string;
      username?: string;
    };
  };
  metadata?: {
    firstNodeIndex?: number;
    nodes?: any[] | null;
    connections?: any[] | null;
    // 若后端已保存 FreeLayout 编辑器的原始文档，则优先使用该字段进行渲染
    flowgramUI?: FlowDocumentJSON | any;
  };
}

export const RuleDetail: React.FC<{
  data: RuleDetailData;
  onBack: () => void;
  initialTab?: 'workflow' | 'design';
}> = ({ data, onBack, initialTab }) => {
  const [name, setName] = useState<string>(data?.ruleChain?.name ?? '');
  const [desc, setDesc] = useState<string>(data?.ruleChain?.additionalInfo?.description ?? '');
  const [debug, setDebug] = useState<boolean>(!!data?.ruleChain?.debugMode);
  const [root] = useState<boolean>(!!data?.ruleChain?.root);
  const [activeKey, setActiveKey] = useState<string>(initialTab ?? 'workflow');
  const [saving, setSaving] = useState<boolean>(false);
  const [deployed, setDeployed] = useState<boolean>(() => !Boolean(data?.ruleChain?.disabled));

  const [configurationSnapshot, setConfigurationSnapshot] = useState<Record<string, unknown>>(
    () => {
      const cfg = (data?.ruleChain as { configuration?: Record<string, unknown> })?.configuration;
      return cfg && typeof cfg === 'object' ? { ...cfg } : {};
    }
  );
  const [requestMetadataParamsJson, setRequestMetadataParamsJson] = useState<string>(
    emptyRuleChainParamsJson()
  );
  const [requestMessageBodyParamsJson, setRequestMessageBodyParamsJson] = useState<string>(
    emptyRuleChainParamsJson()
  );
  const [responseMessageBodyParamsJson, setResponseMessageBodyParamsJson] = useState<string>(
    emptyRuleChainParamsJson()
  );
  const [flowgramEditorJson, setFlowgramEditorJson] = useState<string>('');
  const [flowgramSkillDirName, setFlowgramSkillDirName] = useState<string>('');
  const [skillRefreshVersion, setSkillRefreshVersion] = useState(0);

  const configurationSyncKey = useMemo(() => {
    const id = data?.ruleChain?.id ?? '';
    const cfg = (data?.ruleChain as { configuration?: unknown })?.configuration;
    return `${id}|${JSON.stringify(cfg ?? null)}`;
  }, [data?.ruleChain?.id, (data?.ruleChain as { configuration?: unknown })?.configuration]);

  React.useEffect(() => {
    const cfg = ((data?.ruleChain as any)?.configuration || {}) as Record<string, unknown>;
    setConfigurationSnapshot({ ...cfg });
    const fg = parseRuleChainFlowgramFromConfiguration(cfg);
    setRequestMetadataParamsJson(fg.requestMetadataParamsJson);
    setRequestMessageBodyParamsJson(fg.requestMessageBodyParamsJson);
    setResponseMessageBodyParamsJson(fg.responseMessageBodyParamsJson);
    setFlowgramEditorJson(fg.editorJson);
    setFlowgramSkillDirName(fg.skillDirName);
  }, [configurationSyncKey]);

  React.useEffect(() => {
    if (initialTab) setActiveKey(initialTab);
  }, [initialTab]);
  React.useEffect(() => {
    setDeployed(!Boolean(data?.ruleChain?.disabled));
  }, [data?.ruleChain?.id, data?.ruleChain?.disabled]);
  const menuItems = useMemo(
    () => [
      { itemKey: 'workflow', text: '工作流设置', icon: <SettingConfig {...menuIconProps} /> },
      { itemKey: 'design', text: '工作流设计', icon: <GraphicDesign {...menuIconProps} /> },
    ],
    []
  );

  // 将接口返回的 metadata 转换为 FlowDocumentJSON：
  // 渲染逻辑与“导入 JSON”保持一致：优先使用 flowgramUI，其次将 RuleChain JSON 转换为编辑器文档
  const convertMetadataToDoc = (md?: RuleDetailData['metadata']): FlowDocumentJSON | undefined => {
    if (md && Array.isArray(md.nodes) && Array.isArray(md.connections)) {
      const rc = { ruleChain: data.ruleChain, metadata: md } as any;
      return buildDocumentFromRuleChainJSON(rc) as any;
    }
    const startNode: FlowNodeJSON = {
      id: String(data?.ruleChain?.id ?? 'start_' + Math.random().toString(36).slice(2, 8)),
      type: WorkflowNodeType.Start,
      meta: { position: { x: 180, y: 180 } },
      data: { title: data?.ruleChain?.name ?? 'Start' },
    } as any;
    return { nodes: [startNode], edges: [] };
  };

  const designDoc: FlowDocumentJSON | undefined = convertMetadataToDoc(data?.metadata);
  // 左侧子菜单选中状态（基础信息/变量/运行日志/工作流集成）
  const [subKey, setSubKey] = useState<string>('basic');
  const [timeRange, setTimeRange] = useState<[Date | null, Date | null]>([null, null]);
  const [runs, setRuns] = useState<any[]>([]);
  const [page, setPage] = useState<number>(1);
  const [size, setSize] = useState<number>(10);
  const [total, setTotal] = useState<number>(0);
  const [loadingRuns, setLoadingRuns] = useState<boolean>(false);
  const [viewerOpen, setViewerOpen] = useState<boolean>(false);
  const [viewerDoc, setViewerDoc] = useState<FlowDocumentJSON | undefined>();
  const [viewerLogs, setViewerLogs] = useState<{ list: any[]; startTs?: number; endTs?: number }>();

  const formatDateTime = (d?: Date | null): string => {
    if (!d) return '';
    const pad = (n: number) => (n < 10 ? `0${n}` : String(n));
    const y = d.getFullYear();
    const m = pad(d.getMonth() + 1);
    const dd = pad(d.getDate());
    const hh = pad(d.getHours());
    const mm = pad(d.getMinutes());
    const ss = pad(d.getSeconds());
    return `${y}-${m}-${dd} ${hh}:${mm}:${ss}`;
  };

  const toStartOfDay = (d?: Date | null): Date | null => {
    if (!d) return null;
    const x = new Date(d);
    x.setHours(0, 0, 0, 0);
    return x;
  };
  const toEndOfDay = (d?: Date | null): Date | null => {
    if (!d) return null;
    const x = new Date(d);
    x.setHours(23, 59, 59, 999);
    return x;
  };

  const refreshDeployStatus = React.useCallback(async () => {
    const id = String(data?.ruleChain?.id ?? '');
    if (!id) return;
    const latest = await getRuleDetail(id);
    const latestRule = latest?.ruleChain;
    if (!latestRule || typeof latestRule !== 'object') return;
    setRuleBaseInfo(latestRule as any);
    setDeployed(!Boolean((latestRule as any)?.disabled));
  }, [data?.ruleChain?.id]);

  const fetchRuns = async (p?: number, s?: number) => {
    const current = typeof p === 'number' ? p : page;
    const pageSize = typeof s === 'number' ? s : size;
    const params: Record<string, any> = {
      size: pageSize,
      page: current,
      current: current,
      chainId: String(data?.ruleChain?.id ?? ''),
    };
    const startTime = formatDateTime(timeRange?.[0] || null);
    const endTime = formatDateTime(timeRange?.[1] || null);
    if (startTime) params.startTime = startTime;
    if (endTime) params.endTime = endTime;
    try {
      setLoadingRuns(true);
      const data = await requestJSON<{
        items: any[];
        total?: number;
        size?: number;
        page?: number;
      }>('/logs/runs', { params });
      setRuns(Array.isArray((data as any)?.items) ? (data as any).items : []);
      setTotal(Number((data as any)?.total ?? 0));
      setPage(Number((data as any)?.page ?? current));
      setSize(Number((data as any)?.size ?? pageSize));
      setLoadingRuns(false);
    } catch (e) {
      setLoadingRuns(false);
      Toast.error({ content: String((e as Error)?.message ?? e) });
    }
  };

  React.useEffect(() => {
    if (subKey === 'logs') {
      fetchRuns(1, size);
    }
  }, [subKey]);

  React.useEffect(() => {
    const id = String(data?.ruleChain?.id ?? '');
    if (!id) return;
    const handler = (evt: Event) => {
      const customEvt = evt as CustomEvent<{ id?: string; deployed?: boolean }>;
      const eventId = String(customEvt?.detail?.id ?? '');
      if (eventId !== id) return;
      setDeployed(Boolean(customEvt?.detail?.deployed));
    };
    window.addEventListener(RULE_CHAIN_DEPLOY_STATUS_EVENT, handler as EventListener);
    return () => {
      window.removeEventListener(RULE_CHAIN_DEPLOY_STATUS_EVENT, handler as EventListener);
    };
  }, [data?.ruleChain?.id]);

  React.useEffect(() => {
    if (activeKey !== 'design') return;
    refreshDeployStatus().catch(() => {});
    const timer = window.setInterval(() => {
      refreshDeployStatus().catch(() => {});
    }, 5000);
    return () => window.clearInterval(timer);
  }, [activeKey, refreshDeployStatus]);

  return (
    <div
      style={{ display: 'flex', flexDirection: 'column', height: '100%', background: '#F7F8FA' }}
    >
      <div
        style={{
          borderBottom: '1px solid rgba(6,7,9,0.06)',
          padding: '8px 12px',
          display: 'grid',
          gridTemplateColumns: '1fr auto 1fr',
          alignItems: 'center',
          background: '#fff',
          boxShadow: '0 1px 6px rgba(6,7,9,0.06)',
          position: 'sticky',
          top: 0,
          zIndex: 99,
          minHeight: '56px',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <Button onClick={onBack} type="secondary">
            退出
          </Button>
          <Nav
            mode="horizontal"
            items={menuItems}
            selectedKeys={[activeKey]}
            onSelect={(d) => {
              const key = String(d.itemKey);
              setActiveKey(key);
              const id = String(data?.ruleChain?.id ?? '');
              if (!id) return;
              if (key === 'design')
                window.location.hash = `#/workflow/${encodeURIComponent(id)}/design`;
              if (key === 'workflow') window.location.hash = `#/workflow/${encodeURIComponent(id)}`;
            }}
            style={{ marginLeft: 8 }}
          />
        </div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 8,
            margin: 0,
            maxWidth: 560,
            justifySelf: 'center',
          }}
        >
          <Typography.Title
            heading={5}
            style={{
              margin: 0,
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {name || data?.ruleChain?.id}
          </Typography.Title>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '2px 10px',
              borderRadius: 999,
              background: deployed ? 'rgba(82,196,26,0.10)' : 'rgba(245,34,45,0.10)',
              border: deployed
                ? '1px solid rgba(82,196,26,0.35)'
                : '1px solid rgba(245,34,45,0.35)',
            }}
          >
            <span
              className="workflow-deploy-glow"
              style={{
                background: deployed ? '#52c41a' : '#ff4d4f',
                boxShadow: deployed
                  ? '0 0 8px rgba(82,196,26,0.95), 0 0 14px rgba(82,196,26,0.75)'
                  : '0 0 8px rgba(255,77,79,0.95), 0 0 14px rgba(255,77,79,0.75)',
              }}
            />
            <Typography.Text
              size="small"
              style={{ color: deployed ? '#389e0d' : '#cf1322', fontWeight: 600 }}
            >
              {deployed ? '已部署' : '未部署'}
            </Typography.Text>
          </div>
          <Tag size="small" color={root ? 'green' : 'grey'}>
            {root ? '根规则链' : '子规则链'}
          </Tag>
        </div>
        {/* 保存、测试、导出按钮通过 Portal 渲染到这里 */}
        <div
          id="top-toolbar-portal-container"
          style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 8 }}
        ></div>
      </div>
      {activeKey === 'design' ? (
        <div style={{ height: '100%', display: 'flex' }}>
          <Editor initialDoc={designDoc} showTopToolbar={true} />
        </div>
      ) : (
        <div style={{ padding: 16 }}>
          <div
            style={{
              display: 'flex',
              gap: 16,
              alignItems: 'flex-start',
              maxWidth: 1200,
              margin: '0 auto',
            }}
          >
            {/* 左侧垂直菜单 */}
            <div style={{ width: 240 }}>
              <Nav
                mode="vertical"
                items={[
                  {
                    itemKey: 'basic',
                    text: '基础信息',
                    icon: <SettingConfig {...menuIconProps} />,
                  },
                  { itemKey: 'vars', text: '变量', icon: <DataAll {...menuIconProps} /> },
                  { itemKey: 'logs', text: '运行日志', icon: <Histogram {...menuIconProps} /> },
                  {
                    itemKey: 'integration',
                    text: '工作流集成',
                    icon: <GraphicDesign {...menuIconProps} />,
                  },
                ]}
                selectedKeys={[subKey]}
                onSelect={(d) => setSubKey(String(d.itemKey))}
              />
            </div>
            {/* 右侧卡片内容 */}
            <div style={{ flex: 1 }}>
              <div
                style={{
                  background: '#fff',
                  border: '1px solid rgba(6,7,9,0.06)',
                  boxShadow: '0 1px 6px rgba(6,7,9,0.06)',
                  borderRadius: 12,
                  padding: 16,
                }}
              >
                {subKey === 'basic' && (
                  <>
                    <div
                      style={{
                        display: 'grid',
                        gridTemplateColumns: '200px 1fr',
                        gap: 12,
                        alignItems: 'center',
                      }}
                    >
                      <Typography.Text type="tertiary">ID</Typography.Text>
                      <Typography.Text>{data?.ruleChain?.id}</Typography.Text>

                      <Typography.Text type="tertiary">工作流名称</Typography.Text>
                      <Input value={name} onChange={setName} placeholder="请输入工作流名称" />

                      <Typography.Text type="tertiary">调试模式</Typography.Text>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Switch checked={debug} onChange={(v) => setDebug(!!v)} />
                        <Typography.Text type="tertiary">
                          开启后会显著增加系统负载，并将节点执行时输出日志
                        </Typography.Text>
                      </div>

                      <Typography.Text type="tertiary">根链</Typography.Text>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Switch checked={root} disabled />
                        <Typography.Text type="tertiary">当前为根规则链</Typography.Text>
                      </div>

                      <Typography.Text type="tertiary">技能</Typography.Text>
                      <RuleChainSkillAction
                        ruleChainId={String(data?.ruleChain?.id ?? '')}
                        isRoot={root}
                        showStatusText
                        refreshToken={skillRefreshVersion}
                        onGenerated={async () => {
                          try {
                            const json = await getRuleDetail(String(data?.ruleChain?.id ?? ''));
                            const rc = json?.ruleChain || {};
                            const cfg = ((rc as any)?.configuration || {}) as Record<
                              string,
                              unknown
                            >;
                            setConfigurationSnapshot({ ...cfg });
                            const fg = parseRuleChainFlowgramFromConfiguration(cfg);
                            setFlowgramSkillDirName(fg.skillDirName);
                            setSkillRefreshVersion((value) => value + 1);
                          } catch {
                            /* ignore */
                          }
                        }}
                      />

                      <Typography.Text type="tertiary">描述</Typography.Text>
                      <Input value={desc} onChange={setDesc} placeholder="描述" />
                    </div>

                    <Typography.Title heading={6} style={{ marginTop: 20, marginBottom: 8 }}>
                      规则链入参 / 出参
                    </Typography.Title>
                    <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 12 }}>
                      入参/出参说明写入{' '}
                      <Typography.Text code>ruleChain.configuration.flowgram.io</Typography.Text>
                      ，保存后持久化到数据库 configuration 字段。
                    </Typography.Paragraph>
                    <RuleChainRequestParamsEditor
                      title="请求 — 元数据（metadata）"
                      value={requestMetadataParamsJson}
                      onChange={setRequestMetadataParamsJson}
                    />
                    <RuleChainRequestParamsEditor
                      title="请求 — 消息体（data）"
                      value={requestMessageBodyParamsJson}
                      onChange={setRequestMessageBodyParamsJson}
                    />
                    <RuleChainRequestParamsEditor
                      title="响应 — 消息体（输出 data 结构说明）"
                      value={responseMessageBodyParamsJson}
                      onChange={setResponseMessageBodyParamsJson}
                    />

                    <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end' }}>
                      <Button
                        theme="solid"
                        type="primary"
                        loading={saving}
                        onClick={async () => {
                          const id = String(data?.ruleChain?.id ?? '');
                          if (!id) {
                            Toast.error({ content: '缺少规则链ID，无法保存' });
                            return;
                          }
                          try {
                            setSaving(true);
                            const mergedConfiguration = buildRuleChainConfigurationWithFlowgram(
                              configurationSnapshot,
                              {
                                description: String(desc ?? '').trim(),
                                requestMetadataParamsJson,
                                requestMessageBodyParamsJson,
                                responseMessageBodyParamsJson,
                                editorScratchJson: flowgramEditorJson,
                                skillDirName: flowgramSkillDirName,
                              }
                            );
                            const body = {
                              id,
                              name,
                              debugMode: !!debug,
                              root: !!root,
                              additionalInfo: { description: desc ?? '' },
                              configuration: mergedConfiguration,
                            };
                            await createRuleBase(id, body);
                            // 保存成功后刷新详情
                            const json = await getRuleDetail(id);
                            const rc = json?.ruleChain || {};
                            setName(String(rc?.name ?? name));
                            setDesc(String(rc?.additionalInfo?.description ?? desc ?? ''));
                            const cfg = ((rc as any)?.configuration || {}) as Record<
                              string,
                              unknown
                            >;
                            setConfigurationSnapshot({ ...cfg });
                            const fg = parseRuleChainFlowgramFromConfiguration(cfg);
                            setRequestMetadataParamsJson(fg.requestMetadataParamsJson);
                            setRequestMessageBodyParamsJson(fg.requestMessageBodyParamsJson);
                            setResponseMessageBodyParamsJson(fg.responseMessageBodyParamsJson);
                            setFlowgramEditorJson(fg.editorJson);
                            setFlowgramSkillDirName(fg.skillDirName);
                            try {
                              setRuleBaseInfo(rc);
                            } catch {}
                            setSkillRefreshVersion((value) => value + 1);
                            setSaving(false);
                            Toast.success({ content: '保存成功并已刷新' });
                          } catch (e) {
                            setSaving(false);
                            Toast.error({ content: String((e as Error)?.message ?? e) });
                          }
                        }}
                      >
                        保存
                      </Button>
                    </div>
                  </>
                )}
                {subKey === 'vars' && (
                  <Typography.Text type="tertiary">变量配置功能待接入</Typography.Text>
                )}
                {subKey === 'logs' && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                    <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
                      <DatePicker
                        type="dateTime"
                        value={timeRange[0] as any}
                        placeholder="开始时间"
                        onChange={(v: any) => {
                          const nv = v ? toStartOfDay(v as Date) : null;
                          setTimeRange([nv, timeRange[1]]);
                        }}
                      />
                      <DatePicker
                        type="dateTime"
                        value={timeRange[1] as any}
                        placeholder="结束时间"
                        onChange={(v: any) => {
                          const nv = v ? toEndOfDay(v as Date) : null;
                          setTimeRange([timeRange[0], nv]);
                        }}
                      />
                      <Button
                        theme="solid"
                        type="primary"
                        onClick={() => {
                          setPage(1);
                          fetchRuns(1, size);
                        }}
                      >
                        查询
                      </Button>
                      <Button
                        type="tertiary"
                        onClick={() => {
                          setTimeRange([null, null]);
                          setPage(1);
                          fetchRuns(1, size);
                        }}
                      >
                        重置
                      </Button>
                    </div>
                    <Spin spinning={loadingRuns}>
                      <Table
                        dataSource={runs}
                        rowKey={runLogTableRowKey}
                        columns={[
                          {
                            title: '工作流名称',
                            render: (_: unknown, r: any) => runLogChainDisplay(r).name || '—',
                            width: 200,
                          },
                          {
                            title: '规则链ID',
                            render: (_: unknown, r: any) => runLogChainDisplay(r).id || '—',
                            width: 160,
                          },
                          {
                            title: '开始时间',
                            render: (_, r: any) => {
                              const ts = Number(r?.startTs ?? 0);
                              return ts ? new Date(ts).toLocaleString() : '';
                            },
                            width: 180,
                          },
                          {
                            title: '结束时间',
                            render: (_, r: any) => {
                              const ts = Number(r?.endTs ?? 0);
                              return ts ? new Date(ts).toLocaleString() : '';
                            },
                            width: 180,
                          },
                          {
                            title: '耗时(ms)',
                            render: (_, r: any) => {
                              const s = Number(r?.startTs ?? 0);
                              const e = Number(r?.endTs ?? 0);
                              return s && e ? e - s : '';
                            },
                            width: 120,
                          },
                          {
                            title: '状态',
                            render: (_, r: any) => {
                              const hasErr = Array.isArray(r?.logs)
                                ? r.logs.some((l: any) => String(l?.err || '').length > 0)
                                : false;
                              return (
                                <Tag size="small" color={hasErr ? 'red' : 'green'}>
                                  {hasErr ? '失败' : '成功'}
                                </Tag>
                              );
                            },
                            width: 100,
                          },
                          {
                            title: '操作',
                            render: (_, r: any) => (
                              <Button
                                size="small"
                                onClick={() => {
                                  try {
                                    const dslRoot = r?.ruleChain;
                                    const doc = buildDocumentFromRuleChainJSON(
                                      dslRoot && typeof dslRoot === 'object'
                                        ? dslRoot
                                        : ({ ruleChain: {}, metadata: {} } as any)
                                    ) as any;
                                    setViewerDoc(doc);
                                    const logs = Array.isArray(r?.logs) ? r.logs : [];
                                    setViewerLogs({
                                      list: logs,
                                      startTs: r?.startTs,
                                      endTs: r?.endTs,
                                    });
                                    setViewerOpen(true);
                                  } catch (e) {
                                    Toast.error({ content: String((e as Error)?.message ?? e) });
                                  }
                                }}
                              >
                                查看
                              </Button>
                            ),
                            width: 120,
                          },
                        ]}
                        pagination={false}
                      />
                    </Spin>
                    <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                      <Pagination
                        total={total}
                        pageSize={size}
                        currentPage={page}
                        onPageChange={(p) => {
                          setPage(p);
                          fetchRuns(p, size);
                        }}
                        onPageSizeChange={(ps) => {
                          setSize(ps);
                          setPage(1);
                          fetchRuns(1, ps);
                        }}
                        pageSizeOpts={[10, 20, 50]}
                      />
                    </div>
                  </div>
                )}
                {subKey === 'integration' && (
                  <Typography.Text type="tertiary">工作流集成功能待接入</Typography.Text>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
      <Modal
        visible={viewerOpen}
        title="运行日志查看"
        onCancel={() => setViewerOpen(false)}
        footer={null}
        width="98vw"
        centered
        style={{ maxWidth: '100vw' }}
        bodyStyle={{
          height: 'calc(100vh - 96px)',
          maxHeight: 'calc(100vh - 96px)',
          padding: 8,
          overflow: 'hidden',
          boxSizing: 'border-box',
        }}
      >
        <div style={{ height: '100%', width: '100%', minHeight: 0 }}>
          <Editor
            initialDoc={viewerDoc}
            showTopToolbar={true}
            readonly={true}
            initialLogs={viewerLogs}
            openRunPanel={false}
          />
        </div>
      </Modal>
    </div>
  );
};
