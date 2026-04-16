# DSL Mapping Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将前端节点 DSL 映射改造成单一配置源架构，并沉淀项目级 Skill，确保后续新增组件按统一规则实现。

**Architecture:** 在 `flowgram/src/utils/dsl-mapping/` 新增 `types + engine + specs + specializers` 四层结构，`rulechain-builder.ts` 改为“spec 命中走新引擎、未命中回退旧逻辑”。先迁移 `ai/agentHarness`、`ai/llm`、`restApiCall` 三个节点，再补充 round-trip 测试与项目级 Skill。

**Tech Stack:** TypeScript, Flowgram editor utils, Vitest（新增）, ESLint, pnpm/npm scripts

---

## File Structure

- Create: `flowgram/src/utils/dsl-mapping/types.ts`（映射类型定义）
- Create: `flowgram/src/utils/dsl-mapping/engine.ts`（通用映射引擎）
- Create: `flowgram/src/utils/dsl-mapping/specs.ts`（节点映射配置注册）
- Create: `flowgram/src/utils/dsl-mapping/specializers.ts`（节点特例转换）
- Modify: `flowgram/src/utils/rulechain-builder.ts`（接入映射引擎分发）
- Modify: `flowgram/package.json`（新增测试脚本）
- Create: `flowgram/vitest.config.ts`（测试配置）
- Create: `flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts`（节点映射单测）
- Create: `.cursor/skills/dsl-component-mapping/SKILL.md`（项目级 Skill）

---

### Task 1: 搭建 `dsl-mapping` 基础类型与引擎

**Files:**
- Create: `flowgram/src/utils/dsl-mapping/types.ts`
- Create: `flowgram/src/utils/dsl-mapping/engine.ts`
- Test: `flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts`

- [ ] **Step 1: 先写失败测试（引擎字段映射与默认值）**

```ts
import { describe, expect, it } from 'vitest';
import { mapNodeToDslConfig } from '../engine';
import type { NodeMappingSpec } from '../types';

describe('dsl mapping engine', () => {
  it('maps inputsValues to config with defaults', () => {
    const spec: NodeMappingSpec = {
      nodeType: 'ai/agentHarness',
      fields: [
        { nodePath: 'model', dslPath: 'model', valueType: 'template', defaultValue: '' },
        {
          nodePath: 'enableWorkspaceTools',
          dslPath: 'enableWorkspaceTools',
          valueType: 'boolean',
          defaultValue: false,
        },
      ],
    };
    const nodeData = {
      inputsValues: {
        model: { type: 'template', content: 'gpt-4.1' },
      },
    };
    const cfg = mapNodeToDslConfig(nodeData as any, spec);
    expect(cfg).toEqual({
      model: 'gpt-4.1',
      enableWorkspaceTools: false,
    });
  });
});
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd flowgram && npm run test:unit -- dsl-mapping`  
Expected: FAIL，报错 `Cannot find module '../engine'` 或 `mapNodeToDslConfig is not a function`

- [ ] **Step 3: 写最小实现（types + engine）**

```ts
// types.ts
export type MappingValueType = 'template' | 'constant' | 'string' | 'number' | 'boolean';

export interface MappingField {
  nodePath: string;
  dslPath: string;
  valueType: MappingValueType;
  defaultValue?: unknown;
}

export interface NodeMappingSpec {
  nodeType: string;
  fields: MappingField[];
  transformOut?: (cfg: Record<string, any>, nodeData: any) => Record<string, any>;
  transformIn?: (cfg: Record<string, any>, nodeData: any) => Record<string, any>;
}
```

