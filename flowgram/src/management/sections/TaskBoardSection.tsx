import React, { useState, useEffect, useMemo, useRef } from 'react';

import type { FormApi } from '@douyinfe/semi-ui/lib/es/form/interface';
import {
  Table,
  Button,
  Form,
  Input,
  Select,
  Modal,
  Space,
  Tag,
  Toast,
  Popconfirm,
  Card,
  Typography,
  Radio,
  Spin,
} from '@douyinfe/semi-ui';
import { IconPlus, IconEdit, IconDelete, IconUser, IconInfoCircle } from '@douyinfe/semi-icons';

import { buildDocumentFromRuleChainJSON } from '../../utils/rulechain-builder';
import { FlowDocumentJSON } from '../../typings';
import { fetchRunLogs } from '../../services/test-run-http';
import { requestJSON } from '../../services/http';
import {
  listTasks,
  getTask,
  createTask,
  updateTask,
  deleteTask,
  executeTaskRuleChain,
  createChildTask,
  listChildTasks,
  TaskItem,
  TaskStatus,
  TaskType,
  taskStatusOptions,
  taskTypeOptions,
  priorityOptions,
  CreateTaskParams,
  CreateChildTaskParams,
} from '../../services/api-task';
import { getRuleList } from '../../services/api-rules';
import { Editor } from '../../editor';
import { priorityTagColor, taskStatusTagColor, taskTypeTagColor } from './section-display';

const VIEW_MODE_KEY = 'task-board-view-mode';
const KANBAN_PAGE_SIZE = 300;

type ViewMode = 'table' | 'kanban';

function readStoredViewMode(): ViewMode {
  try {
    const v = localStorage.getItem(VIEW_MODE_KEY);
    if (v === 'kanban' || v === 'table') return v;
  } catch {
    // ignore
  }
  return 'table';
}

