/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FlowNodeRegistry } from '../typings';
import { YapiNodeRegistry } from './yapi';
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
import { VolcTlsSearchLogsNodeRegistry } from './volc-tls-search-logs';
import { TransformNodeRegistry } from './transform';
import { StartNodeRegistry } from './start';
import { RedisClientNodeRegistry } from './redisClient';
import { OpenSearchSearchNodeRegistry } from './opensearch-search';
import { MultiNodeOutputRegistry } from './multi-node-output';
import { LuaTransformNodeRegistry } from './luaTransform';
import { FileWriteNodeRegistry } from './file-write';
import { FileReadNodeRegistry } from './file-read';
import { FileListNodeRegistry } from './file-list';
import { FileDeleteNodeRegistry } from './file-delete';
import { FeishuWebhookNodeRegistry } from './feishuWebhook';
import { ExecNodeRegistry } from './exec';
import { DBClientNodeRegistry } from './dbClient';
import { CursorCliNodeRegistry } from './cursorCli';
import { CursorCliAuthNodeRegistry } from './cursorCliAuth';
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
import { TaskBoardNodeRegistry } from './task-board';
import { ServiceManagementNodeRegistry } from './service-management';
import { JsonExtractNodeRegistry } from './jsonExtract';
import { FeishuCliAuthNodeRegistry } from './feishuCliAuth';
import { WorkspaceSyncNodeRegistry } from './workspace-sync';
import { SourcegraphSearchNodeRegistry } from './sourcegraph-search';
import { SourcegraphTokenVerifyNodeRegistry } from './sourcegraph-token-verify';
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
  TaskBoardNodeRegistry,
  ServiceManagementNodeRegistry,
  FlowSubChainNodeRegistry,
];