```ts
// engine.ts
import type { MappingField, NodeMappingSpec } from './types';

function normalize(value: unknown, field: MappingField): unknown {
  if (value === undefined || value === null || value === '') {
    return field.defaultValue;
  }
  if (field.valueType === 'number') return Number(value);
  if (field.valueType === 'boolean') return Boolean(value);
  return value;
}

export function mapNodeToDslConfig(nodeData: any, spec: NodeMappingSpec): Record<string, any> {
  const cfg: Record<string, any> = {};
  for (const field of spec.fields) {
    const raw = nodeData?.inputsValues?.[field.nodePath]?.content;
    cfg[field.dslPath] = normalize(raw, field);
  }
  return spec.transformOut ? spec.transformOut(cfg, nodeData) : cfg;
}

export function mapDslToNodeInputsValues(cfg: Record<string, any>, spec: NodeMappingSpec): Record<string, any> {
  const normalized = spec.transformIn ? spec.transformIn(cfg, {}) : cfg;
  const inputsValues: Record<string, any> = {};
  for (const field of spec.fields) {
    inputsValues[field.nodePath] = {
      type: field.valueType === 'template' ? 'template' : 'constant',
      content: normalize(normalized[field.dslPath], field),
    };
  }
  return inputsValues;
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd flowgram && npm run test:unit -- dsl-mapping`  
Expected: PASS，`dsl mapping engine` 测试通过

- [ ] **Step 5: 提交本任务**

```bash
git add flowgram/src/utils/dsl-mapping/types.ts flowgram/src/utils/dsl-mapping/engine.ts flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts
git commit -m "refactor(flowgram): add base dsl mapping engine and types"
```

---

### Task 2: 为 `ai/agentHarness` 与 `ai/llm` 建立单一配置源并接入

**Files:**
- Create: `flowgram/src/utils/dsl-mapping/specs.ts`
- Create: `flowgram/src/utils/dsl-mapping/specializers.ts`
- Modify: `flowgram/src/utils/rulechain-builder.ts`
- Test: `flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts`

- [ ] **Step 1: 先写失败测试（agentHarness/llm round-trip）**

```ts
import { getNodeMappingSpec } from '../specs';
import { mapNodeToDslConfig, mapDslToNodeInputsValues } from '../engine';

it('round-trip ai/agentHarness', () => {
  const spec = getNodeMappingSpec('ai/agentHarness');
  const nodeData = {
    inputsValues: {
      model: { type: 'template', content: 'gpt-4.1' },
      enableMcpTool: { type: 'constant', content: true },
      maxToolCalls: { type: 'constant', content: 6 },
    },
  };
  const cfg = mapNodeToDslConfig(nodeData as any, spec!);
  const iv = mapDslToNodeInputsValues(cfg, spec!);
  expect(iv.model.content).toBe('gpt-4.1');
  expect(iv.enableMcpTool.content).toBe(true);
  expect(iv.maxToolCalls.content).toBe(6);
});
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd flowgram && npm run test:unit -- ai/agentHarness`  
Expected: FAIL，报错 `getNodeMappingSpec` 未定义

- [ ] **Step 3: 写最小实现（specs + specializers）**

```ts
// specializers.ts
export function llmTransformOut(cfg: Record<string, any>): Record<string, any> {
  const userPrompt = String(cfg.userPrompt ?? '');
  const next = { ...cfg };
  delete next.userPrompt;
  next.messages = [{ role: 'user', content: userPrompt }];
  next.params = {
    temperature: cfg.temperature,
    topP: cfg.topP,
    maxTokens: cfg.maxTokens,
    responseFormat: cfg.responseFormat,
  };
  delete next.temperature;
  delete next.topP;
  delete next.maxTokens;
  delete next.responseFormat;
  return next;
}

export function llmTransformIn(cfg: Record<string, any>): Record<string, any> {
  const params = cfg.params ?? {};
  return {
    ...cfg,
    userPrompt: Array.isArray(cfg.messages) ? cfg.messages[0]?.content ?? '' : '',
    temperature: params.temperature ?? 0.5,
    topP: params.topP ?? 0.5,
    maxTokens: params.maxTokens ?? 0,
    responseFormat: params.responseFormat ?? 'text',
  };
}
```

