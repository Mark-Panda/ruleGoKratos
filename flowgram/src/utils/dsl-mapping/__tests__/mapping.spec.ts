/* eslint-disable import/no-extraneous-dependencies -- vitest 在 devDependencies，单测合法 */
import { describe, expect, it, vi } from 'vitest';

import type { NodeMappingSpec } from '../types';
import {
  aiAgentHarnessMappingSpec,
  cursorAcpMappingSpec,
  cursorCliMappingSpec,
  cursorCliAuthMappingSpec,
  feishuWebhookMappingSpec,
  feishuCliAuthMappingSpec,
  workspaceSyncMappingSpec,
  dbClientMappingSpec,
  flowMappingSpec,
  jsFilterMappingSpec,
  jsTransformMappingSpec,
  logMappingSpec,
  luaTransformMappingSpec,
  multiNodeOutputMappingSpec,
  opensearchSearchMappingSpec,
  redisClientMappingSpec,
  restApiCallMappingSpec,
  inclusiveMappingSpec,
  switchMappingSpec,
  whileMappingSpec,
  execMappingSpec,
  volcTlsSearchLogsMappingSpec,
  yapiMappingSpec,
  jsonExtractMappingSpec,
} from '../specs';
import { mapDslToNodeInputsValues, mapNodeToDslConfig } from '../engine';
import {
  buildDocumentFromRuleChainJSON,
  buildRuleChainJSONFromDocument,
} from '../../rulechain-builder';

const baseSpec: NodeMappingSpec = {
  nodeType: 'test/node',
  fields: [
    {
      inputKey: 'name',
      dslKey: 'name',
      valueType: 'constant',
      defaultValue: 'anon',
    },
    {
      inputKey: 'count',
      dslKey: 'count',
      valueType: 'number',
      defaultValue: 0,
    },
    {
      inputKey: 'enabled',
      dslKey: 'enabled',
      valueType: 'boolean',
      defaultValue: false,
    },
    {
      inputKey: 'tpl',
      dslKey: 'tpl',
      valueType: 'template',
      defaultValue: 'x',
    },
  ],
};

describe('mapNodeToDslConfig', () => {
  it('reads inputsValues[key].content and applies defaults when missing', () => {
    const node = {
      data: {
        inputsValues: {
          name: { content: 'hello' },
        },
      },
    };
    const cfg = mapNodeToDslConfig(node, baseSpec);
    expect(cfg.name).toBe('hello');
    expect(cfg.count).toBe(0);
    expect(cfg.enabled).toBe(false);
    expect(cfg.tpl).toBe('x');
  });

  it('normalizes number from string', () => {
    const node = {
      data: {
        inputsValues: {
          count: { content: '42' },
        },
      },
    };
    expect(mapNodeToDslConfig(node, baseSpec).count).toBe(42);
  });

  it('normalizes boolean from common string/number forms', () => {
    const spec = { ...baseSpec, fields: [...baseSpec.fields] };
    const node = (enabled: unknown) => ({
      data: { inputsValues: { enabled: { content: enabled } } },
    });
    expect(mapNodeToDslConfig(node('true'), spec).enabled).toBe(true);
    expect(mapNodeToDslConfig(node('0'), spec).enabled).toBe(false);
    expect(mapNodeToDslConfig(node(1), spec).enabled).toBe(true);
  });

  it('treats boolean empty string as empty and applies defaultValue (toDSL)', () => {
    const spec: NodeMappingSpec = {
      nodeType: 'b',
      fields: [
        {
          inputKey: 'flag',
          dslKey: 'flag',
          valueType: 'boolean',
          defaultValue: true,
        },
      ],
    };
    expect(
      mapNodeToDslConfig({ data: { inputsValues: { flag: { content: '' } } } }, spec).flag
    ).toBe(true);
  });

  it('runs spec.transformOut on full config', () => {
    const spec: NodeMappingSpec = {
      ...baseSpec,
      transformOut: (c) => {
        c.tag = 'marked';
        return c;
      },
    };
    const cfg = mapNodeToDslConfig({ data: { inputsValues: {} } }, spec);
    expect(cfg.tag).toBe('marked');
    expect(cfg.name).toBe('anon');
  });

  it('runs field.transformOut after normalization', () => {
    const spec: NodeMappingSpec = {
      nodeType: 't',
      fields: [
        {
          inputKey: 'n',
          dslKey: 'n',
          valueType: 'number',
          defaultValue: 1,
          transformOut: (v) => (typeof v === 'number' ? v * 2 : v),
        },
      ],
    };
    expect(mapNodeToDslConfig({ data: { inputsValues: { n: { content: 3 } } } }, spec).n).toBe(6);
  });

  it('warns and keeps normalized value when field.transformOut throws', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const spec: NodeMappingSpec = {
      nodeType: 'x',
      fields: [
        {
          inputKey: 'a',
          dslKey: 'a',
          valueType: 'constant',
          defaultValue: 'z',
          transformOut: () => {
            throw new Error('boom');
          },
        },
      ],
    };
    const cfg = mapNodeToDslConfig({ data: { inputsValues: { a: { content: 'v' } } } }, spec);
    expect(cfg.a).toBe('v');
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it('warns and returns pre-hook config when spec.transformOut throws', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const spec: NodeMappingSpec = {
      nodeType: 'x',
      fields: [{ inputKey: 'a', dslKey: 'a', valueType: 'constant', defaultValue: 1 }],
      transformOut: () => {
        throw new Error('boom');
      },
    };
    const cfg = mapNodeToDslConfig({ data: { inputsValues: {} } }, spec);
    expect(cfg.a).toBe(1);
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });
});

describe('mapDslToNodeInputsValues', () => {
  it('maps configuration keys back to inputsValues with content', () => {
    const iv = mapDslToNodeInputsValues(
      { name: 'a', count: '7', enabled: 'true', tpl: 't' },
      baseSpec
    );
    expect(iv.name?.content).toBe('a');
    expect(iv.count?.content).toBe(7);
    expect(iv.enabled?.content).toBe(true);
    expect(iv.tpl?.content).toBe('t');
  });

  it('applies spec.transformIn before field mapping', () => {
    const spec: NodeMappingSpec = {
      ...baseSpec,
      transformIn: (c) => {
        c.name = 'from-hook';
        return c;
      },
    };
    const iv = mapDslToNodeInputsValues({}, spec);
    expect(iv.name?.content).toBe('from-hook');
  });

  it('parses json valueType on export and round-trips structure', () => {
    const spec: NodeMappingSpec = {
      nodeType: 'j',
      fields: [
        {
          inputKey: 'meta',
          dslKey: 'meta',
          valueType: 'json',
          defaultValue: null,
        },
      ],
    };
    const cfg = mapNodeToDslConfig(
      { data: { inputsValues: { meta: { content: '{"a":1}' } } } },
      spec
    );
    expect(cfg.meta).toEqual({ a: 1 });
    const iv = mapDslToNodeInputsValues(cfg as Record<string, unknown>, spec);
    expect(iv.meta?.content).toEqual({ a: 1 });
  });

  it('fromDSL: only undefined uses defaultValue; explicit null is preserved', () => {
    const spec: NodeMappingSpec = {
      nodeType: 'j',
      fields: [
        {
          inputKey: 'meta',
          dslKey: 'meta',
          valueType: 'json',
          defaultValue: { d: 1 },
        },
      ],
    };
    const missing = mapDslToNodeInputsValues({}, spec);
    expect(missing.meta?.content).toEqual({ d: 1 });
    const explicitNull = mapDslToNodeInputsValues({ meta: null }, spec);
    expect(explicitNull.meta?.content).toBeNull();
  });

  it('fromDSL: boolean empty string uses defaultValue', () => {
    const spec: NodeMappingSpec = {
      nodeType: 'b',
      fields: [
        {
          inputKey: 'flag',
          dslKey: 'flag',
          valueType: 'boolean',
          defaultValue: true,
        },
      ],
    };
    expect(mapDslToNodeInputsValues({ flag: '' }, spec).flag?.content).toBe(true);
  });

  it('fromDSL: json string is parsed; invalid JSON string is kept', () => {
    const spec: NodeMappingSpec = {
      nodeType: 'j',
      fields: [
        {
          inputKey: 'meta',
          dslKey: 'meta',
          valueType: 'json',
          defaultValue: null,
        },
      ],
    };
    expect(mapDslToNodeInputsValues({ meta: '{"b":2}' }, spec).meta?.content).toEqual({ b: 2 });
    const bad = '{not json';
    expect(mapDslToNodeInputsValues({ meta: bad }, spec).meta?.content).toBe(bad);
  });

  it('warns and keeps pre-hook cfg when spec.transformIn throws', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const spec: NodeMappingSpec = {
      nodeType: 'x',
      fields: [
        {
          inputKey: 'a',
          dslKey: 'a',
          valueType: 'constant',
          defaultValue: 'd',
        },
      ],
      transformIn: () => {
        throw new Error('boom');
      },
    };
    const iv = mapDslToNodeInputsValues({ a: 'ok' } as Record<string, unknown>, spec);
    expect(iv.a?.content).toBe('ok');
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it('warns and keeps normalized value when field.transformIn throws', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const spec: NodeMappingSpec = {
      nodeType: 'x',
      fields: [
        {
          inputKey: 'a',
          dslKey: 'a',
          valueType: 'number',
          defaultValue: 0,
          transformIn: () => {
            throw new Error('boom');
          },
        },
      ],
    };
    const iv = mapDslToNodeInputsValues({ a: 7 } as Record<string, unknown>, spec);
    expect(iv.a?.content).toBe(7);
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });
});

