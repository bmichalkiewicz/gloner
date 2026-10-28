package repositories

import "testing"

func TestParseURL(t *testing.T) {
	tests := []struct {
		url      string
		wantHost string
		wantPath string
		wantErr  bool
	}{
		{url: "git@github.com:user/repo.git", wantHost: "github.com", wantPath: "user/repo"},
		{url: "git@gitlab.com:group/sub/repo.git", wantHost: "gitlab.com", wantPath: "group/sub/repo"},
		{url: "git@github.com:user/repo", wantHost: "github.com", wantPath: "user/repo"},
		{url: "git@github.com:user/my.repo.git", wantHost: "github.com", wantPath: "user/my.repo"},
		{url: "org-123@github.com:user/repo.git", wantHost: "github.com", wantPath: "user/repo"},
		{url: "ssh://git@gitlab.example.com:2222/group/repo.git", wantHost: "gitlab.example.com", wantPath: "group/repo"},
		{url: "ssh://git@gitlab.example.com/group/repo.git", wantHost: "gitlab.example.com", wantPath: "group/repo"},
		{url: "https://github.com/user/repo.git", wantHost: "github.com", wantPath: "user/repo"},
		{url: "https://github.com/user/repo", wantHost: "github.com", wantPath: "user/repo"},
		{url: "", wantErr: true},
		{url: "github.com/user/repo", wantErr: true},
		{url: "/local/path/repo.git", wantErr: true},
		{url: "ftp://host/user/repo.git", wantErr: true},
		{url: "git@github.com:.git", wantErr: true},
		{url: "git@github.com:../../etc/repo.git", wantErr: true},
		{url: "git@github.com:user/../../../repo.git", wantErr: true},
		{url: "git@github.com:user//repo.git", wantErr: true},
		{url: "git@bad_host!:user/repo.git", wantErr: true},
		{url: "https://github.com/user/re po.git", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			host, path, err := ParseURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseURL(%q) = (%q, %q), want error", tt.url, host, path)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseURL(%q) unexpected error: %v", tt.url, err)
			}
			if host != tt.wantHost || path != tt.wantPath {
				t.Errorf("ParseURL(%q) = (%q, %q), want (%q, %q)", tt.url, host, path, tt.wantHost, tt.wantPath)
			}
		})
	}
}

func TestSortGroups(t *testing.T) {
	groups := sortGroups(map[string]*Group{
		"b": {Name: "b", Projects: []Project{{URL: "z"}, {URL: "a"}}},
		"a": {Name: "a"},
	})

	if len(groups) != 2 || groups[0].Name != "a" || groups[1].Name != "b" {
		t.Fatalf("groups not sorted by name: %+v", groups)
	}
	if groups[1].Projects[0].URL != "a" {
		t.Errorf("projects not sorted by URL: %+v", groups[1].Projects)
	}
}
