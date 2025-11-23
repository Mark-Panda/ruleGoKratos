/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useMemo, useState } from 'react';

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

import { buildDocumentFromRuleChainJSON } from '../utils/rulechain-builder';
import { FlowDocumentJSON, FlowNodeJSON } from '../typings';
import { setRuleBaseInfo } from '../services/rule-base-info';
import { requestJSON } from '../services/http';
import { createRuleBase, getRuleDetail } from '../services/api-rules';
import { WorkflowNodeType } from '../nodes';
import { Editor } from '../editor';

export interface RuleDetailData {
  ruleChain: {
    id: string;
    name: string;
    debugMode?: boolean;
    root?: boolean;
    disabled?: boolean;
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
  React.useEffect(() => {
    if (initialTab) setActiveKey(initialTab);
  }, [initialTab]);
  const menuItems = useMemo(
    () => [
      { itemKey: 'workflow', text: '工作流设置' },
      { itemKey: 'design', text: '工作流设计' },
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
                  { itemKey: 'basic', text: '基础信息' },
                  { itemKey: 'vars', text: '变量' },
                  { itemKey: 'logs', text: '运行日志' },
                  { itemKey: 'integration', text: '工作流集成' },
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

                      <Typography.Text type="tertiary">描述</Typography.Text>
                      <Input value={desc} onChange={setDesc} placeholder="描述" />
                    </div>

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
                            const body = {
                              id,
                              name,
                              debugMode: !!debug,
                              root: !!root,
                              additionalInfo: { description: desc ?? '' },
                              configuration: { vars: {} },
                            };
                            await createRuleBase(id, body);
                            // 保存成功后刷新详情
                            const json = await getRuleDetail(id);
                            const rc = json?.ruleChain || {};
                            setName(String(rc?.name ?? name));
                            setDesc(String(rc?.additionalInfo?.description ?? desc ?? ''));
                            try {
                              setRuleBaseInfo(rc);
                            } catch {}
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
                        columns={[
                          {
                            title: '工作流名称',
                            dataIndex: 'ruleChain.name',
                            width: 200,
                          },
                          {
                            title: '规则链ID',
                            dataIndex: 'ruleChain.id',
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
                                    const rcjson = {
                                      ruleChain: r?.ruleChain,
                                      metadata: r?.metadata,
                                    } as any;
                                    const doc = buildDocumentFromRuleChainJSON(rcjson) as any;
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
                        rowKey={(r: any) => String(r?.id || r?.ruleChain?.id || Math.random())}
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
        width={1200}
      >
        <div style={{ height: '70vh' }}>
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
