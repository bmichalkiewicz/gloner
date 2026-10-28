package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "https://gitlab.com", want: "https://gitlab.com"},
		{in: "https://gitlab.com/", want: "https://gitlab.com"},
		{in: "gitlab.example.com", want: "https://gitlab.example.com"},
		{in: "http://gitlab.local:8080", want: "http://gitlab.local:8080"},
		{in: "https://example.com/gitlab", want: "https://example.com/gitlab"},
		{in: "", wantErr: true},
		{in: "ftp://gitlab.com", wantErr: true},
		{in: "https://", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := NormalizeURL(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeURL(%q) = %q, want error", tt.in, got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("NormalizeURL(%q) = (%q, %v), want %q", tt.in, got, err, tt.want)
			}
		})
	}
}

func TestInitCreatesPrivateConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(home, ".config", "gloner", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("config file permissions = %v, want 0600", info.Mode().Perm())
	}
	if Settings.Gitlab.URL != DefaultGitlabURL {
		t.Errorf("default GitLab URL = %q, want %q", Settings.Gitlab.URL, DefaultGitlabURL)
	}
}
