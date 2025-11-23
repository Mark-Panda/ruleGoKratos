import React, { useEffect, useState } from 'react';

import { marked } from 'marked';
import {
  Button,
  Input,
  Modal,
  Pagination,
  Select,
  Spin,
  Table,
  TextArea,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlus } from '@douyinfe/semi-icons';

import { requestJSON } from '../../services/http';

export const DocsSection: React.FC = () => {
  const [docKeywords, setDocKeywords] = useState('');
  const [docLoading, setDocLoading] = useState(false);
  const [docError, setDocError] = useState<string | undefined>();
  const [docItems, setDocItems] = useState<any[]>([]);
  const [docPage, setDocPage] = useState(1);
  const [docSize, setDocSize] = useState(10);
  const [docTotal, setDocTotal] = useState(0);
  const [docCreateVisible, setDocCreateVisible] = useState(false);
  const [docSubmitting, setDocSubmitting] = useState(false);
  const [docEditMode, setDocEditMode] = useState<'create' | 'edit' | 'view'>('create');
  const [docForm, setDocForm] = useState({ id: '', name: '', description: '', content: '' });
  const [docPreview, setDocPreview] = useState(true);

  const fetchDocs = async (page?: number, size?: number) => {
    setDocLoading(true);
    setDocError(undefined);
    try {
      const data = await requestJSON<any>('/doc/list', {
        params: { page: page ?? docPage, size: size ?? docSize },
      });
      const list = Array.isArray(data?.list)
        ? data.list
        : Array.isArray(data?.items)
        ? data.items
        : Array.isArray(data)
        ? data
        : [];
      const norm = list.map((it: any) => ({
        id: String(it?.id ?? Math.random()),
        name: String(it?.name ?? it?.title ?? it?.docName ?? ''),
        description: String(it?.description ?? it?.desc ?? ''),
        content: String(it?.content ?? ''),
        relatedCount: Number(it?.relatedCount ?? it?.workItemCount ?? 0),
        enabled: it?.disabled === false,
        createTime: it?.createdAt ?? it?.createTime ?? null,
        updateTime: it?.updatedAt ?? it?.updateTime ?? null,
      }));
      setDocItems(norm);
      const total = Number(data?.total ?? norm.length);
      setDocTotal(Number.isFinite(total) ? total : norm.length);
    } catch (e) {
      setDocError(String((e as Error)?.message ?? e));
    } finally {
      setDocLoading(false);
    }
  };

  useEffect(() => {
    fetchDocs(1, docSize);
    setDocPage(1);
  }, []);

  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        background: '#F7F8FA',
        height: '100%',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          flex: 1,
          overflow: 'auto',
          padding: 24,
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
              background: '#fff',
              borderRadius: 12,
              border: '1px solid rgba(6,7,9,0.06)',
              boxShadow: '0 2px 8px rgba(6,7,9,0.04)',
              padding: '10px 12px',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, flex: 1 }}>
              <Input
                prefix={<span style={{ color: '#667EEA', fontSize: 16 }}>🔍</span>}
                value={docKeywords}
                onChange={(v) => {
                  setDocKeywords(v);
                  setDocPage(1);
                }}
                placeholder="搜索文档名称/描述..."
                showClear
                style={{ maxWidth: 380, borderRadius: 10 }}
              />
              <Button
                theme="solid"
                type="primary"
                onClick={() => fetchDocs(1, docSize)}
                loading={docLoading}
              >
                查询
              </Button>
              <Button
                type="tertiary"
                onClick={() => {
                  setDocKeywords('');
                  setDocPage(1);
                  fetchDocs(1, docSize);
                }}
              >
                重置
              </Button>
            </div>
            <div>
              <Button
                icon={<IconPlus />}
                theme="solid"
                type="primary"
                onClick={() => {
                  setDocEditMode('create');
                  setDocForm({ id: '', name: '', description: '', content: '' });
                  setDocPreview(true);
                  setDocCreateVisible(true);
                }}
              >
                新建文档
              </Button>
            </div>
          </div>
          {docError ? <Typography.Text type="danger">加载失败：{docError}</Typography.Text> : null}
          <Spin spinning={docLoading}>
            <Table
              dataSource={docItems
                .filter((d) => {
                  const kw = docKeywords.trim().toLowerCase();
                  if (!kw) return true;
                  const name = String(d.name || '').toLowerCase();
                  const desc = String(d.description || '').toLowerCase();
                  return name.includes(kw) || desc.includes(kw);
                })
                .slice((docPage - 1) * docSize, (docPage - 1) * docSize + docSize)}
              rowKey={(r: any) => String(r.id)}
              columns={[
                { title: '文档名称', render: (_, r: any) => String(r.name || '-'), width: 240 },
                {
                  title: '描述',
                  render: (_, r: any) => (
                    <Typography.Text type="tertiary">{String(r.description || '')}</Typography.Text>
                  ),
                },
                {
                  title: '创建时间',
                  width: 180,
                  render: (_, r: any) => {
                    const ts = Number(r.createTime ? Date.parse(String(r.createTime)) : 0);
                    return ts ? new Date(ts).toLocaleString() : '';
                  },
                },
                {
                  title: '操作',
                  width: 220,
                  render: (_, r: any) => (
                    <div style={{ display: 'flex', gap: 8 }}>
                      <Button
                        size="small"
                        type="primary"
                        onClick={() => {
                          setDocEditMode('view');
                          setDocForm({
                            id: String(r.id || ''),
                            name: String(r.name || ''),
                            description: String(r.description || ''),
                            content: String(r.content || ''),
                          });
                          setDocPreview(true);
                          setDocCreateVisible(true);
                        }}
                      >
                        详情
                      </Button>
                      <Button
                        size="small"
                        type="secondary"
                        onClick={() => {
                          setDocEditMode('edit');
                          setDocForm({
                            id: String(r.id || ''),
                            name: String(r.name || ''),
                            description: String(r.description || ''),
                            content: String(r.content || ''),
                          });
                          setDocPreview(true);
                          setDocCreateVisible(true);
                        }}
                      >
                        编辑
                      </Button>
                      <Button
                        size="small"
                        type="danger"
                        onClick={() => {
                          Modal.confirm({
                            title: '删除文档',
                            content: `确认删除文档「${String(r.name || '-')}」？此操作不可恢复`,
                            okText: '删除',
                            cancelText: '取消',
                            onOk: async () => {
                              Toast.success({ content: '已删除（待接入服务端）' });
                            },
                          });
                        }}
                      >
                        删除
                      </Button>
                    </div>
                  ),
                },
              ]}
              pagination={false}
            />
          </Spin>
          <div
            style={{
              background: '#fff',
              borderRadius: 12,
              border: '1px solid rgba(6,7,9,0.06)',
              boxShadow: '0 2px 8px rgba(6,7,9,0.04)',
              padding: '10px 12px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <Typography.Text>共 {docTotal} 条</Typography.Text>
              <Typography.Text type="tertiary" style={{ fontSize: 12 }}>
                显示 {docTotal === 0 ? 0 : (docPage - 1) * docSize + 1}-
                {Math.min(docPage * docSize, docTotal)} 条
              </Typography.Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <Select
                value={docSize}
                style={{ width: 110 }}
                onChange={(v) => {
                  const s = Number(v);
                  setDocSize(s);
                  setDocPage(1);
                  fetchDocs(1, s);
                }}
              >
                <Select.Option value={10}>10 / 页</Select.Option>
                <Select.Option value={20}>20 / 页</Select.Option>
                <Select.Option value={50}>50 / 页</Select.Option>
              </Select>
              <Pagination
                total={docTotal}
                pageSize={docSize}
                currentPage={docPage}
                onChange={(p: number) => {
                  setDocPage(p);
                  fetchDocs(p, docSize);
                }}
              />
            </div>
          </div>
        </div>

        <Modal
          title={
            <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <span style={{ fontSize: 20 }}>📄</span>
              <span style={{ fontSize: 16, fontWeight: 600 }}>
                {docEditMode === 'create'
                  ? '新建文档'
                  : docEditMode === 'edit'
                  ? '编辑文档'
                  : '文档详情'}
              </span>
            </div>
          }
          visible={docCreateVisible}
          onCancel={() => setDocCreateVisible(false)}
          confirmLoading={docSubmitting}
          okText={docEditMode === 'create' ? '保存' : docEditMode === 'edit' ? '更新' : '关闭'}
          onOk={async () => {
            if (!docForm.name.trim()) {
              Toast.warning({ content: '请输入文档名称' });
              return;
            }
            if (docEditMode === 'view') {
              setDocCreateVisible(false);
              return;
            }
            setDocSubmitting(true);
            try {
              if (docEditMode === 'create') {
                await requestJSON('/doc/create', {
                  method: 'POST',
                  body: {
                    title: docForm.name,
                    content: docForm.content,
                    desc: docForm.description,
                  },
                });
                Toast.success({ content: '保存成功' });
              } else {
                await requestJSON('/doc/edit', {
                  method: 'POST',
                  body: {
                    id: Number(docForm.id),
                    title: docForm.name,
                    content: docForm.content,
                    desc: docForm.description,
                  },
                });
                Toast.success({ content: '更新成功' });
              }
              setDocCreateVisible(false);
              await fetchDocs(1, docSize);
            } catch (e) {
              Toast.error({ content: String((e as Error)?.message ?? e) });
            } finally {
              setDocSubmitting(false);
            }
          }}
          style={{ borderRadius: 16, width: 1280 }}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
              <div>
                <Typography.Text strong style={{ display: 'block', marginBottom: 8, fontSize: 14 }}>
                  文档名称 *
                </Typography.Text>
                <Input
                  value={docForm.name}
                  onChange={(v) => setDocForm({ ...docForm, name: v })}
                  placeholder="请输入文档名称"
                  disabled={docEditMode === 'view'}
                  size="large"
                />
              </div>
              <div>
                <Typography.Text strong style={{ display: 'block', marginBottom: 8, fontSize: 14 }}>
                  文档描述
                </Typography.Text>
                <TextArea
                  value={docForm.description}
                  onChange={(v) => setDocForm({ ...docForm, description: String(v) })}
                  autosize={{ minRows: 3, maxRows: 6 }}
                  placeholder="请输入文档描述"
                  disabled={docEditMode === 'view'}
                  style={{ fontSize: 14 }}
                />
              </div>
            </div>
            <div>
              <Typography.Text strong style={{ display: 'block', marginBottom: 8, fontSize: 14 }}>
                文档内容
              </Typography.Text>
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: docPreview ? '1fr 1fr' : '1fr',
                  gap: 16,
                }}
              >
                <TextArea
                  value={docForm.content}
                  onChange={(v) => setDocForm({ ...docForm, content: String(v) })}
                  autosize={{ minRows: 16, maxRows: 32 }}
                  placeholder="请输入文档内容，支持 Markdown 语法"
                  disabled={docEditMode === 'view'}
                  style={{ fontSize: 14 }}
                />
                {docPreview && (
                  <div
                    style={{
                      border: '1px solid rgba(6,7,9,0.08)',
                      borderRadius: 10,
                      padding: 16,
                      background: '#FAFAFA',
                      overflow: 'auto',
                      maxHeight: 560,
                    }}
                    dangerouslySetInnerHTML={{
                      __html: String(marked.parse(docForm.content || '')),
                    }}
                  />
                )}
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8 }}>
                <Typography.Text>实时预览</Typography.Text>
                <Select
                  value={docPreview ? '1' : '0'}
                  onChange={(v) => setDocPreview(v === '1')}
                  disabled={docEditMode === 'view'}
                  style={{ width: 120 }}
                >
                  <Select.Option value="1">开启</Select.Option>
                  <Select.Option value="0">关闭</Select.Option>
                </Select>
              </div>
            </div>
          </div>
        </Modal>
      </div>
    </div>
  );
};

export default DocsSection;
