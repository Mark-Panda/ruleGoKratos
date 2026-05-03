import React, { useCallback, useEffect, useRef, useState } from 'react';

import type { FormApi } from '@douyinfe/semi-ui/lib/es/form/interface';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  TextArea,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconEdit,
  IconHistory,
  IconPause,
  IconPlay,
  IconPlus,
} from '@douyinfe/semi-icons';

import {
  parseRuleChainParamsJson,
  buildRuleChainParamsPreviewValue,
} from '../../utils/rule-chain-request-params';
import { parseRuleChainFlowgramFromConfiguration } from '../../utils/rule-chain-flowgram-dsl';
import {
  createScheduledTask,
  deleteScheduledTask,
  disableScheduledTask,
  enableScheduledTask,
  listScheduledTaskRuns,
  listScheduledTasks,
  ScheduledTask,
  ScheduledTaskRun,
  updateScheduledTask,
} from '../../services/api-scheduled-task';
import { getRuleList, getRuleDetail } from '../../services/api-rules';
import { JsonValueEditor } from '../../components/testrun/json-value-editor';
import {
  buildScheduledTaskPayload,
  describeScheduledTaskRunStatus,
  describeSchedule,
  getScheduledTaskFormInitValues,
  normalizeScheduledTaskRunStatus,
  ScheduleType,
} from './scheduled-task-cron';

type ModalType = 'create' | 'edit';

interface RuleChainOption {
  value: string;
  label: string;
}

const scheduleTypeOptions: { value: ScheduleType; label: string }[] = [
  { value: 'every_minutes', label: '每 N 分钟' },
  { value: 'every_hours', label: '每 N 小时' },
  { value: 'daily', label: '每天' },
  { value: 'weekly', label: '每周' },
  { value: 'monthly', label: '每月' },
  { value: 'advanced', label: '高级 Cron' },
];

const dayOfWeekOptions = [
  { value: 0, label: '周日' },
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
];

function asTotal(value: unknown): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

function formatDateTime(value: ScheduledTask['updatedAt']): string {
  if (!value) return '—';
  if (typeof value === 'object') {
    const seconds = Number((value as { seconds?: unknown }).seconds);
    if (Number.isFinite(seconds) && seconds > 0) {
      return new Date(seconds * 1000).toLocaleString();
    }
    return '—';
  }
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return String(value);
  return d.toLocaleString();
}

function runStatusLabel(status: unknown): string {
  return describeScheduledTaskRunStatus(status);
}

function runStatusColor(status: unknown): 'green' | 'red' | 'grey' {
  const normalized = normalizeScheduledTaskRunStatus(status);
  if (normalized === 'success') return 'green';
  if (normalized === 'failed') return 'red';
  return 'grey';
}

function getRuleChainOption(raw: any): RuleChainOption | null {
  const chain = raw?.ruleChain ?? raw;
  const id = String(chain?.id ?? '').trim();
  if (!id) return null;
  const name = String(chain?.name ?? '').trim();
  return {
    value: id,
    label: name ? `${name}（${id}）` : id,
  };
}

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

