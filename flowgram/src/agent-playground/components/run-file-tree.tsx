/**
 * Run File Tree - 浏览运行工作区生成的文件
 */
import React, { useCallback, useEffect, useState } from 'react';

import { Button, Spin, Tag, Tooltip, Typography } from '@douyinfe/semi-ui';
import {
  IconChevronDown,
  IconChevronRight,
  IconFile,
  IconFolder,
  IconFolderOpen,
  IconRefresh,
  IconTerminal,
} from '@douyinfe/semi-icons';

import { listRunWorkspaceFiles, WorkspaceFileItem } from '../../services/api-playground';

interface RunFileTreeProps {
  runId: string;
  onFileSelect: (file: { path: string; name: string }) => void;
  onOpenTerminal: (cwd: string) => void;
}

function fileExt(path: string): string {
  const i = path.lastIndexOf('.');
  return i >= 0 ? path.slice(i + 1).toLowerCase() : '';
}

function extColor(ext: string): string {
  if (ext === 'md') return '#1677ff';
  if (ext === 'json') return '#d46b08';
  if (ext === 'yaml' || ext === 'yml') return '#389e0d';
  if (ext === 'txt') return '#722ed1';
  if (ext === 'go' || ext === 'py' || ext === 'js' || ext === 'ts') return '#eb2f96';
  if (ext === 'sh' || ext === 'bash') return '#13c2c2';
  return '#595959';
}

