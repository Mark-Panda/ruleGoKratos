import React, { useState, useEffect, useRef } from 'react';

import type { FormApi } from '@douyinfe/semi-ui/lib/es/form/interface';
import {
  Table,
  Button,
  Form,
  Select,
  Modal,
  Space,
  Tag,
  Toast,
  Popconfirm,
  Card,
  Typography,
  Spin,
} from '@douyinfe/semi-ui';
import { IconPlus, IconEdit, IconDelete, IconInfoCircle } from '@douyinfe/semi-icons';

import {
  listServices,
  getService,
  saveServiceByName,
  updateService,
  deleteService,
  ServiceItem,
  ServiceStatus,
  serviceStatusOptions,
  CreateServiceParams,
  UpdateServiceParams,
} from '../../services/api-service';
import { serviceStatusTagColor } from './section-display';

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
  const [formInitSnapshot, setFormInitSnapshot] = useState<Record<string, unknown>>({});
  const [formModalKey, setFormModalKey] = useState(0);
  const serviceFormApiRef = useRef<FormApi<Record<string, unknown>> | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [detailVisible, setDetailVisible] = useState(false);
  const [detailService, setDetailService] = useState<ServiceItem | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

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
    void fetchServices(1, pageSize);
  }, [filters]);

  // 打开新增/编辑弹窗（initValues + key 重挂载，保证编辑回显）
  const openModal = (type: 'create' | 'edit', service?: ServiceItem) => {
    setModalType(type);
    serviceFormApiRef.current = null;
    if (type === 'edit' && service) {
      setEditingService(service);
      setFormInitSnapshot({
        name: service.name,
        status: Number(service.status) as ServiceStatus,
        volc_log_service_id: service.volc_log_service_id ?? '',
        git_repo_url: service.git_repo_url ?? '',
        description: service.description ?? '',
      });
    } else {
      setEditingService(null);
      setFormInitSnapshot({
        name: '',
        status: ServiceStatus.STOPPED,
        volc_log_service_id: '',
        git_repo_url: '',
        description: '',
      });
    }
    setFormModalKey((k) => k + 1);
    setModalVisible(true);
  };

  const openDetail = async (svc: ServiceItem) => {
    setDetailVisible(true);
    setDetailLoading(true);
    setDetailService(svc);
    try {
      const { item } = await getService(svc.id);
      setDetailService(item);
    } catch {
      Toast.warning('拉取详情失败，已显示列表中的数据');
    } finally {
      setDetailLoading(false);
    }
  };

  const closeDetail = () => {
    setDetailVisible(false);
    setDetailService(null);
  };

  // 关闭弹窗
  const closeModal = () => {
    setModalVisible(false);
    setFormInitSnapshot({});
    setEditingService(null);
    serviceFormApiRef.current = null;
  };

  // 提交表单
  const handleSubmit = async () => {
    const api = serviceFormApiRef.current;
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
    const status = Number(v.status) as ServiceStatus;
    if (!name || !Number.isFinite(status)) {
      Toast.warning('请填写必填项');
      return;
    }
    const body = {
      name,
      status,
      volc_log_service_id: String(v.volc_log_service_id ?? '').trim(),
      git_repo_url: String(v.git_repo_url ?? '').trim(),
      description: String(v.description ?? ''),
    };
    setSubmitting(true);
    try {
      if (modalType === 'create') {
        await saveServiceByName(body as CreateServiceParams);
        Toast.success('保存成功');
      } else if (modalType === 'edit' && editingService) {
        await updateService(editingService.id, body as UpdateServiceParams);
        Toast.success('更新成功');
      }
      closeModal();
      void fetchServices(currentPage, pageSize);
    } catch (e) {
      Toast.error(modalType === 'create' ? '保存失败' : '更新失败');
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
      void fetchServices(
        currentPage === 1 ? 1 : services.length === 1 ? currentPage - 1 : currentPage,
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
        return option ? <Tag color={serviceStatusTagColor(val)}>{option.label}</Tag> : val;
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
      render: (_value: unknown, record: ServiceItem) => (
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
            marginBottom: '24px',
          }}
        >
          <Typography.Title heading={3}>服务列表</Typography.Title>
          <Button icon={<IconPlus />} type="primary" onClick={() => openModal('create')}>
            新增服务
          </Button>
        </div>

        {/* 筛选：勿在 Form 内对 field + 受控 value 混用，会导致状态不回显 */}
        <div style={{ marginBottom: '24px', display: 'flex', gap: '16px', alignItems: 'flex-end' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <Typography.Text type="tertiary" size="small">
              状态
            </Typography.Text>
            <Select
              value={filters.status}
              onChange={(val) =>
                setFilters({
                  ...filters,
                  status: (val === '' || val == null ? undefined : val) as
                    | ServiceStatus
                    | undefined,
                })
              }
              placeholder="全部状态"
              style={{ width: 160 }}
              showClear
              optionList={serviceStatusOptions.map((o) => ({ label: o.label, value: o.value }))}
            />
          </div>
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
            onPageChange: (page: number) => {
              void fetchServices(page, pageSize);
            },
            onPageSizeChange: (size: number) => {
              setPageSize(size);
              setCurrentPage(1);
              void fetchServices(1, size);
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
            serviceFormApiRef.current = api;
          }}
          labelPosition="left"
          labelAlign="right"
          labelWidth={120}
        >
          <Form.Input
            field="name"
            label="服务名称"
            placeholder="请输入服务名称"
            rules={[{ required: true, message: '请输入服务名称' }]}
          />
          <Form.Select
            field="status"
            label="服务状态"
            placeholder="请选择服务状态"
            style={{ width: '100%' }}
            rules={[{ required: true, message: '请选择服务状态' }]}
            optionList={serviceStatusOptions.map((o) => ({ label: o.label, value: o.value }))}
          />
          <Form.Input
            field="volc_log_service_id"
            label="火山日志服务ID"
            placeholder="请输入火山日志服务ID"
          />
          <Form.Input field="git_repo_url" label="Git仓库地址" placeholder="请输入Git仓库地址" />
          <Form.TextArea
            field="description"
            label="服务描述"
            placeholder="请输入服务描述"
            rows={4}
          />
        </Form>
      </Modal>

      <Modal
        title="服务详情"
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
          {detailService && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              {(
                [
                  ['服务ID', detailService.id],
                  ['服务名称', detailService.name],
                  [
                    '状态',
                    serviceStatusOptions.find((o) => o.value === detailService.status)?.label ??
                      detailService.status,
                  ],
                  ['火山日志ID', detailService.volc_log_service_id || '—'],
                  [
                    'Git 仓库',
                    detailService.git_repo_url ? (
                      <a
                        href={detailService.git_repo_url}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {detailService.git_repo_url}
                      </a>
                    ) : (
                      '—'
                    ),
                  ],
                  ['创建时间', detailService.created_at || '—'],
                  ['更新时间', detailService.updated_at || '—'],
                  ['描述', detailService.description || '—'],
                ] as const
              ).map(([label, val]) => (
                <div
                  key={String(label)}
                  style={{
                    display: 'grid',
                    gridTemplateColumns: '120px 1fr',
                    gap: 12,
                    alignItems: 'start',
                  }}
                >
                  <Typography.Text type="tertiary">{label}</Typography.Text>
                  <div style={{ wordBreak: 'break-word' }}>{val as React.ReactNode}</div>
                </div>
              ))}
            </div>
          )}
        </Spin>
      </Modal>
    </div>
  );
};

export default ServiceManagementSection;