export const ScheduledTaskSection: React.FC = () => {
  const [tasks, setTasks] = useState<ScheduledTask[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const taskListRequestIdRef = useRef(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState<{
    name?: string;
    disabled?: boolean;
    ruleChainId?: string;
  }>({});

  const [ruleChainOptions, setRuleChainOptions] = useState<RuleChainOption[]>([]);
  const [ruleChainLoading, setRuleChainLoading] = useState(false);

  const [modalVisible, setModalVisible] = useState(false);
  const [modalType, setModalType] = useState<ModalType>('create');
  const [editingTask, setEditingTask] = useState<ScheduledTask | null>(null);
  const [formInitSnapshot, setFormInitSnapshot] = useState<Record<string, unknown>>({});
  const [formModalKey, setFormModalKey] = useState(0);
  const [activeScheduleType, setActiveScheduleType] = useState<ScheduleType>('daily');
  const formApiRef = useRef<FormApi<Record<string, unknown>> | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [selectedRuleChainId, setSelectedRuleChainId] = useState<string | undefined>();
  const [detailLoading, setDetailLoading] = useState(false);
  const [metadataStr, setMetadataStr] = useState('');
  const [body, setBody] = useState<Record<string, unknown>>({});

  const [historyVisible, setHistoryVisible] = useState(false);
  const [historyTask, setHistoryTask] = useState<ScheduledTask | null>(null);
  const [historyRuns, setHistoryRuns] = useState<ScheduledTaskRun[]>([]);
  const [historyTotal, setHistoryTotal] = useState(0);
  const [historyPage, setHistoryPage] = useState(1);
  const [historyPageSize, setHistoryPageSize] = useState(10);
  const [historyLoading, setHistoryLoading] = useState(false);
  const historyRequestIdRef = useRef(0);
  const activeHistoryTaskIdRef = useRef<string | null>(null);

  const fetchTasks = async (page = currentPage, size = pageSize) => {
    const requestId = taskListRequestIdRef.current + 1;
    taskListRequestIdRef.current = requestId;
    setLoading(true);
    try {
      const res = await listScheduledTasks({
        name: filters.name?.trim() || undefined,
        disabled: filters.disabled,
        ruleChainId: filters.ruleChainId?.trim() || undefined,
        page,
        pageSize: size,
      });
      if (requestId !== taskListRequestIdRef.current) return;
      setTasks(Array.isArray(res.tasks) ? res.tasks : []);
      setTotal(asTotal(res.total));
      setCurrentPage(page);
    } catch (e) {
      if (requestId !== taskListRequestIdRef.current) return;
      Toast.error('获取定时任务列表失败');
      console.error(e);
    } finally {
      if (requestId === taskListRequestIdRef.current) {
        setLoading(false);
      }
    }
  };

  const fetchRuleChains = async () => {
    setRuleChainLoading(true);
    try {
      const res = await getRuleList({ page: 1, size: 200, root: true });
      const options = (Array.isArray(res.items) ? res.items : [])
        .map(getRuleChainOption)
        .filter((item): item is RuleChainOption => Boolean(item));
      setRuleChainOptions(options);
    } catch (e) {
      Toast.warning('规则链选项加载失败，可手动输入规则链 ID');
      console.error(e);
      setRuleChainOptions([]);
    } finally {
      setRuleChainLoading(false);
    }
  };

  const applyDetailToPayloadFields = useCallback((detail: any) => {
    const cfg = detail?.ruleChain?.configuration;
    const fg = parseRuleChainFlowgramFromConfiguration(cfg);
    const metaNodes = parseRuleChainParamsJson(fg.requestMetadataParamsJson);
    const bodyNodes = parseRuleChainParamsJson(fg.requestMessageBodyParamsJson);
    const metaPreview = buildRuleChainParamsPreviewValue(metaNodes);
    const bodyPreview = buildRuleChainParamsPreviewValue(bodyNodes);
    setMetadataStr(metadataPreviewToQueryString(metaPreview));
    setBody(bodyPreview);
  }, []);

  useEffect(() => {
    void fetchRuleChains();
  }, []);

  useEffect(() => {
    if (!selectedRuleChainId) {
      setMetadataStr('');
      setBody({});
      return;
    }
    let cancelled = false;
    (async () => {
      setDetailLoading(true);
      try {
        const detail = await getRuleDetail(selectedRuleChainId);
        if (!cancelled) {
          applyDetailToPayloadFields(detail);
        }
      } catch (e) {
        if (!cancelled) {
          Toast.warning('规则链详情加载失败，参数模板未自动填充');
          console.error(e);
        }
      } finally {
        if (!cancelled) setDetailLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedRuleChainId, applyDetailToPayloadFields]);

  useEffect(() => {
    setCurrentPage(1);
    void fetchTasks(1, pageSize);
  }, [filters]);

  const openModal = (type: ModalType, task?: ScheduledTask) => {
    const initValues = getScheduledTaskFormInitValues(type === 'edit' ? task : null);
    setModalType(type);
    setEditingTask(type === 'edit' && task ? task : null);
    setActiveScheduleType(initValues.scheduleType as ScheduleType);

    if (type === 'edit' && task?.payloadTemplate) {
      try {
        const tpl = JSON.parse(task.payloadTemplate);
        if (typeof tpl.metadata === 'string') setMetadataStr(tpl.metadata);
        if (tpl.body && typeof tpl.body === 'object') setBody(tpl.body);
      } catch {
        setMetadataStr('');
        setBody({});
      }
    } else {
      setMetadataStr('');
      setBody({});
    }
    setSelectedRuleChainId(type === 'edit' && task ? task.ruleChainId : undefined);

    setFormInitSnapshot(initValues as Record<string, unknown>);
    formApiRef.current = null;
    setFormModalKey((k) => k + 1);
    setModalVisible(true);
  };

  const closeModal = () => {
    setModalVisible(false);
    setEditingTask(null);
    setFormInitSnapshot({});
    formApiRef.current = null;
    setSelectedRuleChainId(undefined);
    setMetadataStr('');
    setBody({});
  };

  const refreshCurrentPage = () => {
    void fetchTasks(currentPage, pageSize);
  };

  const handleSubmit = async () => {
    const api = formApiRef.current;
    if (!api) {
      Toast.warning('表单未就绪，请稍后重试');
      return;
    }
    try {
      await api.validate();
    } catch {
      return;
    }

    let payload;
    try {
      payload = buildScheduledTaskPayload(api.getValues());
    } catch (e) {
      Toast.error(String((e as Error)?.message ?? e));
      return;
    }
    if (!payload.name || !payload.ruleChainId || !payload.cronExpr) {
      Toast.warning('请填写任务名称、规则链和执行周期');
      return;
    }

    payload.payloadTemplate = JSON.stringify({
      metadata: metadataStr.trim(),
      body,
    });

    setSubmitting(true);
    try {
      if (modalType === 'create') {
        await createScheduledTask(payload);
        Toast.success('创建成功');
      } else if (editingTask) {
        await updateScheduledTask(editingTask.id, payload);
        Toast.success('更新成功');
      }
      closeModal();
      refreshCurrentPage();
    } catch (e) {
      Toast.error(modalType === 'create' ? '创建失败' : '更新失败');
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  const handleEnableToggle = async (task: ScheduledTask) => {
    try {
      if (task.disabled) {
        await enableScheduledTask(task.id);
        Toast.success('已开启');
      } else {
        await disableScheduledTask(task.id);
        Toast.success('已关闭');
      }
      refreshCurrentPage();
    } catch (e) {
      Toast.error(task.disabled ? '开启失败' : '关闭失败');
      console.error(e);
    }
  };

  const handleDelete = async (task: ScheduledTask) => {
    try {
      await deleteScheduledTask(task.id);
      Toast.success('删除成功');
      void fetchTasks(
        currentPage === 1 ? 1 : tasks.length === 1 ? currentPage - 1 : currentPage,
        pageSize
      );
    } catch (e) {
      Toast.error('删除失败');
      console.error(e);
    }
  };

  const fetchHistory = async (task: ScheduledTask, page = historyPage, size = historyPageSize) => {
    const taskId = String(task.id);
    const requestId = historyRequestIdRef.current + 1;
    historyRequestIdRef.current = requestId;
    setHistoryLoading(true);
    try {
      const res = await listScheduledTaskRuns(task.id, { page, pageSize: size });
      if (requestId !== historyRequestIdRef.current || activeHistoryTaskIdRef.current !== taskId) {
        return;
      }
      setHistoryRuns(Array.isArray(res.runs) ? res.runs : []);
      setHistoryTotal(asTotal(res.total));
      setHistoryPage(page);
    } catch (e) {
      if (requestId !== historyRequestIdRef.current || activeHistoryTaskIdRef.current !== taskId) {
        return;
      }
      Toast.error('获取执行历史失败');
      console.error(e);
    } finally {
      if (requestId === historyRequestIdRef.current && activeHistoryTaskIdRef.current === taskId) {
        setHistoryLoading(false);
      }
    }
  };

  const openHistory = (task: ScheduledTask) => {
    activeHistoryTaskIdRef.current = String(task.id);
    setHistoryTask(task);
    setHistoryVisible(true);
    setHistoryRuns([]);
    setHistoryTotal(0);
    setHistoryPage(1);
    void fetchHistory(task, 1, historyPageSize);
  };

  const closeHistory = () => {
    historyRequestIdRef.current += 1;
    activeHistoryTaskIdRef.current = null;
    setHistoryVisible(false);
    setHistoryTask(null);
    setHistoryRuns([]);
    setHistoryTotal(0);
    setHistoryLoading(false);
  };

  const renderRuleChainField = () => {
    if (ruleChainOptions.length > 0) {
      return (
        <Form.Select
          field="ruleChainId"
          label="绑定规则链"
          placeholder="请选择主规则链"
          style={{ width: '100%' }}
          loading={ruleChainLoading}
          filter
          rules={[{ required: true, message: '请选择绑定规则链' }]}
          optionList={ruleChainOptions}
          onChange={(val) => setSelectedRuleChainId(String(val ?? ''))}
        />
      );
    }
    return (
      <Form.Input
        field="ruleChainId"
        label="绑定规则链"
        placeholder="请输入 ruleChainId"
        rules={[{ required: true, message: '请输入绑定规则链' }]}
        onChange={(val) => setSelectedRuleChainId(String(val ?? ''))}
      />
    );
  };

  const renderScheduleFields = () => {
    if (activeScheduleType === 'every_minutes') {
      return (
        <Form.Input
          field="minutes"
          label="分钟间隔"
          placeholder="1-59"
          rules={[{ required: true }]}
        />
      );
    }
    if (activeScheduleType === 'every_hours') {
      return (
        <Form.Input
          field="hours"
          label="小时间隔"
          placeholder="1-23"
          rules={[{ required: true }]}
        />
      );
    }
    if (activeScheduleType === 'weekly') {
      return (
        <>
          <Form.Select
            field="dayOfWeek"
            label="星期"
            style={{ width: '100%' }}
            optionList={dayOfWeekOptions}
            rules={[{ required: true }]}
          />
          <Form.Input field="hour" label="小时" placeholder="0-23" rules={[{ required: true }]} />
          <Form.Input field="minute" label="分钟" placeholder="0-59" rules={[{ required: true }]} />
        </>
      );
    }
    if (activeScheduleType === 'monthly') {
      return (
        <>
          <Form.Input
            field="dayOfMonth"
            label="日期"
            placeholder="1-31"
            rules={[{ required: true }]}
          />
          <Form.Input field="hour" label="小时" placeholder="0-23" rules={[{ required: true }]} />
          <Form.Input field="minute" label="分钟" placeholder="0-59" rules={[{ required: true }]} />
        </>
      );
    }
    if (activeScheduleType === 'advanced') {
      return (
        <Form.Input
          field="cronExpr"
          label="Cron 表达式"
          placeholder="例如：0 2 * * *"
          rules={[{ required: true, message: '请输入 Cron 表达式' }]}
        />
      );
    }
    return (
      <>
        <Form.Input field="hour" label="小时" placeholder="0-23" rules={[{ required: true }]} />
        <Form.Input field="minute" label="分钟" placeholder="0-59" rules={[{ required: true }]} />
      </>
    );
  };

  const columns = [
    {
      title: '任务名称',
      dataIndex: 'name',
      width: 180,
      render: (val: string) => <Typography.Text strong>{val || '—'}</Typography.Text>,
    },
    {
      title: '绑定主规则链',
      dataIndex: 'ruleChainId',
      width: 180,
      render: (val: string) => <Typography.Text copyable>{val || '—'}</Typography.Text>,
    },
    {
      title: '执行周期描述',
      width: 220,
      render: (_: unknown, record: ScheduledTask) =>
        describeSchedule(
          record.scheduleType as ScheduleType,
          record.scheduleConfig,
          record.cronExpr
        ),
    },
    {
      title: '启停状态',
      dataIndex: 'disabled',
      width: 110,
      render: (disabled: boolean) =>
        disabled ? <Tag color="grey">已关闭</Tag> : <Tag color="green">已开启</Tag>,
    },
    {
      title: '最近运行时间',
      dataIndex: 'lastRunAt',
      width: 180,
      render: formatDateTime,
    },
    {
      title: '最近结果',
      dataIndex: 'lastStatus',
      width: 110,
      render: (status: unknown) => (
        <Tag color={runStatusColor(status)}>{runStatusLabel(status)}</Tag>
      ),
    },
    {
      title: '最近错误',
      dataIndex: 'lastError',
      width: 200,
      ellipsis: true,
      render: (val: string) => val || '—',
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      width: 180,
      render: formatDateTime,
    },
    {
      title: '操作',
      fixed: 'right' as const,
      width: 270,
      render: (_: unknown, record: ScheduledTask) => (
        <Space spacing={4} wrap>
          <Button
            icon={<IconEdit />}
            size="small"
            theme="borderless"
            onClick={() => openModal('edit', record)}
          >
            编辑
          </Button>
          <Button
            icon={record.disabled ? <IconPlay /> : <IconPause />}
            size="small"
            theme="borderless"
            onClick={() => void handleEnableToggle(record)}
          >
            {record.disabled ? '开启' : '关闭'}
          </Button>
          <Button
            icon={<IconHistory />}
            size="small"
            theme="borderless"
            onClick={() => openHistory(record)}
          >
            历史
          </Button>
          <Popconfirm
            title="确认删除"
            content="确定要删除这个定时任务吗？删除后不可恢复。"
            okText="确定"
            cancelText="取消"
            onConfirm={() => void handleDelete(record)}
          >
            <Button icon={<IconDelete />} size="small" theme="borderless" type="danger">
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const historyColumns = [
    {
      title: '执行时间',
      dataIndex: 'startedAt',
      width: 180,
      render: formatDateTime,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: unknown) => (
        <Tag color={runStatusColor(status)}>{runStatusLabel(status)}</Tag>
      ),
    },
    {
      title: '规则链 ID',
      dataIndex: 'ruleChainId',
      width: 180,
      render: (val: string) => <Typography.Text copyable>{val || '—'}</Typography.Text>,
    },
    {
      title: '失败原因',
      dataIndex: 'errorMessage',
      width: 220,
      ellipsis: true,
      render: (val: string) => val || '—',
    },
    {
      title: '触发 payload',
      dataIndex: 'triggerPayload',
      ellipsis: true,
      render: (val: string) => val || '—',
    },
  ];

  return (
    <div style={{ padding: '24px', width: '100%', boxSizing: 'border-box' }}>
      <Card>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 20,
            flexWrap: 'wrap',
            gap: 12,
          }}
        >
          <Typography.Title heading={3} style={{ margin: 0 }}>
            定时任务
          </Typography.Title>
          <Button icon={<IconPlus />} type="primary" onClick={() => openModal('create')}>
            新增定时任务
          </Button>
        </div>

        <div
          style={{
            marginBottom: 24,
            display: 'flex',
            gap: 16,
            alignItems: 'flex-end',
            flexWrap: 'wrap',
          }}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <Typography.Text type="tertiary" size="small">
              任务名称
            </Typography.Text>
            <Input
              value={filters.name ?? ''}
              onChange={(val) => setFilters({ ...filters, name: val })}
              placeholder="请输入任务名称"
              style={{ width: 200 }}
              showClear
            />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <Typography.Text type="tertiary" size="small">
              启停状态
            </Typography.Text>
            <Select
              value={filters.disabled == null ? undefined : filters.disabled ? 'true' : 'false'}
              onChange={(val) =>
                setFilters({
                  ...filters,
                  disabled: val == null || val === '' ? undefined : String(val) === 'true',
                })
              }
              placeholder="全部状态"
              style={{ width: 160 }}
              showClear
              optionList={[
                { label: '已开启', value: 'false' },
                { label: '已关闭', value: 'true' },
              ]}
            />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <Typography.Text type="tertiary" size="small">
              绑定规则链
            </Typography.Text>
            {ruleChainOptions.length > 0 ? (
              <Select
                value={filters.ruleChainId}
                onChange={(val) =>
                  setFilters({ ...filters, ruleChainId: val == null ? undefined : String(val) })
                }
                placeholder="全部规则链"
                style={{ width: 260 }}
                loading={ruleChainLoading}
                showClear
                filter
                optionList={ruleChainOptions}
              />
            ) : (
              <Input
                value={filters.ruleChainId ?? ''}
                onChange={(val) => setFilters({ ...filters, ruleChainId: val })}
                placeholder="输入 ruleChainId"
                style={{ width: 220 }}
                showClear
              />
            )}
          </div>
          <Button onClick={() => setFilters({})}>重置筛选</Button>
        </div>

        <Table
          columns={columns}
          dataSource={tasks}
          rowKey="id"
          loading={loading}
          pagination={{
            currentPage,
            pageSize,
            total,
            showSizeChanger: true,
            pageSizeOpts: [10, 20, 50, 100],
            onPageChange: (page: number, size = pageSize) => {
              setPageSize(size);
              void fetchTasks(page, size);
            },
          }}
        />
      </Card>

      <Modal
        title={modalType === 'create' ? '新增定时任务' : '编辑定时任务'}
        visible={modalVisible}
        onCancel={closeModal}
        width={800}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12 }}>
            <Button onClick={closeModal}>取消</Button>
            <Button type="primary" loading={submitting} onClick={() => void handleSubmit()}>
              {modalType === 'create' ? '创建' : '保存'}
            </Button>
          </div>
        }
      >
        <Form
          key={formModalKey}
          initValues={formInitSnapshot}
          getFormApi={(api) => {
            formApiRef.current = api;
          }}
          labelPosition="left"
          labelAlign="right"
          labelWidth={110}
        >
          <Form.Input
            field="name"
            label="任务名称"
            placeholder="请输入任务名称"
            rules={[{ required: true, message: '请输入任务名称' }]}
          />
          {renderRuleChainField()}
          <Form.Select
            field="scheduleType"
            label="执行周期"
            style={{ width: '100%' }}
            optionList={scheduleTypeOptions}
            rules={[{ required: true, message: '请选择执行周期' }]}
            onChange={(val) => setActiveScheduleType(val as ScheduleType)}
          />
          {renderScheduleFields()}
          <Form.TextArea
            field="description"
            label="任务描述"
            placeholder="请输入任务描述"
            rows={3}
          />
        </Form>

        {selectedRuleChainId && (
          <Spin spinning={detailLoading}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginTop: 16 }}>
              <Typography.Title heading={6} style={{ margin: 0 }}>
                工作流入参
              </Typography.Title>
              <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 0 }}>
                根据规则链 configuration.flowgram.io
                中配置的入参自动生成，可修改。触发时将合并到执行 payload 中。
              </Typography.Paragraph>
              <div>
                <Typography.Text size="small" style={{ display: 'block', marginBottom: 8 }}>
                  元数据（query，由规则链入参自动生成，可修改）
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
                  请求体（由规则链入参自动生成，可修改）
                </Typography.Text>
                <JsonValueEditor value={body} onChange={setBody} />
              </div>
            </div>
          </Spin>
        )}
      </Modal>

      <Modal
        title={historyTask ? `执行历史：${historyTask.name}` : '执行历史'}
        visible={historyVisible}
        onCancel={closeHistory}
        width={960}
        footer={
          <Button type="primary" onClick={closeHistory}>
            关闭
          </Button>
        }
      >
        <Table
          columns={historyColumns}
          dataSource={historyRuns}
          rowKey="id"
          loading={historyLoading}
          pagination={{
            currentPage: historyPage,
            pageSize: historyPageSize,
            total: historyTotal,
            showSizeChanger: true,
            pageSizeOpts: [10, 20, 50],
            onPageChange: (page: number, size = historyPageSize) => {
              setHistoryPageSize(size);
              if (historyTask) void fetchHistory(historyTask, page, size);
            },
          }}
        />
      </Modal>
    </div>
  );
};

export default ScheduledTaskSection;