```ts
// specs.ts
import type { NodeMappingSpec } from './types';
import { llmTransformIn, llmTransformOut } from './specializers';

const specs: Record<string, NodeMappingSpec> = {
  'ai/agentHarness': {
    nodeType: 'ai/agentHarness',
    fields: [
      { nodePath: 'model', dslPath: 'model', valueType: 'template', defaultValue: '' },
      { nodePath: 'systemPrompt', dslPath: 'systemPrompt', valueType: 'template', defaultValue: '' },
      { nodePath: 'userPrompt', dslPath: 'userPrompt', valueType: 'template', defaultValue: '' },
      { nodePath: 'enableSkillTool', dslPath: 'enableSkillTool', valueType: 'boolean', defaultValue: true },
      { nodePath: 'enableMcpTool', dslPath: 'enableMcpTool', valueType: 'boolean', defaultValue: true },
      { nodePath: 'enableUUIDTool', dslPath: 'enableUUIDTool', valueType: 'boolean', defaultValue: true },
      {
        nodePath: 'enableWorkspaceTools',
        dslPath: 'enableWorkspaceTools',
        valueType: 'boolean',
        defaultValue: false,
      },
      { nodePath: 'skillAllowlist', dslPath: 'skillAllowlist', valueType: 'template', defaultValue: '' },
      { nodePath: 'mcpAllowlist', dslPath: 'mcpAllowlist', valueType: 'template', defaultValue: '' },
      { nodePath: 'maxIterations', dslPath: 'maxIterations', valueType: 'number', defaultValue: 0 },
      { nodePath: 'maxToolCalls', dslPath: 'maxToolCalls', valueType: 'number', defaultValue: 0 },
      { nodePath: 'toolTimeoutSecs', dslPath: 'toolTimeoutSecs', valueType: 'number', defaultValue: 0 },
    ],
  },
  'ai/llm': {
    nodeType: 'ai/llm',
    fields: [
      { nodePath: 'model', dslPath: 'model', valueType: 'constant', defaultValue: '' },
      { nodePath: 'key', dslPath: 'key', valueType: 'constant', defaultValue: '' },
      { nodePath: 'url', dslPath: 'url', valueType: 'constant', defaultValue: '' },
      { nodePath: 'systemPrompt', dslPath: 'systemPrompt', valueType: 'template', defaultValue: '' },
      { nodePath: 'userPrompt', dslPath: 'userPrompt', valueType: 'template', defaultValue: '' },
      { nodePath: 'temperature', dslPath: 'temperature', valueType: 'number', defaultValue: 0.5 },
      { nodePath: 'topP', dslPath: 'topP', valueType: 'number', defaultValue: 0.5 },
      { nodePath: 'maxTokens', dslPath: 'maxTokens', valueType: 'number', defaultValue: 0 },
      { nodePath: 'responseFormat', dslPath: 'responseFormat', valueType: 'constant', defaultValue: 'text' },
    ],
    transformOut: llmTransformOut,
    transformIn: llmTransformIn,
  },
};

export function getNodeMappingSpec(nodeType: string): NodeMappingSpec | undefined {
  return specs[nodeType];
}
```

- [ ] **Step 4: 在 `rulechain-builder.ts` 接入分发**

```ts
import { getNodeMappingSpec } from './dsl-mapping/specs';
import { mapNodeToDslConfig, mapDslToNodeInputsValues } from './dsl-mapping/engine';

// toDSL 分支
const spec = getNodeMappingSpec(nodeType);
if (spec) {
  base.configuration = mapNodeToDslConfig(n.data ?? {}, spec);
  nodesRC.push(base);
  return;
}

// fromDSL 分支
const spec2 = getNodeMappingSpec(t);
if (spec2) {
  base.data = {
    ...(base.data ?? {}),
    title: n.name ?? t,
    positionType: 'middle',
    inputsValues: mapDslToNodeInputsValues(n.configuration ?? {}, spec2),
  };
  return base as FlowNodeJSON;
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd flowgram && npm run test:unit -- mapping.spec.ts`  
Expected: PASS，`ai/agentHarness` 与 `ai/llm` round-trip 通过

- [ ] **Step 6: 提交本任务**

```bash
git add flowgram/src/utils/dsl-mapping/specs.ts flowgram/src/utils/dsl-mapping/specializers.ts flowgram/src/utils/rulechain-builder.ts flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts
git commit -m "refactor(flowgram): migrate ai nodes to spec-based dsl mapping"
```

---

### Task 3: 迁移 `restApiCall` 并补齐 URL/query 双向映射

**Files:**
- Modify: `flowgram/src/utils/dsl-mapping/specs.ts`
- Modify: `flowgram/src/utils/dsl-mapping/specializers.ts`
- Modify: `flowgram/src/utils/rulechain-builder.ts`
- Test: `flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts`

