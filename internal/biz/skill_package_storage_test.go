package biz

import "testing"

func TestLegacySkillStorageEntryToPackageID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"planner/SKILL.md", "planner"},
		{"foo.md", "foo"},
		{"planner", "planner"},
		{"  x/y.z  ", "x"},
	}
	for _, c := range cases {
		if got := LegacySkillStorageEntryToPackageID(c.in); got != c.want {
			t.Errorf("LegacySkillStorageEntryToPackageID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
