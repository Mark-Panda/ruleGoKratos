/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLink from '../../assets/icon_link-one.svg';
import { feishuWebhookFormMeta } from './form-meta';

let index = 0;

const defaultCardExample =
  '{"config":{"wide_screen_mode":true},"elements":[{"tag":"div","text":{"tag":"lark_md","content":"**规则链通知** 来自 ${msg.type}"}}]}';

export const FeishuWebhookNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.FeishuWebhook,
  info: {
    icon: iconLink,
    description:
      '飞书 Webhook：text；post 可勾选按行分段、@所有人、@成员；interactive 可选「通知卡片」填标题+Markdown 或手写 JSON；raw 整包。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'input', location: 'top', portID: 'input_top' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 560,
    },
    defaultExpanded: false,
    expandable: true,
  },
  formMeta: feishuWebhookFormMeta,
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.FeishuWebhook,
      data: {
        title: `Feishu_Webhook_${++index}`,
        positionType: 'middle',
        inputsValues: {
          msgType: { type: 'constant', content: 'text' },
          webhookUrl: {
            type: 'template',
            content: 'https://open.feishu.cn/open-apis/bot/v2/hook/xxxxxxxx',
          },
          text: {
            type: 'template',
            content: 'Hello from rule chain: ${msg.type}',
          },
          postTitle: { type: 'template', content: '通知标题' },
          postBody: { type: 'template', content: '第一行说明\n第二行 ${msg.type}' },
          postLang: { type: 'constant', content: 'zh_cn' },
          postSplitByLine: { type: 'constant', content: true },
          postAtAllBefore: { type: 'constant', content: false },
          postAtAllAfter: { type: 'constant', content: false },
          postMentionUserIds: { type: 'constant', content: [] as string[] },
          interactivePreset: { type: 'constant', content: 'card_json' },
          cardNoticeTitle: { type: 'template', content: '规则链通知' },
          cardNoticeMarkdown: {
            type: 'template',
            content: '**摘要** 类型 ${msg.type}，可继续写 Markdown。',
          },
          cardJson: { type: 'template', content: defaultCardExample },
          rawJson: {
            type: 'template',
            content: '{"msg_type":"text","content":{"text":"raw 模式整包示例"}}',
          },
          timeoutMs: { type: 'constant', content: 15000 },
          replaceData: { type: 'constant', content: false },
        },
        inputs: {
          type: 'object',
          required: ['msgType', 'webhookUrl', 'timeoutMs'],
          properties: {
            msgType: {
              type: 'string',
              enum: ['text', 'post', 'interactive', 'raw'],
              default: { type: 'constant', content: 'text' } as any,
              extra: {
                label: '消息类型',
                formComponent: 'enum-select',
                description: 'text / post / interactive / raw',
              },
            },
            webhookUrl: {
              type: 'string',
              extra: {
                label: 'Webhook URL',
                formComponent: 'prompt-editor',
                description: '须为 https；建议 ${metadata.feishuHook} 注入。',
              },
            },
            text: {
              type: 'string',
              extra: {
                label: '纯文本（text）',
                formComponent: 'prompt-editor',
                description: '仅 msg_type=text 时使用。',
              },
            },
            postTitle: {
              type: 'string',
              extra: {
                label: '富文本标题（post）',
                formComponent: 'prompt-editor',
                description: 'post.<语言>.title',
              },
            },
            postBody: {
              type: 'string',
              extra: {
                label: '富文本正文（post）',
                formComponent: 'prompt-editor',
                description:
                  '与下方选项组合：可整段一块文本，或按行拆成多段；支持 @ 与成员列表。',
              },
            },
            postLang: {
              type: 'string',
              enum: ['zh_cn', 'en_us', 'ja_jp'],
              default: { type: 'constant', content: 'zh_cn' } as any,
              extra: {
                label: '富文本语言（post）',
                formComponent: 'enum-select',
              },
            },
            postSplitByLine: {
              type: 'boolean',
              extra: {
                label: '正文按「换行」拆成多段',
                description:
                  '勾选：每一非空行单独成一段（适合多条要点）；不勾选：整段正文一块。',
              },
            },
            postAtAllBefore: {
              type: 'boolean',
              extra: {
                label: '在正文前插入 @所有人',
                description: '在正文各段之前插入一行 @所有人（飞书 post 的 at 段）。',
              },
            },
            postAtAllAfter: {
              type: 'boolean',
              extra: {
                label: '在正文后插入 @所有人',
                description: '在正文与各 @成员 之后追加一行 @所有人。',
              },
            },
            postMentionUserIds: {
              type: 'array',
              items: { type: 'string' },
              extra: {
                label: '@指定成员（open_id / user_id）',
                formComponent: 'array-editor',
                arrayAddLabel: '添加成员',
                description:
                  '每条填一个 open_id 或 user_id（如 ou_xxx）；按顺序在正文后各插入一行 @该成员。',
              },
            },
            interactivePreset: {
              type: 'string',
              enum: ['card_json', 'notice_card'],
              default: { type: 'constant', content: 'card_json' } as any,
              extra: {
                label: '卡片配置方式（interactive）',
                formComponent: 'enum-select',
                description:
                  '自定义 JSON：填写下方「卡片 JSON」。通知卡片：只需填标题 + Markdown，无需写 JSON。',
              },
            },
            cardNoticeTitle: {
              type: 'string',
              extra: {
                label: '通知卡片标题',
                formComponent: 'prompt-editor',
                description: '仅「通知卡片」模式；可为空则只显示正文区。',
              },
            },
            cardNoticeMarkdown: {
              type: 'string',
              extra: {
                label: '通知卡片 Markdown',
                formComponent: 'prompt-editor',
                description: '仅「通知卡片」模式；支持 **粗体**、`代码` 等飞书 lark_md。',
              },
            },
            cardJson: {
              type: 'string',
              extra: {
                label: '卡片 JSON（自定义）',
                formComponent: 'prompt-editor',
                description: '仅「自定义 JSON」模式：卡片对象，不含最外层 msg_type。',
              },
            },
            rawJson: {
              type: 'string',
              extra: {
                label: '自定义整包（raw）',
                formComponent: 'prompt-editor',
                description: '完整 Webhook 请求体 JSON，须含 msg_type。',
              },
            },
            timeoutMs: {
              type: 'number',
              extra: {
                label: '超时(毫秒)',
                description: '默认 15000；≤0 时后端按 15000。',
              },
            },
            replaceData: {
              type: 'boolean',
              extra: {
                label: '用响应体替换消息体',
                description: '成功时写入下游 msg 数据。',
              },
            },
          },
        },
      },
    } as any;
  },
};