describe('ai/agentHarness spec round-trip', () => {
  it('preserves model/prompts, configurable toggles, skill allowlist, and limits', () => {
    const node = {
      data: {
        inputsValues: {
          model: { content: '${model}' },
          systemPrompt: { content: 'S' },
          userPrompt: { content: 'U' },
          enableSkillTool: { content: false },
          enableUUIDTool: { content: false },
          enableWorkspaceTools: { content: true },
          skillAllowlist: { content: ['a', 'b'] },
          maxIterations: { content: 3 },
          maxToolCalls: { content: 10 },
          toolTimeoutSecs: { content: 30 },
        },
      },
    };
    const cfg = mapNodeToDslConfig(node, aiAgentHarnessMappingSpec);
    expect(cfg.model).toBe('${model}');
    expect(cfg.enableSkillTool).toBe(false);
    expect(cfg.enableMcpTool).toBeUndefined();
    expect(cfg.enableWorkspaceTools).toBe(true);
    expect(cfg.skillAllowlist).toEqual(['a', 'b']);
    expect(cfg.mcpAllowlist).toBeUndefined();
    expect(cfg.maxIterations).toBe(3);
    expect(cfg.llmConfigId).toBe(0);
    expect(cfg.llmModelEntryId).toBe(0);

    const iv = mapDslToNodeInputsValues(cfg as Record<string, unknown>, aiAgentHarnessMappingSpec);
    expect(iv.model?.content).toBe('${model}');
    expect(iv.systemPrompt?.content).toBe('S');
    expect(iv.userPrompt?.content).toBe('U');
    expect(iv.enableSkillTool?.content).toBe(false);
    expect(iv.enableMcpTool).toBeUndefined();
    expect(iv.enableUUIDTool?.content).toBe(false);
    expect(iv.enableWorkspaceTools?.content).toBe(true);
    expect(iv.skillAllowlist?.content).toEqual(['a', 'b']);
    expect(iv.mcpAllowlist).toBeUndefined();
    expect(iv.maxIterations?.content).toBe(3);
    expect(iv.maxToolCalls?.content).toBe(10);
    expect(iv.toolTimeoutSecs?.content).toBe(30);
    expect(iv.llmConfigId?.content).toBe(0);
    expect(iv.llmModelEntryId?.content).toBe(0);
  });

  it('toDSL: missing inputsValues keys use agentHarness spec defaults', () => {
    const node = {
      data: {
        inputsValues: {
          userPrompt: { content: 'hi' },
        },
      },
    };
    const cfg = mapNodeToDslConfig(node, aiAgentHarnessMappingSpec);
    expect(cfg.model).toBe('');
    expect(cfg.systemPrompt).toBe('');
    expect(cfg.userPrompt).toBe('hi');
    expect(cfg.enableSkillTool).toBe(true);
    expect(cfg.enableMcpTool).toBeUndefined();
    expect(cfg.enableUUIDTool).toBe(true);
    expect(cfg.enableWorkspaceTools).toBe(true);
    expect(cfg.enableSubAgentTool).toBe(true);
    expect(cfg.skillAllowlist).toEqual([]);
    expect(cfg.mcpAllowlist).toBeUndefined();
    expect(cfg.maxIterations).toBe(0);
    expect(cfg.maxToolCalls).toBe(0);
    expect(cfg.toolTimeoutSecs).toBe(0);
    expect(cfg.llmConfigId).toBe(0);
    expect(cfg.llmModelEntryId).toBe(0);
  });

  it('fromDSL：字符串 Skill 白名单转为数组（兼容旧链）', () => {
    const iv = mapDslToNodeInputsValues(
      { skillAllowlist: 'a, b', mcpAllowlist: 's1:t1,s2:*' } as Record<string, unknown>,
      aiAgentHarnessMappingSpec
    );
    expect(iv.skillAllowlist?.content).toEqual(['a', 'b']);
    expect(iv.mcpAllowlist).toBeUndefined();
  });
});