export const RunFileTree: React.FC<RunFileTreeProps> = ({
  runId,
  onFileSelect,
  onOpenTerminal,
}) => {
  const [rootItems, setRootItems] = useState<WorkspaceFileItem[]>([]);
  const [dirContents, setDirContents] = useState<Map<string, WorkspaceFileItem[]>>(new Map());
  const [expandedDirs, setExpandedDirs] = useState<Set<string>>(new Set());
  const [selectedFilePath, setSelectedFilePath] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadingDirs, setLoadingDirs] = useState<Set<string>>(new Set());

  const fetchRoot = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listRunWorkspaceFiles(runId);
      setRootItems(res.items || []);
    } catch {
      setRootItems([]);
    } finally {
      setLoading(false);
    }
  }, [runId]);

  useEffect(() => {
    fetchRoot();
  }, [fetchRoot]);

  const fetchDir = useCallback(
    async (dirPath: string) => {
      if (dirContents.has(dirPath)) return;
      setLoadingDirs((prev) => new Set(prev).add(dirPath));
      try {
        const res = await listRunWorkspaceFiles(runId, dirPath);
        setDirContents((prev) => new Map(prev).set(dirPath, res.items || []));
      } catch {
        setDirContents((prev) => new Map(prev).set(dirPath, []));
      } finally {
        setLoadingDirs((prev) => {
          const next = new Set(prev);
          next.delete(dirPath);
          return next;
        });
      }
    },
    [runId, dirContents]
  );

  const toggleDir = useCallback(
    (dirPath: string) => {
      setExpandedDirs((prev) => {
        const next = new Set(prev);
        if (next.has(dirPath)) {
          next.delete(dirPath);
        } else {
          next.add(dirPath);
          void fetchDir(dirPath);
        }
        return next;
      });
    },
    [fetchDir]
  );

  const handleFileClick = useCallback(
    (filePath: string, fileName: string) => {
      setSelectedFilePath(filePath);
      onFileSelect({ path: filePath, name: fileName });
    },
    [onFileSelect]
  );

  const refreshDir = useCallback(
    async (dirPath: string | null) => {
      if (dirPath === null) {
        setRootItems([]);
        setDirContents(new Map());
        setExpandedDirs(new Set());
        setSelectedFilePath(null);
        await fetchRoot();
      } else {
        try {
          const res = await listRunWorkspaceFiles(runId, dirPath);
          setDirContents((prev) => new Map(prev).set(dirPath, res.items || []));
        } catch {
          /* ignore */
        }
      }
    },
    [runId, fetchRoot]
  );

  const renderTree = (
    items: WorkspaceFileItem[],
    depth: number,
    pathPrefix: string
  ): React.ReactNode =>
    items.map((item) => {
      if (item.type === 'file') {
        const filePath = pathPrefix ? `${pathPrefix}/${item.name}` : item.name;
        const isSelected = selectedFilePath === filePath;
        const ext = fileExt(item.name);
        return (
          <div
            key={filePath}
            onClick={() => handleFileClick(filePath, item.name)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              padding: `3px 8px 3px ${10 + depth * 14}px`,
              cursor: 'pointer',
              borderRadius: 4,
              margin: '1px 4px',
              background: isSelected ? 'rgba(22,119,255,0.10)' : 'transparent',
              borderLeft: isSelected ? '2px solid #1677ff' : '2px solid transparent',
            }}
            onMouseEnter={(e) => {
              if (!isSelected)
                (e.currentTarget as HTMLDivElement).style.background = 'rgba(28,31,35,0.04)';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLDivElement).style.background = isSelected
                ? 'rgba(22,119,255,0.10)'
                : 'transparent';
            }}
          >
            <IconFile size="small" style={{ color: extColor(ext), flexShrink: 0 }} />
            <Typography.Text
              ellipsis={{ showTooltip: true }}
              style={{ flex: 1, fontSize: 12, color: isSelected ? '#1677ff' : undefined }}
            >
              {item.name}
            </Typography.Text>
            <Tag size="small" style={{ flexShrink: 0, fontSize: 10, padding: '0 4px' }}>
              {ext}
            </Tag>
          </div>
        );
      }
      // dir node
      const dirPath = pathPrefix ? `${pathPrefix}/${item.name}` : item.name;
      const isExpanded = expandedDirs.has(dirPath);
      const isLoading = loadingDirs.has(dirPath);
      const children = dirContents.get(dirPath) || [];
      return (
        <div key={dirPath}>
          <div
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
              (e.currentTarget as HTMLDivElement).style.background = 'rgba(28,31,35,0.04)';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLDivElement).style.background = 'transparent';
            }}
          >
            <span
              onClick={() => toggleDir(dirPath)}
              style={{
                display: 'flex',
                alignItems: 'center',
                color: '#9ca3af',
                flexShrink: 0,
                fontSize: 12,
              }}
            >
              {isLoading ? (
                <Spin size="small" />
              ) : isExpanded ? (
                <IconChevronDown size="small" />
              ) : (
                <IconChevronRight size="small" />
              )}
            </span>
            <span
              onClick={() => toggleDir(dirPath)}
              style={{ display: 'flex', alignItems: 'center', gap: 4, flex: 1, minWidth: 0 }}
            >
              {isExpanded ? (
                <IconFolderOpen size="small" style={{ color: '#faad14', flexShrink: 0 }} />
              ) : (
                <IconFolder size="small" style={{ color: '#faad14', flexShrink: 0 }} />
              )}
              <Typography.Text style={{ fontSize: 12, color: '#374151' }}>
                {item.name}
              </Typography.Text>
            </span>
            <Tooltip content="在此目录打开终端">
              <Button
                size="small"
                theme="borderless"
                icon={<IconTerminal size="small" />}
                style={{ flexShrink: 0, padding: '0 4px' }}
                onClick={(e) => {
                  e.stopPropagation();
                  onOpenTerminal(dirPath);
                }}
              />
            </Tooltip>
          </div>
          {isExpanded && renderTree(children, depth + 1, dirPath)}
        </div>
      );
    });

  return (
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
      {/* 顶部操作栏 */}
      <div
        style={{
          padding: '8px 10px',
          borderBottom: '1px solid rgba(28,31,35,0.06)',
          display: 'flex',
          gap: 6,
          alignItems: 'center',
        }}
      >
        <Typography.Text strong style={{ flex: 1, fontSize: 13 }}>
          工作区文件
        </Typography.Text>
        <Tooltip content="刷新">
          <Button
            size="small"
            theme="borderless"
            icon={<IconRefresh />}
            loading={loading}
            onClick={() => void refreshDir(null)}
          />
        </Tooltip>
      </div>

      {/* 文件列表 */}
      <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', padding: '4px 0' }}>
        {loading && rootItems.length === 0 ? (
          <div style={{ padding: '16px 12px', textAlign: 'center' }}>
            <Spin spinning />
          </div>
        ) : rootItems.length === 0 ? (
          <div style={{ padding: '16px 12px', textAlign: 'center' }}>
            <Typography.Text type="tertiary" size="small">
              工作区暂无文件
            </Typography.Text>
          </div>
        ) : (
          renderTree(rootItems, 0, '')
        )}
      </div>
    </div>
  );
};
