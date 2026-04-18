package biz

import (
	"path/filepath"
	"sort"
	"strings"
)

// TrimSkillLikeExtension 去掉常见技能文件后缀（与目录扫描一致）。
func TrimSkillLikeExtension(path string) string {
	lower := strings.ToLower(path)
	for _, ext := range []string{".md", ".txt", ".yaml", ".yml", ".json"} {
		if strings.HasSuffix(lower, ext) {
			return path[:len(path)-len(ext)]
		}
	}
	return path
}

// LegacySkillStorageEntryToPackageID 兼容 DB 中旧版存的「文件相对路径」，统一为技能包 id。
func LegacySkillStorageEntryToPackageID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = filepath.ToSlash(s)
	s = TrimSkillLikeExtension(s)
	return PackageIDFromSkillName(s)
}

// NormalizeStoredSkillPackageIDs 去重排序技能包 id（含旧版路径兼容）。
func NormalizeStoredSkillPackageIDs(raw []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(raw))
	for _, x := range raw {
		id := LegacySkillStorageEntryToPackageID(x)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
