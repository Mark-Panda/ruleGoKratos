/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

export enum WorkflowNodeType {
  Start = 'start',
  End = 'end',
  AgentHarness = 'ai/agentHarness',
  HTTP = 'restApiCall',
  Code = 'code',
  Variable = 'variable',
  Condition = 'condition',
  Loop = 'loop',
  For = 'for',
  BlockStart = 'block-start',
  BlockEnd = 'block-end',
  Comment = 'comment',
  Continue = 'continue',
  Break = 'break',
  CaseCondition = 'switch',
  Transform = 'jsTransform',
  Fork = 'fork',
  Join = 'join',
  FetchNodeOutput = 'fetch-node-output',
  MultiNodeOutput = 'transform/multiNodeOutput',
  LogString = 'log',
  JsFilter = 'jsFilter',
  DBClient = 'dbClient',
  Cron = 'endpoint/schedule',
  Flow = 'flow',
  LuaTransform = 'luaTransform',
  RedisClient = 'x/redisClient',
  OpenSearchSearch = 'opensearch/search',
  VolcTlsSearchLogs = 'volcTls/searchLogs',
  CursorCli = 'x/cursorCli',
  CursorCliAuth = 'x/cursorCliAuth',
  CursorAcp = 'x/cursorAcp',
  FeishuWebhook = 'x/feishuWebhook',
  FeishuCliAuth = 'x/feishuCliAuth',
  Yapi = 'transform/yapi',

  /** RuleGo：包容分支（多分支同时命中） */
  Inclusive = 'inclusive',
  /** RuleGo：While 条件循环 */
  While = 'while',
  /** RuleGo：执行本地命令 */
  Exec = 'exec',
  FileRead = 'x/fileRead',
  /** 任务看板 */
  TaskBoard = 'x/taskBoard',
  /** 服务管理 */
  ServiceManagement = 'x/serviceManagement',
  /** JSON 提取与纠错 */
  JsonExtract = 'x/jsonExtract',
  /** 工作区仓库同步（与「工作区管理」同步仓库一致） */
  WorkspaceSync = 'x/workspaceSync',
  /** Sourcegraph 搜索节点（构建查询并执行搜索） */
  SourcegraphSearch = 'x/sourcegraphSearch',
  /** Sourcegraph Token 校验节点 */
  SourcegraphTokenVerify = 'x/sourcegraphTokenVerify',
  FileWrite = 'x/fileWrite',
  FileDelete = 'x/fileDelete',
  FileList = 'x/fileList',
  GitClone = 'ci/gitClone',
  GitCommit = 'ci/gitCommit',
  GitPush = 'ci/gitPush',
}

export enum OutPutPortType {
  SuccessPort = 'Success',
  FailurePort = 'Failure',
}

/** 画布入口点旁展示的输入端标识文案 */
export const InputPortLabel = '输入';
