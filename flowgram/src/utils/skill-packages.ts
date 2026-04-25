/**
 * 将管理端列出的技能文件归并为「套装」：与后端 FileSkillExecutor 的 key 规则对齐
 * （相对路径去掉扩展名；首段路径为一套，根目录单文件自成一套）。
 */

import type { SkillItem } from '../services/api-agent';

const SKILL_FILE_EXTS = new Set(['.md', '.txt', '.yaml', '.yml', '.json']);

function extOf(rel: string): string {
  const i = rel.lastIndexOf('.');
  if (i <= 0) return '';
  return rel.slice(i).toLowerCase();
}

/** 与 biz loadSkills 一致：相对路径去掉扩展名，正斜杠，无首尾斜杠 */
export function skillContentKey(item: SkillItem): string {
  let rel = String(item.path ?? '')
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
    .replace(/\/+$/, '');
  if (!rel) return '';
  const ext = extOf(rel);
  if (SKILL_FILE_EXTS.has(ext)) {
    rel = rel.slice(0, -ext.length);
  }
  return rel.replace(/\/+$/, '');
}

/** 套装 id：多级路径取首段；单段即整 key */
export function skillPackageId(skillKey: string): string {
  const i = skillKey.indexOf('/');
  return i >= 0 ? skillKey.slice(0, i) : skillKey;
}

export type SkillPackageGroup = {
  /** 套装目录名（或根下单文件的 stem） */
  id: string;
  /** 该套下全部官方 Skill name（写入白名单的精确值） */
  keys: string[];
  files: SkillItem[];
};

export function groupSkillPackages(items: SkillItem[]): SkillPackageGroup[] {
  const map = new Map<string, { keys: Set<string>; files: SkillItem[] }>();
  for (const it of items) {
    const key = skillContentKey(it);
    if (!key) continue;
    const pkgId = skillPackageId(key);
    if (!map.has(pkgId)) {
      map.set(pkgId, { keys: new Set(), files: [] });
    }
    const g = map.get(pkgId)!;
    g.keys.add(key);
    g.files.push(it);
  }
  return Array.from(map.entries())
    .map(([id, v]) => ({
      id,
      keys: Array.from(v.keys).sort(),
      files: v.files.sort((a, b) => String(a.path).localeCompare(String(b.path))),
    }))
    .sort((a, b) => a.id.localeCompare(b.id));
}
