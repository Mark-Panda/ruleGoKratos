/**
 * 规则链入参/出参树形编辑（YApi 风格参数表），Semi UI。
 */

import React, { useEffect, useRef, useState } from 'react';

import { Button, Input, Modal, Select, TextArea, Toast, Typography } from '@douyinfe/semi-ui';

import {
  buildRuleChainParamsCommentedPreview,
  buildRuleChainParamsPreviewValue,
  emptyRuleChainParamsJson,
  importRuleChainParamsFromObjectJson,
  newParamNodeId,
  parseRuleChainParamsJson,
  serializeRuleChainParamsNodes,
  type RuleChainParamNode,
  type RuleChainParamType,
} from '../utils/rule-chain-request-params';
import { tryFormatJsonPretty } from '../utils/format-json-pretty';

export type RuleChainRequestParamsEditorProps = {
  title: string;
  value: string;
  onChange: (json: string) => void;
};

type Path = number[];

function updateAtPath(
  nodes: RuleChainParamNode[],
  path: Path,
  updater: (node: RuleChainParamNode) => RuleChainParamNode
): RuleChainParamNode[] {
  if (path.length === 0) return nodes;
  const [head, ...rest] = path;
  return nodes.map((node, idx) => {
    if (idx !== head) return node;
    if (rest.length === 0) return updater(node);
    return { ...node, children: updateAtPath(node.children, rest, updater) };
  });
}

function removeAtPath(nodes: RuleChainParamNode[], path: Path): RuleChainParamNode[] {
  if (path.length === 0) return nodes;
  const [head, ...rest] = path;
  if (rest.length === 0) return nodes.filter((_, i) => i !== head);
  return nodes.map((n, i) => (i === head ? { ...n, children: removeAtPath(n.children, rest) } : n));
}

function insertSiblingAfter(
  nodes: RuleChainParamNode[],
  path: Path,
  factory: () => RuleChainParamNode
): RuleChainParamNode[] {
  const [head, ...rest] = path;
  if (rest.length === 0) {
    const next = [...nodes];
    next.splice(head + 1, 0, factory());
    return next;
  }
  return nodes.map((n, i) =>
    i === head ? { ...n, children: insertSiblingAfter(n.children, rest, factory) } : n
  );
}

function addChildAtPath(
  nodes: RuleChainParamNode[],
  path: Path,
  factory: () => RuleChainParamNode
): RuleChainParamNode[] {
  return updateAtPath(nodes, path, (node) => ({ ...node, children: [...node.children, factory()] }));
}

function defaultNode(type: RuleChainParamType = 'string'): RuleChainParamNode {
  return {
    id: newParamNodeId(),
    key: '',
    type,
    required: false,
    description: '',
    children: [],
  };
}

const PARAM_TYPES: RuleChainParamType[] = ['string', 'number', 'boolean', 'array', 'object'];

