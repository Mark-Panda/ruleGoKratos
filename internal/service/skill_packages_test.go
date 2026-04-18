package service

import "testing"

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

