import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
  Button,
  Input,
  Modal,
  Select,
  Spin,
  Table,
  Tabs,
  Tag,
  TextArea,
  Toast,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconChevronDown,
  IconChevronRight,
  IconFile,
  IconFolder,
  IconFolderOpen,
  IconRefresh,
  IconSave,
  IconSearch,
  IconUpload,
} from '@douyinfe/semi-icons';

import { groupSkillPackages } from '../../utils/skill-packages';
import {
  MCPConfigItem,
  MCPConfigPayload,
  LlmConfigItem,
  LlmConfigPayload,
  LlmModelEntryItem,
  LlmModelEntryPayload,
  createMCPConfig,
  createLlmConfig,
  createLlmModelEntry,
  deleteMCPConfig,
  deleteLlmConfig,
  deleteLlmModelEntry,
  listMCPConfigs,
  listLlmConfigs,
  listSkills,
  readSkillFile,
  saveSkillFile,
  type SkillScope,
  type SkillItem,
  testMCPConfig,
  updateMCPConfig,
  updateLlmConfig,
  updateLlmModelEntry,
  uploadSkill,
} from '../../services/api-agent';

// ────────── 文件树结构（用于包内子目录分层展示）──────────
type FileTreeLeaf = { type: 'file'; name: string; file: SkillItem };
type FileTreeDir = { type: 'dir'; name: string; children: FileTreeNode[] };
type FileTreeNode = FileTreeLeaf | FileTreeDir;

function buildFileTree(pkgId: string, files: SkillItem[]): FileTreeNode[] {
  const root: FileTreeNode[] = [];
  for (const file of files) {
    let rel = file.path;
    if (rel.startsWith(pkgId + '/')) rel = rel.slice(pkgId.length + 1);
    const parts = rel.split('/');
    let cur: FileTreeNode[] = root;
    for (let i = 0; i < parts.length - 1; i++) {
      let dir = cur.find((n): n is FileTreeDir => n.type === 'dir' && n.name === parts[i]);
      if (!dir) {
        dir = { type: 'dir', name: parts[i], children: [] };
        cur.push(dir);
      }
      cur = dir.children;
    }
    cur.push({ type: 'file', name: parts[parts.length - 1], file });
  }
  return root;
}
// ─────────────────────────────────────────────────────────

const defaultMCPForm: MCPConfigPayload = {
  name: '',
  server: '',
  endpoint: '',
  headers: {},
  enabled: true,
  description: '',
  transport: 'http',
  stdio_command: '',
  stdio_args_json: '[]',
  stdio_env_json: '{}',
};

const defaultLlmConfigForm: LlmConfigPayload = {
  name: '',
  provider: 'openai',
  baseUrl: '',
  apiKey: '',
  enabled: true,
  description: '',
  models: [{ modelName: '', description: '', enabled: true }],
};

const defaultEntryForm: LlmModelEntryPayload = {
  modelName: '',
  description: '',
  enabled: true,
};

