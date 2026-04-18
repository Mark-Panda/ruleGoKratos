package service

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// discoverSkillPackageCounts 与 ListSkills 相同的扫描规则，按技能 id 的首路径段聚合为「技能包」。
func discoverSkillPackageCounts(root string) (map[string]int, error) {
	counts := make(map[string]int)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !isSkillFileExt(ext) {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = filepath.Base(path)
		}
		relSlash := filepath.ToSlash(rel)
		noExt := strings.TrimSuffix(relSlash, filepath.Ext(relSlash))
		pkg := packageIDFromRelNoExt(noExt)
		if pkg != "" {
			counts[pkg]++
		}
		return nil
	})
	if err != nil {
		return nil, err
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
