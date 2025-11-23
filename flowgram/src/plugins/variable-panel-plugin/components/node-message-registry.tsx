import React, { useEffect, useMemo, useState } from 'react';

import { WorkflowDocument, useService } from '@flowgram.ai/free-layout-editor';
import { Button, Input, Select, Table, TextArea, Toast } from '@douyinfe/semi-ui';

import { getRuleBaseInfo } from '../../../services/rule-base-info';

type NodeMsgTpl = {
  id: string;
  ts?: number;
  data?: any;
  msg?: any;
  metadata?: any;
  msgType?: string;
  dataType?: string;
};

const readAll = (): Record<string, NodeMsgTpl> => {
  try {
    const rid = String(getRuleBaseInfo()?.id || '');
    const raw = localStorage.getItem(`NODE_MSG_TPL_${rid}`);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
};

const writeAll = (data: Record<string, NodeMsgTpl>) => {
  try {
    const rid = String(getRuleBaseInfo()?.id || '');
    localStorage.setItem(`NODE_MSG_TPL_${rid}`, JSON.stringify(data));
  } catch {}
};

export const NodeMessageRegistry: React.FC = () => {
  const document = useService(WorkflowDocument);
  const [store, setStore] = useState<Record<string, NodeMsgTpl>>(() => readAll());
  const nodes = useMemo(() => document.getAllNodes(), [document]);

  useEffect(() => {
    const next = { ...store };
    nodes.forEach((n) => {
      const id = String(n.id);
      if (!next[id]) {
        next[id] = {
          id,
          ts: Date.now(),
          data: {},
          msg: {},
          metadata: {},
          msgType: 'default',
          dataType: 'JSON',
        };
      }
    });
    setStore(next);
    writeAll(next);
  }, []);

  const dataSource = nodes.map((n) => {
    const id = String(n.id);
    const tpl = store[id];
    return {
      id,
      ts: tpl?.ts ?? Date.now(),
      data: tpl?.data ?? {},
      msg: tpl?.msg ?? {},
      metadata: tpl?.metadata ?? {},
      msgType: tpl?.msgType ?? 'default',
      dataType: tpl?.dataType ?? 'JSON',
    } as NodeMsgTpl;
  });

  const updateTpl = (id: string, patch: Partial<NodeMsgTpl>) => {
    const next = { ...store, [id]: { ...(store[id] || { id }), ...(patch as any) } };
    setStore(next);
    writeAll(next);
  };

  return (
    <div style={{ padding: 8 }}>
      <Table
        dataSource={dataSource}
        pagination={false}
        rowKey={(r: any) => r.id}
        columns={[
          { title: 'Node ID', dataIndex: 'id', width: 200 },
          {
            title: 'ts',
            render: (_, r) => (
              <Input
                value={String(r.ts ?? '')}
                onChange={(v) => updateTpl(r.id, { ts: Number(v) || Date.now() })}
              />
            ),
            width: 160,
          },
          {
            title: 'msgType',
            render: (_, r) => (
              <Input
                value={String(r.msgType ?? '')}
                onChange={(v) => updateTpl(r.id, { msgType: String(v) })}
              />
            ),
            width: 160,
          },
          {
            title: 'dataType',
            render: (_, r) => (
              <Select
                value={String(r.dataType ?? 'JSON')}
                onChange={(v) => updateTpl(r.id, { dataType: String(v) })}
                style={{ width: 120 }}
              >
                <Select.Option value="JSON">JSON</Select.Option>
                <Select.Option value="TEXT">TEXT</Select.Option>
              </Select>
            ),
            width: 140,
          },
          {
            title: 'data',
            render: (_, r) => (
              <TextArea
                value={JSON.stringify(r.data ?? {}, null, 2)}
                onChange={(v) => {
                  try {
                    const obj = JSON.parse(String(v));
                    updateTpl(r.id, { data: obj });
                  } catch {
                    Toast.error('data 需为合法 JSON');
                  }
                }}
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            ),
          },
          {
            title: 'msg',
            render: (_, r) => (
              <TextArea
                value={JSON.stringify(r.msg ?? {}, null, 2)}
                onChange={(v) => {
                  try {
                    const obj = JSON.parse(String(v));
                    updateTpl(r.id, { msg: obj });
                  } catch {
                    Toast.error('msg 需为合法 JSON');
                  }
                }}
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            ),
          },
          {
            title: 'metadata',
            render: (_, r) => (
              <TextArea
                value={JSON.stringify(r.metadata ?? {}, null, 2)}
                onChange={(v) => {
                  try {
                    const obj = JSON.parse(String(v));
                    updateTpl(r.id, { metadata: obj });
                  } catch {
                    Toast.error('metadata 需为合法 JSON');
                  }
                }}
                autosize={{ minRows: 3, maxRows: 6 }}
              />
            ),
          },
          {
            title: '操作',
            render: (_, r) => (
              <Button
                size="small"
                type="tertiary"
                onClick={() => {
                  updateTpl(r.id, {
                    ts: Date.now(),
                    data: {},
                    msg: {},
                    metadata: {},
                    msgType: 'default',
                    dataType: 'JSON',
                  });
                }}
              >
                重置
              </Button>
            ),
            width: 120,
          },
        ]}
      />
    </div>
  );
};
