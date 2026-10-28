package cmd

import "testing"

func TestMatchesPrefix(t *testing.T) {
	tests := []struct {
		path     string
		prefixes []string
		want     bool
	}{
		{path: "github.com/org/repo", want: true},
		{path: "github.com/org/repo", prefixes: []string{"github.com"}, want: true},
		{path: "github.com/org/repo", prefixes: []string{"github.com/org/"}, want: true},
		{path: "github.com/org/repo", prefixes: []string{"github.com/org/repo"}, want: true},
		{path: "github.com/org/repo", prefixes: []string{"github.com/or"}, want: false},
		{path: "github.com/org/repo", prefixes: []string{"gitlab.com", "github.com/org"}, want: true},
	}

	for _, tt := range tests {
		if got := matchesPrefix(tt.path, tt.prefixes); got != tt.want {
			t.Errorf("matchesPrefix(%q, %v) = %v, want %v", tt.path, tt.prefixes, got, tt.want)
		}
	}
}
