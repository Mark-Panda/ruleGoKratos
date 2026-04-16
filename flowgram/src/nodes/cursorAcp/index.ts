/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLaptop from '../../assets/icon_collect-laptop.svg';
import { cursorAcpFormMeta } from './form-meta';

let index = 0;

/** 占位：ACP 要求每行一条 JSON-RPC；用户需按会话替换 id/params（参见 agent acp 文档）。 */
const defaultAcpStdinExample = [
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{"fs":{"readTextFile":false,"writeTextFile":false},"terminal":false},"clientInfo":{"name":"rulego","version":"0.1"}}}',
];

export const CursorAcpNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.CursorAcp,
  info: {
    icon: iconLaptop,
    description:
      '启动 agent acp。要发给 Cursor 的内容：写在「stdin JSON-RPC 行」里（每行一条 JSON-RPC），例如 session/prompt 请求里带上你的自然语言；不是单独某个「提示词」配置项。authenticate / session/new 等也在同一列表按顺序配置。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 320,
      height: 280,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: cursorAcpFormMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.CursorAcp,
      data: {
        title: `Cursor_ACP_${++index}`,
        positionType: 'middle',
        inputsValues: {
          stdinLines: { type: 'constant', content: defaultAcpStdinExample },
          apiKey: { type: 'template', content: '' },
          workspacePath: { type: 'template', content: '' },
          workDir: { type: 'template', content: '' },
          agentPath: { type: 'constant', content: 'agent' },
          args: { type: 'constant', content: ['acp'] },
          log: { type: 'constant', content: false },
          replaceData: { type: 'constant', content: true },
          timeoutMs: { type: 'constant', content: 120000 },
        },
        inputs: {
          type: 'object',
          required: ['agentPath', 'args', 'stdinLines', 'timeoutMs'],
          properties: {
            stdinLines: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: 'stdin JSON-RPC 行（用户指令在这里）',
                formComponent: 'array-editor',
                description:
                  '每一行是一条完整 JSON-RPC。你的任务/说明放在 method 为 session/prompt（等）的那一行 JSON 里，例如 params 中的文本字段；可用 ${msg.xxx} 模板。initialize、authenticate、session/new、session/prompt 按文档顺序各占一行或多行。',
              },
            },
            apiKey: {
              type: 'string',
              extra: {
                label: 'API 密钥（--api-key）',
                formComponent: 'prompt-editor',
                description:
                  '非空时插入 --api-key；留空则使用环境变量 CURSOR_API_KEY。推荐 ${metadata.xxx}，勿明文落库。',
              },
            },
            workspacePath: {
              type: 'string',
              extra: {
                label: '工作区路径（--workspace）',
                formComponent: 'prompt-editor',
                description:
                  '非空时插入 --workspace，指定仓库根目录作为 Agent 代码上下文。可与「工作目录」同时配置。',
              },
            },
            workDir: {
              type: 'string',
              extra: {
                label: '进程工作目录（cwd）',
                formComponent: 'prompt-editor',
                description: '子进程 cmd.Dir；留空用 metadata.workDir',
              },
            },
            agentPath: {
              type: 'string',
              extra: {
                label: 'agent 可执行文件',
                description: '默认 agent；常见安装路径 ~/.local/bin/agent',
              },
            },
            args: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: 'argv',
                formComponent: 'array-editor',
                description: '首项必须为 acp',
              },
            },
            log: {
              type: 'boolean',
              extra: { label: '输出到调试日志' },
            },
            replaceData: {
              type: 'boolean',
              extra: { label: '用 stdout 替换消息体' },
            },
            timeoutMs: {
              type: 'number',
              extra: { label: '超时(毫秒)', description: '默认 120000' },
            },
          },
        },
      },
    } as any;
  },
};
