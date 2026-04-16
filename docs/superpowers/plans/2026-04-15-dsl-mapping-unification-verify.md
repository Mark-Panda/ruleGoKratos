# dsl-mapping / rulechain-builder 验证说明（2026-04-15）

## 执行命令

在含 `flowgram` 的仓库目录下：

```bash
cd flowgram && npm run lint -- src/utils/rulechain-builder.ts src/utils/dsl-mapping
cd flowgram && npm run test:unit
cd flowgram && npm run ts-check
```

仅检查本次相关文件、避免 `lint` 脚本里 `./src` 连带其它目录告警时，可使用：

```bash
cd flowgram && npx eslint src/utils/rulechain-builder.ts "src/utils/dsl-mapping/**/*.ts"
```

## `ts-check` 基线

- `npm run ts-check`（`tsc --noEmit`）在 flowgram 侧**可能仍失败**，常见为其它源码中的 **TS6133**（未使用的 import/变量等），属**既有基线**，与 `rulechain-builder.ts`、`dsl-mapping/**` 的映射改动无直接关系。
- 回归时建议：查看 `tsc` 输出中是否出现 `src/utils/rulechain-builder.ts` 或 `src/utils/dsl-mapping/` 路径；若无，可视为本次目标文件**未新增** TypeScript 错误。
- 测试文件曾用 `@ts-expect-error` 压制 `vitest` 解析；在 `vitest` 已写入 `devDependencies` 且类型可解析后应**删除**该注释，否则会触发 **TS2578**（Unused `@ts-expect-error`）。

## 与本次改动文件的关系

| 区域 | 说明 |
|------|------|
| `src/utils/rulechain-builder.ts` | 规则链 JSON 与画布文档互转；与 dsl-mapping 引擎衔接。 |
| `src/utils/dsl-mapping/**/*.ts` | 节点映射 spec 与 `mapNodeToDslConfig` / `mapDslToNodeInputsValues`。 |
| `src/utils/dsl-mapping/__tests__/mapping.spec.ts` | 映射与 builder 的单元测试；`ts-check` 会编译该文件。 |
