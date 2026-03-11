package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommitSummary represents a single git commit from yesterday.
type CommitSummary struct {
	Repo    string
	Message string
}

// ReposUnder returns names of immediate subdirectories of rootDir that contain a .git directory.
func ReposUnder(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}
	var repos []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gitPath := filepath.Join(rootDir, e.Name(), ".git")
		if _, err := os.Stat(gitPath); err == nil {
			repos = append(repos, e.Name())
		}
	}
	return repos, nil
}

// ParseGitLog parses the output of `git log --format=%s` for the given repo name.
func ParseGitLog(output, repoName string) []CommitSummary {
	var commits []CommitSummary
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		commits = append(commits, CommitSummary{
			Repo:    repoName,
			Message: line,
		})
	}
	return commits
}

// YesterdayCommits returns all commits from yesterday (local time) across all git repos under rootDir.
// Errors from individual repos are silently skipped — a repo without git history is not an error.
func YesterdayCommits(rootDir string) ([]CommitSummary, error) {
	repos, err := ReposUnder(rootDir)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	afterDate := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local).Format("2006-01-02")
	beforeDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Format("2006-01-02")

	var all []CommitSummary
	for _, repoName := range repos {
		repoPath := filepath.Join(rootDir, repoName)
		out, err := exec.Command("git", "-C", repoPath,
			"log",
			"--after="+afterDate,
			"--before="+beforeDate,
			"--format=%s",
			"--no-merges",
		).Output()
		if err != nil {
			continue // not a git repo or no commits
		}
		all = append(all, ParseGitLog(string(out), repoName)...)
	}
	return all, nil
}
