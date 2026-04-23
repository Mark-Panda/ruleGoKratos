import React, { useCallback, useEffect, useMemo, useState } from 'react';

import {
  Button,
  Card,
  Input,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  Toast,
  Typography,
} from '@douyinfe/semi-ui';
import { IconDelete, IconEdit, IconEyeOpened, IconPlus, IconRefresh } from '@douyinfe/semi-icons';

import {
  createWorkspace,
  deleteWorkspace,
  getWorkspace,
  listWorkspaces,
  syncWorkspaceRepos,
  updateWorkspace,
  type WorkspaceItem,
  type WorkspacePayload,
} from '../../services/api-workspaces';

const { Text, Title } = Typography;

interface WorkspaceFormState {
  id: string;
  name: string;
  description: string;
  repositoryUrls: string[];
}

function generateUUID(): string {
  if (typeof globalThis !== 'undefined' && globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = Math.floor(Math.random() * 16);
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

function emptyForm(): WorkspaceFormState {
  return {
    id: generateUUID(),
    name: '',
    description: '',
    repositoryUrls: [''],
  };
}

function cleanRepositoryURLs(urls: string[]): string[] {
  return urls.map((x) => x.trim()).filter(Boolean);
}

export const WorkspacesSection: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [rows, setRows] = useState<WorkspaceItem[]>([]);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [syncingWorkspaceId, setSyncingWorkspaceId] = useState<string>('');
  const [busyText, setBusyText] = useState('');

  const [modalOpen, setModalOpen] = useState(false);
  const [modalMode, setModalMode] = useState<'create' | 'edit'>('create');
  const [editingId, setEditingId] = useState<string>('');
  const [form, setForm] = useState<WorkspaceFormState>(emptyForm());

  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<WorkspaceItem | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [syncConfirmOpen, setSyncConfirmOpen] = useState(false);
  const [syncTarget, setSyncTarget] = useState<WorkspaceItem | null>(null);
  const [syncConfirmLoading, setSyncConfirmLoading] = useState(false);

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const items = await listWorkspaces();
      setRows(items);
    } catch (e) {
      Toast.error(`加载失败：${e}`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  const openCreate = () => {
    setModalMode('create');
    setEditingId('');
    setForm(emptyForm());
    setModalOpen(true);
  };

  const openEdit = (row: WorkspaceItem) => {
    setModalMode('edit');
    setEditingId(row.id);
    setForm({
      id: row.id,
      name: row.name || '',
      description: row.description || '',
      repositoryUrls:
        row.repositoryUrls && row.repositoryUrls.length > 0 ? [...row.repositoryUrls] : [''],
    });
    setModalOpen(true);
  };

  const openDetail = async (row: WorkspaceItem) => {
    setDetailLoading(true);
    setDetailOpen(true);
    try {
      const item = await getWorkspace(row.id);
      setDetail(item);
    } catch (e) {
      Toast.error(`加载详情失败：${e}`);
      setDetail(null);
    } finally {
      setDetailLoading(false);
    }
  };

  const submit = async () => {
    const id = form.id.trim() || generateUUID();
    const name = form.name.trim();
    const repoURLs = cleanRepositoryURLs(form.repositoryUrls);
    if (!name) {
      Toast.warning('请填写工作区名称');
      return;
    }
    if (repoURLs.length === 0) {
      Toast.warning('请至少填写一个仓库地址');
      return;
    }
    const payload: WorkspacePayload = {
      id,
      name,
      description: form.description.trim(),
      repositoryUrls: repoURLs,
    };
    setBusyText(modalMode === 'create' ? '正在创建工作区并拉取仓库...' : '正在保存工作区并同步仓库...');
    setSubmitLoading(true);
    try {
      if (modalMode === 'create') {
        await createWorkspace(payload);
        Toast.success('已创建并拉取仓库');
      } else {
        await updateWorkspace(editingId, payload);
        Toast.success('已更新并同步仓库');
      }
      setModalOpen(false);
      void loadAll();
    } catch (e) {
      Toast.error(`保存失败：${e}`);
    } finally {
      setSubmitLoading(false);
      setBusyText('');
    }
  };

  const onDelete = (row: WorkspaceItem) => {
    Modal.confirm({
      title: '删除工作区',
      content: `确认删除工作区「${row.name}」吗？会同时删除该工作区目录与 .code-workspace 文件。`,
      onOk: async () => {
        try {
          await deleteWorkspace(row.id);
          Toast.success('已删除');
          void loadAll();
        } catch (e) {
          Toast.error(`删除失败：${e}`);
        }
      },
    });
  };

  const onSyncRepos = (row: WorkspaceItem) => {
    setSyncTarget(row);
    setSyncConfirmOpen(true);
  };

  const confirmSyncRepos = async () => {
    if (!syncTarget) {
      setSyncConfirmOpen(false);
      return;
    }
    setSyncConfirmLoading(true);
    setBusyText(`正在同步「${syncTarget.name}」仓库...`);
    setSyncingWorkspaceId(syncTarget.id);
    try {
      await syncWorkspaceRepos(syncTarget.id);
      Toast.success('仓库同步完成');
      setSyncConfirmOpen(false);
      setSyncTarget(null);
      void loadAll();
    } catch (e) {
      Toast.error(`同步失败：${e}`);
    } finally {
      setSyncConfirmLoading(false);
      setSyncingWorkspaceId('');
      setBusyText('');
    }
  };

  const columns = useMemo(
    () => [
      {
        title: '工作区 ID',
        dataIndex: 'id',
        width: 180,
        render: (value: string) => <Text code>{value}</Text>,
      },
      {
        title: '名称',
        dataIndex: 'name',
      },
      {
        title: '仓库数',
        key: 'repoCount',
        width: 90,
        render: (_: unknown, row: WorkspaceItem) => (
          <Tag color="blue">{row.repositoryUrls?.length || 0}</Tag>
        ),
      },
      {
        title: '工作区目录',
        dataIndex: 'rootDir',
        ellipsis: true,
      },
      {
        title: '更新时间',
        dataIndex: 'updatedAt',
        width: 220,
      },
      {
        title: '操作',
        key: 'op',
        width: 340,
        render: (_: unknown, row: WorkspaceItem) => (
          <Space>
            <Button
              size="small"
              type="tertiary"
              icon={<IconEyeOpened />}
              onClick={() => void openDetail(row)}
            >
              查看
            </Button>
            <Button size="small" type="tertiary" icon={<IconEdit />} onClick={() => openEdit(row)}>
              编辑
            </Button>
            <Button
              size="small"
              loading={syncingWorkspaceId === row.id}
              disabled={!!syncingWorkspaceId && syncingWorkspaceId !== row.id}
              onClick={() => onSyncRepos(row)}
            >
              同步仓库
            </Button>
            <Button size="small" type="danger" icon={<IconDelete />} onClick={() => onDelete(row)}>
              删除
            </Button>
          </Space>
        ),
      },
    ],
    []
  );

  return (
    <div style={{ position: 'relative' }}>
      <div style={{ padding: 24 }}>
      <Card>
        <Space vertical align="start" style={{ width: '100%' }}>
          <Title heading={5} style={{ margin: 0 }}>
            工作区管理
          </Title>
          <Text type="tertiary">
            每个工作区对应容器内独立目录 <Text code>/app/code_workspace/&lt;workspaceId&gt;</Text>
            ，并持久化配置文件
            <Text code>&lt;workspaceId&gt;.code-workspace</Text>。保存时会自动拉取/更新配置的所有
            Git 仓库。
          </Text>
          <Space>
            <Button type="primary" theme="solid" icon={<IconPlus />} onClick={openCreate}>
              新建工作区
            </Button>
            <Button icon={<IconRefresh />} onClick={() => void loadAll()}>
              刷新
            </Button>
          </Space>
        </Space>
      </Card>

      <Card style={{ marginTop: 16 }}>
        <Spin spinning={loading}>
          <Table columns={columns} dataSource={rows} rowKey="id" pagination={{ pageSize: 10 }} />
        </Spin>
      </Card>

      <Modal
        visible={modalOpen}
        title={modalMode === 'create' ? '新建工作区' : '编辑工作区'}
        onCancel={() => setModalOpen(false)}
        onOk={() => void submit()}
        okText={modalMode === 'create' ? '创建并拉取' : '保存并同步'}
        confirmLoading={submitLoading}
        okButtonProps={{ disabled: submitLoading }}
        cancelButtonProps={{ disabled: submitLoading }}
        width={760}
        maskClosable={false}
      >
        <Space vertical align="start" style={{ width: '100%' }}>
          <div style={{ width: '100%' }}>
            <Text type="tertiary">工作区 ID *</Text>
            <Input
              style={{ marginTop: 8 }}
              value={form.id}
              disabled
              placeholder="例如: ai-platform"
            />
            <Text type="tertiary" size="small">
              系统自动生成 UUID，用于目录名和配置文件名。
            </Text>
          </div>
          <div style={{ width: '100%' }}>
            <Text type="tertiary">工作区名称 *</Text>
            <Input
              style={{ marginTop: 8 }}
              value={form.name}
              placeholder="用于菜单和识别"
              onChange={(v) => setForm((f) => ({ ...f, name: v }))}
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text type="tertiary">描述</Text>
            <Input
              style={{ marginTop: 8 }}
              value={form.description}
              onChange={(v) => setForm((f) => ({ ...f, description: v }))}
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text type="tertiary">Git 仓库地址 *</Text>
            <Space vertical align="start" style={{ width: '100%', marginTop: 8 }}>
              {form.repositoryUrls.map((value, idx) => (
                <Space key={`repo-${idx}`} style={{ width: '100%' }}>
                  <Input
                    style={{ flex: 1 }}
                    value={value}
                    placeholder={
                      idx === 0
                        ? 'https://github.com/org/repo-a.git'
                        : 'git@github.com:org/repo-b.git'
                    }
                    onChange={(v) =>
                      setForm((f) => {
                        const next = [...f.repositoryUrls];
                        next[idx] = v;
                        return { ...f, repositoryUrls: next };
                      })
                    }
                  />
                  <Button
                    type="danger"
                    disabled={form.repositoryUrls.length <= 1}
                    onClick={() =>
                      setForm((f) => ({
                        ...f,
                        repositoryUrls:
                          f.repositoryUrls.length <= 1
                            ? f.repositoryUrls
                            : f.repositoryUrls.filter((_, i) => i !== idx),
                      }))
                    }
                  >
                    删除
                  </Button>
                </Space>
              ))}
              <Button
                icon={<IconPlus />}
                onClick={() =>
                  setForm((f) => ({
                    ...f,
                    repositoryUrls: [...f.repositoryUrls, ''],
                  }))
                }
              >
                新增仓库地址
              </Button>
            </Space>
          </div>
        </Space>
      </Modal>

      <Modal
        visible={syncConfirmOpen}
        title="同步工作区仓库"
        onCancel={() => {
          if (syncConfirmLoading) return;
          setSyncConfirmOpen(false);
          setSyncTarget(null);
        }}
        onOk={() => void confirmSyncRepos()}
        confirmLoading={syncConfirmLoading}
        okButtonProps={{ disabled: syncConfirmLoading }}
        cancelButtonProps={{ disabled: syncConfirmLoading }}
        maskClosable={false}
      >
        <Text>
          {syncTarget ? `确认重新同步「${syncTarget.name}」下全部仓库吗？` : '确认同步该工作区下全部仓库吗？'}
        </Text>
      </Modal>

      <Modal
        visible={detailOpen}
        title={detail ? `工作区详情：${detail.name}` : '工作区详情'}
        footer={null}
        width={760}
        onCancel={() => setDetailOpen(false)}
      >
        <Spin spinning={detailLoading}>
          {detail ? (
            <Space vertical align="start" style={{ width: '100%' }}>
              <Text>
                <Text strong>ID：</Text>
                <Text code>{detail.id}</Text>
              </Text>
              <Text>
                <Text strong>工作目录：</Text>
                <Text code>{detail.rootDir}</Text>
              </Text>
              <Text>
                <Text strong>配置文件：</Text>
                <Text code>{detail.configFile}</Text>
              </Text>
              <Text>
                <Text strong>仓库列表：</Text>
              </Text>
              <div
                style={{
                  width: '100%',
                  border: '1px solid var(--semi-color-border)',
                  borderRadius: 8,
                  padding: 12,
                }}
              >
                {detail.repositories?.map((repo) => (
                  <div key={`${repo.url}-${repo.dir}`} style={{ marginBottom: 8 }}>
                    <Text code>{repo.url}</Text>
                    <Text type="tertiary"> → {repo.dir}</Text>
                  </div>
                ))}
              </div>
              <Text type="tertiary">创建时间：{detail.createdAt || '-'}</Text>
              <Text type="tertiary">更新时间：{detail.updatedAt || '-'}</Text>
            </Space>
          ) : (
            <Text type="tertiary">暂无数据</Text>
          )}
        </Spin>
      </Modal>
      </div>
      {busyText ? (
        <div
          style={{
            position: 'fixed',
            inset: 0,
            background: 'rgba(28, 32, 41, 0.42)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 3000,
          }}
        >
          <div
            style={{
              background: '#fff',
              borderRadius: 10,
              padding: '18px 24px',
              minWidth: 320,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 12,
              boxShadow: '0 12px 24px rgba(0, 0, 0, 0.2)',
            }}
          >
            <Spin spinning size="large" />
            <Text>{busyText}</Text>
          </div>
        </div>
      ) : null}
    </div>
  );
};

export default WorkspacesSection;