- [ ] **Step 1: 先写失败测试（URL/query round-trip）**

```ts
it('round-trip restApiCall query mapping', () => {
  const spec = getNodeMappingSpec('restApiCall');
  const nodeData = {
    inputsValues: {
      requestMethod: { type: 'constant', content: 'GET' },
      restEndpointUrlPattern: { type: 'template', content: 'https://a.com/api' },
      params: { type: 'constant', content: { a: '1', b: 'x' } },
      headers: { type: 'constant', content: { Authorization: 'token' } },
      body: { type: 'template', content: '' },
      readTimeoutMs: { type: 'constant', content: 3000 },
    },
  };
  const cfg = mapNodeToDslConfig(nodeData as any, spec!);
  expect(cfg.restEndpointUrlPattern).toBe('https://a.com/api?a=1&b=x');
  const iv = mapDslToNodeInputsValues(cfg, spec!);
  expect(iv.params.content).toEqual({ a: '1', b: 'x' });
});
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `cd flowgram && npm run test:unit -- restApiCall`  
Expected: FAIL，query 未拼装或未拆分

- [ ] **Step 3: 写最小实现（restApiCall spec + transform）**

```ts
// specializers.ts
export function restTransformOut(cfg: Record<string, any>): Record<string, any> {
  const next = { ...cfg };
  const url = String(next.restEndpointUrlPattern ?? '');
  const params = next.params ?? {};
  const qs = Object.entries(params)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v ?? ''))}`)
    .join('&');
  if (url && qs) {
    next.restEndpointUrlPattern = `${url}${url.includes('?') ? '&' : '?'}${qs}`;
  }
  return next;
}

export function restTransformIn(cfg: Record<string, any>): Record<string, any> {
  const next = { ...cfg };
  const fullUrl = String(next.restEndpointUrlPattern ?? '');
  const qm = fullUrl.indexOf('?');
  if (qm < 0) return next;
  const baseUrl = fullUrl.slice(0, qm);
  const query = fullUrl.slice(qm + 1);
  const params: Record<string, string> = {};
  for (const pair of query.split('&')) {
    if (!pair) continue;
    const [k, v] = pair.split('=');
    if (!k) continue;
    params[decodeURIComponent(k)] = decodeURIComponent(v ?? '');
  }
  next.restEndpointUrlPattern = baseUrl;
  next.params = { ...(params || {}), ...(next.params || {}) };
  return next;
}
```

```ts
// specs.ts 中新增
'restApiCall': {
  nodeType: 'restApiCall',
  fields: [
    { nodePath: 'requestMethod', dslPath: 'requestMethod', valueType: 'constant', defaultValue: 'GET' },
    { nodePath: 'restEndpointUrlPattern', dslPath: 'restEndpointUrlPattern', valueType: 'template', defaultValue: '' },
    { nodePath: 'headers', dslPath: 'headers', valueType: 'constant', defaultValue: {} },
    { nodePath: 'params', dslPath: 'params', valueType: 'constant', defaultValue: {} },
    { nodePath: 'body', dslPath: 'body', valueType: 'template', defaultValue: '' },
    { nodePath: 'readTimeoutMs', dslPath: 'readTimeoutMs', valueType: 'number', defaultValue: 0 },
  ],
  transformOut: restTransformOut,
  transformIn: restTransformIn,
},
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd flowgram && npm run test:unit -- mapping.spec.ts`  
Expected: PASS，`restApiCall` URL/query round-trip 通过

- [ ] **Step 5: 提交本任务**

```bash
git add flowgram/src/utils/dsl-mapping/specs.ts flowgram/src/utils/dsl-mapping/specializers.ts flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts
git commit -m "refactor(flowgram): migrate restApiCall to shared dsl mapping spec"
```

---

### Task 4: 建立测试基础设施（Vitest）并接入脚本

**Files:**
- Modify: `flowgram/package.json`
- Create: `flowgram/vitest.config.ts`
- Test: `flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts`

- [ ] **Step 1: 先写失败测试命令（脚本不存在）**

Run: `cd flowgram && npm run test:unit`  
Expected: FAIL，`Missing script: "test:unit"`

- [ ] **Step 2: 写最小配置**

