/**
 * 工作流执行：选择已启用的根规则链，按规则链配置的请求参数填充表单，异步触发执行并轮询日志。
 */

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
  Button,
  Collapse,
  Empty,
  Input,
  Select,
  Table,
  Typography,
  Toast,
  Spin,
  Tag,
  TextArea,
} from '@douyinfe/semi-ui';

import {
  buildCanvasNodeMapsFromRuleDetail,
  formatNodeCellDisplay,
  parseRunLogPayload,
  runLogChainDisplay,
  runLogRowsFromPayload,
  summarizeRuleMsgLike,
  summarizeRunLogErr,
  truncateText,
  type ParsedRunLogPayload,
} from '../../utils/run-log-display';
import {
  buildRuleChainParamsPreviewValue,
  parseRuleChainParamsJson,
} from '../../utils/rule-chain-request-params';
import { parseRuleChainFlowgramFromConfiguration } from '../../utils/rule-chain-flowgram-dsl';
import {
  DEFAULT_INFERRED_NOTIFY_MSG_TYPE,
  inferMsgTypeFromRuleDetail,
} from '../../utils/infer-rule-chain-msg-type';
import { alphaNanoid } from '../../utils';
import { executeTestRun, fetchRunLogsDetailed } from '../../services/test-run-http';
import { getRuleList, getRuleDetail } from '../../services/api-rules';
import { JsonValueEditor } from '../../components/testrun/json-value-editor';

/** 将请求元数据参数预览对象转为 query 字符串（与画布侧测试运行 metadata 语义一致）。 */
function metadataPreviewToQueryString(obj: Record<string, unknown>): string {
  const parts: string[] = [];
  const walk = (prefix: string, val: unknown) => {
    if (val === undefined || val === null) return;
    if (val !== null && typeof val === 'object' && !Array.isArray(val)) {
      for (const [k, v] of Object.entries(val as Record<string, unknown>)) {
        walk(prefix ? `${prefix}.${k}` : k, v);
      }
      return;
    }
    const encVal = typeof val === 'object' ? JSON.stringify(val) : String(val);
    parts.push(`${encodeURIComponent(prefix)}=${encodeURIComponent(encVal)}`);
  };
  walk('', obj);
  return parts.join('&');
}

const POLL_MS = 1000;
const POLL_MAX_MS = 30 * 60 * 1000;

function formatTs(ms: number): string {
  if (!ms) return '—';
  try {
    return new Date(ms).toLocaleString();
  } catch {
    return String(ms);
  }
}

function safeJson(v: unknown): string {
  try {
    return JSON.stringify(v, null, 2);
  } catch {
    return String(v);
  }
}

