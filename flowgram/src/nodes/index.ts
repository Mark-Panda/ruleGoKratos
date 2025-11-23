/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FlowNodeRegistry } from '../typings';
import { YapiNodeRegistry } from './yapi';
import { TransformNodeRegistry } from './transform';
import { StartNodeRegistry } from './start';
import { RedisClientNodeRegistry } from './redisClient';
import { MultiNodeOutputRegistry } from './multi-node-output';
import { LuaTransformNodeRegistry } from './luaTransform';
import { LogStringNodeRegistry } from './logString';
import { LLMNodeRegistry } from './llm';
import { JsFilterNodeRegistry } from './jsFilter';
import { JoinNodeRegistry } from './join';
import { HTTPNodeRegistry } from './http';
import { GroupNodeRegistry } from './group';
import { ForkNodeRegistry } from './fork';
import { ForNodeRegistry } from './for';
import { FlowSubChainNodeRegistry } from './flow';
import { FetchNodeOutputRegistry } from './fetch-node-output';
import { EndNodeRegistry } from './end';
import { DBClientNodeRegistry } from './dbClient';
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
  StartNodeRegistry,
  EndNodeRegistry,
  LLMNodeRegistry,
  ForNodeRegistry,
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
  CronNodeRegistry,
  FlowSubChainNodeRegistry,
];
