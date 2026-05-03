/**
 * Run Workspace Panel - 运行完成后的工作区浏览面板（文件树 + 文件查看 + 终端）
 */
import React, { useState, useCallback } from 'react';

import { Card, Tabs, Typography } from '@douyinfe/semi-ui';
import { IconFile, IconTerminal } from '@douyinfe/semi-icons';

import { RunTerminalPanel } from './run-terminal-panel';
import { RunFileViewer } from './run-file-viewer';
import { RunFileTree } from './run-file-tree';

interface RunWorkspacePanelProps {
  runId: string;
  workspacePath: string;
}

export const RunWorkspacePanel: React.FC<RunWorkspacePanelProps> = ({ runId, workspacePath }) => {
  const [activeSubTab, setActiveSubTab] = useState<'files' | 'terminal'>('files');
  const [selectedFile, setSelectedFile] = useState<{ path: string; name: string } | null>(null);
  const [terminalCwd, setTerminalCwd] = useState<string>(workspacePath);

  const handleFileSelect = useCallback((file: { path: string; name: string }) => {
    setSelectedFile(file);
  }, []);

  const handleOpenTerminal = useCallback(
    (dirPath: string) => {
      setTerminalCwd(`${workspacePath}/${dirPath}`);
      setActiveSubTab('terminal');
    },
    [workspacePath]
  );

  const handleOpenTerminalFromViewer = useCallback(
    (dirPath: string) => {
      // dirPath 来自 viewer 时是相对路径
      setTerminalCwd(`${workspacePath}/${dirPath}`);
      setActiveSubTab('terminal');
    },
    [workspacePath]
  );

  const handleCloseTerminal = useCallback(() => {
    setActiveSubTab('files');
  }, []);

  return (
    <Card
      style={{
        marginTop: 16,
        borderRadius: 14,
        boxShadow: '0 1px 12px rgba(28, 31, 35, 0.06)',
        minHeight: 320,
      }}
      bodyStyle={{ padding: 0, display: 'flex', flexDirection: 'column', minHeight: 280 }}
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <div
            style={{
              width: 32,
              height: 32,
              borderRadius: 8,
              background:
                'linear-gradient(135deg, rgba(22, 100, 255, 0.10), rgba(22, 100, 255, 0.04))',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 16,
            }}
          >
            📂
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Typography.Text strong style={{ fontSize: 14 }}>
              工作区文件
            </Typography.Text>
            <Typography.Text
              type="tertiary"
              size="small"
              ellipsis={{ showTooltip: true }}
              style={{ maxWidth: 400 }}
            >
              {workspacePath}
            </Typography.Text>
          </div>
        </div>
      }
      headerExtraContent={
        <Tabs
          type="button"
          size="small"
          activeKey={activeSubTab}
          onChange={(k) => setActiveSubTab(k as 'files' | 'terminal')}
        >
          <Tabs.TabPane
            itemKey="files"
            tab={
              <span>
                <IconFile style={{ marginRight: 4, verticalAlign: '-2px' }} />
                文件
              </span>
            }
          />
          <Tabs.TabPane
            itemKey="terminal"
            tab={
              <span>
                <IconTerminal style={{ marginRight: 4, verticalAlign: '-2px' }} />
                终端
              </span>
            }
          />
        </Tabs>
      }
    >
      {activeSubTab === 'files' ? (
        <div style={{ display: 'flex', flex: 1, minHeight: 0, gap: 12, padding: 12 }}>
          <RunFileTree
            runId={runId}
            onFileSelect={handleFileSelect}
            onOpenTerminal={handleOpenTerminal}
          />
          <RunFileViewer
            runId={runId}
            filePath={selectedFile?.path ?? null}
            fileName={selectedFile?.name ?? null}
            onOpenTerminal={handleOpenTerminalFromViewer}
          />
        </div>
      ) : (
        <div style={{ display: 'flex', flex: 1, minHeight: 0, padding: 12 }}>
          <RunTerminalPanel cwd={terminalCwd} onClose={handleCloseTerminal} />
        </div>
      )}
    </Card>
  );
};
