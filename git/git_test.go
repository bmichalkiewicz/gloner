package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestDir(t *testing.T) {
	base := filepath.Join(string(filepath.Separator), "home", "user", "git")

	dir, err := Dir("git@github.com:user/repo.git", base)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "github.com", "user", "repo"); dir != want {
		t.Errorf("Dir() = %q, want %q", dir, want)
	}

	if _, err := Dir("git@github.com:../../repo.git", base); err == nil {
		t.Error("Dir() accepted a path escaping the base directory")
	}
}

// gitCmd runs git in dir and fails the test on error.
func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestCloneExistingAndDryRun(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	base := t.TempDir()
	url := "git@example.com:user/repo.git"

	status, err := Clone(context.Background(), url, base, Options{DryRun: true})
	if err != nil || status != StatusPlanned {
		t.Fatalf("dry run: got (%v, %v), want StatusPlanned", status, err)
	}

	dir, _ := Dir(url, base)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Clone(context.Background(), url, base, Options{}); err == nil {
		t.Fatal("expected error for existing directory that is not a repository")
	}

	gitCmd(t, dir, "init", "--quiet")
	status, err = Clone(context.Background(), url, base, Options{})
	if err != nil || status != StatusExists {
		t.Fatalf("existing repo: got (%v, %v), want StatusExists", status, err)
	}
}

func TestPull(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	tmp := t.TempDir()
	upstream := filepath.Join(tmp, "upstream")
	clone := filepath.Join(tmp, "clone")

	if err := os.Mkdir(upstream, 0o755); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, upstream, "init", "--quiet")
	gitCmd(t, upstream, "commit", "--quiet", "--allow-empty", "-m", "first")
	gitCmd(t, tmp, "clone", "--quiet", upstream, clone)

	changed, err := Pull(context.Background(), clone)
	if err != nil || changed {
		t.Fatalf("pull without changes: got (%v, %v), want (false, nil)", changed, err)
	}

	gitCmd(t, upstream, "commit", "--quiet", "--allow-empty", "-m", "second")
	changed, err = Pull(context.Background(), clone)
	if err != nil || !changed {
		t.Fatalf("pull with changes: got (%v, %v), want (true, nil)", changed, err)
	}
}

func TestFindRepositories(t *testing.T) {
	base := t.TempDir()
	for _, dir := range []string{
		"github.com/user/a/.git",
		"github.com/user/a/sub/.git", // nested, must not be listed
		"gitlab.com/group/sub/b/.git",
		"gitlab.com/group/not-a-repo",
	} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	repos, err := FindRepositories(base)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(base, "github.com/user/a"),
		filepath.Join(base, "gitlab.com/group/sub/b"),
	}
	if !slices.Equal(repos, want) {
		t.Errorf("FindRepositories() = %v, want %v", repos, want)
	}
}
