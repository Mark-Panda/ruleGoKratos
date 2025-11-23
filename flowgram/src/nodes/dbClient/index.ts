/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { WorkflowNodeType, OutPutPortType } from '../constants';
import { alphaNanoid } from '../../utils';
import { FlowNodeRegistry } from '../../typings';
import iconDB from '../../assets/icon_database-search.svg';

let index = 0;
export const DBClientNodeRegistry: FlowNodeRegistry = {
  type: WorkflowNodeType.DBClient,
  info: {
    icon: iconDB,
    description:
      '通过标准sql接口对数据库进行增删修改查操作。内置支持mysql和postgres数据库，可以执行SQL查询、更新、插入、删除、DDL等操作',
  },
  meta: {
    // 设置端口：一个输入，两个输出（success / failed）
    defaultPorts: [
      { type: 'input', location: 'left' },
      { type: 'output', location: 'right', portID: OutPutPortType.SuccessPort },
      { type: 'output', location: 'bottom', portID: OutPutPortType.FailurePort },
    ],
    size: {
      width: 360,
      height: 390,
    },
  },
  onAdd() {
    return {
      id: `${alphaNanoid(16)}`,
      type: 'dbClient',
      data: {
        title: `查询数据库_${++index}`,
        positionType: 'middle',
        inputsValues: {
          // SQL 模板内容：在表单中使用 prompt-editor 进行编辑，
          // 支持引用变量（模板插值），并可与下方 params 数组按顺序绑定占位符
          // 例如：`SELECT * FROM user WHERE id = ?`，由 params[0] 提供值
          sql: {
            type: 'template',
            content: '',
          },
          params: {
            // SQL 参数列表：数组形式，按顺序与 SQL 模板中的占位符绑定
            // 例如 SQL 为 `SELECT * FROM user WHERE id = ? AND status = ?`
            // 则 params[0] 对应 id，params[1] 对应 status
            type: 'constant',
            content: [],
          },
          getOne: {
            // 是否仅返回一条记录：true 返回第一条，false 返回全部
            type: 'constant',
            content: false,
          },
          poolSize: {
            // 数据库连接池大小：并发量较高时适当增大以提升性能
            // 留空或 0 使用后端默认值
            type: 'constant',
            content: 0,
          },
          driverName: {
            // 数据库驱动名称：目前支持 'mysql' 与 'postgres'
            type: 'constant',
            content: 'mysql',
          },
          dsn: {
            // DSN 连接字符串模板：例如
            // mysql:   user:pass@tcp(host:port)/db?charset=utf8mb4
            // postgres: postgres://user:pass@host:port/db?sslmode=disable
            // 支持变量插值，可结合环境变量或全局变量
            type: 'template',
            content: '',
          },
        },
        inputs: {
          type: 'object',
          required: ['sql', 'getOne', 'poolSize', 'driverName', 'dsn'],
          properties: {
            driverName: {
              type: 'string',
              enum: ['mysql', 'postgres'],
              extra: { label: '数据库驱动名称', formComponent: 'enum-select' },
            },
            dsn: {
              type: 'string',
              extra: {
                description:
                  '数据库连接字符串，支持模板变量。示例：mysql 使用 user:pass@tcp(host:port)/db?charset=utf8mb4；postgres 使用 postgres://user:pass@host:port/db?sslmode=disable',
              },
            },
            sql: {
              type: 'string',
              extra: {
                label: 'sql',
                formComponent: 'sql-editor',
                description:
                  '可以使用 ${metadata.key} 或者 ${msg.key}变量，SQL参数允许使用 ? 占位符',
              },
            },
            params: {
              type: 'array',
              items: {
                type: 'string',
              },
              extra: {
                label: '参数列表',
                description:
                  '可以使用 ${metadata.key} 读取元数据中的变量或者使用 ${msg.key} 读取消息负荷中的变量进行替换',
                formComponent: 'array-editor',
              },
            },
            getOne: {
              type: 'boolean',
              extra: {
                label: '是否仅返回一条记录',
                description: '开启后仅返回第一条记录；关闭返回全部记录',
              },
            },
            poolSize: {
              type: 'number',
              extra: {
                label: '连接池大小',
                description: '并发量大时适当增大；留空或 0 使用默认值',
              },
            },
          },
        },
        // outputs: {
        //   type: 'object',
        //   properties: {
        //     // result: { type: 'string' },
        //   },
        // },
      },
    };
  },
};
