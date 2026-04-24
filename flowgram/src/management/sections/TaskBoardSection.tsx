import React, { useState, useEffect } from 'react';

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
} from '@douyinfe/semi-ui';
import { IconPlus, IconEdit, IconDelete } from '@douyinfe/semi-icons';

import {
  listTasks,
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

const { Option } = Select;

export const TaskBoardSection: React.FC = () => {
  const [tasks, setTasks] = useState<TaskItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [pageSize, setPageSize] = useState(10);
  const [currentPage, setCurrentPage] = useState(1);
  const [filters, setFilters] = useState<{
    status?: TaskStatus;
    type?: TaskType;
    handler_user_id?: string;
  }>({});
  const [modalVisible, setModalVisible] = useState(false);
  const [modalType, setModalType] = useState<'create' | 'edit'>('create');
  const [editingTask, setEditingTask] = useState<TaskItem | null>(null);
  const [formValues, setFormValues] = useState<Partial<CreateTaskParams>>({});
  const [submitting, setSubmitting] = useState(false);

  // 获取任务列表
  const fetchTasks = async (page = 1, size = pageSize) => {
    setLoading(true);
    try {
      const res = await listTasks({
        ...filters,
        page,
        page_size: size,
      });
      setTasks(res.items);
      setTotal(res.total);
      setCurrentPage(page);
    } catch (e) {
      Toast.error('获取任务列表失败');
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTasks();
  }, [filters]);

  // 打开新增/编辑弹窗
  const openModal = (type: 'create' | 'edit', task?: TaskItem) => {
    setModalType(type);
    if (type === 'edit' && task) {
      setEditingTask(task);
      setFormValues({
        name: task.name,
        priority: task.priority,
        type: task.type,
        handler_user_id: task.handler_user_id,
        description: task.description,
      });
    } else {
      setEditingTask(null);
      setFormValues({
        priority: 99,
        type: TaskType.OTHER,
      });
    }
    setModalVisible(true);
  };

  // 关闭弹窗
  const closeModal = () => {
    setModalVisible(false);
    setFormValues({});
    setEditingTask(null);
  };

  // 提交表单
  const handleSubmit = async () => {
    if (!formValues.name || !formValues.priority || !formValues.type) {
      Toast.warning('请填写必填项');
      return;
    }
    setSubmitting(true);
    try {
      if (modalType === 'create') {
        await createTask(formValues as CreateTaskParams);
        Toast.success('创建成功');
      } else if (modalType === 'edit' && editingTask) {
        await updateTask(editingTask.id, formValues as UpdateTaskParams);
        Toast.success('更新成功');
      }
      closeModal();
      fetchTasks(currentPage);
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
      fetchTasks(currentPage === 1 ? 1 : tasks.length === 1 ? currentPage - 1 : currentPage);
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
    <div style={{ padding: '24px', width: '100%' }}>
      <Card>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: '24px',
          }}
        >
          <Typography.Title heading={3}>任务看板</Typography.Title>
          <Button icon={<IconPlus />} type="primary" onClick={() => openModal('create')}>
            新增任务
          </Button>
        </div>

        {/* 筛选区域 */}
        <div style={{ marginBottom: '24px', display: 'flex', gap: '16px', alignItems: 'flex-end' }}>
          <Form labelPosition="left" labelAlign="right" labelWidth={80}>
            <Form.Input
              field="handler_user_id"
              label="处理人"
              placeholder="请输入处理人ID"
              value={filters.handler_user_id}
              onChange={(val) => setFilters({ ...filters, handler_user_id: val })}
              style={{ width: 200 }}
            />
          </Form>
          <Form labelPosition="left" labelAlign="right" labelWidth={80}>
            <Form.Select
              field="status"
              label="状态"
              placeholder="全部状态"
              value={filters.status}
              onChange={(val) => setFilters({ ...filters, status: val as TaskStatus })}
              style={{ width: 160 }}
            >
              <Option value={null}>全部状态</Option>
              {taskStatusOptions.map((o) => (
                <Option key={o.value} value={o.value}>
                  {o.label}
                </Option>
              ))}
            </Form.Select>
          </Form>
          <Form labelPosition="left" labelAlign="right" labelWidth={80}>
            <Form.Select
              field="type"
              label="类型"
              placeholder="全部类型"
              value={filters.type}
              onChange={(val) => setFilters({ ...filters, type: val as TaskType })}
              style={{ width: 160 }}
            >
              <Option value={null}>全部类型</Option>
              {taskTypeOptions.map((o) => (
                <Option key={o.value} value={o.value}>
                  {o.label}
                </Option>
              ))}
            </Form.Select>
          </Form>
          <Button onClick={() => setFilters({})}>重置筛选</Button>
        </div>

        {/* 表格 */}
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
              fetchTasks(page, size);
            },
          }}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={modalType === 'create' ? '新增任务' : '编辑任务'}
        visible={modalVisible}
        onCancel={closeModal}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
            <Button onClick={closeModal}>取消</Button>
            <Button type="primary" onClick={handleSubmit} loading={submitting}>
              {modalType === 'create' ? '创建' : '保存'}
            </Button>
          </div>
        }
        width={600}
      >
        <Form labelPosition="left" labelAlign="right" labelWidth={100}>
          <Form.Input
            field="name"
            label="任务名称"
            placeholder="请输入任务名称"
            value={formValues.name}
            onChange={(val) => setFormValues({ ...formValues, name: val })}
            rules={[{ required: true, message: '请输入任务名称' }]}
          />
          <Form.Select
            field="priority"
            label="优先级"
            placeholder="请选择优先级"
            value={formValues.priority}
            onChange={(val) => setFormValues({ ...formValues, priority: val as number })}
            style={{ width: '100%' }}
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            {priorityOptions.map((o) => (
              <Option key={o.value} value={o.value}>
                {o.label}
              </Option>
            ))}
          </Form.Select>
          <Form.Select
            field="status"
            label="任务状态"
            placeholder="请选择任务状态"
            value={formValues.status || TaskStatus.PENDING}
            onChange={(val) => setFormValues({ ...formValues, status: val as TaskStatus })}
            style={{ width: '100%' }}
          >
            {taskStatusOptions.map((o) => (
              <Option key={o.value} value={o.value}>
                {o.label}
              </Option>
            ))}
          </Form.Select>
          <Form.Select
            field="type"
            label="任务类型"
            placeholder="请选择任务类型"
            value={formValues.type}
            onChange={(val) => setFormValues({ ...formValues, type: val as TaskType })}
            style={{ width: '100%' }}
            rules={[{ required: true, message: '请选择任务类型' }]}
          >
            {taskTypeOptions.map((o) => (
              <Option key={o.value} value={o.value}>
                {o.label}
              </Option>
            ))}
          </Form.Select>
          <Form.Input
            field="handler_user_id"
            label="处理人ID"
            placeholder="请输入处理人ID"
            value={formValues.handler_user_id}
            onChange={(val) => setFormValues({ ...formValues, handler_user_id: val })}
          />
          <Form.TextArea
            field="description"
            label="任务描述"
            placeholder="请输入任务描述"
            value={formValues.description}
            onChange={(val) => setFormValues({ ...formValues, description: val })}
            rows={4}
          />
        </Form>
      </Modal>
    </div>
  );
};

export default TaskBoardSection;