describe('restApiCall spec URL / query', () => {
  it('toDSL：将 params 序列化拼到 restEndpointUrlPattern', () => {
    const node = {
      data: {
        inputsValues: {
          url: { content: 'https://api.example/items' },
          requestMethod: { content: 'POST' },
          params: { content: { page: '1', q: 'a b' } },
          headers: { content: { 'X-Trace': 't1' } },
        },
      },
    };
    const cfg = mapNodeToDslConfig(node, restApiCallMappingSpec);
    expect(cfg.requestMethod).toBe('POST');
    expect(cfg.restEndpointUrlPattern).toBe(
      'https://api.example/items?page=1&q=' + encodeURIComponent('a b')
    );
    expect(cfg.params).toEqual({ page: '1', q: 'a b' });
    expect(cfg.headers).toEqual({ 'X-Trace': 't1' });
  });

  it('toDSL：pattern 已含 ? 时用 & 追加 query', () => {
    const node = {
      data: {
        inputsValues: {
          url: { content: 'https://h/x?fixed=1' },
          requestMethod: { content: 'GET' },
          params: { content: { a: '2' } },
          headers: { content: {} },
        },
      },
    };
    const cfg = mapNodeToDslConfig(node, restApiCallMappingSpec);
    expect(cfg.restEndpointUrlPattern).toBe('https://h/x?fixed=1&a=2');
    expect(cfg.headers).toBeUndefined();
  });

  it('toDSL：无 URL 时不写入 restEndpointUrlPattern，仍可导出 params', () => {
    const node = {
      data: {
        inputsValues: {
          url: { content: '' },
          requestMethod: { content: 'GET' },
          params: { content: { orphan: '1' } },
          headers: { content: {} },
        },
      },
    };
    const cfg = mapNodeToDslConfig(node, restApiCallMappingSpec);
    expect(cfg.restEndpointUrlPattern).toBeUndefined();
    expect(cfg.params).toEqual({ orphan: '1' });
    expect(cfg.headers).toBeUndefined();
  });

  it('toDSL：空 params / 空 headers 不写入 configuration 键（对齐旧导出）', () => {
    const node = {
      data: {
        inputsValues: {
          url: { content: 'https://only-path' },
          requestMethod: { content: 'GET' },
        },
      },
    };
    const cfg = mapNodeToDslConfig(node, restApiCallMappingSpec);
    expect(cfg.params).toBeUndefined();
    expect(cfg.headers).toBeUndefined();
    expect(cfg.restEndpointUrlPattern).toBe('https://only-path');
  });

  it('fromDSL：拆 query 并与 cfg.params 合并，cfg.params 同键覆盖 URL', () => {
    const iv = mapDslToNodeInputsValues(
      {
        restEndpointUrlPattern: 'https://x/p?a=fromUrl&b=2',
        params: { a: 'fromCfg', c: '3' },
        headers: {},
        requestMethod: 'PUT',
      } as Record<string, unknown>,
      restApiCallMappingSpec
    );
    expect(iv.url?.content).toBe('https://x/p');
    expect(iv.params?.content).toEqual({ a: 'fromCfg', b: '2', c: '3' });
    expect(iv.requestMethod?.content).toBe('PUT');
  });

  it('fromDSL：仅 URL 带 query、无 params 键时仍能回填 params', () => {
    const iv = mapDslToNodeInputsValues(
      {
        restEndpointUrlPattern: 'https://x?q=1',
        requestMethod: 'GET',
      } as Record<string, unknown>,
      restApiCallMappingSpec
    );
    expect(iv.url?.content).toBe('https://x');
    expect(iv.params?.content).toEqual({ q: '1' });
  });

  it('round-trip：URL + params -> DSL -> inputsValues 恢复基址与合并后的 params', () => {
    const node = {
      data: {
        inputsValues: {
          url: { content: 'http://svc/r' },
          requestMethod: { content: 'GET' },
          params: { content: { u: 'v' } },
          headers: { content: {} },
        },
      },
    };
    const cfg = mapNodeToDslConfig(node, restApiCallMappingSpec);
    expect(cfg.headers).toBeUndefined();
    const iv = mapDslToNodeInputsValues(cfg as Record<string, unknown>, restApiCallMappingSpec);
    expect(iv.url?.content).toBe('http://svc/r');
    expect(iv.params?.content).toEqual({ u: 'v' });
    expect(iv.requestMethod?.content).toBe('GET');
  });

  it('rulechain-builder：restApiCall DSL → 文档 → 再导出不丢 URL / query / params', () => {
    const rc = {
      ruleChain: {
        id: 'chain-rt',
        name: 'RT',
        debugMode: false,
        root: true,
        disabled: false,
      },
      metadata: {
        firstNodeIndex: 0,
        nodes: [
          {
            id: 'node-start',
            type: 'start',
            name: 'Start',
            debugMode: false,
            configuration: {},
          },
          {
            id: 'node-http',
            type: 'restApiCall',
            name: 'Http',
            debugMode: false,
            configuration: {
              requestMethod: 'GET',
              restEndpointUrlPattern: 'https://svc.example/api/items?q=fromUrl',
              params: { p: 'fromCfg' },
            },
          },
        ],
        connections: [{ fromId: 'node-start', toId: 'node-http', type: 'Success' }],
        ruleChainConnections: [],
      },
    };

    const flow = buildDocumentFromRuleChainJSON(rc as any);
    const doc = {
      toJSON: () => ({
        id: rc.ruleChain.id,
        name: rc.ruleChain.name,
        nodes: flow.nodes,
        edges: flow.edges ?? [],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: 'chain-rt' });
    const back = JSON.parse(json) as any;
    const httpNode = back.metadata.nodes.find((n: any) => n.type === 'restApiCall');
    expect(httpNode).toBeTruthy();
    const c = httpNode.configuration;
    expect(String(c.restEndpointUrlPattern)).toContain('https://svc.example/api/items');
    expect(String(c.restEndpointUrlPattern)).toMatch(/[?&]q=fromUrl/);
    expect(String(c.restEndpointUrlPattern)).toMatch(/[?&]p=fromCfg/);
    expect(c.params).toMatchObject({ q: 'fromUrl', p: 'fromCfg' });
  });

  it('rulechain-builder：无 params.properties 时仍从 paramsValues 导出到 DSL', () => {
    const doc = {
      toJSON: () => ({
        id: 'chain-fb',
        name: 'FB',
        nodes: [
          {
            id: 'st',
            type: 'start',
            meta: { position: { x: 0, y: 0 } },
            data: { title: 'S' },
          },
          {
            id: 'rx',
            type: 'restApiCall',
            meta: { position: { x: 100, y: 0 } },
            data: {
              title: 'X',
              positionType: 'middle',
              api: {
                method: 'GET',
                url: { type: 'template', content: 'https://a.example/b' },
              },
              params: {},
              paramsValues: { foo: { type: 'constant', content: 'bar' } },
              headers: {},
              headersValues: {},
              body: { bodyType: 'JSON' },
              timeout: { retryTimes: 0, timeout: 0 },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'rx', sourcePortID: 'Success' }],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: 'chain-fb' });
    const cfg = JSON.parse(json).metadata.nodes.find(
      (n: any) => n.type === 'restApiCall'
    ).configuration;
    expect(cfg.params).toEqual({ foo: 'bar' });
    expect(String(cfg.restEndpointUrlPattern)).toContain('foo=' + encodeURIComponent('bar'));
  });

  it('rulechain-builder：Failure 连线回放时恢复到上部输入端口', () => {
    const rc = {
      ruleChain: {
        id: 'chain-failure-top-input',
        name: 'FailureTopInput',
        debugMode: false,
        root: true,
        disabled: false,
      },
      metadata: {
        firstNodeIndex: 0,
        nodes: [
          { id: 'st', type: 'start', name: 'Start', debugMode: false, configuration: {} },
          {
            id: 'cc',
            type: 'x/cursorCli',
            name: 'CursorCli',
            debugMode: false,
            configuration: {},
          },
          { id: 'lg', type: 'log', name: 'Log', debugMode: false, configuration: {} },
        ],
        connections: [
          { fromId: 'st', toId: 'cc', type: 'Success' },
          { fromId: 'cc', toId: 'lg', type: 'Failure' },
        ],
      },
    };

    const flow = buildDocumentFromRuleChainJSON(rc as any) as any;
    const failureEdge = (flow.edges ?? []).find(
      (e: any) => e.sourceNodeID === 'cc' && e.targetNodeID === 'lg'
    );
    expect(failureEdge).toBeTruthy();
    expect(failureEdge.targetPortID).toBe('input_top');
  });

  it('rulechain-builder：Else / False 分支回放时恢复到上部输入端口', () => {
    const rc = {
      ruleChain: {
        id: 'chain-else-false-top-input',
        name: 'ElseFalseTopInput',
        debugMode: false,
        root: true,
        disabled: false,
      },
      metadata: {
        firstNodeIndex: 0,
        nodes: [
          { id: 'st', type: 'start', name: 'Start', debugMode: false, configuration: {} },
          {
            id: 'cond',
            type: 'condition',
            name: 'Condition',
            debugMode: false,
            configuration: {},
          },
          {
            id: 'filter',
            type: 'jsFilter',
            name: 'Filter',
            debugMode: false,
            configuration: {},
          },
          { id: 'log_else', type: 'log', name: 'LogElse', debugMode: false, configuration: {} },
          {
            id: 'log_false',
            type: 'log',
            name: 'LogFalse',
            debugMode: false,
            configuration: {},
          },
        ],
        connections: [
          { fromId: 'st', toId: 'cond', type: 'Success' },
          { fromId: 'cond', toId: 'log_else', type: 'else' },
          { fromId: 'st', toId: 'filter', type: 'Success' },
          { fromId: 'filter', toId: 'log_false', type: 'False' },
        ],
      },
    };

    const flow = buildDocumentFromRuleChainJSON(rc as any) as any;
    const elseEdge = (flow.edges ?? []).find(
      (e: any) => e.sourceNodeID === 'cond' && e.targetNodeID === 'log_else'
    );
    const falseEdge = (flow.edges ?? []).find(
      (e: any) => e.sourceNodeID === 'filter' && e.targetNodeID === 'log_false'
    );

    expect(elseEdge).toBeTruthy();
    expect(elseEdge.targetPortID).toBe('input_top');
    expect(falseEdge).toBeTruthy();
    expect(falseEdge.targetPortID).toBe('input_top');
  });
});

describe('remaining node specs round-trip', () => {
  it('dbClient / redisClient / flow / multiNodeOutput / yapi mappings', () => {
    const dbCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            driverName: { content: 'postgres' },
            dsn: { content: 'postgres://x' },
            sql: { content: 'select 1' },
            params: { content: ['1'] },
            getOne: { content: true },
            poolSize: { content: 4 },
          },
        },
      },
      dbClientMappingSpec
    );
    expect(dbCfg).toMatchObject({
      driverName: 'postgres',
      dsn: 'postgres://x',
      sql: 'select 1',
      params: ['1'],
      getOne: true,
      poolSize: 4,
    });
    const dbIv = mapDslToNodeInputsValues(dbCfg as Record<string, unknown>, dbClientMappingSpec);
    expect(dbIv.driverName?.content).toBe('postgres');
    expect(dbIv.params?.content).toEqual(['1']);

    const redisCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            server: { content: 'redis://x:6379' },
            password: { content: 'pwd' },
            poolSize: { content: 8 },
            db: { content: 2 },
            cmd: { content: 'SET a b' },
            params: { content: ['a', 'b'] },
          },
        },
      },
      redisClientMappingSpec
    );
    expect(redisCfg).toMatchObject({
      server: 'redis://x:6379',
      password: 'pwd',
      poolSize: 8,
      db: 2,
      cmd: 'SET a b',
      params: ['a', 'b'],
    });
    const redisIv = mapDslToNodeInputsValues(
      redisCfg as Record<string, unknown>,
      redisClientMappingSpec
    );
    expect(redisIv.db?.content).toBe(2);

    const osCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            endpoint: { content: 'https://opensearch.example.com:9200' },
            index: { content: 'logs-*' },
            username: { content: 'u' },
            password: { content: 'p' },
            insecureSkipVerify: { content: true },
            timeoutSec: { content: 30 },
            searchType: { content: 'dfs_query_then_fetch' },
            ignoreUnavailable: { content: true },
            defaultSearchBody: { content: '{"size":10,"query":{"match_all":{}}}' },
          },
        },
      },
      opensearchSearchMappingSpec
    );
    expect(osCfg).toMatchObject({
      endpoint: 'https://opensearch.example.com:9200',
      index: 'logs-*',
      username: 'u',
      password: 'p',
      insecureSkipVerify: true,
      timeoutSec: 30,
      searchType: 'dfs_query_then_fetch',
      ignoreUnavailable: true,
      defaultSearchBody: '{"size":10,"query":{"match_all":{}}}',
    });
    const osIv = mapDslToNodeInputsValues(
      osCfg as Record<string, unknown>,
      opensearchSearchMappingSpec
    );
    expect(osIv.searchType?.content).toBe('dfs_query_then_fetch');
    expect(osIv.ignoreUnavailable?.content).toBe(true);

    const tlsCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            endpoint: { content: 'https://tls.cn-beijing.volces.com' },
            region: { content: 'cn-beijing' },
            accessKeyId: { content: 'ak' },
            secretAccessKey: { content: 'sk' },
            sessionToken: { content: 'st' },
            topicId: { content: 'topic-1' },
            defaultQuery: { content: '__source__:*error*' },
            limit: { content: 50 },
            useApiV3: { content: true },
            timeoutSec: { content: 20 },
            timeRangePreset: { content: 'last_1h' },
            defaultStartTimeMs: { content: 1000 },
            defaultEndTimeMs: { content: 2000 },
            defaultSort: { content: 'asc' },
            highLight: { content: true },
          },
        },
      },
      volcTlsSearchLogsMappingSpec
    );
    expect(tlsCfg).toMatchObject({
      endpoint: 'https://tls.cn-beijing.volces.com',
      region: 'cn-beijing',
      accessKeyId: 'ak',
      secretAccessKey: 'sk',
      sessionToken: 'st',
      topicId: 'topic-1',
      defaultQuery: '__source__:*error*',
      limit: 50,
      useApiV3: true,
      timeoutSec: 20,
      timeRangePreset: 'last_1h',
      defaultStartTimeMs: 1000,
      defaultEndTimeMs: 2000,
      defaultSort: 'asc',
      highLight: true,
    });
    const tlsIv = mapDslToNodeInputsValues(
      tlsCfg as Record<string, unknown>,
      volcTlsSearchLogsMappingSpec
    );
    expect(tlsIv.useApiV3?.content).toBe(true);
    expect(tlsIv.timeRangePreset?.content).toBe('last_1h');
    expect(tlsIv.defaultSort?.content).toBe('asc');

    const cursorCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            agentPath: { content: '/opt/homebrew/bin/agent' },
            args: { content: [] },
            printMode: { content: true },
            outputFormat: { content: 'json' },
            prompt: { content: 'find and fix performance issues' },
            model: { content: 'gpt-5.2' },
            workspacePath: { content: '/data/repo' },
            log: { content: true },
            replaceData: { content: false },
            workDir: { content: '/tmp/wd' },
            timeoutMs: { content: 5000 },
          },
        },
      },
      cursorCliMappingSpec
    );
    expect(cursorCfg).toMatchObject({
      agentPath: '/opt/homebrew/bin/agent',
      args: [],
      printMode: true,
      outputFormat: 'json',
      prompt: 'find and fix performance issues',
      model: 'gpt-5.2',
      workspacePath: '/data/repo',
      log: true,
      replaceData: false,
      workDir: '/tmp/wd',
      timeoutMs: 5000,
    });
    const cursorIv = mapDslToNodeInputsValues(
      cursorCfg as Record<string, unknown>,
      cursorCliMappingSpec
    );
    expect(cursorIv.args?.content).toEqual([]);
    expect(cursorIv.outputFormat?.content).toBe('json');
    expect(cursorIv.printMode?.content).toBe(true);
    expect(cursorIv.prompt?.content).toBe('find and fix performance issues');
    expect(cursorIv.model?.content).toBe('gpt-5.2');
    expect(cursorIv.replaceData?.content).toBe(false);

    const cursorAuthCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            agentPath: { content: 'agent' },
            workspacePath: { content: '/repo/root' },
            worktree: { content: true },
            force: { content: false },
            workDir: { content: '/tmp/wd' },
            timeoutMs: { content: 12000 },
            replaceData: { content: true },
          },
        },
      },
      cursorCliAuthMappingSpec
    );
    expect(cursorAuthCfg).toMatchObject({
      agentPath: 'agent',
      workspacePath: '/repo/root',
      worktree: true,
      force: false,
      workDir: '/tmp/wd',
      timeoutMs: 12000,
      replaceData: true,
    });
    const cursorAuthIv = mapDslToNodeInputsValues(
      cursorAuthCfg as Record<string, unknown>,
      cursorCliAuthMappingSpec
    );
    expect(cursorAuthIv.workspacePath?.content).toBe('/repo/root');
    expect(cursorAuthIv.force?.content).toBe(false);

    const legacyIv = mapDslToNodeInputsValues(
      {
        cursorPath: '/legacy/bin/cursor',
        args: ['x'],
      } as Record<string, unknown>,
      cursorCliMappingSpec
    );
    expect(legacyIv.agentPath?.content).toBe('/legacy/bin/cursor');

    const acpCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            agentPath: { content: 'agent' },
            args: { content: ['acp'] },
            stdinLines: { content: ['{"jsonrpc":"2.0","id":1,"method":"ping"}'] },
            workspacePath: { content: '/ws' },
            log: { content: false },
            replaceData: { content: true },
            workDir: { content: '' },
            timeoutMs: { content: 90000 },
          },
        },
      },
      cursorAcpMappingSpec
    );
    expect(acpCfg).toMatchObject({
      agentPath: 'agent',
      args: ['acp'],
      stdinLines: ['{"jsonrpc":"2.0","id":1,"method":"ping"}'],
      workspacePath: '/ws',
      timeoutMs: 90000,
    });
    const acpIv = mapDslToNodeInputsValues(acpCfg as Record<string, unknown>, cursorAcpMappingSpec);
    expect(acpIv.stdinLines?.content).toEqual(['{"jsonrpc":"2.0","id":1,"method":"ping"}']);

    const fwCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            msgType: { content: 'post' },
            webhookUrl: { content: 'https://open.feishu.cn/open-apis/bot/v2/hook/x' },
            text: { content: 'hello ${msg.id}' },
            postTitle: { content: 'T' },
            postBody: { content: 'B' },
            postLang: { content: 'en_us' },
            postSplitByLine: { content: true },
            postAtAllBefore: { content: true },
            postAtAllAfter: { content: false },
            postMentionUserIds: { content: ['ou_a', 'ou_b'] },
            interactivePreset: { content: 'notice_card' },
            cardNoticeTitle: { content: 'N' },
            cardNoticeMarkdown: { content: 'M' },
            cardJson: { content: '{}' },
            rawJson: { content: '{}' },
            timeoutMs: { content: 8000 },
            replaceData: { content: true },
          },
        },
      },
      feishuWebhookMappingSpec
    );
    expect(fwCfg).toMatchObject({
      msgType: 'post',
      webhookUrl: 'https://open.feishu.cn/open-apis/bot/v2/hook/x',
      text: 'hello ${msg.id}',
      postTitle: 'T',
      postBody: 'B',
      postLang: 'en_us',
      postSplitByLine: true,
      postAtAllBefore: true,
      postAtAllAfter: false,
      postMentionUserIds: ['ou_a', 'ou_b'],
      interactivePreset: 'notice_card',
      cardNoticeTitle: 'N',
      cardNoticeMarkdown: 'M',
      cardJson: '{}',
      rawJson: '{}',
      timeoutMs: 8000,
      replaceData: true,
    });
    const fwIv = mapDslToNodeInputsValues(
      fwCfg as Record<string, unknown>,
      feishuWebhookMappingSpec
    );
    expect(fwIv.msgType?.content).toBe('post');
    expect(fwIv.webhookUrl?.content).toBe('https://open.feishu.cn/open-apis/bot/v2/hook/x');
    expect(fwIv.text?.content).toBe('hello ${msg.id}');
    expect(fwIv.postLang?.content).toBe('en_us');
    expect(fwIv.postSplitByLine?.content).toBe(true);
    expect(fwIv.postMentionUserIds?.content).toEqual(['ou_a', 'ou_b']);
    expect(fwIv.interactivePreset?.content).toBe('notice_card');
    expect(fwIv.timeoutMs?.content).toBe(8000);
    expect(fwIv.replaceData?.content).toBe(true);

    const fsAuthCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            cliPath: { content: 'lark-cli' },
            args: { content: ['auth', 'status'] },
            workDir: { content: '/app' },
            timeoutMs: { content: 9000 },
            replaceData: { content: false },
          },
        },
      },
      feishuCliAuthMappingSpec
    );
    expect(fsAuthCfg).toMatchObject({
      cliPath: 'lark-cli',
      args: ['auth', 'status'],
      workDir: '/app',
      timeoutMs: 9000,
      replaceData: false,
    });
    const fsAuthIv = mapDslToNodeInputsValues(
      fsAuthCfg as Record<string, unknown>,
      feishuCliAuthMappingSpec
    );
    expect(fsAuthIv.cliPath?.content).toBe('lark-cli');
    expect(fsAuthIv.timeoutMs?.content).toBe(9000);

    const wsSyncCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            workspaceId: { content: 'ws-uuid-1' },
            replaceData: { content: false },
          },
        },
      },
      workspaceSyncMappingSpec
    );
    expect(wsSyncCfg).toMatchObject({
      workspaceId: 'ws-uuid-1',
      replaceData: false,
    });
    const wsSyncIv = mapDslToNodeInputsValues(
      wsSyncCfg as Record<string, unknown>,
      workspaceSyncMappingSpec
    );
    expect(wsSyncIv.workspaceId?.content).toBe('ws-uuid-1');
    expect(wsSyncIv.replaceData?.content).toBe(false);

    const fwLegacyIv = mapDslToNodeInputsValues(
      {
        webhookUrl: 'https://open.feishu.cn/open-apis/bot/v2/hook/legacy',
        text: 'legacy',
      } as Record<string, unknown>,
      feishuWebhookMappingSpec
    );
    expect(fwLegacyIv.msgType?.content).toBe('text');
    expect(fwLegacyIv.postLang?.content).toBe('zh_cn');
    expect(fwLegacyIv.interactivePreset?.content).toBe('card_json');
    expect(fwLegacyIv.postMentionUserIds?.content).toEqual([]);

    const flowCfg = mapNodeToDslConfig(
      { data: { inputsValues: { targetId: { content: 'sub-1' }, extend: { content: true } } } },
      flowMappingSpec
    );
    expect(flowCfg).toMatchObject({ targetId: 'sub-1', extend: true });
    const flowIv = mapDslToNodeInputsValues(flowCfg as Record<string, unknown>, flowMappingSpec);
    expect(flowIv.extend?.content).toBe(true);

    const multiCfg = mapNodeToDslConfig(
      { data: { inputsValues: { nodeIds: { content: ['n1', 'n2'] } } } },
      multiNodeOutputMappingSpec
    );
    expect(multiCfg.nodeIds).toEqual(['n1', 'n2']);
    const multiIv = mapDslToNodeInputsValues(
      multiCfg as Record<string, unknown>,
      multiNodeOutputMappingSpec
    );
    expect(multiIv.nodeIds?.content).toEqual(['n1', 'n2']);

    const yapiCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            baseUrl: { content: 'https://yapi.xx' },
            userName: { content: 'u' },
            password: { content: 'p' },
            interfacePath: { content: '/api/demo' },
            loginType: { content: 'normal' },
          },
        },
      },
      yapiMappingSpec
    );
    expect(yapiCfg).toMatchObject({
      baseUrl: 'https://yapi.xx',
      userName: 'u',
      password: 'p',
      interfacePath: '/api/demo',
      loginType: 'normal',
    });
  });

  it('switch mapping keeps expression groups round-trip', () => {
    const switchCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            cases: {
              content: [
                {
                  key: 'A',
                  groups: [
                    {
                      operator: 'and',
                      rows: [
                        {
                          type: 'expression',
                          left: { content: 'msg.temp' },
                          operator: '>',
                          right: { content: 10 },
                        },
                      ],
                    },
                  ],
                },
              ],
            },
          },
        },
      },
      switchMappingSpec
    );
    expect(switchCfg.cases).toEqual([{ case: 'msg.temp > 10', then: 'A' }]);
    const switchIv = mapDslToNodeInputsValues(
      switchCfg as Record<string, unknown>,
      switchMappingSpec
    );
    const cases = (switchIv.cases?.content as any[]) || [];
    expect(cases[0].key).toBe('A');
    expect(cases[0].groups?.[0]?.rows?.[0]?.operator).toBe('>');
  });

  it('script node specs map body fields to jsScript/luaScript keys', () => {
    const jsCfg = mapNodeToDslConfig(
      { data: { inputsValues: { scriptBody: { content: 'return msg;' } } } },
      jsTransformMappingSpec
    );
    expect(jsCfg.jsScript).toBe('return msg;');
    const logCfg = mapNodeToDslConfig(
      { data: { inputsValues: { scriptBody: { content: 'return "x";' } } } },
      logMappingSpec
    );
    expect(logCfg.jsScript).toBe('return "x";');
    const filterCfg = mapNodeToDslConfig(
      { data: { inputsValues: { scriptBody: { content: 'return true;' } } } },
      jsFilterMappingSpec
    );
    expect(filterCfg.jsScript).toBe('return true;');
    const luaCfg = mapNodeToDslConfig(
      { data: { inputsValues: { scriptBody: { content: 'return msg' } } } },
      luaTransformMappingSpec
    );
    expect(luaCfg.luaScript).toBe('return msg');
  });

  it('inclusive mapping uses same cases shape as switch', () => {
    const inclusiveCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            cases: {
              content: [
                {
                  key: 'C1',
                  groups: [
                    {
                      operator: 'and',
                      rows: [{ type: 'expression', content: 'msg.ok == true' }],
                    },
                  ],
                },
              ],
            },
          },
        },
      },
      inclusiveMappingSpec
    );
    expect(inclusiveCfg.cases).toEqual([{ case: 'msg.ok == true', then: 'C1' }]);
  });

  it('while / exec map bidirectionally via inputsValues', () => {
    const whileCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            condition: { content: 'msg.n < 3', type: 'template' },
            do: { content: 'node_a', type: 'constant' },
          },
        },
      },
      whileMappingSpec
    );
    expect(whileCfg.condition).toBe('msg.n < 3');
    expect(whileCfg.do).toBe('node_a');
    const ivW = mapDslToNodeInputsValues(whileCfg as Record<string, unknown>, whileMappingSpec);
    expect(ivW.condition?.content).toBe('msg.n < 3');
    expect(ivW.do?.content).toBe('node_a');

    const whileChainCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            condition: { content: 'msg.ok', type: 'template' },
            do: { content: 'chain:sub_chain_1', type: 'constant' },
          },
        },
      },
      whileMappingSpec
    );
    expect(whileChainCfg.do).toBe('chain:sub_chain_1');

    const execCfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            cmd: { content: 'echo', type: 'template' },
            args: { content: ['hi'], type: 'constant' },
            log: { content: true, type: 'constant' },
            replaceData: { content: false, type: 'constant' },
          },
        },
      },
      execMappingSpec
    );
    expect(execCfg.cmd).toBe('echo');
    expect(execCfg.args).toEqual(['hi']);
  });
});

