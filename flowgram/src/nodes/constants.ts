/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

export enum WorkflowNodeType {
  Start = 'start',
  End = 'end',
  LLM = 'ai/llm',
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
  Yapi = 'transform/yapi',
}

export enum OutPutPortType {
  SuccessPort = 'Success',
  FailurePort = 'Failure',
}
