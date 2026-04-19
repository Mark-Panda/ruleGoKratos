import React, { useEffect, useRef, useState } from 'react';

import {
  Button,
  Input,
  Select,
  Typography,
  Tag,
  Modal,
  Toast,
  Table,
  Row,
  Col,
  Switch,
} from '@douyinfe/semi-ui';
import { IconPlus, IconChevronLeft } from '@douyinfe/semi-icons';

import { RuleDetail, RuleDetailData } from '../rule-detail';
import { FlowDocumentJSON } from '../../typings';
import {
  getRuleList,
  createRuleBase,
  startRuleChain,
  stopRuleChain,
  deleteRuleChain,
} from '../../services/api-rules';
import { Editor } from '../../editor';

export const WorkflowSection: React.FC = () => {
  const [showEditor, setShowEditor] = useState(false);
  const [rules, setRules] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const [selectedDoc, setSelectedDoc] = useState<FlowDocumentJSON | undefined>();
  const [showDetail, setShowDetail] = useState(false);
  const [detailData, setDetailData] = useState<RuleDetailData | undefined>();
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(10);
  const [keywords, setKeywords] = useState('');
  const [chainFilter, setChainFilter] = useState<'all' | 'root' | 'sub'>('all');
  const [total, setTotal] = useState(0);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createName, setCreateName] = useState('');
  const [createDesc, setCreateDesc] = useState('');
  const [createRoot, setCreateRoot] = useState(true);
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [nameError, setNameError] = useState(false);
  const [createId, setCreateId] = useState<string>(() => {
    try {
      const { customAlphabet } = require('nanoid');
      return customAlphabet('0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ', 12)();
    } catch {
      return Math.random().toString(36).slice(2, 14);
    }
  });

  const refreshList = async () => {
    if (showEditor) return;
    setLoading(true);
    setError(undefined);
    const rootParam = chainFilter === 'root' ? true : chainFilter === 'sub' ? false : undefined;
    try {
      const data = await getRuleList({
        page,
        size,
        keywords: keywords.trim() || undefined,
        root: rootParam,
      });
      const items = Array.isArray(data.items) ? data.items : [];
      setRules(items);
      const t = Number(data.total ?? data.count ?? items.length);
      setTotal(Number.isFinite(t) ? t : items.length);
    } catch (err) {
      setError(String(err));
    } finally {
      setLoading(false);
    }
  };

  // 跟踪上一次的筛选条件，用于检测筛选条件是否改变
  const prevKeywordsRef = useRef(keywords);
  const prevChainFilterRef = useRef(chainFilter);

  useEffect(() => {
    const keywordsChanged = prevKeywordsRef.current !== keywords;
    const chainFilterChanged = prevChainFilterRef.current !== chainFilter;

    // 如果筛选条件改变且当前不在第一页，重置到第一页
    if ((keywordsChanged || chainFilterChanged) && page !== 1) {
      prevKeywordsRef.current = keywords;
      prevChainFilterRef.current = chainFilter;
      setPage(1);
      return;
    }

    // 更新引用
    prevKeywordsRef.current = keywords;
    prevChainFilterRef.current = chainFilter;

    // 刷新列表
    refreshList();
  }, [page, size, keywords, chainFilter, showEditor]);

  const columns = [
    {
      title: '序号',
      dataIndex: 'index',
      render: (_: any, __: any, index: number) => (page - 1) * size + index + 1,
      width: 60,
    },
    {
      title: '工作流 ID',
      dataIndex: 'ruleChain.id',
      render: (text: string) => <Typography.Text copyable>{text}</Typography.Text>,
    },
    {
      title: '工作流名称',
      dataIndex: 'ruleChain.name',
      render: (text: string) => <Typography.Text strong>{text}</Typography.Text>,
    },
    {
      title: '类型',
      dataIndex: 'ruleChain.root',
      render: (root: boolean) =>
        root ? (
          <Tag color="blue" type="ghost">
            主流程
          </Tag>
        ) : (
          <Tag color="cyan" type="ghost">
            子流程
          </Tag>
        ),
    },
    {
      title: '状态',
      render: (text: any, record: any) => {
        const chain = record.ruleChain;
        const disabled = chain?.disabled;
        if (disabled)
          return (
            <Tag color="grey" style={{ borderRadius: 4 }}>
              已下线
            </Tag>
          );
        return (
          <Tag color="green" style={{ borderRadius: 4 }}>
            已部署
          </Tag>
        );
      },
    },
    {
      title: '调试模式',
      dataIndex: 'ruleChain.debugMode',
      render: (debug: boolean) => {
        if (debug)
          return (
            <Tag color="indigo" style={{ borderRadius: 4 }}>
              开启
            </Tag>
          );
        return <Tag style={{ borderRadius: 4 }}>关闭</Tag>;
      },
    },
    {
      title: '描述',
      dataIndex: 'ruleChain.additionalInfo.description',
      render: (text: string, record: any) => {
        const desc = record?.ruleChain?.additionalInfo?.description || '-';
        return (
          <Typography.Text
            type="tertiary"
            ellipsis={{ showTooltip: true }}
            style={{ maxWidth: 200 }}
          >
            {desc}
          </Typography.Text>
        );
      },
    },
    {
      title: '操作',
      fixed: 'right' as const,
      width: 180,
      render: (text: any, record: any) => (
        <div style={{ display: 'flex', gap: 12 }}>
          <Typography.Text
            style={{ color: '#667EEA', cursor: 'pointer' }}
            onClick={() => {
              const id = record?.ruleChain?.id;
              if (id) window.location.hash = `#/workflow/${encodeURIComponent(id)}`;
            }}
          >
            打开
          </Typography.Text>
          <Typography.Text
            style={{ color: '#667EEA', cursor: 'pointer' }}
            onClick={async () => {
              const id = String(record?.ruleChain?.id ?? '');
              const disabled = Boolean(record?.ruleChain?.disabled);
              if (!id) return;
              try {
                if (disabled) {
                  await startRuleChain(id);
                  Toast.success({ content: '已部署' });
                } else {
                  await stopRuleChain(id);
                  Toast.success({ content: '已下线' });
                }
                refreshList();
              } catch (e) {
                Toast.error({ content: '操作失败' });
              }
            }}
          >
            {record?.ruleChain?.disabled ? '部署' : '下线'}
          </Typography.Text>
          <Typography.Text
            style={{ color: '#FF4B4B', cursor: 'pointer' }}
            onClick={async () => {
              const id = String(record?.ruleChain?.id ?? '');
              if (!id) return;
              Modal.confirm({
                title: '确认删除',
                content: '确认删除该工作流？',
                onOk: async () => {
                  try {
                    await deleteRuleChain(id);
                    Toast.success({ content: '删除成功' });
                    refreshList();
                  } catch (e) {
                    Toast.error({ content: '删除失败' });
                  }
                },
              });
            }}
          >
            删除
          </Typography.Text>
        </div>
      ),
    },
  ];

  const FilterItem = ({ label, children }: { label: string; children: React.ReactNode }) => (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <span style={{ fontSize: 14, color: '#1C2029', minWidth: 70, textAlign: 'right' }}>
        {label}
      </span>
      <div style={{ flex: 1 }}>{children}</div>
    </div>
  );

  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        background: '#F7F8FA', // Keep gray bg for the whole area
        overflow: 'hidden',
        padding: 16,
      }}
    >
      {showDetail ? (
        <RuleDetail
          data={detailData as RuleDetailData}
          onBack={() => {
            setShowDetail(false);
            setDetailData(undefined);
          }}
        />
      ) : showEditor ? (
        <div
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            minHeight: 0,
            overflow: 'hidden',
            background: '#fff',
            borderRadius: 4,
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '12px 16px',
              background: '#fff',
              borderBottom: '1px solid rgba(6,7,9,0.06)',
            }}
          >
            <Button
              icon={<IconChevronLeft />}
              onClick={() => {
                setShowEditor(false);
                setSelectedDoc(undefined);
              }}
            >
              返回管理面板
            </Button>
          </div>
          <Editor initialDoc={selectedDoc} />
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16, height: '100%' }}>
          {error ? <Typography.Text type="danger">加载失败：{error}</Typography.Text> : null}

          {/* Filter Area */}
          <div style={{ background: '#fff', padding: 24, borderRadius: 2 }}>
            <div style={{ width: '100%' }}>
              <Row gutter={24}>
                <Col span={8}>
                  <FilterItem label="关键词">
                    <Input
                      placeholder="搜索名称或 ID"
                      value={keywords}
                      onChange={setKeywords}
                      showClear
                    />
                  </FilterItem>
                </Col>
                <Col span={6}>
                  <FilterItem label="类型">
                    <Select
                      placeholder="全部"
                      style={{ width: '100%' }}
                      value={chainFilter}
                      onChange={(v) => setChainFilter(v as any)}
                    >
                      <Select.Option value="all">全部</Select.Option>
                      <Select.Option value="root">主流程</Select.Option>
                      <Select.Option value="sub">子流程</Select.Option>
                    </Select>
                  </FilterItem>
                </Col>
                <Col
                  span={10}
                  style={{ textAlign: 'left', display: 'flex', gap: 12, paddingLeft: 24 }}
                >
                  <Button
                    type="primary"
                    theme="solid"
                    onClick={() => {
                      setPage(1);
                      refreshList();
                    }}
                  >
                    筛选
                  </Button>
                  <Button
                    type="tertiary"
                    onClick={() => {
                      setKeywords('');
                      setChainFilter('all');
                      setPage(1);
                    }}
                  >
                    重置
                  </Button>
                </Col>
              </Row>
            </div>
          </div>

          {/* Table Area */}
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
            {/* Toolbar */}
            <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end', gap: 12 }}>
              <Button
                icon={<IconPlus />}
                theme="solid"
                type="primary"
                onClick={() => setShowCreateModal(true)}
              >
                新建工作流
              </Button>
            </div>

            {/* Table */}
            <Table
              dataSource={rules}
              rowKey={(r?: any) => String(r?.ruleChain?.id ?? r?.id ?? 'rule-unknown')}
              columns={columns}
              loading={loading}
              pagination={{
                total,
                currentPage: page,
                pageSize: size,
                onChange: (p) => setPage(p),
                onPageSizeChange: (s) => setSize(s),
                showSizeChanger: true,
                pageSizeOpts: [10, 20, 50],
              }}
              rowSelection={{
                onChange: (selectedRowKeys, selectedRows) => {
                  // Handle selection
                },
              }}
              style={{ flex: 1, overflow: 'auto' }}
            />
          </div>
        </div>
      )}

      <Modal
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <span style={{ fontSize: 24 }}>✨</span>
            <span style={{ fontSize: 18, fontWeight: 600 }}>新建工作流</span>
          </div>
        }
        visible={showCreateModal}
        onCancel={() => {
          setShowCreateModal(false);
          setNameError(false);
        }}
        confirmLoading={createSubmitting}
        style={{ borderRadius: 16 }}
        onOk={async () => {
          if (!createName.trim()) {
            setNameError(true);
            return;
          }
          setCreateSubmitting(true);
          try {
            const body = {
              id: createId,
              name: createName.trim(),
              root: !!createRoot,
              additionalInfo: { description: createDesc ?? '' },
            };
            await createRuleBase(createId, body);
            Toast.success({ content: '创建成功' });
            setShowCreateModal(false);
            window.location.hash = `#/workflow/${encodeURIComponent(createId)}`;
            setCreateName('');
            setCreateDesc('');
            setCreateRoot(true);
            setNameError(false);
            try {
              const { customAlphabet } = require('nanoid');
              setCreateId(
                customAlphabet(
                  '0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ',
                  12
                )()
              );
            } catch {
              setCreateId(Math.random().toString(36).slice(2, 14));
            }
            setPage(1);
          } catch (e) {
            Toast.error({ content: String((e as Error)?.message ?? e) });
          } finally {
            setCreateSubmitting(false);
          }
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div>
            <Typography.Text strong style={{ display: 'block', marginBottom: 8, color: '#1C2029' }}>
              工作流名称 *
            </Typography.Text>
            <Input
              value={createName}
              onChange={(v) => {
                setCreateName(v);
                if (v.trim()) setNameError(false);
              }}
              placeholder="请输入工作流名称"
              size="large"
              validateStatus={nameError ? 'error' : 'default'}
              style={{ borderRadius: 10 }}
            />
          </div>
          <div>
            <Typography.Text strong style={{ display: 'block', marginBottom: 8, color: '#1C2029' }}>
              工作流描述
            </Typography.Text>
            <Input
              value={createDesc}
              onChange={setCreateDesc}
              placeholder="请输入工作流描述（可选）"
              size="large"
              style={{ borderRadius: 10 }}
            />
          </div>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              padding: '12px 14px',
              background: 'rgba(102, 126, 234, 0.08)',
              borderRadius: 10,
              border: '1px solid rgba(102, 126, 234, 0.15)',
            }}
          >
            <div>
              <Typography.Text
                strong
                style={{ display: 'block', color: '#1C2029', marginBottom: 4 }}
              >
                流程类型
              </Typography.Text>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <Switch checked={createRoot} onChange={(v) => setCreateRoot(v)} />
                <Typography.Text>{createRoot ? '主流程' : '子流程'}</Typography.Text>
              </div>
            </div>
          </div>
          <div>
            <Typography.Text strong style={{ display: 'block', marginBottom: 8, color: '#1C2029' }}>
              工作流 ID
            </Typography.Text>
            <Input
              value={createId}
              onChange={setCreateId}
              placeholder="自动生成，不可修改"
              size="large"
              disabled
              style={{ borderRadius: 10, fontFamily: 'monospace' }}
            />
            <Typography.Text
              type="tertiary"
              style={{ fontSize: 11, marginTop: 4, display: 'block' }}
            >
              💡 提示：ID 用于唯一标识工作流，建议使用默认生成的随机 ID
            </Typography.Text>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default WorkflowSection;
