/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useCallback, useState } from 'react';

import { usePanelManager } from '@flowgram.ai/panel-manager-plugin';
import {
  useClientContext,
  useService,
  GlobalScope,
  ASTFactory,
} from '@flowgram.ai/free-layout-editor';
import { JsonSchemaUtils } from '@flowgram.ai/form-materials';
import { Button, Modal, TextArea, Toast, Space, Dropdown } from '@douyinfe/semi-ui';
import { IconDownload, IconUpload, IconCopy, IconChevronDown } from '@douyinfe/semi-icons';

import { problemPanelFactory } from '../problem-panel';
import { collectWorkflowProblems } from '../../utils/workflow-validation';
import {
  buildRuleChainJSONFromDocument,
  buildDocumentFromRuleChainJSON,
} from '../../utils/rulechain-builder';
import { FlowDocumentJSON, FlowNodeJSON } from '../../typings';
import { getRuleBaseInfo } from '../../services/rule-base-info';
import { GetGlobalVariableSchema } from '../../plugins/variable-panel-plugin';
import iconVariable from '../../assets/icon-variable.png';

export function ExportImport(props: { disabled?: boolean }) {
  const { document: workflowDocument, get } = useClientContext();
  const globalScope = useService(GlobalScope);
  const panelManager = usePanelManager();

  const [exportVisible, setExportVisible] = useState(false);
  const [importVisible, setImportVisible] = useState(false);
  const [exportText, setExportText] = useState('');
  const [importText, setImportText] = useState('');
  const [ruleChainVisible, setRuleChainVisible] = useState(false);
  const [ruleChainText, setRuleChainText] = useState('');

  const buildExportJSON = useCallback(() => {
    try {
      const getter = get<GetGlobalVariableSchema>(GetGlobalVariableSchema);
      const raw = workflowDocument.toJSON();
      // 标准化节点数据以满足 FlowNodeJSON 的必需字段约束
      const normalizedNodes: FlowNodeJSON[] = raw.nodes.map((n: any) => ({
        ...n,
        data: n.data ?? {},
      }));
      const full: FlowDocumentJSON = {
        nodes: normalizedNodes,
        edges: raw.edges,
        globalVariable: getter ? getter() : undefined,
      };
      return JSON.stringify(full, null, 2);
    } catch (e) {
      console.error(e);
      Toast.error('导出失败：序列化异常');
      return '';
    }
  }, [workflowDocument, get]);

  const openExport = useCallback(async () => {
    const problems = await collectWorkflowProblems(workflowDocument);
    if (problems.length > 0) {
      panelManager.open(problemPanelFactory.key, 'bottom', { props: { problems } });
      Toast.error('存在未填写的必填项');
      return;
    }
    const text = buildExportJSON();
    setExportText(text);
    setExportVisible(true);
  }, [buildExportJSON, workflowDocument]);

  const openRuleChainExport = useCallback(async () => {
    const problems = await collectWorkflowProblems(workflowDocument);
    if (problems.length > 0) {
      panelManager.open(problemPanelFactory.key, 'bottom', { props: { problems } });
      Toast.error('存在未填写的必填项');
      return;
    }
    const baseInfo = getRuleBaseInfo();
    const text = buildRuleChainJSONFromDocument(workflowDocument, baseInfo);
    setRuleChainText(text);
    setRuleChainVisible(true);
  }, [workflowDocument]);

  const copyExport = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(exportText);
      Toast.success('JSON 已复制到剪贴板');
    } catch {
      Toast.error('复制失败');
    }
  }, [exportText]);

  const downloadExport = useCallback(() => {
    try {
      const blob = new Blob([exportText], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = window.document.createElement('a');
      a.href = url;
      a.download = 'workflow.json';
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      Toast.error('下载失败');
    }
  }, [exportText]);

  const onImportConfirm = useCallback(() => {
    try {
      const raw = JSON.parse(importText);

      const isFlowDoc = (o: any): o is FlowDocumentJSON =>
        !!o && Array.isArray(o.nodes) && Array.isArray(o.edges);

      const isRuleChain = (o: any): o is { ruleChain: any; metadata: any } =>
        !!o &&
        !!o.ruleChain &&
        !!o.metadata &&
        Array.isArray(o.metadata.nodes) &&
        Array.isArray(o.metadata.connections);

      let doc: FlowDocumentJSON | null = null;
      if (isFlowDoc(raw)) {
        doc = raw;
      } else if (isRuleChain(raw)) {
        const rc = raw as any;
        doc = buildDocumentFromRuleChainJSON(rc);
      }

      if (!doc) {
        Toast.error('导入失败：无法识别的 JSON 格式');
        return;
      }

      try {
        const existingNodes = workflowDocument.getAllNodes();
        existingNodes.forEach((n) => {
          if (workflowDocument.canRemove(n)) {
            n.dispose();
          }
        });
      } catch {}

      workflowDocument.fromJSON({ nodes: doc.nodes, edges: doc.edges });

      if ((doc as any).globalVariable) {
        globalScope.setVar(
          ASTFactory.createVariableDeclaration({
            key: 'global',
            meta: { title: 'Global', icon: iconVariable },
            type: JsonSchemaUtils.schemaToAST((doc as any).globalVariable),
          })
        );
      }

      Toast.success('导入成功');
      setImportVisible(false);
    } catch (e) {
      console.error(e);
      Toast.error('导入失败：JSON 解析异常');
    }
  }, [workflowDocument, globalScope, importText]);

  const disabled = props.disabled;

  const menuItems = [
    {
      node: 'item',
      name: '导出 JSON',
      icon: <IconDownload size="large" />,
      onClick: openExport,
    },
    {
      node: 'item',
      name: '导出 RuleChain',
      icon: <IconDownload size="large" />,
      onClick: openRuleChainExport,
    },
    {
      node: 'divider',
    },
    {
      node: 'item',
      name: '导入 JSON',
      icon: <IconUpload size="large" />,
      onClick: () => setImportVisible(true),
    },
  ];

  return (
    <>
      <Dropdown
        trigger="click"
        position="bottomRight"
        menu={menuItems.map((item) => ({ ...item, node: 'item' as const }))}
        disabled={disabled}
      >
        <Button
          theme="light"
          disabled={disabled}
          icon={<IconDownload size="default" />}
          iconPosition="left"
        >
          导出/导入
          <IconChevronDown size="small" style={{ marginLeft: 4 }} />
        </Button>
      </Dropdown>

      <Modal
        title="导出工作流 JSON"
        visible={exportVisible}
        onCancel={() => setExportVisible(false)}
        footer={
          <Space>
            <Button icon={<IconCopy />} onClick={copyExport}>
              复制
            </Button>
            <Button icon={<IconDownload />} onClick={downloadExport}>
              下载
            </Button>
            <Button type="primary" onClick={() => setExportVisible(false)}>
              关闭
            </Button>
          </Space>
        }
        width={720}
      >
        <TextArea
          value={exportText}
          onChange={(v) => setExportText(String(v))}
          autosize={{ minRows: 18, maxRows: 32 }}
        />
      </Modal>

      <Modal
        title="导出 RuleChain JSON"
        visible={ruleChainVisible}
        onCancel={() => setRuleChainVisible(false)}
        footer={
          <Space>
            <Button
              icon={<IconCopy />}
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(ruleChainText);
                  Toast.success('RuleChain JSON 已复制到剪贴板');
                } catch {
                  Toast.error('复制失败');
                }
              }}
            >
              复制
            </Button>
            <Button
              icon={<IconDownload />}
              onClick={() => {
                try {
                  const blob = new Blob([ruleChainText], { type: 'application/json' });
                  const url = URL.createObjectURL(blob);
                  const a = window.document.createElement('a');
                  a.href = url;
                  a.download = 'rulechain.json';
                  a.click();
                  URL.revokeObjectURL(url);
                } catch {
                  Toast.error('下载失败');
                }
              }}
            >
              下载
            </Button>
            <Button type="primary" onClick={() => setRuleChainVisible(false)}>
              关闭
            </Button>
          </Space>
        }
        width={720}
      >
        <TextArea
          value={ruleChainText}
          onChange={(v) => setRuleChainText(String(v))}
          autosize={{ minRows: 18, maxRows: 32 }}
        />
      </Modal>

      <Modal
        title="导入工作流 JSON"
        visible={importVisible}
        onCancel={() => setImportVisible(false)}
        onOk={onImportConfirm}
        okText="导入"
        width={720}
      >
        <TextArea
          placeholder="在此粘贴工作流 JSON"
          value={importText}
          onChange={(v) => setImportText(String(v))}
          autosize={{ minRows: 18, maxRows: 32 }}
        />
      </Modal>
    </>
  );
}
