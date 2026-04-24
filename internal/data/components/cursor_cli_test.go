package data

import "testing"

func TestCursorAgentStatusLooksAuthed(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"", false},
		{"   ", false},
		{"Not logged in", false},
		{"Error: not logged in\n", false},
		{"✓ Login successful!\nLogged in (unable to fetch user details)", true},
		{"Login successful", true},
		{"Logged in as user@example.com", true},
		{"Logged in (unable to fetch user details)", true},
		{"something else", false},
	}
	for _, tc := range cases {
		if got := cursorAgentStatusLooksAuthed(tc.raw); got != tc.want {
			t.Errorf("cursorAgentStatusLooksAuthed(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}