describe('structure nodes: rulechain round-trip (for, then endpoint/schedule)', () => {
  it('for：range / do / mode、子图 blocks 与 inner edges 在 文档→RuleChain→文档 后保持一致', () => {
    const chainId = 'chain-for-rt';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'ForRoundTrip',
        nodes: [
          {
            id: 'st',
            type: 'start',
            meta: { position: { x: 100, y: 100 } },
            data: { title: 'S' },
          },
          {
            id: 'for_loop',
            type: 'for',
            meta: { position: { x: 200, y: 100 } },
            data: {
              title: '循环',
              positionType: 'middle',
              note: { type: 'constant', content: 'i in items' },
              nodeId: { type: 'constant', content: 'inner_flow' },
              operationMode: { type: 'constant', content: 2 },
            },
            blocks: [
              {
                id: 'block_start_rt1',
                type: 'block-start',
                meta: { position: { x: 0, y: 0 } },
                data: { positionType: 'middle' },
              },
              {
                id: 'inner_flow',
                type: 'flow',
                meta: { position: { x: 50, y: 0 } },
                data: {
                  title: 'Sub',
                  positionType: 'middle',
                  inputsValues: {
                    targetId: { type: 'constant', content: 'sub-chain-1' },
                    extend: { type: 'constant', content: true },
                  },
                },
              },
              {
                id: 'block_end_rt1',
                type: 'block-end',
                meta: { position: { x: 150, y: 0 } },
                data: { positionType: 'middle' },
              },
            ],
            edges: [
              { sourceNodeID: 'block_start_rt1', targetNodeID: 'inner_flow' },
              { sourceNodeID: 'inner_flow', targetNodeID: 'block_end_rt1' },
            ],
          },
          {
            id: 'log_tail',
            type: 'log',
            meta: { position: { x: 500, y: 100 } },
            data: {
              title: 'Tail',
              positionType: 'middle',
              script: {
                language: 'javascript',
                content:
                  '// 函数签名不可修改\nasync function ToString(msg, metadata, msgType, dataType) {\nreturn msg;\n}',
              },
            },
          },
        ],
        edges: [
          { sourceNodeID: 'st', targetNodeID: 'for_loop', sourcePortID: 'Success' },
          { sourceNodeID: 'for_loop', targetNodeID: 'log_tail', sourcePortID: 'Success' },
        ],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const forMeta = parsed.metadata.nodes.find((n: any) => n.type === 'for');
    expect(forMeta).toBeTruthy();
    expect(forMeta.configuration.range).toBe('i in items');
    expect(forMeta.configuration.do).toBe('inner_flow');
    expect(forMeta.configuration.mode).toBe(2);
    expect(forMeta.configuration.extra?.blocks).toHaveLength(3);
    expect(forMeta.configuration.extra?.edges).toHaveLength(2);

    const flowMeta = parsed.metadata.nodes.find((n: any) => n.id === 'inner_flow');
    expect(flowMeta?.configuration?.targetId).toBe('sub-chain-1');
    expect(flowMeta?.configuration?.extend).toBe(true);

    const flowBack = buildDocumentFromRuleChainJSON(parsed);
    const forNode = flowBack.nodes.find((n: any) => n.type === 'for');
    expect(forNode).toBeTruthy();
    expect(forNode!.data.note.content).toBe('i in items');
    expect(forNode!.data.nodeId.content).toBe('inner_flow');
    expect(forNode!.data.operationMode.content).toBe(2);
    const blockTypes = (forNode!.blocks ?? []).map((b: any) => String(b.type));
    expect(blockTypes).toEqual(['block-start', 'flow', 'block-end']);
    const inner = (forNode!.blocks ?? []).find((b: any) => b.id === 'inner_flow');
    expect(inner?.data?.inputsValues?.targetId?.content).toBe('sub-chain-1');
    expect(inner?.data?.inputsValues?.extend?.content).toBe(true);
    const innerEdges = forNode!.edges ?? [];
    expect(innerEdges).toHaveLength(2);
  });

  it('endpoint/schedule：cron 与指向首节点的边在 文档→RuleChain→文档 后保持一致（首节点须为 schedule 的 to 目标）', () => {
    const chainId = 'chain-ep-rt';
    const cronExpr = '0 0/5 * * * ?';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'EpRoundTrip',
        nodes: [
          {
            id: 'st_first',
            type: 'start',
            meta: { position: { x: 180, y: 180 } },
            data: { title: '入口' },
          },
          {
            id: 'log_a',
            type: 'log',
            meta: { position: { x: 400, y: 180 } },
            data: {
              title: 'L',
              positionType: 'middle',
              script: {
                language: 'javascript',
                content:
                  '// 函数签名不可修改\nasync function ToString(msg, metadata, msgType, dataType) {\nreturn msg;\n}',
              },
            },
          },
          {
            id: 'cron_ep',
            type: 'endpoint/schedule',
            meta: { position: { x: -260, y: 180 } },
            data: {
              title: '定时',
              positionType: 'header',
              inputsValues: {
                cron: { type: 'constant', content: cronExpr },
              },
              inputs: {
                type: 'object',
                required: ['cron'],
                properties: {
                  cron: {
                    type: 'string',
                    extra: {
                      label: 'Cron 表达式',
                      formComponent: 'cron-editor',
                    },
                  },
                },
              },
            },
          },
        ],
        edges: [
          { sourceNodeID: 'cron_ep', targetNodeID: 'st_first', sourcePortID: 'Success' },
          { sourceNodeID: 'st_first', targetNodeID: 'log_a', sourcePortID: 'Success' },
        ],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const eps = parsed.metadata.endpoints ?? [];
    expect(eps.length).toBeGreaterThanOrEqual(1);
    const ep = eps.find((e: any) => String(e.type) === 'endpoint/schedule');
    expect(ep).toBeTruthy();
    expect(String(ep.routers?.[0]?.from?.path ?? '')).toBe(cronExpr);
    expect(String(ep.routers?.[0]?.to?.path ?? '')).toBe(`${chainId}:st_first`);

    const flowBack = buildDocumentFromRuleChainJSON(parsed);
    const cronNode = flowBack.nodes.find((n: any) => n.type === 'endpoint/schedule');
    expect(cronNode).toBeTruthy();
    const cronContent = (cronNode as any)?.data?.inputsValues?.cron?.content;
    expect(cronContent).toBe(cronExpr);

    const edgeToStart = (flowBack.edges ?? []).find(
      (e: any) => e.sourceNodeID === 'cron_ep' && e.targetNodeID === 'st_first'
    );
    expect(edgeToStart).toBeTruthy();
  });

  it('x/cursorCli：文档→RuleChain→文档 round-trip 保持 configuration 与 inputsValues', () => {
    const chainId = 'chain-cursor-rt';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'CursorCliRT',
        nodes: [
          {
            id: 'st',
            type: 'start',
            meta: { position: { x: 0, y: 0 } },
            data: { title: 'S' },
          },
          {
            id: 'cc1',
            type: 'x/cursorCli',
            meta: { position: { x: 200, y: 0 } },
            data: {
              title: 'RunCursor',
              positionType: 'middle',
              inputsValues: {
                agentPath: { type: 'constant', content: 'agent' },
                printMode: { type: 'constant', content: false },
                prompt: { type: 'template', content: '' },
                model: { type: 'template', content: '' },
                outputFormat: { type: 'constant', content: 'text' },
                args: { type: 'constant', content: ['--version'] },
                workspacePath: { type: 'template', content: '/repo/root' },
                log: { type: 'constant', content: false },
                replaceData: { type: 'constant', content: true },
                workDir: { type: 'template', content: '${metadata.proj}' },
                timeoutMs: { type: 'constant', content: 3000 },
              },
              inputs: {
                type: 'object',
                required: ['agentPath', 'args'],
                properties: {
                  agentPath: { type: 'string' },
                  args: { type: 'array', items: { type: 'string' } },
                },
              },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'cc1', sourcePortID: 'Success' }],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const nodeMeta = parsed.metadata.nodes.find((n: any) => n.id === 'cc1');
    expect(nodeMeta?.type).toBe('x/cursorCli');
    expect(nodeMeta.configuration).toMatchObject({
      agentPath: 'agent',
      printMode: false,
      outputFormat: 'text',
      args: ['--version'],
      workspacePath: '/repo/root',
      log: false,
      replaceData: true,
      workDir: '${metadata.proj}',
      timeoutMs: 3000,
    });

    const back = buildDocumentFromRuleChainJSON(parsed);
    const cc = back.nodes.find((n: any) => n.id === 'cc1');
    expect(cc?.data?.inputsValues?.args?.content).toEqual(['--version']);
    expect(cc?.data?.inputsValues?.workspacePath?.content).toBe('/repo/root');
    expect(cc?.data?.inputsValues?.workDir?.content).toBe('${metadata.proj}');
    expect(cc?.data?.inputsValues?.timeoutMs?.content).toBe(3000);
    expect(cc?.data?.inputsValues?.replaceData?.content).toBe(true);
  });

  it('x/cursorAcp：文档→RuleChain→文档 round-trip 保持 configuration', () => {
    const chainId = 'chain-acp-rt';
    const initLine =
      '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{},"clientInfo":{"name":"t","version":"1"}}}';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'AcpRT',
        nodes: [
          {
            id: 'st',
            type: 'start',
            meta: { position: { x: 0, y: 0 } },
            data: { title: 'S' },
          },
          {
            id: 'acp1',
            type: 'x/cursorAcp',
            meta: { position: { x: 220, y: 0 } },
            data: {
              title: 'ACP',
              positionType: 'middle',
              inputsValues: {
                agentPath: { type: 'constant', content: 'agent' },
                args: { type: 'constant', content: ['acp'] },
                stdinLines: { type: 'constant', content: [initLine] },
                workspacePath: { type: 'template', content: '/repo/acp' },
                log: { type: 'constant', content: false },
                replaceData: { type: 'constant', content: true },
                workDir: { type: 'template', content: '' },
                timeoutMs: { type: 'constant', content: 60000 },
              },
              inputs: { type: 'object', required: ['stdinLines'], properties: {} },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'acp1', sourcePortID: 'Success' }],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const meta = parsed.metadata.nodes.find((n: any) => n.id === 'acp1');
    expect(meta?.type).toBe('x/cursorAcp');
    expect(meta.configuration.args).toEqual(['acp']);
    expect(meta.configuration.stdinLines).toEqual([initLine]);
    expect(meta.configuration.workspacePath).toBe('/repo/acp');

    const back = buildDocumentFromRuleChainJSON(parsed);
    const node = back.nodes.find((n: any) => n.id === 'acp1');
    expect((node as any)?.data?.inputsValues?.stdinLines?.content).toEqual([initLine]);
    expect((node as any)?.data?.inputsValues?.workspacePath?.content).toBe('/repo/acp');
    expect((node as any)?.data?.inputsValues?.timeoutMs?.content).toBe(60000);
  });

  it('x/cursorCliAuth：文档→RuleChain→文档 round-trip 保持 configuration', () => {
    const chainId = 'chain-cursor-auth-rt';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'CursorAuthRT',
        nodes: [
          { id: 'st', type: 'start', meta: { position: { x: 0, y: 0 } }, data: { title: 'S' } },
          {
            id: 'cca',
            type: 'x/cursorCliAuth',
            meta: { position: { x: 220, y: 0 } },
            data: {
              title: 'Auth',
              positionType: 'middle',
              inputsValues: {
                agentPath: { type: 'constant', content: 'agent' },
                workspacePath: { type: 'template', content: '/repo/auth' },
                worktree: { type: 'constant', content: true },
                force: { type: 'constant', content: false },
                workDir: { type: 'template', content: '/tmp/wd' },
                timeoutMs: { type: 'constant', content: 16000 },
                replaceData: { type: 'constant', content: true },
              },
              inputs: { type: 'object', properties: {} },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'cca', sourcePortID: 'Success' }],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const meta = parsed.metadata.nodes.find((n: any) => n.id === 'cca');
    expect(meta?.type).toBe('x/cursorCliAuth');
    expect(meta.configuration).toMatchObject({
      agentPath: 'agent',
      workspacePath: '/repo/auth',
      worktree: true,
      force: false,
      workDir: '/tmp/wd',
      timeoutMs: 16000,
      replaceData: true,
    });

    const back = buildDocumentFromRuleChainJSON(parsed);
    const node = back.nodes.find((n: any) => n.id === 'cca');
    expect((node as any)?.data?.inputsValues?.workspacePath?.content).toBe('/repo/auth');
    expect((node as any)?.data?.inputsValues?.worktree?.content).toBe(true);
    expect((node as any)?.data?.inputsValues?.force?.content).toBe(false);
  });

  it('x/feishuWebhook：文档→RuleChain→文档 round-trip 保持 configuration', () => {
    const chainId = 'chain-fs-rt';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'FsRT',
        nodes: [
          { id: 'st', type: 'start', meta: { position: { x: 0, y: 0 } }, data: { title: 'S' } },
          {
            id: 'f1',
            type: 'x/feishuWebhook',
            meta: { position: { x: 220, y: 0 } },
            data: {
              title: 'FH',
              positionType: 'middle',
              inputsValues: {
                msgType: { type: 'constant', content: 'text' },
                webhookUrl: {
                  type: 'template',
                  content: 'https://open.feishu.cn/open-apis/bot/v2/hook/abc',
                },
                text: { type: 'template', content: 'ping ${msg.type}' },
                postTitle: { type: 'template', content: '' },
                postBody: { type: 'template', content: '' },
                postLang: { type: 'constant', content: 'zh_cn' },
                postSplitByLine: { type: 'constant', content: false },
                postAtAllBefore: { type: 'constant', content: false },
                postAtAllAfter: { type: 'constant', content: false },
                postMentionUserIds: { type: 'constant', content: [] },
                interactivePreset: { type: 'constant', content: 'card_json' },
                cardNoticeTitle: { type: 'template', content: '' },
                cardNoticeMarkdown: { type: 'template', content: '' },
                cardJson: { type: 'template', content: '' },
                rawJson: { type: 'template', content: '' },
                timeoutMs: { type: 'constant', content: 12000 },
                replaceData: { type: 'constant', content: false },
              },
              inputs: { type: 'object', required: ['webhookUrl'], properties: {} },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'f1', sourcePortID: 'Success' }],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const meta = parsed.metadata.nodes.find((n: any) => n.id === 'f1');
    expect(meta?.type).toBe('x/feishuWebhook');
    expect(meta.configuration.webhookUrl).toBe('https://open.feishu.cn/open-apis/bot/v2/hook/abc');
    expect(meta.configuration.msgType).toBe('text');
    expect(meta.configuration.text).toBe('ping ${msg.type}');
    expect(meta.configuration.interactivePreset).toBe('card_json');
    expect(meta.configuration.timeoutMs).toBe(12000);
    expect(meta.configuration.replaceData).toBe(false);

    const back = buildDocumentFromRuleChainJSON(parsed);
    const node = back.nodes.find((n: any) => n.id === 'f1');
    expect((node as any)?.data?.inputsValues?.webhookUrl?.content).toBe(
      'https://open.feishu.cn/open-apis/bot/v2/hook/abc'
    );
    expect((node as any)?.data?.inputsValues?.msgType?.content).toBe('text');
    expect((node as any)?.data?.inputsValues?.text?.content).toBe('ping ${msg.type}');
    expect((node as any)?.data?.inputsValues?.interactivePreset?.content).toBe('card_json');
    expect((node as any)?.data?.inputsValues?.timeoutMs?.content).toBe(12000);
    expect((node as any)?.data?.inputsValues?.replaceData?.content).toBe(false);
  });

  it('x/feishuCliAuth：文档→RuleChain→文档 round-trip 保持 configuration', () => {
    const chainId = 'chain-feishu-auth-rt';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'FeishuAuthRT',
        nodes: [
          { id: 'st', type: 'start', meta: { position: { x: 0, y: 0 } }, data: { title: 'S' } },
          {
            id: 'fca',
            type: 'x/feishuCliAuth',
            meta: { position: { x: 220, y: 0 } },
            data: {
              title: 'Auth',
              positionType: 'middle',
              inputsValues: {
                cliPath: { type: 'constant', content: 'lark-cli' },
                args: { type: 'constant', content: ['auth', 'status'] },
                workDir: { type: 'template', content: '/app' },
                timeoutMs: { type: 'constant', content: 8000 },
                replaceData: { type: 'constant', content: false },
              },
              inputs: { type: 'object', properties: {} },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'fca', sourcePortID: 'Success' }],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const meta = parsed.metadata.nodes.find((n: any) => n.id === 'fca');
    expect(meta?.type).toBe('x/feishuCliAuth');
    expect(meta.configuration).toMatchObject({
      cliPath: 'lark-cli',
      args: ['auth', 'status'],
      workDir: '/app',
      timeoutMs: 8000,
      replaceData: false,
    });

    const back = buildDocumentFromRuleChainJSON(parsed);
    const node = back.nodes.find((n: any) => n.id === 'fca');
    expect((node as any)?.data?.inputsValues?.cliPath?.content).toBe('lark-cli');
    expect((node as any)?.data?.inputsValues?.args?.content).toEqual(['auth', 'status']);
    expect((node as any)?.data?.inputsValues?.replaceData?.content).toBe(false);
  });

  it('x/workspaceSync：文档→RuleChain→文档 round-trip 保持 configuration', () => {
    const chainId = 'chain-ws-sync-rt';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'WsSyncRT',
        nodes: [
          { id: 'st', type: 'start', meta: { position: { x: 0, y: 0 } }, data: { title: 'S' } },
          {
            id: 'ws',
            type: 'x/workspaceSync',
            meta: { position: { x: 220, y: 0 } },
            data: {
              title: 'Sync',
              positionType: 'middle',
              inputsValues: {
                workspaceId: { type: 'constant', content: 'abc-123-workspace' },
                replaceData: { type: 'constant', content: true },
              },
              inputs: { type: 'object', properties: {} },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'ws', sourcePortID: 'Success' }],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const meta = parsed.metadata.nodes.find((n: any) => n.id === 'ws');
    expect(meta?.type).toBe('x/workspaceSync');
    expect(meta.configuration).toMatchObject({
      workspaceId: 'abc-123-workspace',
      replaceData: true,
    });

    const back = buildDocumentFromRuleChainJSON(parsed);
    const node = back.nodes.find((n: any) => n.id === 'ws');
    expect((node as any)?.data?.inputsValues?.workspaceId?.content).toBe('abc-123-workspace');
    expect((node as any)?.data?.inputsValues?.replaceData?.content).toBe(true);
  });

  it('x/feishuWebhook post：文档→RuleChain→文档 round-trip', () => {
    const chainId = 'chain-fs-post';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'FsPost',
        nodes: [
          { id: 'st', type: 'start', meta: { position: { x: 0, y: 0 } }, data: { title: 'S' } },
          {
            id: 'fp',
            type: 'x/feishuWebhook',
            meta: { position: { x: 200, y: 0 } },
            data: {
              title: 'Post',
              positionType: 'middle',
              inputsValues: {
                msgType: { type: 'constant', content: 'post' },
                webhookUrl: {
                  type: 'template',
                  content: 'https://open.feishu.cn/open-apis/bot/v2/hook/posthook',
                },
                text: { type: 'template', content: '' },
                postTitle: { type: 'template', content: '告警 ${msg.type}' },
                postBody: { type: 'template', content: '详情见 metadata' },
                postLang: { type: 'constant', content: 'ja_jp' },
                postSplitByLine: { type: 'constant', content: true },
                postAtAllBefore: { type: 'constant', content: true },
                postAtAllAfter: { type: 'constant', content: false },
                postMentionUserIds: { type: 'constant', content: ['ou_x'] },
                interactivePreset: { type: 'constant', content: 'card_json' },
                cardNoticeTitle: { type: 'template', content: '' },
                cardNoticeMarkdown: { type: 'template', content: '' },
                cardJson: { type: 'template', content: '' },
                rawJson: { type: 'template', content: '' },
                timeoutMs: { type: 'constant', content: 10000 },
                replaceData: { type: 'constant', content: false },
              },
              inputs: { type: 'object', required: ['msgType'], properties: {} },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'fp', sourcePortID: 'Success' }],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const meta = parsed.metadata.nodes.find((n: any) => n.id === 'fp');
    expect(meta.configuration.msgType).toBe('post');
    expect(meta.configuration.postTitle).toBe('告警 ${msg.type}');
    expect(meta.configuration.postLang).toBe('ja_jp');
    expect(meta.configuration.postSplitByLine).toBe(true);
    expect(meta.configuration.postAtAllBefore).toBe(true);
    expect(meta.configuration.postMentionUserIds).toEqual(['ou_x']);

    const back = buildDocumentFromRuleChainJSON(parsed);
    const node = back.nodes.find((n: any) => n.id === 'fp');
    expect((node as any)?.data?.inputsValues?.msgType?.content).toBe('post');
    expect((node as any)?.data?.inputsValues?.postTitle?.content).toBe('告警 ${msg.type}');
    expect((node as any)?.data?.inputsValues?.postLang?.content).toBe('ja_jp');
    expect((node as any)?.data?.inputsValues?.postMentionUserIds?.content).toEqual(['ou_x']);
  });

  it('x/feishuWebhook interactive 通知卡片：round-trip', () => {
    const chainId = 'chain-fs-notice';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'FsNotice',
        nodes: [
          { id: 'st', type: 'start', meta: { position: { x: 0, y: 0 } }, data: { title: 'S' } },
          {
            id: 'fn',
            type: 'x/feishuWebhook',
            meta: { position: { x: 200, y: 0 } },
            data: {
              title: 'Notice',
              positionType: 'middle',
              inputsValues: {
                msgType: { type: 'constant', content: 'interactive' },
                webhookUrl: {
                  type: 'template',
                  content: 'https://open.feishu.cn/open-apis/bot/v2/hook/h',
                },
                text: { type: 'template', content: '' },
                postTitle: { type: 'template', content: '' },
                postBody: { type: 'template', content: '' },
                postLang: { type: 'constant', content: 'zh_cn' },
                postSplitByLine: { type: 'constant', content: false },
                postAtAllBefore: { type: 'constant', content: false },
                postAtAllAfter: { type: 'constant', content: false },
                postMentionUserIds: { type: 'constant', content: [] },
                interactivePreset: { type: 'constant', content: 'notice_card' },
                cardNoticeTitle: { type: 'template', content: '标题A' },
                cardNoticeMarkdown: { type: 'template', content: '**正文**' },
                cardJson: { type: 'template', content: '' },
                rawJson: { type: 'template', content: '' },
                timeoutMs: { type: 'constant', content: 15000 },
                replaceData: { type: 'constant', content: false },
              },
              inputs: { type: 'object', properties: {} },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'fn', sourcePortID: 'Success' }],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const meta = parsed.metadata.nodes.find((n: any) => n.id === 'fn');
    expect(meta.configuration.interactivePreset).toBe('notice_card');
    expect(meta.configuration.cardNoticeTitle).toBe('标题A');

    const back = buildDocumentFromRuleChainJSON(parsed);
    const node = back.nodes.find((n: any) => n.id === 'fn');
    expect((node as any)?.data?.inputsValues?.interactivePreset?.content).toBe('notice_card');
    expect((node as any)?.data?.inputsValues?.cardNoticeTitle?.content).toBe('标题A');
  });
});

describe('x/jsonExtract spec', () => {
  it('toDSL / fromDSL 与 RuleChain round-trip', () => {
    const chainId = 'chain-json-extract-rt';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'JeRT',
        nodes: [
          { id: 'st', type: 'start', meta: { position: { x: 0, y: 0 } }, data: { title: 'S' } },
          {
            id: 'je1',
            type: 'x/jsonExtract',
            meta: { position: { x: 200, y: 0 } },
            data: {
              title: 'Extract',
              positionType: 'middle',
              inputsValues: {
                source: { type: 'template', content: '${msg.data}' },
              },
              inputs: {
                type: 'object',
                required: ['source'],
                properties: {
                  source: { type: 'string' },
                },
              },
            },
          },
        ],
        edges: [{ sourceNodeID: 'st', targetNodeID: 'je1', sourcePortID: 'Success' }],
      }),
    } as any;

    const cfg = mapNodeToDslConfig(
      {
        data: {
          inputsValues: {
            source: { content: '${msg.data}' },
          },
        },
      },
      jsonExtractMappingSpec
    );
    expect(cfg.source).toBe('${msg.data}');

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const meta = parsed.metadata.nodes.find((n: any) => n.id === 'je1');
    expect(meta?.type).toBe('x/jsonExtract');
    expect(meta.configuration).toEqual({
      source: '${msg.data}',
    });

    const back = buildDocumentFromRuleChainJSON(parsed);
    const node = back.nodes.find((n: any) => n.id === 'je1');
    expect((node as any)?.data?.inputsValues?.source?.type).toBe('template');
    expect((node as any)?.data?.inputsValues?.source?.content).toBe('${msg.data}');
    expect((node as any)?.data?.inputs?.properties?.source).toBeTruthy();
    expect((node as any)?.data?.outputs?.properties?.extractedJson).toBeTruthy();
  });

  it('fromDSL：configuration 误含嵌套 inputsValues 时仍能拆出字段', () => {
    const iv = mapDslToNodeInputsValues(
      {
        title: 'bad',
        inputsValues: {
          source: { type: 'template', content: '${data}' },
        },
      } as Record<string, unknown>,
      jsonExtractMappingSpec
    );
    expect(iv.source?.content).toBe('${data}');
  });
});

describe('opensearch/tls nodes: rulechain round-trip', () => {
  it('opensearch/search + volcTls/searchLogs：文档→RuleChain→文档 保持关键 configuration', () => {
    const chainId = 'chain-search-rt';
    const doc = {
      toJSON: () => ({
        id: chainId,
        name: 'SearchRT',
        nodes: [
          { id: 'st', type: 'start', meta: { position: { x: 0, y: 0 } }, data: { title: 'S' } },
          {
            id: 'os1',
            type: 'opensearch/search',
            meta: { position: { x: 180, y: 0 } },
            data: {
              title: 'OS',
              positionType: 'middle',
              inputsValues: {
                endpoint: { type: 'template', content: 'https://os:9200' },
                index: { type: 'template', content: 'app-*' },
                username: { type: 'constant', content: 'u' },
                password: { type: 'constant', content: 'p' },
                insecureSkipVerify: { type: 'constant', content: true },
                timeoutSec: { type: 'constant', content: 25 },
                searchType: { type: 'constant', content: 'query_then_fetch' },
                ignoreUnavailable: { type: 'constant', content: true },
                defaultSearchBody: { type: 'template', content: '{"size":50}' },
              },
              inputs: { type: 'object', properties: {} },
            },
          },
          {
            id: 'tls1',
            type: 'volcTls/searchLogs',
            meta: { position: { x: 380, y: 0 } },
            data: {
              title: 'TLS',
              positionType: 'middle',
              inputsValues: {
                endpoint: { type: 'constant', content: 'https://tls.cn-beijing.volces.com' },
                region: { type: 'constant', content: 'cn-beijing' },
                accessKeyId: { type: 'template', content: 'ak' },
                secretAccessKey: { type: 'template', content: 'sk' },
                sessionToken: { type: 'template', content: '' },
                topicId: { type: 'template', content: 't-1' },
                defaultQuery: { type: 'template', content: '*' },
                limit: { type: 'constant', content: 80 },
                useApiV3: { type: 'constant', content: true },
                timeoutSec: { type: 'constant', content: 10 },
                timeRangePreset: { type: 'constant', content: 'last_6h' },
                defaultStartTimeMs: { type: 'constant', content: 0 },
                defaultEndTimeMs: { type: 'constant', content: 0 },
                defaultSort: { type: 'constant', content: 'desc' },
                highLight: { type: 'constant', content: false },
              },
              inputs: { type: 'object', properties: {} },
            },
          },
        ],
        edges: [
          { sourceNodeID: 'st', targetNodeID: 'os1', sourcePortID: 'Success' },
          { sourceNodeID: 'os1', targetNodeID: 'tls1', sourcePortID: 'Success' },
        ],
      }),
    } as any;

    const json = buildRuleChainJSONFromDocument(doc, { id: chainId });
    const parsed = JSON.parse(json) as any;
    const osMeta = parsed.metadata.nodes.find((n: any) => n.id === 'os1');
    expect(osMeta.type).toBe('opensearch/search');
    expect(osMeta.configuration).toMatchObject({
      endpoint: 'https://os:9200',
      index: 'app-*',
      timeoutSec: 25,
      ignoreUnavailable: true,
    });
    const tlsMeta = parsed.metadata.nodes.find((n: any) => n.id === 'tls1');
    expect(tlsMeta.type).toBe('volcTls/searchLogs');
    expect(tlsMeta.configuration).toMatchObject({
      region: 'cn-beijing',
      topicId: 't-1',
      useApiV3: true,
      timeRangePreset: 'last_6h',
      limit: 80,
    });

    const back = buildDocumentFromRuleChainJSON(parsed);
    const osNode = back.nodes.find((n: any) => n.id === 'os1');
    expect((osNode as any)?.data?.inputsValues?.index?.content).toBe('app-*');
    expect((osNode as any)?.data?.inputsValues?.timeoutSec?.content).toBe(25);
    const tlsNode = back.nodes.find((n: any) => n.id === 'tls1');
    expect((tlsNode as any)?.data?.inputsValues?.region?.content).toBe('cn-beijing');
    expect((tlsNode as any)?.data?.inputsValues?.useApiV3?.content).toBe(true);
    expect((tlsNode as any)?.data?.inputsValues?.timeRangePreset?.content).toBe('last_6h');
  });
});
