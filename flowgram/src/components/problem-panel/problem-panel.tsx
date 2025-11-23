/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { PanelFactory, usePanelManager } from '@flowgram.ai/panel-manager-plugin';
import { useClientContext } from '@flowgram.ai/free-layout-editor';
import { Space, Button, Typography, Divider, IconButton } from '@douyinfe/semi-ui';
import { IconUploadError, IconClose } from '@douyinfe/semi-icons';

import { nodeFormPanelFactory } from '../sidebar';
import { scrollToView } from '../base-node/utils';
export const PROBLEM_PANEL = 'problem-panel';
interface ProblemItem {
  nodeId: string;
  title: string;
  messages?: string[];
}
interface ProblemPanelProps {
  problems: ProblemItem[];
}

export const ProblemPanel: React.FC<ProblemPanelProps> = ({ problems }) => {
  const panelManager = usePanelManager();
  const ctx = useClientContext();

  return (
    <div
      style={{
        width: '100%',
        height: '100%',
        borderRadius: '8px',
        background: 'rgb(251, 251, 251)',
        border: '1px solid rgba(82,100,154, 0.13)',
      }}
    >
      <div style={{ display: 'flex', height: '50px', alignItems: 'center', justifyContent: 'end' }}>
        <IconButton
          type="tertiary"
          theme="borderless"
          icon={<IconClose />}
          onClick={() => panelManager.close(PROBLEM_PANEL)}
        />
      </div>
      <div style={{ padding: '8px 16px' }}>
        <Typography.Title heading={5}>未填写必填信息的节点</Typography.Title>
        <Divider margin={12} />
        <Space vertical spacing={8} style={{ width: '100%' }}>
          {problems.length === 0 ? (
            <Typography.Text>无问题</Typography.Text>
          ) : (
            problems.map((p) => (
              <div
                key={p.nodeId}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '6px 8px',
                  borderRadius: 6,
                  border: '1px solid rgba(82,100,154, 0.13)',
                  background: 'white',
                }}
              >
                <div style={{ maxWidth: '60%' }}>
                  <Typography.Text strong>{p.title || p.nodeId}</Typography.Text>
                  <div style={{ color: 'red', fontSize: 12 }}>
                    {(p.messages && p.messages.length > 0 ? p.messages : ['表单未通过校验']).join(
                      '、'
                    )}
                  </div>
                </div>
                <Space>
                  <Button
                    size="small"
                    onClick={() => {
                      const node = ctx.document.getNode(p.nodeId);
                      if (node) {
                        scrollToView(ctx, node);
                      }
                      panelManager.open(nodeFormPanelFactory.key, 'right', {
                        props: { nodeId: p.nodeId },
                      });
                    }}
                  >
                    定位并打开
                  </Button>
                </Space>
              </div>
            ))
          )}
        </Space>
      </div>
    </div>
  );
};

export const problemPanelFactory: PanelFactory<ProblemPanelProps> = {
  key: PROBLEM_PANEL,
  defaultSize: 200,
  render: (props) => <ProblemPanel {...props} />,
};

export const ProblemButton = () => {
  const panelManager = usePanelManager();

  return (
    <IconButton
      type="tertiary"
      theme="borderless"
      icon={<IconUploadError />}
      onClick={() => panelManager.open(PROBLEM_PANEL, 'bottom')}
    />
  );
};