export const RuleChainRequestParamsEditor: React.FC<RuleChainRequestParamsEditorProps> = ({
  title,
  value,
  onChange,
}) => {
  const [nodes, setNodes] = useState<RuleChainParamNode[]>(() => parseRuleChainParamsJson(value));
  const [viewMode, setViewMode] = useState<'table' | 'jsonOut'>('table');
  const [importOpen, setImportOpen] = useState(false);
  const [importText, setImportText] = useState('');
  const [importError, setImportError] = useState<string | null>(null);
  const lastEmittedJsonRef = useRef<string>(value);

  useEffect(() => {
    if (value === lastEmittedJsonRef.current) return;
    lastEmittedJsonRef.current = value;
    setNodes(parseRuleChainParamsJson(value));
  }, [value]);

  const pushChange = (next: RuleChainParamNode[]) => {
    setNodes(next);
    const json = serializeRuleChainParamsNodes(next);
    lastEmittedJsonRef.current = json;
    onChange(json);
  };

  const updateNode = (path: Path, patch: Partial<RuleChainParamNode>) => {
    const next = updateAtPath(nodes, path, (node) => {
      const merged = { ...node, ...patch };
      if (patch.type && patch.type !== 'object' && patch.type !== 'array') {
        merged.children = [];
      }
      return merged;
    });
    pushChange(next);
  };

  const addRoot = () => pushChange([...nodes, defaultNode()]);

  const applyImport = () => {
    setImportError(null);
    try {
      const imported = importRuleChainParamsFromObjectJson(importText);
      pushChange(imported);
      setImportOpen(false);
      setImportText('');
    } catch (e) {
      setImportError(e instanceof Error ? e.message : String(e));
    }
  };

  const previewObj = buildRuleChainParamsPreviewValue(nodes);
  const previewObjText = JSON.stringify(previewObj, null, 2);
  const previewWithComments = buildRuleChainParamsCommentedPreview(nodes);

  const renderTreeRows = (list: RuleChainParamNode[], level: number, prefix: Path = []) =>
    list.map((node, idx) => {
      const path = [...prefix, idx];
      const canNest = node.type === 'object' || node.type === 'array';
      return (
        <div key={node.id}>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: '14px 1fr 100px 36px 1fr auto',
              alignItems: 'center',
              gap: 6,
              padding: '4px 0',
              borderBottom: '1px solid rgba(6,7,9,0.06)',
            }}
          >
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                background: node.key.trim() ? 'var(--semi-color-success)' : 'var(--semi-color-tertiary)',
              }}
            />
            <Input
              value={node.key}
              onChange={(v) => updateNode(path, { key: String(v) })}
              placeholder={level === 0 ? 'key' : 'child_key'}
              style={{ paddingLeft: 8 + level * 14 }}
            />
            <Select
              value={node.type}
              style={{ width: 100 }}
              onChange={(v) => updateNode(path, { type: v as RuleChainParamType })}
            >
              {PARAM_TYPES.map((t) => (
                <Select.Option key={t} value={t}>
                  {t}
                </Select.Option>
              ))}
            </Select>
            <Button
              size="small"
              type={node.required ? 'primary' : 'tertiary'}
              onClick={() => updateNode(path, { required: !node.required })}
              title={node.required ? '必填' : '可选'}
            >
              *
            </Button>
            <Input
              value={node.description}
              onChange={(v) => updateNode(path, { description: String(v) })}
              placeholder="描述（输出预览中为 // 注释）"
            />
            <div style={{ display: 'flex', gap: 4, flexWrap: 'nowrap' }}>
              {canNest ? (
                <Button size="small" type="tertiary" onClick={() => pushChange(addChildAtPath(nodes, path, () => defaultNode()))}>
                  +子
                </Button>
              ) : null}
              <Button size="small" type="tertiary" onClick={() => pushChange(insertSiblingAfter(nodes, path, () => defaultNode()))}>
                +
              </Button>
              <Button size="small" type="danger" theme="borderless" onClick={() => pushChange(removeAtPath(nodes, path))}>
                删
              </Button>
            </div>
          </div>
          {node.children.length > 0 ? renderTreeRows(node.children, level + 1, path) : null}
        </div>
      );
    });

  return (
    <div style={{ marginTop: 8 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8, flexWrap: 'wrap', gap: 8 }}>
        <Typography.Text strong>{title}</Typography.Text>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
          <Button size="small" type={viewMode === 'table' ? 'primary' : 'tertiary'} onClick={() => setViewMode('table')}>
            表格
          </Button>
          <Button size="small" type={viewMode === 'jsonOut' ? 'primary' : 'tertiary'} onClick={() => setViewMode('jsonOut')}>
            输出 JSON
          </Button>
          <Button size="small" type="tertiary" onClick={() => setImportOpen(true)}>
            导入 JSON
          </Button>
          <Button
            size="small"
            type="tertiary"
            onClick={() => {
              pushChange(parseRuleChainParamsJson(emptyRuleChainParamsJson()));
            }}
          >
            清空
          </Button>
        </div>
      </div>
      <Typography.Paragraph type="tertiary" size="small" style={{ marginBottom: 8 }}>
        树形参数说明；描述会出现在带注释的 JSON 预览中。
      </Typography.Paragraph>

      {viewMode === 'jsonOut' ? (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
          <TextArea value={previewWithComments} rows={10} readOnly style={{ fontFamily: 'monospace', fontSize: 12 }} />
          <TextArea value={previewObjText} rows={10} readOnly style={{ fontFamily: 'monospace', fontSize: 12 }} />
        </div>
      ) : null}

      {viewMode === 'table' ? (
        <>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: '14px 1fr 100px 36px 1fr auto',
              gap: 6,
              padding: '6px 0',
              fontSize: 12,
              color: 'var(--semi-color-text-2)',
              borderBottom: '1px solid rgba(6,7,9,0.12)',
            }}
          >
            <span />
            <span>参数名</span>
            <span>类型</span>
            <span title="必填">*</span>
            <span>说明</span>
            <span>操作</span>
          </div>
          {nodes.length === 0 ? (
            <Typography.Text type="tertiary" size="small" style={{ display: 'block', padding: '12px 0' }}>
              暂无参数，点击下方「添加根参数」
            </Typography.Text>
          ) : (
            renderTreeRows(nodes, 0)
          )}
          <Button style={{ marginTop: 8 }} type="tertiary" onClick={addRoot}>
            添加根参数
          </Button>
        </>
      ) : null}

      <Modal
        title={`导入 JSON — ${title}`}
        visible={importOpen}
        onCancel={() => setImportOpen(false)}
        onOk={applyImport}
        width={640}
      >
        <Typography.Paragraph type="tertiary" size="small">
          粘贴对象 JSON（将覆盖当前表格）。支持 // 与 /* */ 注释。
        </Typography.Paragraph>
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
          <Button
            size="small"
            type="tertiary"
            onClick={() => {
              const r = tryFormatJsonPretty(importText);
              if (!r.ok) {
                Toast.warning({
                  content: `无法格式化：${r.error}。含 // 注释时需先删掉注释再格式化（导入仍可按注释语法解析）。`,
                });
                return;
              }
              setImportText(r.text);
              Toast.success({ content: 'JSON 已格式化' });
            }}
          >
            格式化 JSON
          </Button>
        </div>
        <TextArea
          value={importText}
          onChange={setImportText}
          rows={10}
          placeholder='{ "repos": [ { "repo": "https://..." } ] }'
          style={{ fontFamily: 'monospace', fontSize: 12 }}
        />
        {importError ? (
          <Typography.Text type="danger" size="small" style={{ display: 'block', marginTop: 8 }}>
            {importError}
          </Typography.Text>
        ) : null}
      </Modal>
    </div>
  );
};
