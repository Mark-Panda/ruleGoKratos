/**
 * 工作流引擎 · 全局执行日志：列出所有规则链的运行记录。
 */

import React, { useEffect, useState } from 'react';

import {
  Button,
  DatePicker,
  Typography,
  Tag,
  Toast,
  Table,
  Pagination,
  Spin,
  Modal,
} from '@douyinfe/semi-ui';

import { FlowDocumentJSON } from '../../typings';
import { requestJSON } from '../../services/http';
import { buildDocumentFromRuleChainJSON } from '../../utils/rulechain-builder';
import { Editor } from '../../editor';
import { runLogChainDisplay, runLogTableRowKey } from '../../utils/run-log-display';

export const WorkflowRunLogsSection: React.FC = () => {
  const [timeRange, setTimeRange] = useState<[Date | null, Date | null]>([null, null]);
  const [runs, setRuns] = useState<any[]>([]);
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [viewerOpen, setViewerOpen] = useState(false);
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

  const fetchRuns = async (
    p?: number,
    s?: number,
    range?: [Date | null, Date | null] | null
  ) => {
    const current = typeof p === 'number' ? p : page;
    const pageSize = typeof s === 'number' ? s : size;
    const tr = range !== undefined ? range : timeRange;
    const params: Record<string, string | number | boolean | undefined> = {
      size: pageSize,
      page: current,
    };
    const startTime = formatDateTime(tr?.[0] || null);
    const endTime = formatDateTime(tr?.[1] || null);
    if (startTime) params.startTime = startTime;
    if (endTime) params.endTime = endTime;
    try {
      setLoading(true);
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
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRuns(1, 10);
  }, []);

  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        background: '#F7F8FA',
        overflow: 'hidden',
        padding: 16,
        minHeight: 0,
      }}
    >
      <div style={{ background: '#fff', padding: 24, borderRadius: 2, marginBottom: 16 }}>
        <Typography.Title heading={6} style={{ margin: '0 0 12px' }}>
          执行日志
        </Typography.Title>
        <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 16 }}>
          展示所有规则链的运行记录（全站），可按时间范围筛选；与单工作流详情内「运行日志」接口相同，此处不做规则链过滤。
        </Typography.Paragraph>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, alignItems: 'center' }}>
          <Typography.Text size="small" type="secondary">
            开始
          </Typography.Text>
          <DatePicker
            type="dateTime"
            density="compact"
            value={timeRange[0] ?? undefined}
            onChange={(v: any) => {
              const nv = v ? toStartOfDay(v as Date) : null;
              setTimeRange([nv, timeRange[1]]);
            }}
          />
          <Typography.Text size="small" type="secondary">
            结束
          </Typography.Text>
          <DatePicker
            type="dateTime"
            density="compact"
            value={timeRange[1] ?? undefined}
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
              const cleared: [Date | null, Date | null] = [null, null];
              setTimeRange(cleared);
              setPage(1);
              void fetchRuns(1, size, cleared);
            }}
          >
            重置
          </Button>
        </div>
      </div>

      <div
        style={{
          flex: 1,
          background: '#fff',
          padding: 16,
          borderRadius: 2,
          display: 'flex',
          flexDirection: 'column',
          minHeight: 0,
        }}
      >
        <Spin spinning={loading}>
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
                render: (_: unknown, r: any) => {
                  const ts = Number(r?.startTs ?? 0);
                  return ts ? new Date(ts).toLocaleString() : '';
                },
                width: 180,
              },
              {
                title: '结束时间',
                render: (_: unknown, r: any) => {
                  const ts = Number(r?.endTs ?? 0);
                  return ts ? new Date(ts).toLocaleString() : '';
                },
                width: 180,
              },
              {
                title: '耗时(ms)',
                render: (_: unknown, r: any) => {
                  const s = Number(r?.startTs ?? 0);
                  const e = Number(r?.endTs ?? 0);
                  return s && e ? e - s : '';
                },
                width: 120,
              },
              {
                title: '状态',
                render: (_: unknown, r: any) => {
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
                render: (_: unknown, r: any) => {
                  const chainId = runLogChainDisplay(r).id;
                  return (
                    <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                      {chainId ? (
                        <Button
                          size="small"
                          type="tertiary"
                          onClick={() => {
                            window.location.hash = `#/workflow/${encodeURIComponent(chainId)}`;
                          }}
                        >
                          打开工作流
                        </Button>
                      ) : null}
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
                        查看日志
                      </Button>
                    </div>
                  );
                },
                width: 220,
              },
            ]}
            pagination={false}
          />
        </Spin>
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16 }}>
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

export default WorkflowRunLogsSection;