export const WorkflowExecuteSection: React.FC = () => {
  const [chains, setChains] = useState<any[]>([]);
  const [loadingList, setLoadingList] = useState(false);
  const [selectedId, setSelectedId] = useState<string | undefined>();
  const [detailLoading, setDetailLoading] = useState(false);

  const [msgType, setMsgType] = useState('');
  const [metadataStr, setMetadataStr] = useState('');
  const [headers, setHeaders] = useState<Record<string, unknown>>({
    'Content-Type': 'application/json',
  });
  const [body, setBody] = useState<Record<string, unknown>>({});

  const [running, setRunning] = useState(false);
  const [activeMsgId, setActiveMsgId] = useState<string | null>(null);
  /** 最近一次成功拉取或解析后的展示数据；新一轮执行前清空 */
  const [logBundle, setLogBundle] = useState<{
    parsed: ParsedRunLogPayload;
    raw: unknown;
  } | null>(null);
  /** 轮询中最近一次 HTTP 异常说明（404=记录尚未写入） */
  const [pollHint, setPollHint] = useState<string | null>(null);
  const pollStartRef = useRef<number>(0);
  const loadedDetailRef = useRef<any>(null);
  /** 与画布节点标题映射同源（getRuleDetail 结果） */
  const [ruleDetailSnapshot, setRuleDetailSnapshot] = useState<any>(null);

  const loadChains = useCallback(async () => {
    setLoadingList(true);
    try {
      const data = await getRuleList({
        page: 1,
        size: 500,
        root: true,
        disabled: false,
      });
      const items = Array.isArray(data.items) ? data.items : [];
      setChains(items);
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
      setChains([]);
    } finally {
      setLoadingList(false);
    }
  }, []);

  useEffect(() => {
    void loadChains();
  }, [loadChains]);

  const applyDetailToForm = useCallback((detail: any) => {
    loadedDetailRef.current = detail;
    setRuleDetailSnapshot(detail);
    const cfg = detail?.ruleChain?.configuration;
    const fg = parseRuleChainFlowgramFromConfiguration(cfg);
    const metaNodes = parseRuleChainParamsJson(fg.requestMetadataParamsJson);
    const bodyNodes = parseRuleChainParamsJson(fg.requestMessageBodyParamsJson);
    const metaPreview = buildRuleChainParamsPreviewValue(metaNodes);
    const bodyPreview = buildRuleChainParamsPreviewValue(bodyNodes);
    setMetadataStr(metadataPreviewToQueryString(metaPreview));
    setBody(bodyPreview);
    setHeaders({ 'Content-Type': 'application/json' });
    setMsgType(inferMsgTypeFromRuleDetail(detail, fg.entryMsgType));
  }, []);

  useEffect(() => {
    if (!selectedId) {
      loadedDetailRef.current = null;
      setRuleDetailSnapshot(null);
      return;
    }
    setRuleDetailSnapshot(null);
    let cancelled = false;
    (async () => {
      setDetailLoading(true);
      try {
        const detail = await getRuleDetail(selectedId);
        if (cancelled) return;
        applyDetailToForm(detail);
      } catch (e) {
        if (!cancelled) {
          Toast.error({ content: String((e as Error)?.message ?? e) });
        }
      } finally {
        if (!cancelled) setDetailLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedId, applyDetailToForm]);

  const selectOptions = useMemo(
    () =>
      chains
        .map((r) => {
          const rc = r?.ruleChain;
          const id = String(rc?.id ?? '');
          const name = String(rc?.name ?? id);
          return { value: id, label: `${name} (${id})` };
        })
        .filter((o) => o.value),
    [chains]
  );

  const selectedChain = useMemo(
    () => chains.find((r) => String(r?.ruleChain?.id ?? '') === selectedId),
    [chains, selectedId]
  );

  const displayName = selectedChain?.ruleChain?.name ? String(selectedChain.ruleChain.name) : '';

  useEffect(() => {
    if (!activeMsgId) return;
    pollStartRef.current = Date.now();
    let cancelled = false;

    const tick = async () => {
      if (cancelled) return;
      if (Date.now() - pollStartRef.current > POLL_MAX_MS) {
        setActiveMsgId(null);
        Toast.warning({ content: '轮询已达上限（30 分钟），已停止刷新' });
        return;
      }
      try {
        const r = await fetchRunLogsDetailed(activeMsgId);
        if (cancelled) return;
        if (!r.ok) {
          setPollHint(
            r.status === 404
              ? '运行记录尚未写入（HTTP 404）：规则链刚开始执行或日志尚未落库，将自动重试。'
              : `获取日志失败 HTTP ${r.status}，将自动重试。`
          );
          return;
        }
        setPollHint(null);
        const parsed = parseRunLogPayload(r.data);
        setLogBundle({ parsed, raw: r.data });
        if (parsed.endTs > 0) {
          setActiveMsgId(null);
          Toast.success({ content: '本次执行已结束' });
        }
      } catch {
        if (!cancelled) {
          setPollHint('网络异常，将自动重试');
        }
      }
    };

    void tick();
    const timer = window.setInterval(() => void tick(), POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [activeMsgId]);

  const onExecute = async () => {
    if (!selectedId) {
      Toast.warning({ content: '请先选择规则链' });
      return;
    }
    const mt =
      msgType.trim() ||
      inferMsgTypeFromRuleDetail(loadedDetailRef.current) ||
      DEFAULT_INFERRED_NOTIFY_MSG_TYPE;
    setRunning(true);
    setLogBundle(null);
    setPollHint(null);
    try {
      const msgId = `${alphaNanoid(24)}11`;
      const resp = await executeTestRun({
        ruleId: selectedId,
        msgType: mt,
        metadata: metadataStr.trim(),
        headers: headers as Record<string, any>,
        body,
        debugMode: true,
        msgId,
      });
      if (!resp.ok) {
        Toast.error({
          content: `触发失败 HTTP ${resp.status}: ${JSON.stringify(resp.data ?? {})}`,
        });
        return;
      }
      setActiveMsgId(msgId);
      Toast.success({ content: '已提交异步执行，正在轮询日志…' });
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setRunning(false);
    }
  };

  const tableRows = useMemo(
    () => runLogRowsFromPayload(logBundle?.parsed.logs ?? []),
    [logBundle?.parsed.logs]
  );

  const chainUi = useMemo(() => {
    const row = { ruleChain: logBundle?.parsed.ruleChain };
    return runLogChainDisplay(row);
  }, [logBundle?.parsed.ruleChain]);

  const hasErr = useMemo(
    () => summarizeRunLogErr(logBundle?.parsed.logs ?? []),
    [logBundle?.parsed.logs]
  );

  const runPhase = useMemo(() => {
    if (!logBundle) return 'idle';
    const s = logBundle.parsed.startTs;
    const e = logBundle.parsed.endTs;
    if (!s) return activeMsgId ? 'running' : 'idle';
    if (!e) return 'running';
    return 'done';
  }, [logBundle, activeMsgId]);

  const canvasMaps = useMemo(
    () => buildCanvasNodeMapsFromRuleDetail(ruleDetailSnapshot),
    [ruleDetailSnapshot]
  );

  const logColumns = useMemo(
    () => [
      {
        title: '#',
        width: 44,
        render: (_: unknown, r: Record<string, unknown>) => Number(r._idx ?? 0) + 1,
      },
      {
        title: '节点（画布）',
        width: 228,
        render: (_: unknown, r: Record<string, unknown>) => {
          const nid = String(r.nodeId ?? '');
          const { titleLine, subLine } = formatNodeCellDisplay(nid, canvasMaps);
          return (
            <div>
              <div style={{ fontWeight: 600, fontSize: 13 }}>{titleLine}</div>
              <Typography.Text size="small" type="tertiary" style={{ display: 'block' }}>
                {subLine}
              </Typography.Text>
            </div>
          );
        },
      },
      {
        title: 'inMsg 摘要',
        width: 200,
        ellipsis: true,
        render: (_: unknown, r: Record<string, unknown>) => summarizeRuleMsgLike(r.inMsg, 140),
      },
      {
        title: 'outMsg 摘要',
        width: 200,
        ellipsis: true,
        render: (_: unknown, r: Record<string, unknown>) => summarizeRuleMsgLike(r.outMsg, 140),
      },
      {
        title: '关系',
        width: 100,
        dataIndex: 'relationType',
        ellipsis: true,
      },
      {
        title: '错误',
        width: 160,
        dataIndex: 'err',
        ellipsis: true,
        render: (t: unknown) => {
          const s = t == null ? '' : String(t);
          return s ? truncateText(s, 120) : '—';
        },
      },
    ],
    [canvasMaps]
  );

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
        <Typography.Title heading={6} style={{ margin: '0 0 8px' }}>
          工作流执行
        </Typography.Title>
        <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 16 }}>
          仅列出根规则链且未禁用（disabled=false）。参数来自规则链 configuration.flowgram.io
          中配置的「请求元数据」「请求体」参数说明；提交后走异步接口，下方日志每秒刷新一次直至检测到结束时间。
        </Typography.Paragraph>

        <Spin spinning={loadingList || detailLoading}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16, maxWidth: 960 }}>
            <div>
              <Typography.Text size="small" style={{ display: 'block', marginBottom: 8 }}>
                根规则链
              </Typography.Text>
              <Select
                style={{ width: '100%' }}
                placeholder="请选择要执行的根规则链"
                optionList={selectOptions}
                value={selectedId}
                onChange={(v) => setSelectedId(v as string)}
                filter
                showClear
              />
              {selectedId ? (
                <Typography.Text
                  type="tertiary"
                  size="small"
                  style={{ display: 'block', marginTop: 8 }}
                >
                  当前：{displayName || selectedId}
                  {selectedChain?.ruleChain?.disabled ? (
                    <Tag color="red" size="small" style={{ marginLeft: 8 }}>
                      已禁用
                    </Tag>
                  ) : (
                    <Tag color="green" size="small" style={{ marginLeft: 8 }}>
                      未禁用
                    </Tag>
                  )}
                </Typography.Text>
              ) : null}
            </div>

            <div>
              <Typography.Text size="small" style={{ display: 'block', marginBottom: 8 }}>
                消息类型（选填）
              </Typography.Text>
              <Input
                value={msgType}
                onChange={setMsgType}
                placeholder={`推断失败时使用 ${DEFAULT_INFERRED_NOTIFY_MSG_TYPE}；须与异步 API 路径中的 msgType 段一致`}
              />
              <Typography.Paragraph
                type="tertiary"
                size="small"
                style={{ marginTop: 6, marginBottom: 0 }}
              >
                加载规则链后自动填入：configuration.flowgram.entry_msg_type →
                additionalInfo（msgType 等） → metadata.endpoints 中非定时端点的路由 path（HTTP
                取末段）；定时 Cron 路径会跳过。 输入框清空后，点击「异步执行」仍会按上述顺序推断。
              </Typography.Paragraph>
            </div>

            <div>
              <Typography.Text size="small" style={{ display: 'block', marginBottom: 8 }}>
                元数据（query，由「请求元数据参数」自动生成，可改）
              </Typography.Text>
              <TextArea
                value={metadataStr}
                onChange={setMetadataStr}
                rows={3}
                placeholder="key=value&..."
              />
            </div>

            <div>
              <Typography.Text size="small" style={{ display: 'block', marginBottom: 8 }}>
                请求头
              </Typography.Text>
              <JsonValueEditor value={headers} onChange={setHeaders} />
            </div>

            <div>
              <Typography.Text size="small" style={{ display: 'block', marginBottom: 8 }}>
                请求体（由「请求体参数」自动生成，可改）
              </Typography.Text>
              <JsonValueEditor value={body} onChange={setBody} />
            </div>

            <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
              <Button
                theme="solid"
                type="primary"
                loading={running}
                onClick={() => void onExecute()}
              >
                异步执行
              </Button>
              <Button type="tertiary" onClick={() => void loadChains()}>
                刷新规则链列表
              </Button>
              {activeMsgId ? (
                <Typography.Text type="tertiary" size="small">
                  轮询中 msgId: {activeMsgId}（{POLL_MS / 1000}s）
                </Typography.Text>
              ) : null}
            </div>
          </div>
        </Spin>
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
          overflow: 'hidden',
        }}
      >
        <Typography.Title heading={6} style={{ margin: '0 0 12px', flexShrink: 0 }}>
          执行日志
        </Typography.Title>
        <Typography.Paragraph
          type="tertiary"
          size="small"
          style={{ marginBottom: 12, flexShrink: 0 }}
        >
          数据来源 GET /api/v1/logs/runs/msgId；节点名称来自当前规则链 metadata.flowgramUI（及
          metadata.nodes 补全）。inMsg/outMsg 为 RuleMsg 摘要，展开行可看完整 JSON。轮询间隔{' '}
          {POLL_MS / 1000}s，endTs 有值后自动停止。
        </Typography.Paragraph>

        {pollHint ? (
          <div
            style={{
              flexShrink: 0,
              marginBottom: 12,
              padding: '10px 12px',
              background: 'rgba(255, 170, 0, 0.12)',
              border: '1px solid rgba(255, 170, 0, 0.35)',
              borderRadius: 6,
              fontSize: 13,
              color: '#5c3c00',
            }}
          >
            {pollHint}
          </div>
        ) : null}

        {!logBundle && !activeMsgId ? (
          <Empty
            title="暂无执行记录"
            description="点击「异步执行」后，将在此展示节点级日志与时间线。"
            style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
          />
        ) : null}

        {activeMsgId && !logBundle ? (
          <Empty
            imageStyle={{ height: 72 }}
            title="等待运行日志"
            description="后端根据 msgId 写入 run_log 后即可展示；若持续无数据请确认规则链已部署。"
            style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
          />
        ) : null}

        {logBundle ? (
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, gap: 12 }}>
            <div
              style={{
                flexShrink: 0,
                display: 'flex',
                flexWrap: 'wrap',
                gap: 12,
                alignItems: 'center',
              }}
            >
              <Tag color={runPhase === 'running' ? 'blue' : runPhase === 'done' ? 'green' : 'grey'}>
                {runPhase === 'idle' ? '未开始' : runPhase === 'running' ? '运行中' : '已结束'}
              </Tag>
              {hasErr ? (
                <Tag color="red">节点错误</Tag>
              ) : runPhase === 'done' ? (
                <Tag color="green">无节点错误</Tag>
              ) : null}
              <Typography.Text type="tertiary" size="small">
                规则链：{chainUi.name || '—'}（{chainUi.id || '—'}）
              </Typography.Text>
              <Typography.Text type="tertiary" size="small">
                开始 {formatTs(logBundle.parsed.startTs)} · 结束{' '}
                {logBundle.parsed.endTs ? formatTs(logBundle.parsed.endTs) : '—'}
                {logBundle.parsed.startTs && logBundle.parsed.endTs
                  ? ` · 耗时 ${logBundle.parsed.endTs - logBundle.parsed.startTs} ms`
                  : ''}
              </Typography.Text>
            </div>

            <div style={{ flex: 1, minHeight: 0, overflow: 'hidden' }}>
              <Table
                size="small"
                dataSource={tableRows}
                rowKey="_rowKey"
                scroll={{ y: 360, x: 1320 }}
                pagination={false}
                columns={logColumns}
                expandedRowRender={(record: Record<string, unknown>) => {
                  const inMsg = record.inMsg;
                  const outMsg = record.outMsg;
                  const rest = { ...record };
                  delete rest._idx;
                  delete rest._rowKey;
                  delete rest.inMsg;
                  delete rest.outMsg;
                  return (
                    <div
                      style={{
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 12,
                        paddingRight: 8,
                      }}
                    >
                      <div>
                        <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                          inMsg（完整）
                        </Typography.Text>
                        <pre
                          style={{
                            margin: 0,
                            padding: 12,
                            fontSize: 12,
                            lineHeight: 1.45,
                            whiteSpace: 'pre-wrap',
                            wordBreak: 'break-word',
                            background: '#f0f5ff',
                            borderRadius: 4,
                            maxHeight: 280,
                            overflow: 'auto',
                          }}
                        >
                          {safeJson(inMsg)}
                        </pre>
                      </div>
                      <div>
                        <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                          outMsg（完整）
                        </Typography.Text>
                        <pre
                          style={{
                            margin: 0,
                            padding: 12,
                            fontSize: 12,
                            lineHeight: 1.45,
                            whiteSpace: 'pre-wrap',
                            wordBreak: 'break-word',
                            background: '#f6ffed',
                            borderRadius: 4,
                            maxHeight: 280,
                            overflow: 'auto',
                          }}
                        >
                          {safeJson(outMsg)}
                        </pre>
                      </div>
                      <div>
                        <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                          其它字段
                        </Typography.Text>
                        <pre
                          style={{
                            margin: 0,
                            padding: 12,
                            fontSize: 12,
                            lineHeight: 1.45,
                            whiteSpace: 'pre-wrap',
                            wordBreak: 'break-word',
                            background: '#f7f8fa',
                            borderRadius: 4,
                            maxHeight: 200,
                            overflow: 'auto',
                          }}
                        >
                          {safeJson(rest)}
                        </pre>
                      </div>
                    </div>
                  );
                }}
              />
            </div>

            <Collapse style={{ flexShrink: 0 }}>
              <Collapse.Panel header="原始响应 JSON（调试）" itemKey="raw">
                <pre
                  style={{
                    margin: 0,
                    maxHeight: 240,
                    overflow: 'auto',
                    fontSize: 11,
                    lineHeight: 1.4,
                    padding: 12,
                    background: '#0d1117',
                    color: '#c9d1d9',
                    borderRadius: 4,
                  }}
                >
                  {safeJson(logBundle.raw)}
                </pre>
              </Collapse.Panel>
            </Collapse>
          </div>
        ) : null}
      </div>
    </div>
  );
};

export default WorkflowExecuteSection;