export const AgentSection: React.FC<{ view?: 'skills' | 'mcps' | 'models' }> = ({
  view = 'skills',
}) => {
  const [skillScope, setSkillScope] = useState<SkillScope>('system');
  const [skillRoot, setSkillRoot] = useState('skills');
  const [skills, setSkills] = useState<SkillItem[]>([]);
  const [skillLoading, setSkillLoading] = useState(false);
  const [skillKeyword, setSkillKeyword] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  // 文件系统浏览器状态
  const [expandedPkgs, setExpandedPkgs] = useState<Set<string>>(new Set());
  /** key = `${pkgId}::${dirRelPath}`，控制包内子目录的折叠 */
  const [expandedDirs, setExpandedDirs] = useState<Set<string>>(new Set());
  const [selectedFile, setSelectedFile] = useState<SkillItem | null>(null);
  const [editContent, setEditContent] = useState('');
  const [fileLoading, setFileLoading] = useState(false);
  const [fileSaving, setFileSaving] = useState(false);
  const [isDirty, setIsDirty] = useState(false);

  const [mcpList, setMcpList] = useState<MCPConfigItem[]>([]);
  const [mcpLoading, setMcpLoading] = useState(false);
  const [mcpModalVisible, setMcpModalVisible] = useState(false);
  const [mcpSubmitting, setMcpSubmitting] = useState(false);
  const [mcpEditing, setMcpEditing] = useState<MCPConfigItem | null>(null);
  const [mcpForm, setMcpForm] = useState<MCPConfigPayload>(defaultMCPForm);
  const [headersText, setHeadersText] = useState('{}');
  const [mcpTestLoadingId, setMcpTestLoadingId] = useState<number | null>(null);

  const [llmConfigList, setLlmConfigList] = useState<LlmConfigItem[]>([]);
  const [llmLoading, setLlmLoading] = useState(false);
  const [llmConfigModalVisible, setLlmConfigModalVisible] = useState(false);
  const [llmConfigSubmitting, setLlmConfigSubmitting] = useState(false);
  const [llmConfigEditing, setLlmConfigEditing] = useState<LlmConfigItem | null>(null);
  const [llmConfigForm, setLlmConfigForm] = useState<LlmConfigPayload>(defaultLlmConfigForm);
  const [entryModalVisible, setEntryModalVisible] = useState(false);
  const [entrySubmitting, setEntrySubmitting] = useState(false);
  const [entryEditing, setEntryEditing] = useState<LlmModelEntryItem | null>(null);
  const [entryConfigId, setEntryConfigId] = useState<number | null>(null);
  const [entryForm, setEntryForm] = useState<LlmModelEntryPayload>(defaultEntryForm);

  const fetchSkills = async (scope: SkillScope = skillScope) => {
    setSkillLoading(true);
    try {
      const data = await listSkills(scope);
      setSkillRoot(data.root || 'skills');
      setSkills(Array.isArray(data.items) ? data.items : []);
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setSkillLoading(false);
    }
  };

  const openSkillFile = useCallback(
    async (file: SkillItem) => {
      if (isDirty) {
        const ok = window.confirm('当前文件有未保存的修改，是否放弃？');
        if (!ok) return;
      }
      setSelectedFile(file);
      setIsDirty(false);
      setEditContent('');
      setFileLoading(true);
      try {
        const res = await readSkillFile(file.path, skillScope);
        setEditContent(res.content);
      } catch (e) {
        Toast.error({ content: `读取失败: ${String((e as Error)?.message ?? e)}` });
        setEditContent('');
      } finally {
        setFileLoading(false);
      }
    },
    [isDirty, skillScope]
  );

  const saveCurrentFile = useCallback(async () => {
    if (!selectedFile) return;
    setFileSaving(true);
    try {
      await saveSkillFile(selectedFile.path, editContent, skillScope);
      Toast.success({ content: '保存成功' });
      setIsDirty(false);
      await fetchSkills(skillScope);
    } catch (e) {
      Toast.error({ content: `保存失败: ${String((e as Error)?.message ?? e)}` });
    } finally {
      setFileSaving(false);
    }
  }, [selectedFile, editContent, skillScope]);

  const togglePackage = useCallback((pkgId: string) => {
    setExpandedPkgs((prev) => {
      const next = new Set(prev);
      if (next.has(pkgId)) {
        next.delete(pkgId);
      } else {
        next.add(pkgId);
      }
      return next;
    });
  }, []);

  const toggleDir = useCallback((dirKey: string) => {
    setExpandedDirs((prev) => {
      const next = new Set(prev);
      if (next.has(dirKey)) {
        next.delete(dirKey);
      } else {
        next.add(dirKey);
      }
      return next;
    });
  }, []);

  const fetchMCPs = async () => {
    setMcpLoading(true);
    try {
      const data = await listMCPConfigs();
      setMcpList(data);
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setMcpLoading(false);
    }
  };

  const fetchLlmConfigs = async () => {
    setLlmLoading(true);
    try {
      const data = await listLlmConfigs();
      setLlmConfigList(Array.isArray(data) ? data : []);
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setLlmLoading(false);
    }
  };

  useEffect(() => {
    if (view === 'skills') fetchSkills(skillScope);
    if (view === 'mcps') fetchMCPs();
    if (view === 'models') fetchLlmConfigs();
  }, [view, skillScope]);

  const skillPackages = useMemo(() => groupSkillPackages(skills), [skills]);

  const filteredSkillPackages = useMemo(() => {
    const kw = skillKeyword.trim().toLowerCase();
    if (!kw) return skillPackages;
    return skillPackages.filter((pkg) => {
      if (pkg.id.toLowerCase().includes(kw)) return true;
      return pkg.files.some((f) => {
        const p = String(f.path || '').toLowerCase();
        const n = String(f.name || '').toLowerCase();
        return p.includes(kw) || n.includes(kw);
      });
    });
  }, [skillPackages, skillKeyword]);

  const handleSkillScopeChange = useCallback((next: string) => {
    const scope: SkillScope =
      next === 'agent' ? 'agent' : next === 'workflow' ? 'workflow' : 'system';
    setSkillScope(scope);
    setSelectedFile(null);
    setEditContent('');
    setIsDirty(false);
    setExpandedPkgs(new Set());
    setExpandedDirs(new Set());
  }, []);

  const openCreateMCP = () => {
    setMcpEditing(null);
    setMcpForm(defaultMCPForm);
    setHeadersText('{}');
    setMcpModalVisible(true);
  };

  const openEditMCP = (item: MCPConfigItem) => {
    const headers = item.headers || {};
    const tr = (item.transport || 'http').toLowerCase() === 'stdio' ? 'stdio' : 'http';
    setMcpEditing(item);
    setMcpForm({
      name: item.name,
      server: item.server,
      endpoint: item.endpoint || '',
      headers,
      enabled: !!item.enabled,
      description: item.description || '',
      transport: tr,
      stdio_command: item.stdio_command || '',
      stdio_args_json: item.stdio_args_json?.trim() ? item.stdio_args_json : '[]',
      stdio_env_json: item.stdio_env_json?.trim() ? item.stdio_env_json : '{}',
    });
    setHeadersText(JSON.stringify(headers, null, 2));
    setMcpModalVisible(true);
  };

  const runMcpTest = async (r: MCPConfigItem) => {
    setMcpTestLoadingId(r.id);
    try {
      const res = await testMCPConfig(r.id);
      if (res.ok) {
        Toast.success({ content: res.message || '测试成功' });
        Modal.info({
          title: 'MCP 测试成功',
          width: 640,
          content: (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <Typography.Text type="secondary">{res.message}</Typography.Text>
              {res.serverName ? (
                <Typography.Text>
                  服务端名称：<Typography.Text strong>{res.serverName}</Typography.Text>
                </Typography.Text>
              ) : null}
              {res.protocolVersion ? (
                <Typography.Text>
                  协议版本：<Typography.Text code>{res.protocolVersion}</Typography.Text>
                </Typography.Text>
              ) : null}
              {res.toolNames && res.toolNames.length > 0 ? (
                <div>
                  <Typography.Text strong>工具列表（{res.toolNames.length}）</Typography.Text>
                  <pre
                    style={{
                      marginTop: 8,
                      maxHeight: 280,
                      overflow: 'auto',
                      padding: 8,
                      background: 'rgba(6,7,9,0.04)',
                      borderRadius: 8,
                      fontSize: 12,
                    }}
                  >
                    {res.toolNames.join('\n')}
                  </pre>
                </div>
              ) : null}
            </div>
          ),
        });
      } else {
        Toast.error({ content: res.message || '测试失败' });
      }
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setMcpTestLoadingId(null);
    }
  };

  const submitMCP = async () => {
    if (!mcpForm.name.trim() || !mcpForm.server.trim()) {
      Toast.warning({ content: '请填写名称与 Server' });
      return;
    }
    const tr = (mcpForm.transport || 'http').toLowerCase() === 'stdio' ? 'stdio' : 'http';
    if (tr === 'http' && !mcpForm.endpoint.trim()) {
      Toast.warning({ content: 'HTTP 模式请填写 Endpoint' });
      return;
    }
    if (tr === 'stdio' && !mcpForm.stdio_command?.trim()) {
      Toast.warning({ content: 'stdio 模式请填写启动命令（stdio_command）' });
      return;
    }
    let parsedHeaders: Record<string, any> = {};
    try {
      parsedHeaders = headersText.trim() ? JSON.parse(headersText) : {};
    } catch {
      Toast.error({ content: 'headers 必须是合法 JSON' });
      return;
    }
    if (tr === 'stdio') {
      try {
        const argsRaw = mcpForm.stdio_args_json?.trim() || '[]';
        const args = JSON.parse(argsRaw);
        if (!Array.isArray(args) || !args.every((x) => typeof x === 'string')) {
          Toast.error({
            content:
              'stdio_args_json 须为 JSON 字符串数组，例如 ["-y","@modelcontextprotocol/server-filesystem"]',
          });
          return;
        }
      } catch {
        Toast.error({ content: 'stdio_args_json 须为合法 JSON 数组' });
        return;
      }
      try {
        const envRaw = mcpForm.stdio_env_json?.trim() || '{}';
        const env = JSON.parse(envRaw);
        if (env === null || typeof env !== 'object' || Array.isArray(env)) {
          Toast.error({ content: 'stdio_env_json 须为 JSON 对象' });
          return;
        }
      } catch {
        Toast.error({ content: 'stdio_env_json 须为合法 JSON 对象' });
        return;
      }
    }
    const payload: MCPConfigPayload = {
      ...mcpForm,
      transport: tr,
      headers: parsedHeaders,
      stdio_args_json: mcpForm.stdio_args_json?.trim() || '[]',
      stdio_env_json: mcpForm.stdio_env_json?.trim() || '{}',
    };
    setMcpSubmitting(true);
    try {
      if (mcpEditing) {
        await updateMCPConfig(mcpEditing.id, payload);
        Toast.success({ content: '更新成功' });
      } else {
        await createMCPConfig(payload);
        Toast.success({ content: '创建成功' });
      }
      setMcpModalVisible(false);
      await fetchMCPs();
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setMcpSubmitting(false);
    }
  };

  const openCreateLlmConfig = () => {
    setLlmConfigEditing(null);
    setLlmConfigForm(defaultLlmConfigForm);
    setLlmConfigModalVisible(true);
  };

  const openEditLlmConfig = (item: LlmConfigItem) => {
    setLlmConfigEditing(item);
    setLlmConfigForm({
      name: item.name,
      provider: item.provider || 'openai',
      baseUrl: item.baseUrl || '',
      apiKey: '',
      enabled: !!item.enabled,
      description: item.description || '',
    });
    setLlmConfigModalVisible(true);
  };

  const addLlmModelDraftRow = () => {
    setLlmConfigForm((prev) => ({
      ...prev,
      models: [...(prev.models || []), { modelName: '', description: '', enabled: true }],
    }));
  };

  const removeLlmModelDraftRow = (index: number) => {
    setLlmConfigForm((prev) => {
      const models = [...(prev.models || [])];
      models.splice(index, 1);
      return { ...prev, models };
    });
  };

  const updateLlmModelDraftRow = (index: number, patch: Partial<LlmModelEntryPayload>) => {
    setLlmConfigForm((prev) => {
      const models = [...(prev.models || [])];
      const cur = models[index];
      if (!cur) return prev;
      models[index] = { ...cur, ...patch };
      return { ...prev, models };
    });
  };

  const submitLlmConfig = async () => {
    if (!llmConfigForm.name.trim()) {
      Toast.warning({ content: '请填写配置名称' });
      return;
    }
    const base: LlmConfigPayload = {
      name: llmConfigForm.name.trim(),
      provider: (llmConfigForm.provider || 'openai').trim(),
      baseUrl: llmConfigForm.baseUrl.trim(),
      description: llmConfigForm.description.trim(),
      apiKey: llmConfigForm.apiKey.trim(),
      enabled: llmConfigForm.enabled,
    };
    setLlmConfigSubmitting(true);
    try {
      if (llmConfigEditing) {
        await updateLlmConfig(llmConfigEditing.id, base);
        Toast.success({ content: '更新成功' });
      } else {
        const models = (llmConfigForm.models || [])
          .map((m) => ({
            modelName: m.modelName.trim(),
            description: m.description.trim(),
            enabled: m.enabled,
          }))
          .filter((m) => m.modelName !== '');
        const names = models.map((m) => m.modelName);
        if (names.length !== new Set(names).size) {
          Toast.warning({ content: '模型 ID 列表中存在重复，请检查' });
          return;
        }
        await createLlmConfig({ ...base, models });
        Toast.success({ content: '创建成功' });
      }
      setLlmConfigModalVisible(false);
      await fetchLlmConfigs();
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setLlmConfigSubmitting(false);
    }
  };

  const openCreateEntry = (configId: number) => {
    setEntryEditing(null);
    setEntryConfigId(configId);
    setEntryForm(defaultEntryForm);
    setEntryModalVisible(true);
  };

  const openEditEntry = (_configId: number, row: LlmModelEntryItem) => {
    setEntryEditing(row);
    setEntryConfigId(_configId);
    setEntryForm({
      modelName: row.modelName,
      description: row.description || '',
      enabled: !!row.enabled,
    });
    setEntryModalVisible(true);
  };

  const submitEntry = async () => {
    if (!entryForm.modelName.trim()) {
      Toast.warning({ content: '请填写模型 ID（modelName）' });
      return;
    }
    const payload: LlmModelEntryPayload = {
      modelName: entryForm.modelName.trim(),
      description: entryForm.description.trim(),
      enabled: entryForm.enabled,
    };
    setEntrySubmitting(true);
    try {
      if (entryEditing) {
        await updateLlmModelEntry(entryEditing.id, payload);
        Toast.success({ content: '更新成功' });
      } else if (entryConfigId != null) {
        await createLlmModelEntry(entryConfigId, payload);
        Toast.success({ content: '已添加模型' });
      }
      setEntryModalVisible(false);
      await fetchLlmConfigs();
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) });
    } finally {
      setEntrySubmitting(false);
    }
  };

  const fileExt = (path: string) => {
    const i = path.lastIndexOf('.');
    return i >= 0 ? path.slice(i + 1).toLowerCase() : '';
  };

  const extColor = (ext: string): string => {
    if (ext === 'md') return '#1677ff';
    if (ext === 'json') return '#d46b08';
    if (ext === 'yaml' || ext === 'yml') return '#389e0d';
    if (ext === 'txt') return '#722ed1';
    return '#595959';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
      {view === 'skills' && (
        <>
          <div style={{ padding: '12px 16px 0' }}>
            <Tabs
              type="line"
              activeKey={skillScope}
              onChange={(k) => handleSkillScopeChange(String(k))}
            >
              <Tabs.TabPane itemKey="system" tab="系统技能" />
              <Tabs.TabPane itemKey="agent" tab="Agent技能" />
              <Tabs.TabPane itemKey="workflow" tab="工作流技能" />
            </Tabs>
          </div>
          <div
            style={{
              display: 'flex',
              flex: 1,
              minHeight: 0,
              overflow: 'hidden',
              padding: '8px 16px 12px',
              gap: 12,
            }}
          >
            {/* 左侧文件树 */}
            <div
              style={{
                width: 260,
                flexShrink: 0,
                display: 'flex',
                flexDirection: 'column',
                border: '1px solid rgba(28,31,35,0.08)',
                borderRadius: 8,
                overflow: 'hidden',
                background: '#fff',
              }}
            >
              {/* 文件树顶部操作栏 */}
              <div
                style={{
                  padding: '8px 10px',
                  borderBottom: '1px solid rgba(28,31,35,0.06)',
                  display: 'flex',
                  gap: 6,
                  alignItems: 'center',
                }}
              >
                <Input
                  prefix={<IconSearch size="small" />}
                  value={skillKeyword}
                  onChange={setSkillKeyword}
                  placeholder="搜索文件"
                  showClear
                  size="small"
                  style={{ flex: 1 }}
                />
                <Tooltip content="刷新">
                  <Button
                    size="small"
                    theme="borderless"
                    icon={<IconRefresh />}
                    loading={skillLoading}
                    onClick={() => fetchSkills(skillScope)}
                  />
                </Tooltip>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".zip,application/zip"
                  style={{ display: 'none' }}
                  onChange={async (e) => {
                    const file = e.target.files?.[0];
                    if (!file) return;
                    if (skillScope !== 'system') {
                      Toast.warning({ content: '仅系统技能允许上传技能包' });
                      e.target.value = '';
                      return;
                    }
                    try {
                      await uploadSkill(file, file.name, skillScope);
                      Toast.success({ content: '上传成功' });
                      await fetchSkills(skillScope);
                    } catch (err) {
                      Toast.error({ content: String((err as Error)?.message ?? err) });
                    } finally {
                      e.target.value = '';
                    }
                  }}
                />
                {skillScope === 'system' && (
                  <Tooltip content="上传技能包(zip)">
                    <Button
                      size="small"
                      theme="borderless"
                      icon={<IconUpload />}
                      onClick={() => fileInputRef.current?.click()}
                    />
                  </Tooltip>
                )}
              </div>

              {/* 目录提示 */}
              <div
                style={{
                  padding: '4px 10px',
                  background: 'rgba(22,119,255,0.04)',
                  borderBottom: '1px solid rgba(28,31,35,0.06)',
                }}
              >
                <Typography.Text type="tertiary" size="small" ellipsis={{ showTooltip: true }}>
                  {skillRoot}
                </Typography.Text>
              </div>

              {/* 文件树列表：flex: 1 + minHeight:0 保证可滚动；不用 Spin 包裹以免破坏高度链 */}
              <div style={{ flex: 1, minHeight: 0, position: 'relative', overflow: 'hidden' }}>
                {skillLoading && (
                  <div
                    style={{
                      position: 'absolute',
                      inset: 0,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: 'rgba(255,255,255,0.7)',
                      zIndex: 1,
                    }}
                  >
                    <Spin spinning />
                  </div>
                )}
                <div style={{ height: '100%', overflowY: 'auto', padding: '4px 0' }}>
                  {filteredSkillPackages.length === 0 && !skillLoading && (
                    <div style={{ padding: '16px 12px', textAlign: 'center' }}>
                      <Typography.Text type="tertiary" size="small">
                        暂无技能文件
                      </Typography.Text>
                    </div>
                  )}
                  {filteredSkillPackages.map((pkg) => {
                    const isExpanded = expandedPkgs.has(pkg.id);
                    const tree = isExpanded ? buildFileTree(pkg.id, pkg.files) : [];

                    const renderTree = (
                      nodes: FileTreeNode[],
                      depth: number,
                      pathPrefix: string
                    ): React.ReactNode =>
                      nodes.map((node) => {
                        if (node.type === 'file') {
                          const f = node.file;
                          const isSelected = selectedFile?.path === f.path;
                          const ext = fileExt(f.path);
                          return (
                            <div
                              key={f.path}
                              onClick={() => void openSkillFile(f)}
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: 4,
                                padding: `3px 8px 3px ${10 + depth * 14}px`,
                                cursor: 'pointer',
                                borderRadius: 4,
                                margin: '1px 4px',
                                background: isSelected ? 'rgba(22,119,255,0.10)' : 'transparent',
                                borderLeft: isSelected
                                  ? '2px solid #1677ff'
                                  : '2px solid transparent',
                              }}
                              onMouseEnter={(e) => {
                                if (!isSelected)
                                  (e.currentTarget as HTMLDivElement).style.background =
                                    'rgba(28,31,35,0.04)';
                              }}
                              onMouseLeave={(e) => {
                                (e.currentTarget as HTMLDivElement).style.background = isSelected
                                  ? 'rgba(22,119,255,0.10)'
                                  : 'transparent';
                              }}
                            >
                              <IconFile
                                size="small"
                                style={{ color: extColor(ext), flexShrink: 0 }}
                              />
                              <Typography.Text
                                ellipsis={{ showTooltip: true }}
                                style={{
                                  flex: 1,
                                  fontSize: 12,
                                  color: isSelected ? '#1677ff' : undefined,
                                }}
                              >
                                {node.name}
                              </Typography.Text>
                              <Tag
                                size="small"
                                style={{ flexShrink: 0, fontSize: 10, padding: '0 4px' }}
                              >
                                {ext}
                              </Tag>
                            </div>
                          );
                        }
                        // dir node：支持折叠
                        const dirFullKey = `${pkg.id}::${pathPrefix}${node.name}`;
                        const isDirExpanded = expandedDirs.has(dirFullKey);
                        return (
                          <div key={dirFullKey}>
                            <div
                              onClick={() => toggleDir(dirFullKey)}
                              style={{
                                display: 'flex',
                                alignItems: 'center',
                                gap: 4,
                                padding: `3px 8px 3px ${10 + depth * 14}px`,
                                cursor: 'pointer',
                                userSelect: 'none',
                                borderRadius: 4,
                                margin: '1px 4px',
                              }}
                              onMouseEnter={(e) => {
                                (e.currentTarget as HTMLDivElement).style.background =
                                  'rgba(28,31,35,0.04)';
                              }}
                              onMouseLeave={(e) => {
                                (e.currentTarget as HTMLDivElement).style.background =
                                  'transparent';
                              }}
                            >
                              <span
                                style={{
                                  color: '#9ca3af',
                                  flexShrink: 0,
                                  fontSize: 12,
                                  lineHeight: 1,
                                }}
                              >
                                {isDirExpanded ? (
                                  <IconChevronDown size="small" />
                                ) : (
                                  <IconChevronRight size="small" />
                                )}
                              </span>
                              {isDirExpanded ? (
                                <IconFolderOpen
                                  size="small"
                                  style={{ color: '#faad14', flexShrink: 0 }}
                                />
                              ) : (
                                <IconFolder
                                  size="small"
                                  style={{ color: '#faad14', flexShrink: 0 }}
                                />
                              )}
                              <Typography.Text style={{ fontSize: 12, color: '#374151' }}>
                                {node.name}
                              </Typography.Text>
                            </div>
                            {isDirExpanded &&
                              renderTree(node.children, depth + 1, `${pathPrefix}${node.name}/`)}
                          </div>
                        );
                      });

                    return (
                      <div key={pkg.id}>
                        {/* 套装行 */}
                        <div
                          onClick={() => togglePackage(pkg.id)}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 4,
                            padding: '4px 8px',
                            cursor: 'pointer',
                            userSelect: 'none',
                            borderRadius: 4,
                            margin: '1px 4px',
                            background: isExpanded ? 'rgba(22,119,255,0.06)' : 'transparent',
                          }}
                          onMouseEnter={(e) => {
                            if (!isExpanded)
                              (e.currentTarget as HTMLDivElement).style.background =
                                'rgba(28,31,35,0.04)';
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLDivElement).style.background = isExpanded
                              ? 'rgba(22,119,255,0.06)'
                              : 'transparent';
                          }}
                        >
                          <span style={{ color: '#6b7280', flexShrink: 0, fontSize: 12 }}>
                            {isExpanded ? (
                              <IconChevronDown size="small" />
                            ) : (
                              <IconChevronRight size="small" />
                            )}
                          </span>
                          {isExpanded ? (
                            <IconFolderOpen
                              size="small"
                              style={{ color: '#faad14', flexShrink: 0 }}
                            />
                          ) : (
                            <IconFolder size="small" style={{ color: '#faad14', flexShrink: 0 }} />
                          )}
                          <Typography.Text
                            ellipsis={{ showTooltip: true }}
                            style={{ flex: 1, fontSize: 13, fontWeight: 500 }}
                          >
                            {pkg.id}
                          </Typography.Text>
                          <Tag size="small" color="blue" style={{ flexShrink: 0 }}>
                            {pkg.files.length}
                          </Tag>
                        </div>

                        {/* 递归文件树 */}
                        {isExpanded && renderTree(tree, 1, '')}
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>

            {/* 右侧编辑器 */}
            <div
              style={{
                flex: 1,
                display: 'flex',
                flexDirection: 'column',
                border: '1px solid rgba(28,31,35,0.08)',
                borderRadius: 8,
                overflow: 'hidden',
                background: '#fff',
              }}
            >
              {/* 编辑器顶部工具栏 */}
              <div
                style={{
                  padding: '8px 12px',
                  borderBottom: '1px solid rgba(28,31,35,0.06)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  background: 'rgba(28,31,35,0.02)',
                }}
              >
                {selectedFile ? (
                  <>
                    <IconFile
                      size="small"
                      style={{ color: extColor(fileExt(selectedFile.path)), flexShrink: 0 }}
                    />
                    <Typography.Text style={{ flex: 1, fontSize: 13, fontFamily: 'monospace' }}>
                      {selectedFile.path}
                    </Typography.Text>
                    {isDirty && (
                      <Tag color="orange" size="small">
                        未保存
                      </Tag>
                    )}
                    <Button
                      size="small"
                      type="primary"
                      theme="solid"
                      icon={<IconSave />}
                      loading={fileSaving}
                      disabled={!isDirty}
                      onClick={() => void saveCurrentFile()}
                    >
                      保存
                    </Button>
                  </>
                ) : (
                  <Typography.Text type="tertiary" size="small">
                    从左侧选择文件进行查看或编辑
                  </Typography.Text>
                )}
              </div>

              {/* 编辑区：flex:1 + minHeight:0 保证高度填满；不用 Spin 包裹（Spin 内部 wrapper 会截断高度） */}
              <div style={{ flex: 1, minHeight: 0, position: 'relative' }}>
                {selectedFile ? (
                  <>
                    {fileLoading && (
                      <div
                        style={{
                          position: 'absolute',
                          inset: 0,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          background: 'rgba(255,255,255,0.75)',
                          zIndex: 2,
                        }}
                      >
                        <Spin spinning />
                      </div>
                    )}
                    <textarea
                      value={editContent}
                      onChange={(e) => {
                        setEditContent(e.target.value);
                        setIsDirty(true);
                      }}
                      spellCheck={false}
                      style={{
                        display: 'block',
                        width: '100%',
                        height: '100%',
                        resize: 'none',
                        border: 'none',
                        outline: 'none',
                        padding: '12px 16px',
                        fontFamily: '"JetBrains Mono", "Fira Code", "Cascadia Code", monospace',
                        fontSize: 13,
                        lineHeight: 1.6,
                        background: 'transparent',
                        color: '#1c1f23',
                        boxSizing: 'border-box',
                        overflowY: 'auto',
                        tabSize: 2,
                      }}
                      onKeyDown={(e) => {
                        if ((e.metaKey || e.ctrlKey) && e.key === 's') {
                          e.preventDefault();
                          void saveCurrentFile();
                        }
                        if (e.key === 'Tab') {
                          e.preventDefault();
                          const el = e.currentTarget;
                          const start = el.selectionStart;
                          const end = el.selectionEnd;
                          const newVal =
                            editContent.substring(0, start) + '  ' + editContent.substring(end);
                          setEditContent(newVal);
                          setIsDirty(true);
                          requestAnimationFrame(() => {
                            el.selectionStart = start + 2;
                            el.selectionEnd = start + 2;
                          });
                        }
                      }}
                    />
                  </>
                ) : (
                  <div
                    style={{
                      height: '100%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexDirection: 'column',
                      gap: 8,
                      color: 'rgba(28,31,35,0.25)',
                    }}
                  >
                    <IconFile size="extra-large" />
                    <Typography.Text type="quaternary">选择左侧文件开始编辑</Typography.Text>
                  </div>
                )}
              </div>
            </div>
          </div>
        </>
      )}

      {view === 'mcps' && (
        <div style={{ padding: '12px 16px', display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div
            style={{
              background: '#fff',
              borderRadius: 8,
              border: '1px solid rgba(6,7,9,0.06)',
              padding: '8px 12px',
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 8,
            }}
          >
            <Button onClick={() => fetchMCPs()}>刷新</Button>
            <Button theme="solid" type="primary" onClick={openCreateMCP}>
              新增 MCP
            </Button>
          </div>
          <Spin spinning={mcpLoading}>
            <Table
              dataSource={mcpList}
              rowKey="id"
              pagination={{ pageSize: 10 }}
              columns={[
                { title: '名称', dataIndex: 'name', width: 180 },
                { title: 'Server', dataIndex: 'server', width: 180 },
                {
                  title: '传输',
                  width: 88,
                  render: (_, r) =>
                    String(r.transport || 'http').toLowerCase() === 'stdio' ? 'stdio' : 'http',
                },
                {
                  title: 'Endpoint / 命令',
                  ellipsis: true,
                  render: (_, r) => {
                    if (String(r.transport || 'http').toLowerCase() === 'stdio') {
                      return r.stdio_command || '—';
                    }
                    return r.endpoint || '—';
                  },
                },
                {
                  title: '状态',
                  width: 100,
                  render: (_, r) => (r.enabled ? '启用' : '禁用'),
                },
                {
                  title: '操作',
                  width: 260,
                  render: (_, r) => (
                    <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                      <Button
                        size="small"
                        loading={mcpTestLoadingId === r.id}
                        onClick={() => runMcpTest(r)}
                      >
                        测试
                      </Button>
                      <Button size="small" onClick={() => openEditMCP(r)}>
                        编辑
                      </Button>
                      <Button
                        size="small"
                        type="danger"
                        onClick={async () => {
                          try {
                            await deleteMCPConfig(r.id);
                            Toast.success({ content: '删除成功' });
                            fetchMCPs();
                          } catch (e) {
                            Toast.error({ content: String((e as Error)?.message ?? e) });
                          }
                        }}
                      >
                        删除
                      </Button>
                    </div>
                  ),
                },
              ]}
            />
          </Spin>
        </div>
      )}

      {view === 'models' && (
        <div style={{ padding: '12px 16px', display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div
            style={{
              background: '#fff',
              borderRadius: 8,
              border: '1px solid rgba(6,7,9,0.06)',
              padding: '8px 12px',
              display: 'flex',
              justifyContent: 'flex-end',
              gap: 8,
            }}
          >
            <Button onClick={() => fetchLlmConfigs()}>刷新</Button>
            <Button theme="solid" type="primary" onClick={openCreateLlmConfig}>
              新增配置
            </Button>
          </div>
          <Spin spinning={llmLoading}>
            <Table
              dataSource={llmConfigList}
              rowKey="id"
              pagination={{ pageSize: 8 }}
              expandRowByClick
              expandedRowRender={(record) => (
                <div
                  style={{
                    padding: '8px 0 16px 24px',
                    background: 'rgba(6,7,9,0.02)',
                    borderRadius: 8,
                  }}
                >
                  <div
                    style={{
                      marginBottom: 8,
                      display: 'flex',
                      justifyContent: 'flex-end',
                    }}
                  >
                    <Button
                      size="small"
                      theme="solid"
                      disabled={!record?.id}
                      onClick={() => record?.id && openCreateEntry(record.id)}
                    >
                      添加模型
                    </Button>
                  </div>
                  <Table
                    size="small"
                    dataSource={record?.models || []}
                    rowKey={(r?: LlmModelEntryItem) =>
                      String(r?.id ?? r?.modelName ?? 'model-unknown')
                    }
                    pagination={false}
                    columns={[
                      { title: '模型 ID', dataIndex: 'modelName', width: 200 },
                      {
                        title: '说明',
                        dataIndex: 'description',
                        ellipsis: true,
                      },
                      {
                        title: '状态',
                        width: 72,
                        render: (_, r) => (r.enabled ? '启用' : '禁用'),
                      },
                      {
                        title: '操作',
                        width: 160,
                        render: (_, r) => (
                          <div style={{ display: 'flex', gap: 8 }}>
                            <Button
                              size="small"
                              disabled={!record?.id}
                              onClick={() => record?.id && openEditEntry(record.id, r)}
                            >
                              编辑
                            </Button>
                            <Button
                              size="small"
                              type="danger"
                              onClick={async () => {
                                try {
                                  await deleteLlmModelEntry(r.id);
                                  Toast.success({ content: '已删除' });
                                  fetchLlmConfigs();
                                } catch (e) {
                                  Toast.error({
                                    content: String((e as Error)?.message ?? e),
                                  });
                                }
                              }}
                            >
                              删除
                            </Button>
                          </div>
                        ),
                      },
                    ]}
                  />
                </div>
              )}
              columns={[
                { title: '配置名称', dataIndex: 'name', width: 160 },
                { title: '厂商', dataIndex: 'provider', width: 100 },
                {
                  title: '模型数',
                  width: 80,
                  render: (_, r) => r.models?.length ?? 0,
                },
                {
                  title: 'Base URL',
                  dataIndex: 'baseUrl',
                  ellipsis: true,
                  render: (v: string) => v || '—',
                },
                {
                  title: 'API Key',
                  width: 120,
                  render: (_, r) => r.apiKey || '未配置',
                },
                {
                  title: '状态',
                  width: 72,
                  render: (_, r) => (r.enabled ? '启用' : '禁用'),
                },
                {
                  title: '操作',
                  width: 180,
                  render: (_, r) => (
                    <div style={{ display: 'flex', gap: 8 }}>
                      <Button size="small" onClick={() => openEditLlmConfig(r)}>
                        编辑
                      </Button>
                      <Button
                        size="small"
                        type="danger"
                        onClick={async () => {
                          try {
                            await deleteLlmConfig(r.id);
                            Toast.success({ content: '删除成功' });
                            fetchLlmConfigs();
                          } catch (e) {
                            Toast.error({ content: String((e as Error)?.message ?? e) });
                          }
                        }}
                      >
                        删除
                      </Button>
                    </div>
                  ),
                },
              ]}
            />
          </Spin>
        </div>
      )}

      <Modal
        title={llmConfigEditing ? '编辑 LLM 配置' : '新增 LLM 配置'}
        visible={llmConfigModalVisible}
        onCancel={() => setLlmConfigModalVisible(false)}
        onOk={submitLlmConfig}
        confirmLoading={llmConfigSubmitting}
        style={{ width: llmConfigEditing ? 640 : 720 }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <Input
            value={llmConfigForm.name}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, name: String(v) })}
            placeholder="配置名称（唯一），其下可添加多条模型 ID"
          />
          <Input
            value={llmConfigForm.provider}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, provider: String(v) })}
            placeholder="厂商，如 openai / anthropic / azure / ollama"
          />
          <Input
            value={llmConfigForm.baseUrl}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, baseUrl: String(v) })}
            placeholder="API Base URL（可选）"
          />
          <Input
            mode="password"
            value={llmConfigForm.apiKey}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, apiKey: String(v) })}
            placeholder={
              llmConfigEditing
                ? `当前: ${llmConfigEditing.apiKey || '未配置'}，留空则不修改`
                : 'API Key（可选）'
            }
          />
          {llmConfigEditing && (
            <Typography.Text type="tertiary" size="small">
              编辑时留空 API Key 将保留原值。模型请在下方表格展开行中增删改。
            </Typography.Text>
          )}
          <Select
            value={llmConfigForm.enabled ? '1' : '0'}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, enabled: v === '1' })}
          >
            <Select.Option value="1">启用</Select.Option>
            <Select.Option value="0">禁用</Select.Option>
          </Select>
          {!llmConfigEditing && (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 10,
                padding: 12,
                background: 'rgba(6,7,9,0.03)',
                borderRadius: 8,
                border: '1px solid rgba(6,7,9,0.06)',
              }}
            >
              <Typography.Text strong size="small">
                模型列表（可选）
              </Typography.Text>
              <Typography.Text type="tertiary" size="small">
                同一套 Key/Base 下可一次添加多条模型 ID；空行会在保存时忽略。
              </Typography.Text>
              {(llmConfigForm.models || []).map((row, idx) => (
                <div
                  key={idx}
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 8,
                    paddingBottom: 10,
                    borderBottom:
                      idx < (llmConfigForm.models || []).length - 1
                        ? '1px dashed rgba(6,7,9,0.08)'
                        : undefined,
                  }}
                >
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
                    <Input
                      style={{ flex: '1 1 200px', minWidth: 160 }}
                      value={row.modelName}
                      onChange={(v) => updateLlmModelDraftRow(idx, { modelName: String(v) })}
                      placeholder="模型 ID，如 gpt-4o"
                    />
                    <Select
                      style={{ width: 108 }}
                      value={row.enabled ? '1' : '0'}
                      onChange={(v) => updateLlmModelDraftRow(idx, { enabled: v === '1' })}
                    >
                      <Select.Option value="1">启用</Select.Option>
                      <Select.Option value="0">禁用</Select.Option>
                    </Select>
                    <Button type="danger" size="small" onClick={() => removeLlmModelDraftRow(idx)}>
                      删除本行
                    </Button>
                  </div>
                  <Input
                    value={row.description}
                    onChange={(v) => updateLlmModelDraftRow(idx, { description: String(v) })}
                    placeholder="该行说明（可选）"
                  />
                </div>
              ))}
              <div>
                <Button size="small" onClick={addLlmModelDraftRow}>
                  添加一行模型
                </Button>
              </div>
            </div>
          )}
          <TextArea
            value={llmConfigForm.description}
            onChange={(v) => setLlmConfigForm({ ...llmConfigForm, description: String(v) })}
            autosize={{ minRows: 2, maxRows: 6 }}
            placeholder="整项配置描述（可选）"
          />
        </div>
      </Modal>

      <Modal
        title={entryEditing ? '编辑模型 ID' : '添加模型 ID'}
        visible={entryModalVisible}
        onCancel={() => setEntryModalVisible(false)}
        onOk={submitEntry}
        confirmLoading={entrySubmitting}
        style={{ width: 520 }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <Input
            value={entryForm.modelName}
            onChange={(v) => setEntryForm({ ...entryForm, modelName: String(v) })}
            placeholder="模型 ID，如 gpt-4o、gpt-4o-mini"
          />
          <Select
            value={entryForm.enabled ? '1' : '0'}
            onChange={(v) => setEntryForm({ ...entryForm, enabled: v === '1' })}
          >
            <Select.Option value="1">启用</Select.Option>
            <Select.Option value="0">禁用</Select.Option>
          </Select>
          <TextArea
            value={entryForm.description}
            onChange={(v) => setEntryForm({ ...entryForm, description: String(v) })}
            autosize={{ minRows: 2, maxRows: 6 }}
            placeholder="说明（可选）"
          />
        </div>
      </Modal>

      <Modal
        title={mcpEditing ? '编辑 MCP 配置' : '新增 MCP 配置'}
        visible={mcpModalVisible}
        onCancel={() => setMcpModalVisible(false)}
        onOk={submitMCP}
        confirmLoading={mcpSubmitting}
        style={{ width: 820 }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <Input
            value={mcpForm.name}
            onChange={(v) => setMcpForm({ ...mcpForm, name: String(v) })}
            placeholder="名称（唯一）"
          />
          <Input
            value={mcpForm.server}
            onChange={(v) => setMcpForm({ ...mcpForm, server: String(v) })}
            placeholder="Server 名称（与 Agent 里引用的 MCP server 名一致）"
          />
          <Select
            value={(mcpForm.transport || 'http').toLowerCase() === 'stdio' ? 'stdio' : 'http'}
            onChange={(v) =>
              setMcpForm({
                ...mcpForm,
                transport: v === 'stdio' ? 'stdio' : 'http',
              })
            }
          >
            <Select.Option value="http">HTTP / SSE（远程 URL）</Select.Option>
            <Select.Option value="stdio">stdio（本机子进程）</Select.Option>
          </Select>
          {(mcpForm.transport || 'http').toLowerCase() !== 'stdio' ? (
            <>
              <Input
                value={mcpForm.endpoint}
                onChange={(v) => setMcpForm({ ...mcpForm, endpoint: String(v) })}
                placeholder="Endpoint（例如 https://example.com/mcp）"
              />
              <TextArea
                value={headersText}
                onChange={(v) => setHeadersText(String(v))}
                autosize={{ minRows: 6, maxRows: 12 }}
                placeholder='Headers JSON，例如 {"Authorization":"Bearer xxx"}'
              />
            </>
          ) : (
            <>
              <Input
                value={mcpForm.stdio_command}
                onChange={(v) => setMcpForm({ ...mcpForm, stdio_command: String(v) })}
                placeholder="启动命令，例如 npx 或 /usr/local/bin/mcp-server"
              />
              <TextArea
                value={mcpForm.stdio_args_json}
                onChange={(v) => setMcpForm({ ...mcpForm, stdio_args_json: String(v) })}
                autosize={{ minRows: 3, maxRows: 8 }}
                placeholder='参数 JSON 数组，例如 ["-y","@modelcontextprotocol/server-filesystem","/tmp"]'
              />
              <TextArea
                value={mcpForm.stdio_env_json}
                onChange={(v) => setMcpForm({ ...mcpForm, stdio_env_json: String(v) })}
                autosize={{ minRows: 3, maxRows: 8 }}
                placeholder='环境变量 JSON 对象，例如 {} 或 {"NODE_PATH":"/app/node_modules"}'
              />
              <Typography.Text type="tertiary" size="small">
                stdio 由运行本服务的机器拉起子进程；「测试」会真实启动进程并列出工具（超时 60s）。
              </Typography.Text>
            </>
          )}
          <Select
            value={mcpForm.enabled ? '1' : '0'}
            onChange={(v) => setMcpForm({ ...mcpForm, enabled: v === '1' })}
          >
            <Select.Option value="1">启用</Select.Option>
            <Select.Option value="0">禁用</Select.Option>
          </Select>
          <TextArea
            value={mcpForm.description}
            onChange={(v) => setMcpForm({ ...mcpForm, description: String(v) })}
            autosize={{ minRows: 2, maxRows: 6 }}
            placeholder="描述（可选）"
          />
        </div>
      </Modal>
    </div>
  );
};

export default AgentSection;
