/**
 * Run File Viewer - 只读查看运行工作区文件内容
 */
import React, { useEffect, useState } from 'react';

import { IconFile, IconCopy, IconTerminal, IconEyeOpened } from '@douyinfe/semi-icons';
import { Button, Spin, Toast, Tooltip, Typography } from '@douyinfe/semi-ui';

import { readRunWorkspaceFile } from '../../services/api-playground';

interface RunFileViewerProps {
  runId: string;
  filePath: string | null;
  fileName: string | null;
  onOpenTerminal?: (cwd: string) => void;
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
  if (ext === 'html' || ext === 'htm') return '#52c41a';
  return '#595959';
}

export const RunFileViewer: React.FC<RunFileViewerProps> = ({ runId, filePath, fileName, onOpenTerminal }) => {
  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isPreview, setIsPreview] = useState(false);

  useEffect(() => {
    if (!filePath) {
      setContent('');
      setError(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    readRunWorkspaceFile(runId, filePath)
      .then((res) => {
        if (!cancelled) setContent(res.content);
      })
      .catch((err) => {
        if (!cancelled) setError(String((err as Error)?.message ?? err));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [runId, filePath]);

  const ext = filePath ? fileExt(filePath) : '';
  // 获取文件所在目录
  const fileDir = filePath ? filePath.split('/').slice(0, -1).join('/') : null;

  if (!filePath || !fileName) {
    return (
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
        <Typography.Text type="quaternary">从左侧选择文件查看内容</Typography.Text>
      </div>
    );
  }

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, background: '#fff', borderRadius: 8, border: '1px solid rgba(28,31,35,0.08)', overflow: 'hidden' }}>
      {/* 顶部工具栏 */}
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
        <IconFile size="small" style={{ color: extColor(ext), flexShrink: 0 }} />
        <Typography.Text style={{ flex: 1, fontSize: 13, fontFamily: 'monospace' }} ellipsis={{ showTooltip: true }}>
          {filePath}
        </Typography.Text>
        <Button
          size="small"
          type="tertiary"
          icon={<IconCopy />}
          onClick={() => {
            void navigator.clipboard.writeText(content).then(
              () => Toast.success({ content: '已复制文件内容', duration: 2 }),
              () => Toast.warning({ content: '复制失败', duration: 2 })
            );
          }}
          disabled={loading || !!error}
        >
          复制
        </Button>
        {(ext === 'html' || ext === 'htm') && (
          <Tooltip content={isPreview ? '切换到源码' : '预览 HTML'}>
            <Button
              size="small"
              type="tertiary"
              icon={<IconEyeOpened />}
              onClick={() => setIsPreview(!isPreview)}
              disabled={loading || !!error}
            >
              {isPreview ? '源码' : '预览'}
            </Button>
          </Tooltip>
        )}
        {onOpenTerminal && fileDir != null && (
          <Tooltip content="在文件目录打开终端">
            <Button
              size="small"
              type="tertiary"
              icon={<IconTerminal />}
              onClick={() => onOpenTerminal(fileDir || '/')}
            />
          </Tooltip>
        )}
      </div>

      {/* 内容区 */}
      <div style={{ flex: 1, minHeight: 0, position: 'relative' }}>
        {loading && (
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
        {error ? (
          <div style={{ padding: '16px 20px' }}>
            <Typography.Text type="danger" size="small">
              {error}
            </Typography.Text>
          </div>
        ) : isPreview && (ext === 'html' || ext === 'htm') ? (
          <iframe
            srcDoc={content}
            style={{
              width: '100%',
              height: '100%',
              border: 'none',
              background: '#fff',
            }}
            title="HTML Preview"
            sandbox="allow-scripts allow-same-origin"
          />
        ) : (
          <pre
            style={{
              margin: 0,
              padding: '12px 16px',
              height: '100%',
              overflow: 'auto',
              fontFamily: '"JetBrains Mono", "Fira Code", "Cascadia Code", monospace',
              fontSize: 13,
              lineHeight: 1.6,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              background: 'transparent',
              color: '#1c1f23',
              tabSize: 2,
            }}
          >
            {content}
          </pre>
        )}
      </div>
    </div>
  );
};
