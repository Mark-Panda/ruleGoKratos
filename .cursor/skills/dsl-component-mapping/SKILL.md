---
name: dsl-component-mapping
description: Use when 新增或修改 flowgram 节点组件，且涉及 DSL 映射（rulechain-builder、toDSL/fromDSL 双向映射、NodeMappingSpec、导入导出、round-trip、画布与 configuration 对齐）时。
---

# DSL 组件映射（项目约束）

在动 `flowgram` 节点与 RuleGo DSL 的对应关系前，先读完本清单再改代码。

## 已迁移节点如何认定

以 `flowgram/src/utils/dsl-mapping/specs.ts` 为唯一事实源：**`SPEC_BY_TYPE` 中出现的 `nodeType` 键**，或 **`getNodeMappingSpec(nodeType)` 返回值非 `undefined`** 的节点，视为已走映射引擎；其余仍由 `rulechain-builder.ts` 旧分支处理，直到补注册表。

## Checklist

- [ ] **节点注册**：新 `nodeType` 已在画布侧完成注册（节点面板、表单 schema、`inputsValues` 字段与 UI 一致）。
- [ ] **spec 映射**：在 `specs.ts` 增加或更新对应 `NodeMappingSpec`（`fields` 中每条 `MappingField` 含 `inputKey`、`dslKey`、`valueType`、`defaultValue`；按需补充字段级或 spec 级 `transformIn`/`transformOut`）。**当前不设独立的 `required` 语义**：缺值/空值由 **`defaultValue` + `engine` 空值规则**（如 `isEmptyForMapping`）决定如何写出或回填。
- [ ] **specializer 特例**：非通用路径拼装/拆分、结构体重组等，放在 `flowgram/src/utils/dsl-mapping/specializers.ts`，并在代码旁用一行注释写清「为何不能仅靠 spec」。
- [ ] **测试**：在 `flowgram/src/utils/dsl-mapping/__tests__/` 为**每个已迁移节点**至少补三类用例：`toDSL`、`fromDSL`、**round-trip**（`node -> dsl -> node`）。**断言范围**至少覆盖该 spec 的 **`fields` 所涉 input/configuration 路径**，以及 **字段级 / spec 级 `transform` 与 specializer 的预期产物**（避免只测「能跑」不测映射形状）。
- [ ] **执行命令**：在 `flowgram` 目录运行 `npm run test:unit`（`vitest run`）；改动了映射相关逻辑须本地通过后再视为完成。

## 必守规则

1. **缺映射或缺测试视为未完成**：仅有 UI 或仅有 DSL 一侧可跑通，不算交付；CR 中应打回。
2. **默认值只在 spec 声明**：禁止在组件或散落 helper 里悄悄写「隐式默认」；与 `inputsValues` 回填一致的行为由 `NodeMappingSpec` / `MappingField.defaultValue` 显式描述，并与引擎空值规则一致。
3. **复杂逻辑必须放 specializers**：`engine` 只做通用读写与类型归一化；特例集中、最小化，避免大 `switch` 再度膨胀。

## 参考锚点

- 引擎与类型：`flowgram/src/utils/dsl-mapping/engine.ts`、`types.ts`
- 接入分发：`flowgram/src/utils/rulechain-builder.ts`（命中 spec 走引擎，未命中保留旧分支直至迁移）
