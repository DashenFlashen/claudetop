package git_test

import (
	"os"
	"path/filepath"
	"testing"

	"claudetop/internal/git"
)

func TestReposUnder(t *testing.T) {
	root := t.TempDir()

	// Create two git repos and one non-git dir
	for _, name := range []string{"repo-a", "repo-b"} {
		if err := os.MkdirAll(filepath.Join(root, name, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "not-a-repo"), 0755); err != nil {
		t.Fatal(err)
	}

	repos, err := git.ReposUnder(root)
	if err != nil {
		t.Fatalf("ReposUnder: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("got %d repos, want 2: %v", len(repos), repos)
	}
	for _, r := range repos {
		if r != "repo-a" && r != "repo-b" {
			t.Errorf("unexpected repo name: %q", r)
		}
	}
}

func TestParseGitLog(t *testing.T) {
	input := "fix timeout in pipeline\nadd retry logic to connector\n"
	commits := git.ParseGitLog(input, "annotation-service")

	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}
	if commits[0].Repo != "annotation-service" {
		t.Errorf("got repo %q, want %q", commits[0].Repo, "annotation-service")
	}
	if commits[0].Message != "fix timeout in pipeline" {
		t.Errorf("got message %q, want %q", commits[0].Message, "fix timeout in pipeline")
	}
}

func TestParseGitLogEmpty(t *testing.T) {
	commits := git.ParseGitLog("", "my-repo")
	if len(commits) != 0 {
		t.Errorf("expected no commits for empty input, got %d", len(commits))
	}
}
