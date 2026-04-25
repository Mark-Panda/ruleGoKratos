package service

import (
	"os"
	"path/filepath"
	"testing"
)

func writeServiceSkillPackage(t *testing.T, root, pkg, content string) {
	t.Helper()
	dir := filepath.Join(root, pkg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill package failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md failed: %v", err)
	}
}

func TestDiscoverSkillPackagesUsesOfficialSkillPackageEntry(t *testing.T) {
	root := t.TempDir()
	writeServiceSkillPackage(t, root, "pkg-dir", "---\nname: canonical-skill\n---\nbody")
	if err := os.WriteFile(filepath.Join(root, "loose.md"), []byte("loose"), 0o644); err != nil {
		t.Fatalf("write loose file failed: %v", err)
	}
	refDir := filepath.Join(root, "pkg-dir", "reference")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatalf("mkdir reference dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refDir, "guide.md"), []byte("guide"), 0o644); err != nil {
		t.Fatalf("write reference file failed: %v", err)
	}

	items, err := discoverSkillPackages(root)
	if err != nil {
		t.Fatalf("discoverSkillPackages failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one skill package, got %v", items)
	}
	if items[0].ID != "canonical-skill" || items[0].SkillFileCount != 1 {
		t.Fatalf("unexpected skill package info: %#v", items[0])
	}
}

func TestPackageIDFromRelNoExt(t *testing.T) {
	cases := []struct {
		rel  string
		want string
	}{
		{"planner/SKILL", "planner"},
		{"foo", "foo"},
		{"a/b/c", "a"},
		{"/x/y", "x"},
	}
	for _, c := range cases {
		if got := packageIDFromRelNoExt(c.rel); got != c.want {
			t.Errorf("packageIDFromRelNoExt(%q) = %q, want %q", c.rel, got, c.want)
		}
	}
}
