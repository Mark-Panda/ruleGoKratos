/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */
/* eslint-disable import/order */

import { FlowNodeRegistry } from '../typings';
import { CursorAcpNodeRegistry } from './cursorAcp';
import { CronNodeRegistry } from './cron';
import { CommentNodeRegistry } from './comment';
import { CiGitPushNodeRegistry } from './ci-git-push';
import { CiGitCommitNodeRegistry } from './ci-git-commit';
import { CiGitCloneNodeRegistry } from './ci-git-clone';
import { CaseConditionNodeRegistry } from './case-condition';
import { BreakNodeRegistry } from './break';
import { BlockStartNodeRegistry } from './block-start';
import { BlockEndNodeRegistry } from './block-end';
import { AgentHarnessNodeRegistry } from './agent-harness';
import { CursorCliNodeRegistry } from './cursorCli';
import { CursorCliAuthNodeRegistry } from './cursorCliAuth';
import { DBClientNodeRegistry } from './dbClient';
import { EndNodeRegistry } from './end';
import { ExecNodeRegistry } from './exec';
import { FeishuCliAuthNodeRegistry } from './feishuCliAuth';
import { FeishuWebhookNodeRegistry } from './feishuWebhook';
import { FetchNodeOutputRegistry } from './fetch-node-output';
import { FileDeleteNodeRegistry } from './file-delete';
import { FileListNodeRegistry } from './file-list';
import { FileReadNodeRegistry } from './file-read';
import { FileWriteNodeRegistry } from './file-write';
import { FlowSubChainNodeRegistry } from './flow';
import { ForNodeRegistry } from './for';
import { ForkNodeRegistry } from './fork';
import { GroupNodeRegistry } from './group';
import { HTTPNodeRegistry } from './http';
import { InclusiveNodeRegistry } from './inclusive';
import { JoinNodeRegistry } from './join';
import { JsFilterNodeRegistry } from './jsFilter';
import { JsonExtractNodeRegistry } from './jsonExtract';
import { LogStringNodeRegistry } from './logString';
import { LuaTransformNodeRegistry } from './luaTransform';
import { MultiNodeOutputRegistry } from './multi-node-output';
import { OpenSearchSearchNodeRegistry } from './opensearch-search';
import { RedisClientNodeRegistry } from './redisClient';
import { ServiceManagementNodeRegistry } from './service-management';
import { SourcegraphSearchNodeRegistry } from './sourcegraph-search';
import { SourcegraphTokenCreateNodeRegistry } from './sourcegraph-token-create';
import { SourcegraphTokenVerifyNodeRegistry } from './sourcegraph-token-verify';
import { StartNodeRegistry } from './start';
import { TaskBoardNodeRegistry } from './task-board';
import { TransformNodeRegistry } from './transform';
import { VolcTlsSearchLogsNodeRegistry } from './volc-tls-search-logs';
import { WhileNodeRegistry } from './while';
import { WorkspaceSyncNodeRegistry } from './workspace-sync';
import { YapiNodeRegistry } from './yapi';
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
  CursorCliAuthNodeRegistry,
  CursorAcpNodeRegistry,
  FeishuWebhookNodeRegistry,
  FeishuCliAuthNodeRegistry,
  CronNodeRegistry,
  JsonExtractNodeRegistry,
  WorkspaceSyncNodeRegistry,
  SourcegraphSearchNodeRegistry,
  SourcegraphTokenVerifyNodeRegistry,
  SourcegraphTokenCreateNodeRegistry,
  TaskBoardNodeRegistry,
  ServiceManagementNodeRegistry,
  FlowSubChainNodeRegistry,
];
