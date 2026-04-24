import React, { useState, useEffect, useMemo, useRef } from 'react';

import {
  Table,
  Button,
  Form,
  Input,
  Select,
  TextArea,
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

import {
  listTasks,
  getTask,
  createTask,
  updateTask,
  deleteTask,
  TaskItem,
  TaskStatus,
  TaskType,
  taskStatusOptions,
  taskTypeOptions,
  priorityOptions,
  CreateTaskParams,
  UpdateTaskParams,
} from '../../services/api-task';

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
    columnBg: 'rgba(134, 144, 156, 0.08)',
    tagTone: 'grey',
  },
  {
    status: TaskStatus.PROCESSING,
    title: '处理中',
    dot: '#165dff',
    columnBg: 'rgba(22, 93, 255, 0.06)',
    tagTone: 'blue',
  },
  {
    status: TaskStatus.COMPLETED,
    title: '已完成',
    dot: '#00b578',
    columnBg: 'rgba(0, 181, 120, 0.06)',
    tagTone: 'green',
  },
  {
    status: TaskStatus.FAILED,
    title: '失败',
    dot: '#f53f3f',
    columnBg: 'rgba(245, 63, 63, 0.06)',
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
  const taskFormApiRef = useRef<{ validate: () => Promise<void>; getValues: () => Record<string, unknown> } | null>(
    null
  );
  const [submitting, setSubmitting] = useState(false);

  const [detailVisible, setDetailVisible] = useState(false);
  const [detailTask, setDetailTask] = useState<TaskItem | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

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
      });
    } else {
      setEditingTask(null);
      setFormInitSnapshot({
        name: '',
        priority: 99,
        type: TaskType.OTHER,
        handler_user_id: '',
        description: '',
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
    const payload = {
      name,
      priority,
      type,
      handler_user_id: String(v.handler_user_id ?? '').trim(),
      description: String(v.description ?? ''),
    };
    setSubmitting(true);
    try {
      if (modalType === 'create') {
        await createTask(payload as CreateTaskParams);
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
      render: (val: number) => (
        <Tag color={val <= 10 ? 'red' : val <= 30 ? 'orange' : 'grey'}>{val}</Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (val: TaskStatus) => {
        const option = taskStatusOptions.find((o) => o.value === val);
        return option ? <Tag color={option.color}>{option.label}</Tag> : val;
      },
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 120,
      render: (val: TaskType) => {
        const option = taskTypeOptions.find((o) => o.value === val);
        return option ? <Tag color={option.color}>{option.label}</Tag> : val;
      },
    },
    {
      title: '处理人',
      dataIndex: 'handler_user_id',
      width: 120,
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
      ellipsis: true,
    },
    {
      title: '操作',
      width: 160,
      render: (_, record: TaskItem) => (
        <Space>
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
          <Popconfirm
            title="确认删除"
            content="确定要删除这个任务吗？删除后不可恢复。"
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
    <div style={{ padding: '24px', width: '100%', boxSizing: 'border-box' }}>
      <Card>
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
              allowClear
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
              allowClear
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
              onPageChange: (page, size) => {
                setPageSize(size);
                void fetchTasks(page, size);
              },
            }}
          />
        ) : (
          <Spin spinning={loading}>
            <div style={{ marginBottom: 8 }}>
              <Typography.Text type="tertiary" size="small">
                看板模式下列表最多加载 {KANBAN_PAGE_SIZE} 条；变更状态请在卡片上点「编辑」。
                {total > tasks.length ? `（当前 ${tasks.length} / 共 ${total} 条）` : `（共 ${total} 条）`}
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
                const statusOpt = taskStatusOptions.find((o) => o.value === col.status);
                return (
                  <div
                    key={col.status}
                    style={{
                      flex: '1 1 240px',
                      minWidth: 240,
                      maxWidth: 360,
                      background: col.columnBg,
                      borderRadius: 10,
                      padding: '10px 10px 12px',
                      border: '1px solid rgba(28,31,35,0.06)',
                    }}
                  >
                    <div
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        marginBottom: 10,
                        padding: '0 4px',
                      }}
                    >
                      <Typography.Text strong style={{ fontSize: 14 }}>
                        <span style={{ color: col.dot, marginRight: 8 }}>●</span>
                        {col.title}
                      </Typography.Text>
                      <Typography.Text type="tertiary" size="small">
                        共 {list.length} 个
                      </Typography.Text>
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                      {list.map((task) => {
                        const typeOpt = taskTypeOptions.find((o) => o.value === task.type);
                        const pl = priorityLabel(task.priority);
                        return (
                          <div
                            key={task.id}
                            style={{
                              background: '#fff',
                              borderRadius: 8,
                              padding: '12px 12px 10px',
                              boxShadow: '0 1px 2px rgba(28,31,35,0.06)',
                              border: '1px solid rgba(28,31,35,0.06)',
                            }}
                          >
                            <div
                              style={{
                                display: 'flex',
                                alignItems: 'flex-start',
                                gap: 8,
                                marginBottom: 10,
                              }}
                            >
                              <span
                                style={{
                                  width: 6,
                                  height: 6,
                                  borderRadius: '50%',
                                  background: col.dot,
                                  marginTop: 6,
                                  flexShrink: 0,
                                }}
                              />
                              <Typography.Text
                                strong
                                style={{
                                  fontSize: 14,
                                  lineHeight: 1.45,
                                  wordBreak: 'break-word',
                                }}
                              >
                                {task.name}
                              </Typography.Text>
                            </div>
                            <div
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: 6,
                                marginBottom: 10,
                                color: 'var(--semi-color-text-2)',
                                fontSize: 12,
                              }}
                            >
                              <IconUser style={{ opacity: 0.65 }} />
                              <span>处理人</span>
                              <Typography.Text size="small">
                                {task.handler_user_id?.trim() ? task.handler_user_id : '—'}
                              </Typography.Text>
                            </div>
                            <div
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'space-between',
                                flexWrap: 'wrap',
                                gap: 8,
                              }}
                            >
                              <Space spacing={8}>
                                <Tag size="small" color={col.tagTone}>
                                  {statusOpt?.label ?? col.title}
                                </Tag>
                                <Typography.Text type="tertiary" size="small">
                                  {relativeDurationCn(task.created_at)}
                                </Typography.Text>
                              </Space>
                              <Space spacing={4}>
                                {typeOpt && (
                                  <Tag size="small" color={typeOpt.color}>
                                    {typeOpt.label}
                                  </Tag>
                                )}
                                {pl ? (
                                  <Tag size="small" color={pl === 'P0' ? 'red' : 'orange'}>
                                    {pl}
                                  </Tag>
                                ) : (
                                  <Typography.Text type="tertiary" size="small">
                                    暂无
                                  </Typography.Text>
                                )}
                              </Space>
                            </div>
                            <div
                              style={{
                                marginTop: 10,
                                paddingTop: 8,
                                borderTop: '1px solid rgba(28,31,35,0.06)',
                                display: 'flex',
                                justifyContent: 'flex-end',
                                gap: 4,
                                flexWrap: 'wrap',
                              }}
                            >
                              <Button
                                icon={<IconInfoCircle />}
                                size="small"
                                theme="borderless"
                                onClick={() => void openDetail(task)}
                              >
                                详情
                              </Button>
                              <Button
                                icon={<IconEdit />}
                                size="small"
                                theme="borderless"
                                onClick={() => openModal('edit', task)}
                              >
                                编辑
                              </Button>
                              <Popconfirm
                                title="确认删除"
                                content="确定删除该任务？"
                                okText="确定"
                                cancelText="取消"
                                onConfirm={() => handleDelete(task.id)}
                              >
                                <Button icon={<IconDelete />} size="small" theme="borderless" type="danger">
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
          />
          <Form.Select
            field="priority"
            label="优先级"
            placeholder="请选择优先级"
            style={{ width: '100%' }}
            rules={[{ required: true, message: '请选择优先级' }]}
            optionList={priorityOptions.map((o) => ({ label: o.label, value: o.value }))}
          />
          {modalType === 'edit' && (
            <Form.Select
              field="status"
              label="任务状态"
              placeholder="请选择任务状态"
              style={{ width: '100%' }}
              optionList={taskStatusOptions.map((o) => ({ label: o.label, value: o.value }))}
            />
          )}
          <Form.Select
            field="type"
            label="任务类型"
            placeholder="请选择任务类型"
            style={{ width: '100%' }}
            rules={[{ required: true, message: '请选择任务类型' }]}
            optionList={taskTypeOptions.map((o) => ({ label: o.label, value: o.value }))}
          />
          <Form.Input field="handler_user_id" label="处理人ID" placeholder="请输入处理人ID" />
          <Form.TextArea field="description" label="任务描述" placeholder="请输入任务描述" rows={4} />
        </Form>
      </Modal>

      {/* 详情（只读） */}
      <Modal
        title="任务详情"
        visible={detailVisible}
        onCancel={closeDetail}
        footer={
          <Button type="primary" onClick={closeDetail}>
            关闭
          </Button>
        }
        width={560}
      >
        <Spin spinning={detailLoading}>
          {detailTask && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              {(
                [
                  ['任务ID', detailTask.id],
                  ['任务名称', detailTask.name],
                  [
                    '状态',
                    taskStatusOptions.find((o) => o.value === detailTask.status)?.label ?? detailTask.status,
                  ],
                  [
                    '类型',
                    taskTypeOptions.find((o) => o.value === detailTask.type)?.label ?? detailTask.type,
                  ],
                  ['优先级', detailTask.priority],
                  ['处理人', detailTask.handler_user_id || '—'],
                  ['创建时间', detailTask.created_at || '—'],
                  ['更新时间', detailTask.updated_at || '—'],
                  ['描述', detailTask.description || '—'],
                ] as const
              ).map(([label, val]) => (
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
                  <Typography.Text style={{ wordBreak: 'break-word' }}>{String(val)}</Typography.Text>
                </div>
              ))}
            </div>
          )}
        </Spin>
      </Modal>
    </div>
  );
};

export default TaskBoardSection;
