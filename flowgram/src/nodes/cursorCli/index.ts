/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLaptop from '../../assets/icon_collect-laptop.svg';
import { cursorCliFormMeta } from './form-meta';

let index = 0;

export const CursorCliNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.CursorCli,
  info: {
    icon: iconLaptop,
    description:
      '无头调用：打印模式 + 任务说明 + 输出格式（text/json/stream-json）+ 模型，对应 agent -p "…" --output-format … --model …。密钥用 apiKey 或环境变量；仓库根用 workspacePath。「额外参数」仅作补充。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 320,
      height: 260,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: cursorCliFormMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.CursorCli,
      data: {
        title: `Cursor_CLI_${++index}`,
        positionType: 'middle',
        inputsValues: {
          printMode: { type: 'constant', content: true },
          prompt: {
            type: 'template',
            content: 'find and fix performance issues',
          },
          outputFormat: { type: 'constant', content: 'text' },
          model: { type: 'template', content: 'gpt-5.2' },
          agentPath: { type: 'constant', content: 'agent' },
          apiKey: { type: 'template', content: '' },
          workspacePath: { type: 'template', content: '' },
          workDir: { type: 'template', content: '' },
          args: { type: 'constant', content: [] },
          log: { type: 'constant', content: false },
          replaceData: { type: 'constant', content: true },
          timeoutMs: { type: 'constant', content: 0 },
        },
        inputs: {
          type: 'object',
          required: ['agentPath', 'args', 'log', 'replaceData', 'timeoutMs'],
          properties: {
            printMode: {
              type: 'boolean',
              extra: {
                label: '打印模式（-p）',
                description:
                  '开启后由后端插入 -p（无头 agent），与官方 agent -p 一致；任务说明请填「任务说明」字段，勿再手写到「额外参数」里。',
              },
            },
            prompt: {
              type: 'string',
              extra: {
                label: '任务说明（-p 后的内容）',
                formComponent: 'prompt-editor',
                description:
                  '对应命令行中带引号的提示部分，例如 agent -p "find and fix ..." 中的双引号内文案。支持 ${msg.body} 等模板；仅在「打印模式」开启时生效。',
              },
            },
            outputFormat: {
              type: 'string',
              enum: ['text', 'json', 'stream-json'],
              default: { type: 'constant', content: 'text' } as any,
              extra: {
                label: '输出格式（--output-format）',
                formComponent: 'enum-select',
                description:
                  '仅在「打印模式」开启时由后端插入 --output-format；可选 text、json、stream-json（与官方 CLI 文档一致，默认 text）。勿再写入「额外参数」。',
              },
            },
            model: {
              type: 'string',
              extra: {
                label: '模型（--model）',
                formComponent: 'prompt-editor',
                description:
                  '非空时追加 --model <值>，例如 gpt-5.2；与 apiKey 一样为独立配置项，勿拆进「额外参数」。',
              },
            },
            agentPath: {
              type: 'string',
              extra: {
                label: 'agent 可执行文件',
                description:
                  '安装后多为 ~/.local/bin/agent（见官方安装文档）；留空或 agent 表示 PATH；后端允许 basename 为 agent 或 cursor',
              },
            },
            apiKey: {
              type: 'string',
              extra: {
                label: 'API 密钥（--api-key）',
                formComponent: 'prompt-editor',
                description:
                  '非空时插入 --api-key；留空则由运行进程读取环境变量 CURSOR_API_KEY 并同样以 --api-key 传入。建议用 ${metadata.xxx} 注入，避免明文落库。',
              },
            },
            workspacePath: {
              type: 'string',
              extra: {
                label: '工作区路径（--workspace）',
                formComponent: 'prompt-editor',
                description:
                  '指定代码仓库根目录，Agent 以此作为代码上下文（官方全局参数 --workspace）。示例：/path/to/repo 或 ${metadata.repoRoot}。与「进程工作目录」不同：后者是子进程 cwd（workDir/metadata.workDir）。',
              },
            },
            workDir: {
              type: 'string',
              extra: {
                label: '进程工作目录（cwd）',
                formComponent: 'prompt-editor',
                description:
                  '子进程当前工作目录（cmd.Dir）；留空则用 metadata.workDir。与「工作区路径」不同：后者对应 agent 的 --workspace，用于指定仓库根以加载规则与代码上下文。',
              },
            },
            args: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: '额外命令行参数',
                formComponent: 'array-editor',
                description:
                  '在 -p、任务说明、--output-format、--model 之后追加的其它 argv（如 --force）。勿重复写 --output-format。',
              },
            },
            log: {
              type: 'boolean',
              extra: {
                label: '输出到调试日志',
                description: '开启时将子进程 stdout/stderr 写入规则引擎调试回调',
              },
            },
            replaceData: {
              type: 'boolean',
              extra: {
                label: '用 stdout 替换消息体',
                description: '成功时以标准输出（若为空则用 stderr）覆盖下游 msg 数据',
              },
            },
            timeoutMs: {
              type: 'number',
              extra: {
                label: '超时(毫秒)',
                description: '0 表示不额外限制',
              },
            },
          },
        },
      },
    } as any;
  },
};