```json
{
  "scripts": {
    "test:unit": "vitest run",
    "test:unit:watch": "vitest"
  },
  "devDependencies": {
    "vitest": "^2.1.9"
  }
}
```

```ts
// vitest.config.ts
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['src/**/*.spec.ts'],
    environment: 'node',
    globals: true,
  },
});
```

- [ ] **Step 3: 安装依赖并执行测试**

Run: `cd flowgram && npm i && npm run test:unit`  
Expected: PASS，输出 `1+ passed`

- [ ] **Step 4: 提交本任务**

```bash
git add flowgram/package.json flowgram/vitest.config.ts flowgram/package-lock.json flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts
git commit -m "test(flowgram): add vitest and dsl mapping unit tests"
```

---

### Task 5: 沉淀项目级 Skill（新增组件统一映射流程）

**Files:**
- Create: `.cursor/skills/dsl-component-mapping/SKILL.md`
- Test: `manual skill invocation checklist`

- [ ] **Step 1: 先写失败场景（没有 Skill 时容易遗漏）**

```md
场景：新增一个节点，只改 node registry 和 UI 表单，不写 dsl mapping 与 round-trip。
预期失败标准：审查时发现无法 round-trip 或导出字段丢失。
```

- [ ] **Step 2: 写最小 Skill 内容**

```md
---
name: dsl-component-mapping
description: Use when adding or modifying Flowgram node components that require DSL export/import mapping and round-trip consistency checks.
---

# DSL Component Mapping

## Checklist
- [ ] 节点注册完成（`flowgram/src/nodes/**`）
- [ ] 在 `flowgram/src/utils/dsl-mapping/specs.ts` 新增/更新 `NodeMappingSpec`
- [ ] 如有复杂字段，在 `specializers.ts` 添加 `transformOut/transformIn`
- [ ] 在 `__tests__/mapping.spec.ts` 增加该节点 `toDSL/fromDSL/round-trip` 测试
- [ ] 运行 `npm run test:unit` 与 `npm run lint`

## Required Rules
1. 新增组件必须包含映射与测试，否则视为未完成。
2. 默认值只允许在 spec 定义，禁止散落在 `rulechain-builder.ts` 分支。
3. 特例转换必须最小化并写明原因。
```

- [ ] **Step 3: 运行验证**

Run: `cd flowgram && npm run lint && npm run test:unit`  
Expected: PASS，无新增 lint/test 失败

- [ ] **Step 4: 提交本任务**

```bash
git add .cursor/skills/dsl-component-mapping/SKILL.md
git commit -m "docs(skill): add project skill for dsl component mapping workflow"
```

---

### Task 6: 清理重复逻辑并做最终回归

**Files:**
- Modify: `flowgram/src/utils/rulechain-builder.ts`
- Test: `flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts`

- [ ] **Step 1: 删除已迁移节点的重复分支代码（保留回退机制）**

```ts
// 删除 ai/agentHarness / ai/llm / restApiCall 的旧 switch 分支内部映射细节
// 保留未迁移节点 default 与复杂节点（switch/for/jsTransform）现有逻辑
if (spec) {
  // 新引擎路径
} else {
  // 旧逻辑路径（未迁移节点）
}
```

- [ ] **Step 2: 执行完整检查**

Run:
- `cd flowgram && npm run lint`
- `cd flowgram && npm run ts-check`
- `cd flowgram && npm run test:unit`

Expected:
- lint 通过
- ts-check 通过
- unit tests 全部通过

- [ ] **Step 3: 提交最终收口**

```bash
git add flowgram/src/utils/rulechain-builder.ts
git commit -m "refactor(flowgram): remove duplicated dsl mapping branches after spec migration"
```

---

## Self-Review

- **Spec coverage:** 已覆盖架构拆分、三节点迁移、round-trip 测试、项目级 Skill 沉淀。
- **Placeholder scan:** 无 `TODO/TBD`，每个步骤包含文件、命令、代码示例与预期结果。
- **Type consistency:** 统一使用 `NodeMappingSpec`、`mapNodeToDslConfig`、`mapDslToNodeInputsValues` 命名；`transformOut/transformIn` 在 `specializers.ts` 一致。

