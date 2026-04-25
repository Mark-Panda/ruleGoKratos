package service

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ruleGoKratos/internal/biz"
)

// SkillPackageInfo 技能包：相对 skill 根路径的首段目录名（或根目录下单文件的「包 id」）。
type SkillPackageInfo struct {
	ID             string `json:"id"`
	SkillFileCount int    `json:"skillFileCount"`
}

func packageIDFromRelNoExt(relNoExt string) string {
	relNoExt = strings.Trim(strings.ReplaceAll(relNoExt, "\\", "/"), "/")
	if relNoExt == "" {
		return ""
	}
	if i := strings.Index(relNoExt, "/"); i >= 0 {
		return relNoExt[:i]
	}
	return relNoExt
}

func isSkillFileExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".md", ".txt", ".yaml", ".yml", ".json":
		return true
	default:
		return false
	}
}

// discoverSkillPackageCounts 按 Eino 官方 Skill 包约定扫描一级子目录下的 SKILL.md。
func discoverSkillPackageCounts(root string) (map[string]int, error) {
	counts := make(map[string]int)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(root, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		id := biz.SkillNameFromFrontMatter(string(data))
		if id == "" {
			id = packageIDFromRelNoExt(entry.Name())
		}
		if id != "" {
			counts[id] = 1
		}
	}
	return counts, nil
}

func discoverSkillPackages(root string) ([]SkillPackageInfo, error) {
	counts, err := discoverSkillPackageCounts(root)
	if err != nil {
		return nil, err
	}
	out := make([]SkillPackageInfo, 0, len(counts))
	for id, n := range counts {
		out = append(out, SkillPackageInfo{ID: id, SkillFileCount: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func discoverSkillPackageSet(root string) (map[string]struct{}, error) {
	counts, err := discoverSkillPackageCounts(root)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(counts))
	for id := range counts {
		set[id] = struct{}{}
	}
	return set, nil
}
