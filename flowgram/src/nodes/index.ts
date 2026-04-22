/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FlowNodeRegistry } from '../typings';
import { YapiNodeRegistry } from './yapi';
import { TransformNodeRegistry } from './transform';
import { StartNodeRegistry } from './start';
import { RedisClientNodeRegistry } from './redisClient';
import { OpenSearchSearchNodeRegistry } from './opensearch-search';
import { VolcTlsSearchLogsNodeRegistry } from './volc-tls-search-logs';
import { CursorCliNodeRegistry } from './cursorCli';
import { CursorAcpNodeRegistry } from './cursorAcp';
import { FeishuWebhookNodeRegistry } from './feishuWebhook';
import { MultiNodeOutputRegistry } from './multi-node-output';
import { LuaTransformNodeRegistry } from './luaTransform';
import { LogStringNodeRegistry } from './logString';
import { AgentHarnessNodeRegistry } from './agent-harness';
import { JsFilterNodeRegistry } from './jsFilter';
import { JoinNodeRegistry } from './join';
import { HTTPNodeRegistry } from './http';
import { GroupNodeRegistry } from './group';
import { ForkNodeRegistry } from './fork';
import { ForNodeRegistry } from './for';
import { FlowSubChainNodeRegistry } from './flow';
import { FetchNodeOutputRegistry } from './fetch-node-output';
import { EndNodeRegistry } from './end';
import { InclusiveNodeRegistry } from './inclusive';
import { WhileNodeRegistry } from './while';
import { ExecNodeRegistry } from './exec';
import { FileReadNodeRegistry } from './file-read';
import { FileWriteNodeRegistry } from './file-write';
import { FileDeleteNodeRegistry } from './file-delete';
import { FileListNodeRegistry } from './file-list';
import { CiGitCloneNodeRegistry } from './ci-git-clone';
import { CiGitCommitNodeRegistry } from './ci-git-commit';
import { CiGitPushNodeRegistry } from './ci-git-push';
import { DBClientNodeRegistry } from './dbClient';
import { BreakNodeRegistry } from './break';
import { CronNodeRegistry } from './cron';
import { CommentNodeRegistry } from './comment';
import { CaseConditionNodeRegistry } from './case-condition';
import { BlockStartNodeRegistry } from './block-start';
import { BlockEndNodeRegistry } from './block-end';
export { WorkflowNodeType } from './constants';
export { NODE_TYPE_NAMES, getNodeTypeName, getNodeDisplayName } from './node-type-names';

// 节点注册
export const nodeRegistries: FlowNodeRegistry[] = [
  TransformNodeRegistry,
  YapiNodeRegistry,
  LuaTransformNodeRegistry,
  JsFilterNodeRegistry,
  LogStringNodeRegistry,
  CaseConditionNodeRegistry,
  InclusiveNodeRegistry,
  StartNodeRegistry,
  EndNodeRegistry,
  WhileNodeRegistry,
  ExecNodeRegistry,
  FileReadNodeRegistry,
  FileWriteNodeRegistry,
  FileDeleteNodeRegistry,
  FileListNodeRegistry,
  CiGitCloneNodeRegistry,
  CiGitCommitNodeRegistry,
  CiGitPushNodeRegistry,
  AgentHarnessNodeRegistry,
  ForNodeRegistry,
  BreakNodeRegistry,
  ForkNodeRegistry,
  JoinNodeRegistry,
  FetchNodeOutputRegistry,
  MultiNodeOutputRegistry,
  CommentNodeRegistry,
  BlockStartNodeRegistry,
  BlockEndNodeRegistry,
  HTTPNodeRegistry,
  GroupNodeRegistry,
  DBClientNodeRegistry,
  RedisClientNodeRegistry,
  OpenSearchSearchNodeRegistry,
  VolcTlsSearchLogsNodeRegistry,
  CursorCliNodeRegistry,
  CursorAcpNodeRegistry,
  FeishuWebhookNodeRegistry,
  CronNodeRegistry,
  FlowSubChainNodeRegistry,
];
