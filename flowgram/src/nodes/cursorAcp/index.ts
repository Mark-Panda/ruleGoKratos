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

export const CursorAcpNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.CursorAcp,
  info: {
    icon: iconLaptop,
    description:
      'ACP（Agent Client Protocol）：简易模式下只需填写 API Key 与任务说明，后端按官方文档自动完成 initialize → authenticate(cursor_login) → session/new → session/prompt；认证优先使用下方 API 密钥（等价 CLI --api-key）。详见 cursor.com/docs/cli/acp。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 340,
      height: 300,
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
          acpSimpleMode: { type: 'constant', content: true },
          acpTask: {
            type: 'template',
            content: '请根据当前消息说明要完成的工作。',
          },
          agentPath: { type: 'constant', content: 'agent' },
          apiKey: { type: 'template', content: '' },
          workspacePath: { type: 'template', content: '' },
          workDir: { type: 'template', content: '' },
          replaceData: { type: 'constant', content: true },
          timeoutMs: { type: 'constant', content: 120000 },
          log: { type: 'constant', content: false },
          stdinLines: { type: 'constant', content: [] as string[] },
          args: { type: 'constant', content: ['acp'] },
        },
        inputs: {
          type: 'object',
          required: ['agentPath', 'args', 'acpSimpleMode', 'timeoutMs'],
          properties: {
            acpSimpleMode: {
              type: 'boolean',
              extra: {
                label: '简易模式（推荐）',
                description:
                  '开启后只需配置 API Key 与「任务说明」，由规则引擎按 Cursor ACP 文档自动发送 JSON-RPC（initialize → authenticate → session/new → session/prompt）；关闭后可自行编辑下方 stdin 行。',
              },
            },
            acpTask: {
              type: 'string',
              extra: {
                label: '任务说明（session/prompt）',
                formComponent: 'prompt-editor',
                description:
                  '发给 Agent 的自然语言任务；支持 ${msg.body} 等模板。仅在简易模式下写入 session/prompt。',
              },
            },
            agentPath: {
              type: 'string',
              extra: {
                label: 'agent 可执行文件',
                description: '默认 agent；常见路径 ~/.local/bin/agent',
              },
            },
            apiKey: {
              type: 'string',
              extra: {
                label: 'API 密钥（推荐）',
                formComponent: 'prompt-editor',
                description:
                  '对应官方 CLI 的 --api-key；留空则使用运行环境变量 CURSOR_API_KEY。与 agent login 二选一即可，建议 ${metadata.xxx} 注入。authenticate 步骤仍会发送 methodId: cursor_login（与文档一致）。',
              },
            },
            workspacePath: {
              type: 'string',
              extra: {
                label: '工作区（--workspace）',
                formComponent: 'prompt-editor',
                description: '代码仓库根目录，供 CLI 加载上下文；简易模式下同时作为 session/new 的 cwd。',
              },
            },
            workDir: {
              type: 'string',
              extra: {
                label: '进程工作目录（cwd）',
                formComponent: 'prompt-editor',
                description: '子进程当前目录；留空用 metadata.workDir',
              },
            },
            replaceData: {
              type: 'boolean',
              extra: {
                label: '用 stdout 替换消息体',
                description: '成功时写入最后一轮 RPC 返回及后续 stdout（简易模式）',
              },
            },
            timeoutMs: {
              type: 'number',
              extra: {
                label: '超时(毫秒)',
                description: '默认 120000',
              },
            },
            log: {
              type: 'boolean',
              extra: { label: '输出到调试日志' },
            },
            stdinLines: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: 'stdin JSON-RPC 行（高级）',
                formComponent: 'array-editor',
                description:
                  '仅在关闭「简易模式」时使用：每行一条 JSON-RPC。需要自行维护 initialize / authenticate / session 等序列。',
              },
            },
            args: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: '命令行 argv（高级）',
                formComponent: 'array-editor',
                description: '首项须为 acp，一般无需修改。',
              },
            },
          },
        },
      },
    } as any;
  },
};
