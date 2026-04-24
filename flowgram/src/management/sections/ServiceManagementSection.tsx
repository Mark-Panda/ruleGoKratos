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
  listServices,
  createService,
  updateService,
  deleteService,
  ServiceItem,
  ServiceStatus,
  serviceStatusOptions,
  CreateServiceParams,
  UpdateServiceParams,
} from '../../services/api-service';

const { Option } = Select;

export const ServiceManagementSection: React.FC = () => {
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [pageSize, setPageSize] = useState(10);
  const [currentPage, setCurrentPage] = useState(1);
  const [filters, setFilters] = useState<{
    status?: ServiceStatus;
  }>({});
  const [modalVisible, setModalVisible] = useState(false);
  const [modalType, setModalType] = useState<'create' | 'edit'>('create');
  const [editingService, setEditingService] = useState<ServiceItem | null>(null);
  const [formValues, setFormValues] = useState<Partial<CreateServiceParams>>({});
  const [submitting, setSubmitting] = useState(false);

  // 获取服务列表
  const fetchServices = async (page = 1, size = pageSize) => {
    setLoading(true);
    try {
      const res = await listServices({
        ...filters,
        page,
        page_size: size,
      });
      setServices(res.items);
      setTotal(res.total);
      setCurrentPage(page);
    } catch (e) {
      Toast.error('获取服务列表失败');
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServices();
  }, [filters]);

  // 打开新增/编辑弹窗
  const openModal = (type: 'create' | 'edit', service?: ServiceItem) => {
    setModalType(type);
    if (type === 'edit' && service) {
      setEditingService(service);
      setFormValues({
        name: service.name,
        status: service.status,
        volc_log_service_id: service.volc_log_service_id,
        git_repo_url: service.git_repo_url,
        description: service.description,
      });
    } else {
      setEditingService(null);
      setFormValues({
        status: ServiceStatus.STOPPED,
      });
    }
    setModalVisible(true);
  };

  // 关闭弹窗
  const closeModal = () => {
    setModalVisible(false);
    setFormValues({});
    setEditingService(null);
  };

  // 提交表单
  const handleSubmit = async () => {
    if (!formValues.name || !formValues.status) {
      Toast.warning('请填写必填项');
      return;
    }
    setSubmitting(true);
    try {
      if (modalType === 'create') {
        await createService(formValues as CreateServiceParams);
        Toast.success('创建成功');
      } else if (modalType === 'edit' && editingService) {
        await updateService(editingService.id, formValues as UpdateServiceParams);
        Toast.success('更新成功');
      }
      closeModal();
      fetchServices(currentPage);
    } catch (e) {
      Toast.error(modalType === 'create' ? '创建失败' : '更新失败');
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  // 删除服务
  const handleDelete = async (id: number) => {
    try {
      await deleteService(id);
      Toast.success('删除成功');
      fetchServices(currentPage === 1 ? 1 : services.length === 1 ? currentPage - 1 : currentPage);
    } catch (e) {
      Toast.error('删除失败');
      console.error(e);
    }
  };

  // 表格列配置
  const columns = [
    {
      title: '服务ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '服务名称',
      dataIndex: 'name',
      width: 200,
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (val: ServiceStatus) => {
        const option = serviceStatusOptions.find((o) => o.value === val);
        return option ? <Tag color={option.color}>{option.label}</Tag> : val;
      },
    },
    {
      title: '火山日志ID',
      dataIndex: 'volc_log_service_id',
      width: 180,
    },
    {
      title: 'Git仓库地址',
      dataIndex: 'git_repo_url',
      width: 240,
      ellipsis: true,
      render: (val: string) =>
        val ? (
          <a href={val} target="_blank" rel="noopener noreferrer">
            {val}
          </a>
        ) : (
          '-'
        ),
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
      render: (_, record: ServiceItem) => (
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
            content="确定要删除这个服务吗？删除后不可恢复。"
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
          <Typography.Title heading={3}>服务管理</Typography.Title>
          <Button icon={<IconPlus />} type="primary" onClick={() => openModal('create')}>
            新增服务
          </Button>
        </div>

        {/* 筛选区域 */}
        <div style={{ marginBottom: '24px', display: 'flex', gap: '16px', alignItems: 'flex-end' }}>
          <Form labelPosition="left" labelAlign="right" labelWidth={80}>
            <Form.Select
              field="status"
              label="状态"
              placeholder="全部状态"
              value={filters.status}
              onChange={(val) => setFilters({ ...filters, status: val as ServiceStatus })}
              style={{ width: 160 }}
            >
              <Option value={null}>全部状态</Option>
              {serviceStatusOptions.map((o) => (
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
          dataSource={services}
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
              fetchServices(page, size);
            },
          }}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={modalType === 'create' ? '新增服务' : '编辑服务'}
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
        <Form labelPosition="left" labelAlign="right" labelWidth={120}>
          <Form.Input
            field="name"
            label="服务名称"
            placeholder="请输入服务名称"
            value={formValues.name}
            onChange={(val) => setFormValues({ ...formValues, name: val })}
            rules={[{ required: true, message: '请输入服务名称' }]}
          />
          <Form.Select
            field="status"
            label="服务状态"
            placeholder="请选择服务状态"
            value={formValues.status}
            onChange={(val) => setFormValues({ ...formValues, status: val as ServiceStatus })}
            style={{ width: '100%' }}
            rules={[{ required: true, message: '请选择服务状态' }]}
          >
            {serviceStatusOptions.map((o) => (
              <Option key={o.value} value={o.value}>
                {o.label}
              </Option>
            ))}
          </Form.Select>
          <Form.Input
            field="volc_log_service_id"
            label="火山日志服务ID"
            placeholder="请输入火山日志服务ID"
            value={formValues.volc_log_service_id}
            onChange={(val) => setFormValues({ ...formValues, volc_log_service_id: val })}
          />
          <Form.Input
            field="git_repo_url"
            label="Git仓库地址"
            placeholder="请输入Git仓库地址"
            value={formValues.git_repo_url}
            onChange={(val) => setFormValues({ ...formValues, git_repo_url: val })}
          />
          <Form.TextArea
            field="description"
            label="服务描述"
            placeholder="请输入服务描述"
            value={formValues.description}
            onChange={(val) => setFormValues({ ...formValues, description: val })}
            rows={4}
          />
        </Form>
      </Modal>
    </div>
  );
};

export default ServiceManagementSection;
