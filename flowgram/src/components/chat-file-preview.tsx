import React, { useEffect, useMemo, useState, useRef } from 'react';

import { Button, Spin, Toast, Tooltip, Typography } from '@douyinfe/semi-ui';
import { IconClose, IconDownload, IconFile, IconEyeOpened } from '@douyinfe/semi-icons';

import { buildChatFileUrl } from '../services/api-chat';

interface ChatFilePreviewProps {
  filePath: string;
  onClose: () => void;
}

type FileCategory = 'image' | 'html' | 'text' | 'pdf' | 'binary';

const IMAGE_EXTS = ['svg', 'png', 'jpg', 'jpeg', 'gif', 'bmp', 'webp', 'ico', 'avif'];
const TEXT_EXTS = [
  'txt',
  'md',
  'json',
  'yaml',
  'yml',
  'csv',
  'ts',
  'tsx',
  'js',
  'jsx',
  'go',
  'py',
  'rs',
  'java',
  'kt',
  'sql',
  'xml',
  'css',
  'less',
  'scss',
  'sh',
  'bash',
  'env',
  'proto',
  'toml',
  'ini',
  'conf',
  'log',
  'dart',
  'rb',
  'php',
  'c',
  'cpp',
  'h',
  'hpp',
];

function classifyFile(filePath: string): FileCategory {
  const i = filePath.lastIndexOf('.');
  if (i < 0) return 'text';
  const ext = filePath.slice(i + 1).toLowerCase();
  if (IMAGE_EXTS.includes(ext)) return 'image';
  if (['html', 'htm'].includes(ext)) return 'html';
  if (ext === 'pdf') return 'pdf';
  if (TEXT_EXTS.includes(ext)) return 'text';
  return 'binary';
}

function extColor(ext: string): string {
  if (ext === 'md') return '#1677ff';
  if (ext === 'json') return '#d46b08';
  if (ext === 'yaml' || ext === 'yml') return '#389e0d';
  if (ext === 'go' || ext === 'py' || ext === 'js' || ext === 'ts') return '#eb2f96';
  if (ext === 'sh' || ext === 'bash') return '#13c2c2';
  if (ext === 'html' || ext === 'htm') return '#52c41a';
  if (IMAGE_EXTS.includes(ext)) return '#722ed1';
  return '#595959';
}

function getAuthToken(): string {
  if (typeof window === 'undefined') return '';
  return window.localStorage.getItem('AUTH_TOKEN') || window.localStorage.getItem('token') || '';
}

async function fetchFileBlob(filePath: string): Promise<Blob> {
  const url = buildChatFileUrl(filePath);
  const headers: Record<string, string> = {};
  const token = getAuthToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const resp = await fetch(url, { headers });
  if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${await resp.text()}`);
  return resp.blob();
}

async function fetchFileText(filePath: string): Promise<string> {
  const url = buildChatFileUrl(filePath);
  const headers: Record<string, string> = { Accept: 'text/plain' };
  const token = getAuthToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const resp = await fetch(url, { headers });
  if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${await resp.text()}`);
  return resp.text();
}