/** 创建/距今的简短中文时长，用于看板卡片 */
function relativeDurationCn(iso: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  const diff = Date.now() - d.getTime();
  if (diff < 0) return '刚刚';
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return mins < 1 ? '刚刚' : `${mins} 分钟`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours} 小时`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days} 天`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months} 个月`;
  return `${Math.floor(days / 365)} 年`;
}

function priorityLabel(p: number): string | null {
  if (p <= 9) return 'P0';
  if (p <= 19) return 'P1';
  if (p <= 39) return 'P2';
  return null;
}

const KANBAN_COLUMNS: {
  status: TaskStatus;
  title: string;
  dot: string;
  columnBg: string;
  tagTone: 'grey' | 'blue' | 'green' | 'red';
}[] = [
  {
    status: TaskStatus.PENDING,
    title: '待处理',
    dot: '#86909c',
    columnBg: 'rgba(134, 144, 156, 0.1)',
    tagTone: 'grey',
  },
  {
    status: TaskStatus.PROCESSING,
    title: '处理中',
    dot: '#165dff',
    columnBg: 'rgba(22, 93, 255, 0.08)',
    tagTone: 'blue',
  },
  {
    status: TaskStatus.COMPLETED,
    title: '已完成',
    dot: '#00b578',
    columnBg: 'rgba(0, 181, 120, 0.08)',
    tagTone: 'green',
  },
  {
    status: TaskStatus.FAILED,
    title: '失败',
    dot: '#f53f3f',
    columnBg: 'rgba(245, 63, 63, 0.08)',
    tagTone: 'red',
  },
];

export const TaskBoardSection: React.FC = () => {
  const [tasks, setTasks] = useState<TaskItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [pageSize, setPageSize] = useState(10);
  const [currentPage, setCurrentPage] = useState(1);
  const [viewMode, setViewMode] = useState<ViewMode>(() => readStoredViewMode());
  const [filters, setFilters] = useState<{
    status?: TaskStatus;
    type?: TaskType;
    handler_user_id?: string;
  }>({});
  const [modalVisible, setModalVisible] = useState(false);
  const [modalType, setModalType] = useState<'create' | 'edit'>('create');
  const [editingTask, setEditingTask] = useState<TaskItem | null>(null);
  /** 仅用于 Form initValues；与 Semi Form 内部状态同步靠 formKey 重挂载 */
  const [formInitSnapshot, setFormInitSnapshot] = useState<Record<string, unknown>>({});
  const [formModalKey, setFormModalKey] = useState(0);
  const taskFormApiRef = useRef<FormApi<Record<string, unknown>> | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [detailVisible, setDetailVisible] = useState(false);
  const [detailTask, setDetailTask] = useState<TaskItem | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  // 规则链列表（用于下拉选择）
  const [ruleChainOptions, setRuleChainOptions] = useState<{ label: string; value: string }[]>([]);
  const [ruleChainNameMap, setRuleChainNameMap] = useState<Record<string, string>>({});
  const [ruleChainLoading, setRuleChainLoading] = useState(false);

  // 子任务列表弹窗
  const [childTasksVisible, setChildTasksVisible] = useState(false);
  const [childTasks, setChildTasks] = useState<TaskItem[]>([]);
  const [childTasksTotal, setChildTasksTotal] = useState(0);
  const [childTasksLoading, setChildTasksLoading] = useState(false);
  const [childTasksPage, setChildTasksPage] = useState(1);

  // 创建子任务弹窗
  const [createChildVisible, setCreateChildVisible] = useState(false);
  const [createChildSuffix, setCreateChildSuffix] = useState('');
  const [createChildSubmitting, setCreateChildSubmitting] = useState(false);

  // 执行规则链
  const [executeLoading, setExecuteLoading] = useState(false);

  // 规则链执行日志查看器
  const [runLogViewerOpen, setRunLogViewerOpen] = useState(false);
  const [runLogViewerDoc, setRunLogViewerDoc] = useState<FlowDocumentJSON | undefined>();
  const [runLogViewerLogs, setRunLogViewerLogs] = useState<{
    list: any[];
    startTs?: number;
    endTs?: number;
  }>();
  const [runLogViewerLoading, setRunLogViewerLoading] = useState(false);

  // 详情弹窗数据行
  const detailRows = useMemo(() => {
    if (!detailTask) return [];
    const rows: [string, unknown][] = [
      ['任务ID', detailTask.id],
      ['任务名称', detailTask.name],
      [
        '状态',
        taskStatusOptions.find((o) => o.value === detailTask.status)?.label ?? detailTask.status,
      ],
      ['类型', taskTypeOptions.find((o) => o.value === detailTask.type)?.label ?? detailTask.type],
      ['优先级', detailTask.priority],
      ['处理人', detailTask.handler_user_id || '—'],
      [
        '关联规则链',
        detailTask.rule_chain_id
          ? `${ruleChainNameMap[detailTask.rule_chain_id] || detailTask.rule_chain_id}（${
              detailTask.rule_chain_id
            }）`
          : '—',
      ],
      ['最近执行ID', detailTask.last_run_id || '—'],
      ['父任务ID', detailTask.parent_id ?? '—'],
      ['创建时间', detailTask.created_at || '—'],
      ['更新时间', detailTask.updated_at || '—'],
      ['描述', detailTask.description || '—'],
    ];
    return rows;
  }, [detailTask, ruleChainNameMap]);

  // 获取任务列表（表格分页 / 看板大批量）
  const fetchTasks = async (page = 1, size = pageSize) => {
    setLoading(true);
    try {
      const isKanban = viewMode === 'kanban';
      const pageSizeReq = isKanban ? (filters.status ? size : KANBAN_PAGE_SIZE) : size;
      const pageReq = isKanban ? 1 : page;
      const res = await listTasks({
        ...filters,
        page: pageReq,
        page_size: pageSizeReq,
      });
      setTasks(res.items);
      setTotal(res.total);
      if (!isKanban) setCurrentPage(page);
    } catch (e) {
      Toast.error('获取任务列表失败');
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  // 获取规则链列表（用于下拉选择）
  const fetchRuleChains = async () => {
    setRuleChainLoading(true);
    try {
      const res = await getRuleList({ page: 1, size: 500, disabled: false, root: true });
      const list = Array.isArray(res.items) ? res.items : [];
      const nameMap: Record<string, string> = {};
      const options = list
        .map((raw: any) => {
          const chain = raw?.ruleChain ?? raw;
          const id = String(chain?.id ?? '').trim();
          if (!id) return null;
          const name = String(chain?.name ?? '').trim();
          if (name) nameMap[id] = name;
          return { label: name ? `${name}（${id}）` : id, value: id };
        })
        .filter((item): item is { label: string; value: string } => item !== null);
      setRuleChainOptions(options);
      setRuleChainNameMap(nameMap);
    } catch (e) {
      console.error('获取规则链列表失败', e);
    } finally {
      setRuleChainLoading(false);
    }
  };

  useEffect(() => {
    setCurrentPage(1);
    void fetchTasks(1, pageSize);
  }, [filters, viewMode]);

  useEffect(() => {
    try {
      localStorage.setItem(VIEW_MODE_KEY, viewMode);
    } catch {
      // ignore
    }
  }, [viewMode]);

  // 初始化时获取规则链列表
  useEffect(() => {
    void fetchRuleChains();
  }, []);

  // 看板模式每5分钟自动刷新（按当前筛选条件）
  useEffect(() => {
    if (viewMode !== 'kanban') return;
    const interval = setInterval(() => {
      void fetchTasks(1, pageSize);
    }, 300000);
    return () => clearInterval(interval);
  }, [viewMode, filters, pageSize]);

  const tasksByStatus = useMemo(() => {
    const m = new Map<TaskStatus, TaskItem[]>();
    for (const s of [
      TaskStatus.PENDING,
      TaskStatus.PROCESSING,
      TaskStatus.COMPLETED,
      TaskStatus.FAILED,
    ]) {
      m.set(s, []);
    }
    for (const t of tasks) {
      const key = m.has(t.status) ? t.status : TaskStatus.PENDING;
      m.get(key)!.push(t);
    }
    return m;
  }, [tasks]);

  // 计算各列任务数，用于自适应列宽
  const maxTaskCount = useMemo(() => {
    let max = 0;
    tasksByStatus.forEach((list) => {
      if (list.length > max) max = list.length;
    });
    return max;
  }, [tasksByStatus]);

  // 是否为只读状态（处理中/已完成/失败时仅描述可编辑）
  const isReadonlyStatus = (status: TaskStatus): boolean =>
    status === TaskStatus.PROCESSING ||
    status === TaskStatus.COMPLETED ||
    status === TaskStatus.FAILED;

  // 打开新增/编辑弹窗（initValues + key 强制重挂载，避免 Semi Form 忽略受控 value）
  const openModal = (type: 'create' | 'edit', task?: TaskItem) => {
    setModalType(type);
    taskFormApiRef.current = null;
    if (type === 'edit' && task) {
      setEditingTask(task);
      setFormInitSnapshot({
        name: task.name,
        priority: Number(task.priority),
        type: Number(task.type) as TaskType,
        status: Number(task.status) as TaskStatus,
        handler_user_id: task.handler_user_id ?? '',
        description: task.description ?? '',
        rule_chain_id: task.rule_chain_id ?? '',
      });
    } else {
      setEditingTask(null);
      setFormInitSnapshot({
        name: '',
        priority: 99,
        type: TaskType.OTHER,
        handler_user_id: '',
        description: '',
        rule_chain_id: '',
      });
    }
    setFormModalKey((k) => k + 1);
    setModalVisible(true);
  };

  const openDetail = async (task: TaskItem) => {
    setDetailVisible(true);
    setDetailLoading(true);
    setDetailTask(task);
    try {
      const { item } = await getTask(task.id);
      setDetailTask(item);
    } catch {
      Toast.warning('拉取详情失败，已显示列表中的数据');
    } finally {
      setDetailLoading(false);
    }
  };

  const closeDetail = () => {
    setDetailVisible(false);
    setDetailTask(null);
  };

  // 关闭弹窗
  const closeModal = () => {
    setModalVisible(false);
    setFormInitSnapshot({});
    setEditingTask(null);
    taskFormApiRef.current = null;
  };

  // 提交表单（从 FormApi 取数）
  const handleSubmit = async () => {
    const api = taskFormApiRef.current;
    if (!api) {
      Toast.warning('表单未就绪，请稍后重试');
      return;
    }
    try {
      await api.validate();
    } catch {
      return;
    }
    const v = api.getValues() as Record<string, unknown>;
    const name = String(v.name ?? '').trim();
    const priority = Number(v.priority);
    const type = Number(v.type) as TaskType;
    if (!name || !Number.isFinite(priority) || type == null || Number.isNaN(type)) {
      Toast.warning('请填写必填项');
      return;
    }
    const isPending = editingTask?.status === TaskStatus.PENDING;
    const readonly =
      modalType === 'edit' && editingTask != null && isReadonlyStatus(editingTask.status);
    const ruleChainId = isPending ? String(v.rule_chain_id ?? '').trim() : '';
    const isClearingRuleChain = isPending && ruleChainId === '' && editingTask?.rule_chain_id;
    const payload: Record<string, unknown> = {
      description: String(v.description ?? ''),
    };
    if (!readonly) {
      payload.name = name;
      payload.priority = priority;
      payload.type = type;
      payload.handler_user_id = String(v.handler_user_id ?? '').trim();
    }
    if (isPending && ruleChainId) {
      payload.rule_chain_id = ruleChainId;
    } else if (isClearingRuleChain) {
      payload.clear_rule_chain_id = true;
    }
    setSubmitting(true);
    try {
      if (modalType === 'create') {
        await createTask(payload as unknown as CreateTaskParams);
        Toast.success('创建成功');
      } else if (modalType === 'edit' && editingTask) {
        await updateTask(editingTask.id, {
          ...payload,
          status: Number(v.status ?? editingTask.status) as TaskStatus,
        });
        Toast.success('更新成功');
      }
      closeModal();
      void fetchTasks(viewMode === 'kanban' ? 1 : currentPage, pageSize);
    } catch (e) {
      Toast.error(modalType === 'create' ? '创建失败' : '更新失败');
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  // 删除任务
  const handleDelete = async (id: number) => {
    try {
      await deleteTask(id);
      Toast.success('删除成功');
      void fetchTasks(
        viewMode === 'kanban'
          ? 1
          : currentPage === 1
          ? 1
          : tasks.length === 1
          ? currentPage - 1
          : currentPage,
        pageSize
      );
    } catch (e) {
      Toast.error('删除失败');
      console.error(e);
    }
  };

  // 执行任务关联的规则链
  const handleExecuteRuleChain = async (task: TaskItem) => {
    setExecuteLoading(true);
    try {
      const res = await executeTaskRuleChain(task.id);
      Toast.success(res.message || '规则链执行已触发');
      void closeDetail();
      void fetchTasks(viewMode === 'kanban' ? 1 : currentPage, pageSize);
    } catch (e) {
      Toast.error('执行规则链失败');
      console.error(e);
    } finally {
      setExecuteLoading(false);
    }
  };

  // 打开创建子任务弹窗（先拉取最新任务数据）
  const handleCreateChildTask = async (task: TaskItem) => {
    try {
      const { item } = await getTask(task.id);
      setDetailTask(item);
    } catch {
      setDetailTask(task);
    }
    setCreateChildSuffix('');
    setCreateChildVisible(true);
  };

  // 确认创建子任务
  const confirmCreateChildTask = async () => {
    if (!detailTask) return;
    setCreateChildSubmitting(true);
    try {
      const params: CreateChildTaskParams = {};
      if (createChildSuffix.trim()) {
        params.name_suffix = createChildSuffix.trim();
      }
      await createChildTask(detailTask.id, params);
      Toast.success('子任务创建成功');
      setCreateChildVisible(false);
      void closeDetail();
    } catch (e) {
      Toast.error('创建子任务失败');
      console.error(e);
    } finally {
      setCreateChildSubmitting(false);
    }
  };

  // 查看子任务列表
  const handleViewChildTasks = async (task: TaskItem) => {
    setChildTasksVisible(true);
    setChildTasksLoading(true);
    try {
      const res = await listChildTasks(task.id, { page: 1, page_size: 20 });
      setChildTasks(res.items);
      setChildTasksTotal(res.total);
      setChildTasksPage(1);
    } catch (e) {
      Toast.error('获取子任务列表失败');
      console.error(e);
    } finally {
      setChildTasksLoading(false);
    }
  };

  // 加载更多子任务
  const loadMoreChildTasks = async () => {
    if (!detailTask || childTasksLoading) return;
    setChildTasksLoading(true);
    try {
      const nextPage = childTasksPage + 1;
      const res = await listChildTasks(detailTask.id, { page: nextPage, page_size: 20 });
      setChildTasks((prev) => [...prev, ...res.items]);
      setChildTasksTotal(res.total);
      setChildTasksPage(nextPage);
    } catch (e) {
      Toast.error('加载更多失败');
      console.error(e);
    } finally {
      setChildTasksLoading(false);
    }
  };

  // 查看任务执行日志（处理中/已完成/失败状态）
  const handleViewRunLog = async (task: TaskItem) => {
    if (!task.rule_chain_id) {
      Toast.warning('该任务未关联规则链，无执行日志');
      return;
    }
    setRunLogViewerLoading(true);
    try {
      let latest: any;

      // 优先通过 last_run_id 精确查找
      if (task.last_run_id) {
        const precise = await fetchRunLogs(task.last_run_id);
        if (precise) {
          latest = precise;
        }
      }

      // 精确查找失败或无 last_run_id，回退到按 task_id 匹配
      if (!latest) {
        const data = await requestJSON<{ items: any[]; total?: number }>('/logs/runs', {
          params: { chainId: task.rule_chain_id, size: 50, page: 1 },
        });
        const items = Array.isArray(data?.items) ? data.items : [];
        if (items.length === 0) {
          Toast.info('暂无该规则链的执行日志');
          return;
        }
        const taskMatch = items.find((r: any) => {
          try {
            const md = typeof r?.metadata === 'string' ? JSON.parse(r.metadata) : r?.metadata;
            return md?.task_id === String(task.id);
          } catch {
            return false;
          }
        });
        latest = taskMatch || items[0];
      }

      const dslRoot = latest?.ruleChain;
      const doc = buildDocumentFromRuleChainJSON(
        dslRoot && typeof dslRoot === 'object' ? dslRoot : ({ ruleChain: {}, metadata: {} } as any)
      ) as any;
      setRunLogViewerDoc(doc);
      setRunLogViewerLogs({
        list: Array.isArray(latest?.logs) ? latest.logs : [],
        startTs: latest?.startTs,
        endTs: latest?.endTs,
      });
      setRunLogViewerOpen(true);
    } catch (e) {
      Toast.error('获取执行日志失败');
      console.error(e);
    } finally {
      setRunLogViewerLoading(false);
    }
  };

  // 表格列配置
  const columns = [
    {
      title: '任务ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '任务名称',
      dataIndex: 'name',
      width: 200,
      ellipsis: true,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      width: 100,
      render: (val: number) => <Tag color={priorityTagColor(val)}>{val}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (val: TaskStatus) => {
        const option = taskStatusOptions.find((o) => o.value === val);
        return option ? <Tag color={taskStatusTagColor(val)}>{option.label}</Tag> : val;
      },
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 120,
      render: (val: TaskType) => {
        const option = taskTypeOptions.find((o) => o.value === val);
        return option ? <Tag color={taskTypeTagColor(val)}>{option.label}</Tag> : val;
      },
    },
    {
      title: '处理人',
      dataIndex: 'handler_user_id',
      width: 120,
    },
    {
      title: '规则链',
      dataIndex: 'rule_chain_id',
      width: 180,
      render: (val: string) =>
        val ? (
          <Typography.Text size="small" type="tertiary" ellipsis={{ showTooltip: true }}>
            {ruleChainNameMap[val] || val}
          </Typography.Text>
        ) : (
          '—'
        ),
    },
    {
      title: '父任务ID',
      dataIndex: 'parent_id',
      width: 100,
      render: (val: number) => (val ? String(val) : '—'),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      width: 180,
    },
    {
      title: '描述',
      dataIndex: 'description',
      width: 150,
      ellipsis: true,
    },
    {
      title: '操作',
      width: 320,
      render: (_value: unknown, record: TaskItem) => (
        <Space spacing={4}>
          {record.status === TaskStatus.PENDING && record.rule_chain_id && (
            <Popconfirm
              title="执行规则链"
              content="确定要触发执行关联的规则链吗？任务状态将变为处理中。"
              onConfirm={() => void handleExecuteRuleChain(record)}
              okText="确定"
              cancelText="取消"
            >
              <Button size="small" theme="borderless" type="warning">
                执行
              </Button>
            </Popconfirm>
          )}
          <Button
            icon={<IconInfoCircle />}
            size="small"
            theme="borderless"
            onClick={() => void openDetail(record)}
          >
            详情
          </Button>
          <Button
            icon={<IconEdit />}
            size="small"
            theme="borderless"
            onClick={() => openModal('edit', record)}
          >
            编辑
          </Button>
          {(record.status === TaskStatus.COMPLETED || record.status === TaskStatus.FAILED) && (
            <Button
              size="small"
              theme="borderless"
              type="secondary"
              onClick={() => void handleCreateChildTask(record)}
            >
              子任务
            </Button>
          )}
          {record.rule_chain_id && record.status !== TaskStatus.PENDING && (
            <Button
              size="small"
              theme="borderless"
              loading={runLogViewerLoading}
              onClick={() => void handleViewRunLog(record)}
            >
              日志
            </Button>
          )}
          <Popconfirm
            title="确认删除"
            content="确定删除该任务？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button icon={<IconDelete />} size="small" theme="borderless" type="danger">
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div
      style={{
        padding: '24px',
        width: '100%',
        boxSizing: 'border-box',
        height: '100%',
        minHeight: 0,
        overflow: 'auto',
      }}
    >
      <Card style={{ minHeight: '100%' }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '20px',
            flexWrap: 'wrap',
            gap: 12,
          }}
        >
          <Typography.Title heading={3} style={{ margin: 0 }}>
            任务看板
          </Typography.Title>
          <Space wrap>
            <Radio.Group
              type="button"
              buttonSize="middle"
              value={viewMode}
              onChange={(e) => {
                const v = (e?.target?.value ?? 'table') as ViewMode;
                setViewMode(v);
              }}
            >
              <Radio value="table">表格</Radio>
              <Radio value="kanban">看板</Radio>
            </Radio.Group>
            <Button icon={<IconPlus />} type="primary" onClick={() => openModal('create')}>
              新增任务
            </Button>
          </Space>
        </div>

        {/* 筛选：勿在 Form 内对带 field 的控件再绑 value/onChange，会与 Form 内部状态冲突导致不回显 */}
        <div
          style={{
            marginBottom: '24px',
            display: 'flex',
            gap: '16px',
            alignItems: 'flex-end',
            flexWrap: 'wrap',
          }}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <Typography.Text type="tertiary" size="small">
              处理人
            </Typography.Text>
            <Input
              value={filters.handler_user_id ?? ''}
              onChange={(val) => setFilters({ ...filters, handler_user_id: val })}
              placeholder="请输入处理人ID"
              style={{ width: 200 }}
            />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <Typography.Text type="tertiary" size="small">
              状态
            </Typography.Text>
            <Select
              value={filters.status}
              onChange={(val) =>
                setFilters({
                  ...filters,
                  status: (val === '' || val == null ? undefined : val) as TaskStatus | undefined,
                })
              }
              placeholder="全部状态"
              style={{ width: 160 }}
              showClear
              optionList={taskStatusOptions.map((o) => ({ label: o.label, value: o.value }))}
            />
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <Typography.Text type="tertiary" size="small">
              类型
            </Typography.Text>
            <Select
              value={filters.type}
              onChange={(val) =>
                setFilters({
                  ...filters,
                  type: (val === '' || val == null ? undefined : val) as TaskType | undefined,
                })
              }
              placeholder="全部类型"
              style={{ width: 160 }}
              showClear
              optionList={taskTypeOptions.map((o) => ({ label: o.label, value: o.value }))}
            />
          </div>
          <Button onClick={() => setFilters({})}>重置筛选</Button>
        </div>

        {viewMode === 'table' ? (
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
              onPageChange: (page: number) => {
                void fetchTasks(page, pageSize);
              },
              onPageSizeChange: (size: number) => {
                setPageSize(size);
                setCurrentPage(1);
                void fetchTasks(1, size);
              },
            }}
          />
        ) : (
          <Spin spinning={loading}>
            <div style={{ marginBottom: 8 }}>
              <Typography.Text type="tertiary" size="small">
                看板模式下列表最多加载 {KANBAN_PAGE_SIZE} 条；变更状态请在卡片上点「编辑」。
                {total > tasks.length
                  ? `（当前 ${tasks.length} / 共 ${total} 条）`
                  : `（共 ${total} 条）`}
              </Typography.Text>
            </div>
            <div
              style={{
                display: 'flex',
                gap: 12,
                alignItems: 'stretch',
                overflowX: 'auto',
                paddingBottom: 8,
                minHeight: 320,
              }}
            >
              {KANBAN_COLUMNS.map((col) => {
                const list = tasksByStatus.get(col.status) ?? [];
                const colFlex =
                  maxTaskCount > 0 && list.length / maxTaskCount > 0.3
                    ? '1.5 1 300px'
                    : maxTaskCount > 0 && list.length / maxTaskCount > 0.1
                    ? '1.2 1 260px'
                    : '1 1 240px';
                return (
                  <div
                    key={col.status}
                    style={{
                      flex: colFlex,
                      minWidth: 240,
                      maxWidth: 400,
                      background: col.columnBg,
                      borderRadius: 12,
                      padding: '14px 14px 16px',
                      border: '1px solid rgba(28,31,35,0.08)',
                      boxShadow: '0 2px 8px rgba(28,31,35,0.04)',
                    }}
                  >
                    {/* 列头部 */}
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        marginBottom: 14,
                        padding: '0 4px',
                      }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <span style={{ color: col.dot, fontSize: 18 }}>●</span>
                        <Typography.Text strong style={{ fontSize: 15 }}>
                          {col.title}
                        </Typography.Text>
                      </div>
                      <div
                        style={{
                          background: col.dot,
                          color: '#fff',
                          borderRadius: 10,
                          padding: '2px 10px',
                          fontSize: 12,
                          fontWeight: 600,
                          minWidth: 28,
                          textAlign: 'center',
                        }}
                      >
                        {list.length}
                      </div>
                    </div>
                    {/* 卡片列表 */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                      {list.map((task) => {
                        const typeOpt = taskTypeOptions.find((o) => o.value === task.type);
                        const pl = priorityLabel(task.priority);
                        return (
                          <div
                            key={task.id}
                            style={{
                              background: '#fff',
                              borderRadius: 8,
                              border: '1px solid rgba(28,31,35,0.08)',
                              cursor: 'pointer',
                              transition: 'all 0.2s ease',
                              padding: '12px 14px',
                            }}
                            onClick={() => void openDetail(task)}
                            onMouseEnter={(e) => {
                              e.currentTarget.style.boxShadow = '0 4px 12px rgba(28,31,35,0.12)';
                              e.currentTarget.style.borderColor = 'rgba(28,31,35,0.15)';
                            }}
                            onMouseLeave={(e) => {
                              e.currentTarget.style.boxShadow = 'none';
                              e.currentTarget.style.borderColor = 'rgba(28,31,35,0.08)';
                            }}
                          >
                            {/* 任务名称行 */}
                            <div
                              style={{
                                display: 'flex',
                                alignItems: 'flex-start',
                                gap: 8,
                                marginBottom: 8,
                              }}
                            >
                              <span
                                style={{
                                  width: 8,
                                  height: 8,
                                  borderRadius: '50%',
                                  background: col.dot,
                                  marginTop: 5,
                                  flexShrink: 0,
                                }}
                              />
                              <Typography.Text
                                strong
                                style={{
                                  fontSize: 14,
                                  lineHeight: 1.5,
                                  wordBreak: 'break-word',
                                  flex: 1,
                                }}
                              >
                                {task.name}
                              </Typography.Text>
                            </div>
                            {/* 标签行 */}
                            <div
                              style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 8 }}
                            >
                              {typeOpt && (
                                <Tag size="small" color={taskTypeTagColor(task.type)}>
                                  {typeOpt.label}
                                </Tag>
                              )}
                              {pl ? (
                                <Tag size="small" color={pl === 'P0' ? 'red' : 'orange'}>
                                  {pl}
                                </Tag>
                              ) : null}
                              {task.parent_id && (
                                <Tag size="small" color="blue">
                                  子任务
                                </Tag>
                              )}
                              {task.rule_chain_id && (
                                <Tag
                                  size="small"
                                  color="violet"
                                  style={{
                                    maxWidth: 160,
                                    overflow: 'hidden',
                                    textOverflow: 'ellipsis',
                                    whiteSpace: 'nowrap',
                                  }}
                                >
                                  {ruleChainNameMap[task.rule_chain_id] || task.rule_chain_id}
                                </Tag>
                              )}
                            </div>
                            {/* 底部信息栏 */}
                            <div
                              style={{
                                display: 'flex',
                                justifyContent: 'space-between',
                                alignItems: 'center',
                              }}
                            >
                              <div
                                style={{
                                  display: 'flex',
                                  alignItems: 'center',
                                  gap: 4,
                                  color: 'var(--semi-color-text-2)',
                                  fontSize: 12,
                                }}
                              >
                                <IconUser style={{ opacity: 0.6 }} />
                                <Typography.Text size="small" type="tertiary">
                                  {task.handler_user_id?.trim() || '—'}
                                </Typography.Text>
                              </div>
                              <Typography.Text type="tertiary" size="small">
                                {relativeDurationCn(task.created_at)}
                              </Typography.Text>
                            </div>
                            {/* 操作按钮（点击不冒泡） */}
                            <div
                              style={{
                                marginTop: 10,
                                paddingTop: 8,
                                borderTop: '1px solid rgba(28,31,35,0.06)',
                                display: 'flex',
                                gap: 4,
                                flexWrap: 'wrap',
                              }}
                              onClick={(e) => e.stopPropagation()}
                            >
                              {task.status === TaskStatus.PENDING && task.rule_chain_id && (
                                <Popconfirm
                                  title="执行规则链"
                                  content="确定要触发执行关联的规则链吗？任务状态将变为处理中。"
                                  onConfirm={() => void handleExecuteRuleChain(task)}
                                  okText="确定"
                                  cancelText="取消"
                                >
                                  <Button size="small" theme="borderless" type="warning">
                                    执行
                                  </Button>
                                </Popconfirm>
                              )}
                              <Button
                                icon={<IconEdit />}
                                size="small"
                                theme="borderless"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  openModal('edit', task);
                                }}
                              >
                                编辑
                              </Button>
                              {(task.status === TaskStatus.COMPLETED ||
                                task.status === TaskStatus.FAILED) && (
                                <Button
                                  size="small"
                                  theme="borderless"
                                  type="secondary"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    void handleCreateChildTask(task);
                                  }}
                                >
                                  子任务
                                </Button>
                              )}
                              {task.rule_chain_id && task.status !== TaskStatus.PENDING && (
                                <Button
                                  size="small"
                                  theme="borderless"
                                  loading={runLogViewerLoading}
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    void handleViewRunLog(task);
                                  }}
                                >
                                  日志
                                </Button>
                              )}
                              <Popconfirm
                                title="确认删除"
                                content="确定删除该任务？"
                                okText="确定"
                                cancelText="取消"
                                onConfirm={(e) => {
                                  e?.stopPropagation();
                                  handleDelete(task.id);
                                }}
                              >
                                <Button
                                  icon={<IconDelete />}
                                  size="small"
                                  theme="borderless"
                                  type="danger"
                                >
                                  删除
                                </Button>
                              </Popconfirm>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                );
              })}
            </div>
          </Spin>
        )}
      </Card>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={modalType === 'create' ? '新增任务' : '编辑任务'}
        visible={modalVisible}
        onCancel={closeModal}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
            <Button onClick={closeModal}>取消</Button>
            <Button type="primary" onClick={() => void handleSubmit()} loading={submitting}>
              {modalType === 'create' ? '创建' : '保存'}
            </Button>
          </div>
        }
        width={600}
      >
        <Form
          key={formModalKey}
          initValues={formInitSnapshot}
          getFormApi={(api) => {
            taskFormApiRef.current = api;
          }}
          labelPosition="left"
          labelAlign="right"
          labelWidth={100}
        >
          <Form.Input
            field="name"
            label="任务名称"
            placeholder="请输入任务名称"
            rules={[{ required: true, message: '请输入任务名称' }]}
            disabled={
              modalType === 'edit' && editingTask != null && isReadonlyStatus(editingTask.status)
            }
          />
          <Form.Select
            field="priority"
            label="优先级"
            placeholder="请选择优先级"
            style={{ width: '100%' }}
            rules={[{ required: true, message: '请选择优先级' }]}
            optionList={priorityOptions.map((o) => ({ label: o.label, value: o.value }))}
            disabled={
              modalType === 'edit' && editingTask != null && isReadonlyStatus(editingTask.status)
            }
          />
          {modalType === 'edit' && (
            <Form.Select
              field="status"
              label="任务状态"
              placeholder="请选择任务状态"
              style={{ width: '100%' }}
              optionList={taskStatusOptions.map((o) => ({ label: o.label, value: o.value }))}
              disabled
            />
          )}
          <Form.Select
            field="type"
            label="任务类型"
            placeholder="请选择任务类型"
            style={{ width: '100%' }}
            rules={[{ required: true, message: '请选择任务类型' }]}
            optionList={taskTypeOptions.map((o) => ({ label: o.label, value: o.value }))}
            disabled={
              modalType === 'edit' && editingTask != null && isReadonlyStatus(editingTask.status)
            }
          />
          <Form.Input
            field="handler_user_id"
            label="处理人ID"
            placeholder="请输入处理人ID"
            disabled={
              modalType === 'edit' && editingTask != null && isReadonlyStatus(editingTask.status)
            }
          />
          <Form.TextArea
            field="description"
            label="任务描述"
            placeholder="请输入任务描述"
            rows={4}
          />
          {modalType === 'edit' && editingTask?.status === TaskStatus.PENDING ? (
            <Form.Select
              field="rule_chain_id"
              label="关联规则链"
              placeholder="请选择关联的规则链"
              style={{ width: '100%' }}
              loading={ruleChainLoading}
              optionList={ruleChainOptions}
              showClear
              extraText="不选择则不关联规则链"
            />
          ) : modalType === 'edit' ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <Typography.Text type="tertiary" style={{ width: 100, textAlign: 'right' }}>
                关联规则链
              </Typography.Text>
              <Typography.Text>{editingTask?.rule_chain_id || '—'}</Typography.Text>
            </div>
          ) : (
            <Form.Select
              field="rule_chain_id"
              label="关联规则链"
              placeholder="请选择关联的规则链"
              style={{ width: '100%' }}
              loading={ruleChainLoading}
              optionList={ruleChainOptions}
              showClear
              extraText="不选择则不关联规则链"
            />
          )}
          {modalType === 'edit' &&
            editingTask != null &&
            (editingTask.status === TaskStatus.COMPLETED ||
              editingTask.status === TaskStatus.FAILED) && (
              <div style={{ marginTop: 8 }}>
                <Button type="secondary" onClick={() => void handleCreateChildTask(editingTask!)}>
                  创建子任务
                </Button>
              </div>
            )}
        </Form>
      </Modal>

      {/* 详情（只读） */}
      <Modal
        title="任务详情"
        visible={detailVisible}
        onCancel={closeDetail}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            {detailTask?.status === TaskStatus.PENDING && detailTask?.rule_chain_id && (
              <Button
                type="warning"
                onClick={() => void handleExecuteRuleChain(detailTask!)}
                loading={executeLoading}
              >
                执行规则链
              </Button>
            )}
            {(detailTask?.status === TaskStatus.COMPLETED ||
              detailTask?.status === TaskStatus.FAILED) && (
              <Button type="secondary" onClick={() => void handleCreateChildTask(detailTask!)}>
                创建子任务
              </Button>
            )}
            {detailTask?.rule_chain_id && detailTask.status !== TaskStatus.PENDING && (
              <Button
                type="tertiary"
                loading={runLogViewerLoading}
                onClick={() => void handleViewRunLog(detailTask)}
              >
                查看执行日志
              </Button>
            )}
            {detailTask && (
              <Button type="tertiary" onClick={() => void handleViewChildTasks(detailTask!)}>
                查看子任务
              </Button>
            )}
            <Button type="primary" onClick={closeDetail}>
              关闭
            </Button>
          </div>
        }
        width={600}
      >
        <Spin spinning={detailLoading}>
          {detailTask && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              {detailRows.map(([label, val]) => (
                <div
                  key={String(label)}
                  style={{
                    display: 'grid',
                    gridTemplateColumns: '100px 1fr',
                    gap: 12,
                    alignItems: 'start',
                  }}
                >
                  <Typography.Text type="tertiary">{label}</Typography.Text>
                  <Typography.Text style={{ wordBreak: 'break-word' }}>
                    {String(val)}
                  </Typography.Text>
                </div>
              ))}
            </div>
          )}
        </Spin>
      </Modal>

      {/* 子任务列表弹窗 */}
      <Modal
        title="子任务列表"
        visible={childTasksVisible}
        onCancel={() => setChildTasksVisible(false)}
        footer={
          <Button type="primary" onClick={() => setChildTasksVisible(false)}>
            关闭
          </Button>
        }
        width={800}
      >
        <Spin spinning={childTasksLoading}>
          {childTasks.length > 0 ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {childTasks.map((child) => (
                <div
                  key={child.id}
                  style={{
                    border: '1px solid var(--semi-border-color)',
                    borderRadius: 8,
                    padding: '12px 16px',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                  }}
                >
                  <div>
                    <Typography.Text strong>{child.name}</Typography.Text>
                    <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
                      <Tag size="small" color={taskStatusTagColor(child.status)}>
                        {taskStatusOptions.find((o) => o.value === child.status)?.label ?? '未知'}
                      </Tag>
                      {child.rule_chain_id && (
                        <Typography.Text type="tertiary" size="small">
                          规则链: {child.rule_chain_id}
                        </Typography.Text>
                      )}
                    </div>
                  </div>
                  <Space>
                    <Button
                      size="small"
                      theme="borderless"
                      onClick={() => {
                        setChildTasksVisible(false);
                        void openDetail(child);
                      }}
                    >
                      详情
                    </Button>
                  </Space>
                </div>
              ))}
              {childTasksTotal > childTasks.length && (
                <Button
                  style={{ alignSelf: 'center', marginTop: 8 }}
                  onClick={() => void loadMoreChildTasks()}
                >
                  加载更多 ({childTasks.length}/{childTasksTotal})
                </Button>
              )}
            </div>
          ) : (
            <Typography.Text type="tertiary">暂无子任务</Typography.Text>
          )}
        </Spin>
      </Modal>

      {/* 创建子任务弹窗 */}
      <Modal
        title="创建子任务"
        visible={createChildVisible}
        onCancel={() => setCreateChildVisible(false)}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => setCreateChildVisible(false)}>取消</Button>
            <Button
              type="primary"
              onClick={() => void confirmCreateChildTask()}
              loading={createChildSubmitting}
            >
              创建
            </Button>
          </div>
        }
        width={480}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {detailTask && (
            <Typography.Text>
              父任务: <strong>{detailTask.name}</strong>（将继承所有内容）
            </Typography.Text>
          )}
          <Input
            prefix="子任务名称后缀"
            placeholder="默认为 '-子任务'"
            value={createChildSuffix}
            onChange={(val) => setCreateChildSuffix(val)}
          />
          <Typography.Text type="tertiary" size="small">
            说明：子任务将继承父任务的名称、优先级、类型、描述、规则链关联等全部内容
          </Typography.Text>
        </div>
      </Modal>

      {/* 规则链执行日志查看器 */}
      <Modal
        visible={runLogViewerOpen}
        title="执行日志查看"
        onCancel={() => setRunLogViewerOpen(false)}
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
            initialDoc={runLogViewerDoc}
            showTopToolbar={true}
            readonly={true}
            initialLogs={runLogViewerLogs}
            openRunPanel={false}
          />
        </div>
      </Modal>
    </div>
  );
};

export default TaskBoardSection;
