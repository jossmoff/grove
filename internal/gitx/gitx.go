// Package gitx wraps the git binary.
//
// We deliberately shell out rather than use go-git: `git worktree add` carries
// branch creation, tracking setup, sparse-checkout and `repair` semantics that
// go-git does not fully implement and that are not worth reimplementing.
// Read-side status is also shelled out; if it ever gets slow across many
// repos, measure before reaching for a library.
package gitx

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Run executes git with args in dir, returning trimmed stdout.
// Errors carry git's stderr, because "exit status 128" alone is useless.
func Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s in %s: %s: %w",
			strings.Join(args, " "), dir, strings.TrimSpace(errb.String()), err)
	}
	return strings.TrimSpace(out.String()), nil
}

// Try is Run for probes where failure is a legitimate answer.
func Try(dir string, args ...string) (string, bool) {
	s, err := Run(dir, args...)
	return s, err == nil
}

// Clone clones url to dest, streaming progress to the user's terminal.
func Clone(url, dest string) error {
	if err := os.MkdirAll(parentDir(dest), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stdout = os.Stderr // progress is narration, not output
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func parentDir(p string) string {
	i := strings.LastIndexByte(p, os.PathSeparator)
	if i <= 0 {
		return "."
	}
	return p[:i]
}

// WorktreeAddBranch creates a worktree at path on a new branch from base.
// Used for write repos.
func WorktreeAddBranch(repo, path, branch, base string) error {
	_, err := Run(repo, "worktree", "add", "-b", branch, path, base)
	return err
}

// WorktreeAddDetached creates a detached worktree at path from base.
// Used for read repos: no branch is created, so there is nothing to push and
// nothing to clean up in refs/heads afterwards.
func WorktreeAddDetached(repo, path, base string) error {
	_, err := Run(repo, "worktree", "add", "--detach", path, base)
	return err
}

// WorktreeRemove removes the worktree at path from repo.
func WorktreeRemove(repo, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := Run(repo, args...)
	return err
}

// WorktreePrune prunes stale worktree metadata. Only useful after the
// directory is gone, which is why it always follows WorktreeRemove.
func WorktreePrune(repo string) error {
	_, err := Run(repo, "worktree", "prune")
	return err
}

// Fetch updates all remotes.
func Fetch(repo string) error {
	_, err := Run(repo, "fetch", "--all", "--prune", "--quiet")
	return err
}

// DefaultBase resolves the ref new work should start from: origin/HEAD when
// known, else whatever HEAD points at.
func DefaultBase(repo string) string {
	if s, ok := Try(repo, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); ok && s != "" {
		return s
	}
	if s, ok := Try(repo, "rev-parse", "--abbrev-ref", "HEAD"); ok && s != "" {
		return s
	}
	return "HEAD"
}

// RemoteURL returns origin's URL, if configured.
func RemoteURL(repo string) (string, bool) {
	s, ok := Try(repo, "remote", "get-url", "origin")
	return s, ok && s != ""
}

// CurrentBranch returns the checked-out branch, or false when detached.
func CurrentBranch(dir string) (string, bool) {
	s, ok := Try(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if !ok || s == "HEAD" {
		return "", false
	}
	return s, true
}

// DirtyCount counts changed entries (staged + unstaged + untracked).
func DirtyCount(dir string) int {
	s, ok := Try(dir, "status", "--porcelain")
	if !ok || s == "" {
		return 0
	}
	return len(strings.Split(s, "\n"))
}

// AheadBehind returns commits (ahead, behind) of upstream, if one exists.
func AheadBehind(dir string) (ahead, behind int, ok bool) {
	s, o := Try(dir, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if !o {
		return 0, 0, false
	}
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0, 0, false
	}
	b, err1 := strconv.Atoi(parts[0])
	a, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return a, b, true
}

// BranchExists reports whether refs/heads/<branch> exists in repo.
func BranchExists(repo, branch string) bool {
	_, ok := Try(repo, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return ok
}

// LastCommitUnix returns the epoch seconds of the last commit.
func LastCommitUnix(repo string) (int64, bool) {
	s, ok := Try(repo, "log", "-1", "--format=%ct")
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n, err == nil
}

// IsRepo reports whether dir is a git repository root.
func IsRepo(dir string) bool {
	_, err := os.Stat(dir + string(os.PathSeparator) + ".git")
	return err == nil
}
