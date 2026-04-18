/**
 * ListRunLogs 返回的 ruleChain 为 RuleGo RuleChain DSL（含嵌套 ruleChain / metadata），
 * 工作流展示名与规则链 ID 在内层 ruleChain.ruleChain。
 */
export function runLogChainDisplay(row: any): { id: string; name: string } {
  const pack = row?.ruleChain;
  const inner = pack?.ruleChain;
  if (inner != null && typeof inner === 'object') {
    return {
      id: String((inner as { id?: unknown }).id ?? ''),
      name: String((inner as { name?: unknown }).name ?? ''),
    };
  }
  return {
    id: String(pack?.id ?? ''),
    name: String(pack?.name ?? ''),
  };
}
