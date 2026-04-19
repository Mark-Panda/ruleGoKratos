/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconLink from '../../assets/icon_link-one.svg';
import { WorkflowNodeType, OutPutPortType } from '../constants';

let index = 0;
export const CiGitCommitNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.GitCommit,
  info: {
    icon: iconLink,
    description: 'ci/gitCommit：在工作区提交变更；提交 hash 写入 metadata。',
  },
  meta: {
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: { width: 420, height: 360 },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: WorkflowNodeType.GitCommit,
      data: {
        title: `Git提交_${++index}`,
        positionType: 'middle',
        inputsValues: {
          directory: { type: 'template', content: '' },
          pattern: { type: 'constant', content: '' },
          message: { type: 'template', content: 'chore: commit from rulego' },
          signature: {
            type: 'constant',
            content: { authorName: '', authorEmail: '' },
          } as any,
        },
        inputs: {
          type: 'object',
          properties: {
            directory: {
              type: 'string',
              extra: { label: '本地仓库目录', formComponent: 'prompt-editor' },
            },
            pattern: {
              type: 'string',
              extra: {
                label: '添加文件 Glob',
                description: '相对工作区的模式，如 /example/*.go',
              },
            },
            message: {
              type: 'string',
              extra: { label: '提交说明', formComponent: 'prompt-editor' },
            },
            signature: {
              type: 'object',
              extra: {
                label: '作者（Signature）',
                description: 'authorName / authorEmail',
              },
            },
          },
        },
      },
    };
  },
};
