import ReactQuill from 'react-quill';
import React, { useEffect, useMemo, useState } from 'react';

import { marked } from 'marked';
import {
  Button,
  Input,
  Modal,
  Pagination,
  Select,
  Spin,
  Table,
  Tag,
  TextArea,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';

import { requestJSON } from '../../services/http';
import 'react-quill/dist/quill.snow.css';

export const ComponentsSection: React.FC<{ view?: 'installed' | 'rules' }> = ({
  view = 'installed',
}) => {
  const [compLoading, setCompLoading] = useState(false);
  const [compError, setCompError] = useState<string | undefined>();
  const [components, setComponents] = useState<any[]>([]);
  const [compKeywords, setCompKeywords] = useState('');
  const [compKind, setCompKind] = useState<'all' | 'endpoint' | 'node'>('all');
  const [compPage, setCompPage] = useState(1);
  const [compSize, setCompSize] = useState(10);
  const compKinds = ['all', 'endpoint', 'node'] as const;
  const [compViewVisible, setCompViewVisible] = useState(false);
  const [compViewText, setCompViewText] = useState('');

  const filteredComponents = useMemo(() => {
    const kw = compKeywords.trim().toLowerCase();
    const byKw = (components || []).filter((c: any) => {
      if (!kw) return true;
      const name = String(c.name ?? c.title ?? c.type ?? '').toLowerCase();
      const desc = String(c.description ?? c.desc ?? '').toLowerCase();
      return name.includes(kw) || desc.includes(kw);
    });
    const byKind = byKw.filter((c: any) => {
      if (compKind === 'all') return true;
      return String(c.kind) === compKind;
    });
    return byKind;
  }, [components, compKeywords, compKind]);

  const pagedComponents = useMemo(() => {
    const start = (compPage - 1) * compSize;
    return filteredComponents.slice(start, start + compSize);
  }, [filteredComponents, compPage, compSize]);

  const componentOptions = useMemo(() => {
    const seen = new Set<string>();
    const opts = (components || []).map((c: any) => {
      const label = String(c.label || c.name || c.type || '-');
      const value = String(c.type || c.name || '');
      const category = String(c.category || '');
      return { label, value, kind: String(c.kind || ''), category };
    });
    return opts.filter((o) => {
      if (!o.value) return false;
      if (seen.has(o.value)) return false;
      seen.add(o.value);
      return true;
    });
  }, [components]);

  const fetchComponents = async () => {
    setCompLoading(true);
    setCompError(undefined);
    try {
      const data = await requestJSON<any>('/components');
      const epList: any[] = Array.isArray(data?.endpoints) ? data.endpoints : [];
      const nodeList: any[] = Array.isArray(data?.nodes) ? data.nodes : [];
      const pool: Record<string, any[]> = (data?.builtins?.nodePool || {}) as any;
      const normEp = epList.map((ep: any) => {
        const type = String(ep?.type ?? '');
        const inst = Array.isArray(pool?.[type]) ? pool[type] : [];
        return {
          id: type,
          type,
          name: String(ep?.name ?? ''),
          label: String(ep?.label ?? ''),
          category: String(ep?.category ?? (type.split('/')?.[0] || '未分类')),
          kind: 'endpoint',
          description: String(ep?.desc ?? ''),
          version: String(ep?.version ?? ''),
          disabled: !!ep?.disabled,
          fields: Array.isArray(ep?.fields) ? ep.fields : [],
          fieldsLen: Array.isArray(ep?.fields) ? ep.fields.length : 0,
          relationTypes: Array.isArray(ep?.relationTypes) ? ep.relationTypes : [],
          relationTypesLen: Array.isArray(ep?.relationTypes) ? ep.relationTypes.length : 0,
          instancesCount: inst.length,
        };
      });
      const normNode = nodeList.map((nd: any) => {
        const type = String(nd?.type ?? '');
        return {
          id: type,
          type,
          name: String(nd?.name ?? ''),
          label: String(nd?.label ?? ''),
          category: String(nd?.category ?? (type.split('/')?.[0] || '未分类')),
          kind: 'node',
          description: String(nd?.desc ?? ''),
          version: String(nd?.version ?? ''),
          disabled: !!nd?.disabled,
          fields: Array.isArray(nd?.fields) ? nd.fields : [],
          fieldsLen: Array.isArray(nd?.fields) ? nd.fields.length : 0,
          relationTypes: Array.isArray(nd?.relationTypes) ? nd.relationTypes : [],
          relationTypesLen: Array.isArray(nd?.relationTypes) ? nd.relationTypes.length : 0,
          instancesCount: 0,
        };
      });
      setComponents([...normEp, ...normNode]);
    } catch (e) {
      setCompError(String((e as Error)?.message ?? e));
    } finally {
      setCompLoading(false);
    }
  };

  useEffect(() => {
    if (view === 'installed') fetchComponents();
  }, [view]);

  const [ruleLoading, setRuleLoading] = useState(false);
  const [ruleError, setRuleError] = useState<string | undefined>();
  const [ruleItems, setRuleItems] = useState<any[]>([]);
  const [ruleKeywords, setRuleKeywords] = useState('');
  const [ruleKind, setRuleKind] = useState<'all' | 'endpoint' | 'node' | 'external'>('all');
  const [ruleStatus, setRuleStatus] = useState<'all' | 'enabled' | 'disabled'>('all');
  const [rulePage, setRulePage] = useState(1);
  const [ruleSize, setRuleSize] = useState(10);
  const [ruleTotal, setRuleTotal] = useState(0);
  const [ruleEditVisible, setRuleEditVisible] = useState(false);
  const [ruleSubmitting, setRuleSubmitting] = useState(false);
  const [ruleEditMode, setRuleEditMode] = useState<'edit' | 'create' | 'view'>('edit');
  const [ruleForm, setRuleForm] = useState({
    componentName: '',
    componentType: 'action',
    disabled: false,
    useDesc: '',
    useRuleDesc: '',
    id: '',
  });
  const [ruleDescMode, setRuleDescMode] = useState<'rich' | 'markdown'>('markdown');
  const [ruleDescPreview, setRuleDescPreview] = useState(true);

  const filteredRules = useMemo(() => {
    const kw = ruleKeywords.trim().toLowerCase();
    let arr = ruleItems.filter((r) => {
      if (!kw) return true;
      const name = String(r.name ?? r.ruleName ?? r.id ?? '').toLowerCase();
      const desc = String(r.description ?? r.desc ?? '').toLowerCase();
      return name.includes(kw) || desc.includes(kw);
    });
    if (ruleKind !== 'all') arr = arr.filter((r) => String(r.kind) === ruleKind);
    if (ruleStatus !== 'all') {
      const needEnabled = ruleStatus === 'enabled';
      arr = arr.filter((r) => !!r.enabled === needEnabled);
    }
    return arr;
  }, [ruleItems, ruleKeywords, ruleKind, ruleStatus]);

  const pagedRules = useMemo(() => {
    const start = (rulePage - 1) * ruleSize;
    return filteredRules.slice(start, start + ruleSize);
  }, [filteredRules, rulePage, ruleSize]);

  const fetchRules = async (page?: number, size?: number) => {
    setRuleLoading(true);
    setRuleError(undefined);
    try {
      const data = await requestJSON<any>('/componentUseRule/page', {
        params: { page: page ?? rulePage, size: size ?? ruleSize },
      });
      const list = Array.isArray(data?.list)
        ? data.list
        : Array.isArray(data?.items)
        ? data.items
        : Array.isArray(data)
        ? data
        : [];
      const norm = list.map((it: any) => {
        const compType = String(it?.componentType ?? it?.type ?? '');
        const kind = compType.startsWith('endpoint')
          ? 'endpoint'
          : compType === 'external'
          ? 'external'
          : 'node';
        const updatedISO = String(it?.updatedAt ?? it?.updateTime ?? '');
        const updatedTs = updatedISO ? Date.parse(updatedISO) : null;
        return {
          id: String(it?.id ?? Math.random()),
          name: String(it?.componentName ?? it?.name ?? compType ?? ''),
          type: compType,
          kind,
          category: String(compType.split('/')?.[0] || compType || '未知'),
          description: String(it?.useDesc ?? it?.description ?? it?.desc ?? ''),
          ruleDesc: String(it?.useRuleDesc ?? ''),
          enabled: it?.disabled === false,
          updateTime: updatedTs,
        };
      });
      setRuleItems(norm);
      const total = Number(data?.total ?? norm.length);
      setRuleTotal(Number.isFinite(total) ? total : norm.length);
    } catch (e) {
      setRuleError(String((e as Error)?.message ?? e));
    } finally {
      setRuleLoading(false);
    }
  };

  useEffect(() => {
    if (view === 'rules') {
      fetchRules(1, ruleSize);
      setRulePage(1);
    }
  }, [view]);

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
        {view === 'installed' && (
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
                  value={compKeywords}
                  onChange={(v) => {
                    setCompKeywords(v);
                    setCompPage(1);
                  }}
                  placeholder="搜索组件名称/描述..."
                  showClear
                  style={{ maxWidth: 380, borderRadius: 10 }}
                />
                <Select
                  value={compKind}
                  style={{ width: 180 }}
                  onChange={(v) => {
                    setCompKind(v as any);
                    setCompPage(1);
                  }}
                >
                  {compKinds.map((k) => (
                    <Select.Option key={k} value={k}>
                      {k === 'all' ? '全部种类' : k === 'endpoint' ? '端点' : '节点'}
                    </Select.Option>
                  ))}
                </Select>
                <Button
                  theme="solid"
                  type="primary"
                  onClick={() => fetchComponents()}
                  loading={compLoading}
                >
                  查询
                </Button>
                <Button
                  type="tertiary"
                  onClick={() => {
                    setCompKeywords('');
                    setCompKind('all');
                    setCompPage(1);
                    fetchComponents();
                  }}
                >
                  重置
                </Button>
              </div>
            </div>
            {compError ? (
              <Typography.Text type="danger">加载失败：{compError}</Typography.Text>
            ) : null}
            <Spin spinning={compLoading}>
              <Table
                dataSource={pagedComponents}
                rowKey={(r: any) => String(r.id ?? r.type ?? Math.random())}
                columns={[
                  {
                    title: '组件名称',
                    render: (_, r: any) => String(r.label || r.name || r.type || '-'),
                    width: 240,
                  },
                  {
                    title: '组件分类',
                    render: (_, r: any) => String(r.category ?? '未分类'),
                    width: 160,
                  },
                  {
                    title: '组件种类',
                    render: (_, r: any) => (String(r.kind) === 'endpoint' ? '端点' : '节点'),
                    width: 120,
                  },
                  {
                    title: '操作',
                    width: 120,
                    render: (_, r: any) => (
                      <div style={{ display: 'flex', gap: 8 }}>
                        <Button
                          size="small"
                          type="primary"
                          onClick={() => {
                            try {
                              const text = JSON.stringify(r, null, 2);
                              setCompViewText(text);
                              setCompViewVisible(true);
                            } catch (e) {
                              setCompViewText(String(r));
                              setCompViewVisible(true);
                            }
                          }}
                        >
                          详情
                        </Button>
                      </div>
                    ),
                  },
                ]}
                pagination={false}
              />
            </Spin>
            <Modal
              title={
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span style={{ fontSize: 20 }}>🧩</span>
                  <span style={{ fontSize: 16, fontWeight: 600 }}>组件详情</span>
                </div>
              }
              visible={compViewVisible}
              onCancel={() => setCompViewVisible(false)}
              okText="关闭"
              onOk={() => setCompViewVisible(false)}
              style={{ borderRadius: 16, width: 720 }}
            >
              <TextArea
                value={compViewText}
                readOnly
                rows={18}
                style={{ fontFamily: 'SFMono-Regular, Menlo, Monaco, Consolas, monospace' }}
              />
            </Modal>
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
                <Typography.Text>共 {filteredComponents.length} 条</Typography.Text>
                <Typography.Text type="tertiary" style={{ fontSize: 12 }}>
                  显示 {filteredComponents.length === 0 ? 0 : (compPage - 1) * compSize + 1}-
                  {Math.min(compPage * compSize, filteredComponents.length)} 条
                </Typography.Text>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <Select
                  value={compSize}
                  style={{ width: 110 }}
                  onChange={(v) => {
                    setCompSize(Number(v));
                    setCompPage(1);
                  }}
                >
                  <Select.Option value={10}>10 / 页</Select.Option>
                  <Select.Option value={20}>20 / 页</Select.Option>
                  <Select.Option value={50}>50 / 页</Select.Option>
                </Select>
                <Pagination
                  total={filteredComponents.length}
                  pageSize={compSize}
                  currentPage={compPage}
                  onChange={(p: number) => setCompPage(p)}
                />
              </div>
            </div>
          </div>
        )}

        {view === 'rules' && (
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
                  value={ruleKeywords}
                  onChange={(v) => {
                    setRuleKeywords(v);
                    setRulePage(1);
                  }}
                  placeholder="搜索规则名称/描述..."
                  showClear
                  style={{ maxWidth: 380, borderRadius: 10 }}
                />
                <Select
                  value={ruleKind}
                  style={{ width: 160 }}
                  onChange={(v) => {
                    setRuleKind(v as any);
                    setRulePage(1);
                  }}
                >
                  <Select.Option value="all">全部类型</Select.Option>
                  <Select.Option value="endpoint">端点</Select.Option>
                  <Select.Option value="node">节点</Select.Option>
                  <Select.Option value="external">外部</Select.Option>
                </Select>
                <Select
                  value={ruleStatus}
                  style={{ width: 160 }}
                  onChange={(v) => {
                    setRuleStatus(v as any);
                    setRulePage(1);
                  }}
                >
                  <Select.Option value="all">全部状态</Select.Option>
                  <Select.Option value="enabled">启用</Select.Option>
                  <Select.Option value="disabled">禁用</Select.Option>
                </Select>
                <Button
                  theme="solid"
                  type="primary"
                  onClick={() => fetchRules(1, ruleSize)}
                  loading={ruleLoading}
                >
                  查询
                </Button>
                <Button
                  type="tertiary"
                  onClick={() => {
                    setRuleKeywords('');
                    setRuleKind('all');
                    setRuleStatus('all');
                    setRulePage(1);
                    fetchRules(1, ruleSize);
                  }}
                >
                  重置
                </Button>
              </div>
              <div>
                <Button
                  theme="solid"
                  type="primary"
                  onClick={() => {
                    if (!components.length) fetchComponents();
                    setRuleEditMode('create');
                    setRuleForm({
                      componentName: '',
                      componentType: 'action',
                      disabled: false,
                      useDesc: '',
                      useRuleDesc: '',
                      id: '',
                    });
                    setRuleDescMode('markdown');
                    setRuleDescPreview(true);
                    setRuleEditVisible(true);
                  }}
                >
                  新增规则
                </Button>
              </div>
            </div>
            {ruleError ? (
              <Typography.Text type="danger">加载失败：{ruleError}</Typography.Text>
            ) : null}
            <Spin spinning={ruleLoading}>
              <Table
                dataSource={pagedRules}
                rowKey={(r: any) => String(r.id)}
                columns={[
                  { title: '规则名称', render: (_, r: any) => String(r.name ?? '-'), width: 220 },
                  {
                    title: '类型',
                    render: (_, r: any) =>
                      String(r.kind) === 'endpoint'
                        ? '端点'
                        : String(r.kind) === 'node'
                        ? '节点'
                        : '外部',
                    width: 120,
                  },
                  {
                    title: '描述',
                    render: (_, r: any) => (
                      <Typography.Text type="tertiary">
                        {String(r.description ?? '')}
                      </Typography.Text>
                    ),
                  },
                  {
                    title: '状态',
                    width: 120,
                    render: (_, r: any) => (
                      <Tag size="small" color={r.enabled ? 'green' : 'orange'}>
                        {r.enabled ? '启用' : '禁用'}
                      </Tag>
                    ),
                  },
                  {
                    title: '更新时间',
                    width: 180,
                    render: (_, r: any) => {
                      const ts = Number(r.updateTime ?? 0);
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
                            const compType = String(r.type || '');
                            const mappedType = compType.startsWith('endpoint')
                              ? 'endpoint'
                              : r.kind === 'external'
                              ? 'external'
                              : 'action';
                            setRuleEditMode('view');
                            setRuleForm({
                              componentName: String(r.name || ''),
                              componentType: mappedType,
                              disabled: !Boolean(r.enabled),
                              useDesc: String(r.description || ''),
                              useRuleDesc: String(r.ruleDesc || ''),
                              id: String(r.id || ''),
                            });
                            const isHTML = /<[^>]+>/.test(String(r.ruleDesc || ''));
                            setRuleDescMode(isHTML ? 'rich' : 'markdown');
                            setRuleDescPreview(isHTML ? false : true);
                            setRuleEditVisible(true);
                          }}
                        >
                          详情
                        </Button>
                        <Button
                          size="small"
                          type="secondary"
                          onClick={() => {
                            const compType = String(r.type || '');
                            const mappedType = compType.startsWith('endpoint')
                              ? 'endpoint'
                              : r.kind === 'external'
                              ? 'external'
                              : 'action';
                            setRuleEditMode('edit');
                            setRuleForm({
                              componentName: String(r.name || ''),
                              componentType: mappedType,
                              disabled: !Boolean(r.enabled),
                              useDesc: String(r.description || ''),
                              useRuleDesc: String(r.ruleDesc || ''),
                              id: String(r.id || ''),
                            });
                            setRuleDescMode('markdown');
                            setRuleDescPreview(true);
                            setRuleEditVisible(true);
                          }}
                        >
                          编辑
                        </Button>
                        <Button
                          size="small"
                          type="danger"
                          onClick={() => Toast.info({ content: '删除功能待接入' })}
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
            <Modal
              title={
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span style={{ fontSize: 20 }}>🛠️</span>
                  <span style={{ fontSize: 16, fontWeight: 600 }}>
                    {ruleEditMode === 'edit'
                      ? '编辑组件规则'
                      : ruleEditMode === 'create'
                      ? '新增组件规则'
                      : '组件规则详情'}
                  </span>
                </div>
              }
              visible={ruleEditVisible}
              onCancel={() => setRuleEditVisible(false)}
              confirmLoading={ruleSubmitting}
              okText={
                ruleEditMode === 'edit' ? '更新' : ruleEditMode === 'create' ? '新增' : '关闭'
              }
              onOk={async () => {
                if (!ruleForm.componentName.trim()) {
                  Toast.warning({ content: '请输入组件名称' });
                  return;
                }
                if (ruleEditMode === 'view') {
                  setRuleEditVisible(false);
                  return;
                }
                setRuleSubmitting(true);
                try {
                  const content = String(ruleForm.useRuleDesc || '');
                  if (ruleEditMode === 'edit') {
                    if (!ruleForm.id) {
                      Toast.warning({ content: '缺少规则ID' });
                      setRuleSubmitting(false);
                      return;
                    }
                    await requestJSON('/componentUseRule/update', {
                      method: 'POST',
                      body: {
                        componentName: ruleForm.componentName,
                        componentType: ruleForm.componentType,
                        disabled: !!ruleForm.disabled,
                        useDesc: ruleForm.useDesc,
                        useRuleDesc: content,
                        id: String(ruleForm.id),
                      },
                    });
                    Toast.success({ content: '更新成功' });
                  } else {
                    await requestJSON('/componentUseRule/create', {
                      method: 'POST',
                      body: {
                        componentName: ruleForm.componentName,
                        componentType: ruleForm.componentType,
                        disabled: !!ruleForm.disabled,
                        useDesc: ruleForm.useDesc,
                        useRuleDesc: content,
                      },
                    });
                    Toast.success({ content: '新增成功' });
                  }
                  setRuleEditVisible(false);
                  await fetchRules(rulePage, ruleSize);
                } catch (e) {
                  Toast.error({ content: String((e as Error)?.message ?? e) });
                } finally {
                  setRuleSubmitting(false);
                }
              }}
              style={{ borderRadius: 16, width: 980 }}
            >
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <div>
                  <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                    组件名称 *
                  </Typography.Text>
                  {ruleEditMode === 'edit' ? (
                    <Input value={ruleForm.componentName} placeholder="组件名称不可修改" disabled />
                  ) : ruleEditMode === 'create' ? (
                    <Select
                      value={ruleForm.componentName}
                      style={{ width: '100%' }}
                      onChange={(v) => {
                        const name = String(v);
                        const opt = componentOptions.find((o) => o.value === name);
                        const compType =
                          opt?.category || (name.startsWith('endpoint') ? 'endpoint' : 'action');
                        setRuleForm({ ...ruleForm, componentName: name, componentType: compType });
                      }}
                    >
                      {componentOptions.map((o) => (
                        <Select.Option key={o.value} value={o.value}>
                          {o.label}
                        </Select.Option>
                      ))}
                    </Select>
                  ) : (
                    <Input value={ruleForm.componentName} disabled />
                  )}
                </div>
                <div style={{ display: 'flex', gap: 12 }}>
                  <div style={{ flex: 1 }}>
                    <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                      组件类型 *
                    </Typography.Text>
                    <Select
                      value={ruleForm.componentType}
                      onChange={(v) => setRuleForm({ ...ruleForm, componentType: String(v) })}
                      disabled={ruleEditMode === 'view'}
                    >
                      <Select.Option value="action">动作</Select.Option>
                      <Select.Option value="endpoint">端点</Select.Option>
                      <Select.Option value="external">外部</Select.Option>
                    </Select>
                  </div>
                  <div style={{ width: 160 }}>
                    <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                      组件状态
                    </Typography.Text>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Typography.Text>启用</Typography.Text>
                      <Select
                        value={!ruleForm.disabled ? '1' : '0'}
                        onChange={(v) => setRuleForm({ ...ruleForm, disabled: v !== '1' })}
                        disabled={ruleEditMode === 'view'}
                        style={{ width: 100 }}
                      >
                        <Select.Option value="1">是</Select.Option>
                        <Select.Option value="0">否</Select.Option>
                      </Select>
                      <Typography.Text type="tertiary">禁用</Typography.Text>
                    </div>
                  </div>
                </div>
                <div>
                  <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                    使用描述
                  </Typography.Text>
                  <TextArea
                    value={ruleForm.useDesc}
                    onChange={(v) => setRuleForm({ ...ruleForm, useDesc: String(v) })}
                    autosize={{ minRows: 3, maxRows: 6 }}
                    placeholder="请输入使用描述"
                    disabled={ruleEditMode === 'view'}
                  />
                </div>
                <div>
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                    }}
                  >
                    <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
                      使用规则描述
                    </Typography.Text>
                    <Select
                      value={ruleDescMode}
                      onChange={(v) => {
                        setRuleDescMode(v as any);
                        if (String(v) === 'rich') setRuleDescPreview(false);
                        else setRuleDescPreview(true);
                      }}
                      disabled={ruleEditMode === 'view'}
                      style={{ width: 140 }}
                    >
                      <Select.Option value="rich">富文本</Select.Option>
                      <Select.Option value="markdown">Markdown</Select.Option>
                    </Select>
                  </div>
                  <div
                    style={{
                      display: 'grid',
                      gridTemplateColumns:
                        ruleDescMode === 'markdown' && ruleDescPreview ? '1fr 1fr' : '1fr',
                      gap: 12,
                    }}
                  >
                    <div>
                      {ruleDescMode === 'rich' ? (
                        <ReactQuill
                          value={ruleForm.useRuleDesc}
                          onChange={(html) =>
                            setRuleForm({ ...ruleForm, useRuleDesc: String(html) })
                          }
                          theme="snow"
                          modules={{
                            toolbar:
                              ruleEditMode === 'view'
                                ? false
                                : [
                                    [{ header: [1, 2, 3, false] }],
                                    [
                                      'bold',
                                      'italic',
                                      'underline',
                                      'strike',
                                      'blockquote',
                                      'code-block',
                                    ],
                                    [{ list: 'ordered' }, { list: 'bullet' }],
                                    ['link'],
                                    ['clean'],
                                  ],
                          }}
                          readOnly={ruleEditMode === 'view'}
                          formats={[
                            'header',
                            'bold',
                            'italic',
                            'underline',
                            'strike',
                            'blockquote',
                            'code-block',
                            'list',
                            'ordered',
                            'bullet',
                            'link',
                          ]}
                        />
                      ) : (
                        <TextArea
                          value={ruleForm.useRuleDesc}
                          onChange={(v) => setRuleForm({ ...ruleForm, useRuleDesc: String(v) })}
                          autosize={{ minRows: 10, maxRows: 24 }}
                          placeholder="支持 Markdown 语法，提交时将自动转换为 HTML"
                          disabled={ruleEditMode === 'view'}
                        />
                      )}
                    </div>
                    {ruleDescMode === 'markdown' && ruleDescPreview && (
                      <div
                        style={{
                          border: '1px solid rgba(6,7,9,0.08)',
                          borderRadius: 8,
                          padding: 12,
                          background: '#FAFAFB',
                          overflowY: 'auto',
                          maxHeight: 420,
                        }}
                        dangerouslySetInnerHTML={{
                          __html:
                            ruleDescMode === 'markdown'
                              ? String(marked.parse(ruleForm.useRuleDesc || ''))
                              : String(ruleForm.useRuleDesc || ''),
                        }}
                      />
                    )}
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8 }}>
                    <Typography.Text>实时预览</Typography.Text>
                    <Select
                      value={ruleDescPreview ? '1' : '0'}
                      onChange={(v) => setRuleDescPreview(v === '1')}
                      disabled={ruleEditMode === 'view'}
                      style={{ width: 120 }}
                    >
                      <Select.Option value="1">开启</Select.Option>
                      <Select.Option value="0">关闭</Select.Option>
                    </Select>
                  </div>
                </div>
              </div>
            </Modal>
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
                <Typography.Text>共 {filteredRules.length} 条</Typography.Text>
                <Typography.Text type="tertiary" style={{ fontSize: 12 }}>
                  显示 {filteredRules.length === 0 ? 0 : (rulePage - 1) * ruleSize + 1}-
                  {Math.min(rulePage * ruleSize, filteredRules.length)} 条
                </Typography.Text>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <Select
                  value={ruleSize}
                  style={{ width: 110 }}
                  onChange={(v) => {
                    const s = Number(v);
                    setRuleSize(s);
                    setRulePage(1);
                    fetchRules(1, s);
                  }}
                >
                  <Select.Option value={10}>10 / 页</Select.Option>
                  <Select.Option value={20}>20 / 页</Select.Option>
                  <Select.Option value={50}>50 / 页</Select.Option>
                </Select>
                <Pagination
                  total={ruleTotal}
                  pageSize={ruleSize}
                  currentPage={rulePage}
                  onChange={(p: number) => {
                    setRulePage(p);
                    fetchRules(p, ruleSize);
                  }}
                />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ComponentsSection;