export const ChatFilePreview: React.FC<ChatFilePreviewProps> = ({ filePath, onClose }) => {
  const category = useMemo(() => classifyFile(filePath), [filePath]);
  const fileName = useMemo(() => filePath.split('/').pop() || 'file', [filePath]);
  const ext = useMemo(() => {
    const i = filePath.lastIndexOf('.');
    return i >= 0 ? filePath.slice(i + 1).toLowerCase() : '';
  }, [filePath]);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [textContent, setTextContent] = useState<string | null>(null);
  const [objectUrl, setObjectUrl] = useState<string | null>(null);
  const [isPreview, setIsPreview] = useState(true);
  const prevObjectUrlRef = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    setTextContent(null);

    // Revoke previous object URL
    if (prevObjectUrlRef.current) {
      URL.revokeObjectURL(prevObjectUrlRef.current);
      prevObjectUrlRef.current = null;
    }
    setObjectUrl(null);

    const loadFile = async () => {
      try {
        if (category === 'image' || category === 'pdf') {
          const blob = await fetchFileBlob(filePath);
          if (cancelled) return;
          const url = URL.createObjectURL(blob);
          prevObjectUrlRef.current = url;
          setObjectUrl(url);
        } else if (category === 'html') {
          const text = await fetchFileText(filePath);
          if (cancelled) return;
          setTextContent(text);
        } else if (category === 'text') {
          const text = await fetchFileText(filePath);
          if (cancelled) return;
          setTextContent(text);
        } else {
          // binary: try text first, fall back to blob
          try {
            const text = await fetchFileText(filePath);
            if (cancelled) return;
            setTextContent(text);
          } catch {
            const blob = await fetchFileBlob(filePath);
            if (cancelled) return;
            const url = URL.createObjectURL(blob);
            prevObjectUrlRef.current = url;
            setObjectUrl(url);
          }
        }
      } catch (err) {
        if (!cancelled) setError(String((err as Error)?.message ?? err));
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    loadFile();
    return () => {
      cancelled = true;
    };
  }, [filePath, category]);

  // Cleanup on unmount
  useEffect(
    () => () => {
      if (prevObjectUrlRef.current) {
        URL.revokeObjectURL(prevObjectUrlRef.current);
      }
    },
    []
  );

  const handleDownload = () => {
    const url = buildChatFileUrl(filePath, true);
    const headers: Record<string, string> = {};
    const token = getAuthToken();
    if (token) headers['Authorization'] = `Bearer ${token}`;
    fetch(url, { headers })
      .then((r) => r.blob())
      .then((blob) => {
        const a = document.createElement('a');
        a.href = URL.createObjectURL(blob);
        a.download = fileName;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(a.href);
      })
      .catch((err) => Toast.error({ content: `下载失败: ${err}` }));
  };

  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
        background: '#fff',
      }}
    >
      {/* Toolbar */}
      <div
        style={{
          padding: '8px 12px',
          borderBottom: '1px solid rgba(28,31,35,0.06)',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          background: 'rgba(28,31,35,0.02)',
          flexShrink: 0,
        }}
      >
        <IconFile size="small" style={{ color: extColor(ext), flexShrink: 0 }} />
        <Typography.Text
          ellipsis={{ showTooltip: true }}
          style={{ flex: 1, fontSize: 13, fontFamily: 'monospace' }}
        >
          {fileName}
        </Typography.Text>
        {category === 'html' && (
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
        <Button size="small" type="tertiary" icon={<IconDownload />} onClick={handleDownload}>
          下载
        </Button>
        <Button size="small" type="tertiary" icon={<IconClose />} onClick={onClose} />
      </div>

      {/* Content */}
      <div
        style={{
          flex: 1,
          minHeight: 0,
          display: 'flex',
          flexDirection: 'column',
          position: 'relative',
        }}
      >
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
        ) : category === 'image' && objectUrl ? (
          <div
            style={{
              flex: 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: 12,
              overflow: 'auto',
            }}
          >
            <img
              src={objectUrl}
              alt={fileName}
              style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain' }}
            />
          </div>
        ) : category === 'html' && textContent !== null ? (
          isPreview ? (
            <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
              <iframe
                srcDoc={textContent}
                style={{ width: '100%', height: '100%', border: 'none', background: '#fff' }}
                title="HTML Preview"
                sandbox="allow-scripts allow-same-origin"
              />
            </div>
          ) : (
            <pre
              style={{
                margin: 0,
                padding: '12px 16px',
                height: '100%',
                overflow: 'auto',
                fontFamily: '"JetBrains Mono","Fira Code",monospace',
                fontSize: 13,
                lineHeight: 1.6,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                background: 'transparent',
                color: '#1c1f23',
                tabSize: 2,
              }}
            >
              {textContent}
            </pre>
          )
        ) : category === 'pdf' && objectUrl ? (
          <iframe
            src={objectUrl}
            style={{ width: '100%', height: '100%', border: 'none' }}
            title="PDF Preview"
          />
        ) : textContent !== null ? (
          <pre
            style={{
              margin: 0,
              padding: '12px 16px',
              height: '100%',
              overflow: 'auto',
              fontFamily: '"JetBrains Mono","Fira Code",monospace',
              fontSize: 13,
              lineHeight: 1.6,
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              background: 'transparent',
              color: '#1c1f23',
              tabSize: 2,
            }}
          >
            {textContent}
          </pre>
        ) : objectUrl ? (
          <div
            style={{
              flex: 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: 16,
            }}
          >
            <div style={{ textAlign: 'center' }}>
              <Typography.Text type="tertiary">此文件类型不支持预览</Typography.Text>
              <br />
              <Button style={{ marginTop: 8 }} onClick={handleDownload}>
                下载文件
              </Button>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
};
