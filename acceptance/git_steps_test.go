package acceptance_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
)

func registerGitSteps(sc *godog.ScenarioContext, w *World) {
	// Given — state setup
	sc.Step(`^a repo "([^"]*)" exists$`, w.aRepoExists)
	sc.Step(`^a grove "([^"]*)" exists with "([^"]*)" as write$`, w.aGroveExistsWithWrite)
	sc.Step(`^a grove "([^"]*)" exists with "([^"]*)" as write and "([^"]*)" as read$`, w.aGroveExistsWithWriteAndRead)
	sc.Step(`^the worktree "([^"]*)" has an uncommitted file "([^"]*)"$`, w.worktreeHasUncommittedFile)
	sc.Step(`^the file "([^"]*)" has been deleted$`, w.fileHasBeenDeleted)
	sc.Step(`^a profile "([^"]*)" with "([^"]*)" as write$`, w.aProfileWithWrite)
	sc.Step(`^a profile "([^"]*)" with "([^"]*)" as write and "([^"]*)" as read$`, w.aProfileWithWriteAndRead)
	sc.Step(`^the profile "([^"]*)" has a LOCAL\.md containing "([^"]*)"$`, w.profileHasLocalMd)

	// Then — git state assertions
	sc.Step(`^"([^"]*)" is on branch "([^"]*)"$`, w.worktreeIsOnBranch)
	sc.Step(`^"([^"]*)" is in detached HEAD state$`, w.worktreeIsDetached)
	sc.Step(`^the repo "([^"]*)" has branch "([^"]*)"$`, w.repoHasBranch)
	sc.Step(`^the repo "([^"]*)" still has branch "([^"]*)"$`, w.repoHasBranch)
	sc.Step(`^the repo "([^"]*)" has no branches matching "([^"]*)"$`, w.repoHasNoBranchesMatching)
	sc.Step(`^the repo "([^"]*)" has exactly (\d+) worktree entr(?:y|ies)$`, w.repoHasExactlyNWorktreeEntries)
	sc.Step(`^the grove "([^"]*)" still exists$`, w.groveStillExists)
	sc.Step(`^the grove "([^"]*)" does not exist$`, w.groveDoesNotExist)
	sc.Step(`^the file "([^"]*)" exists$`, w.assertFileExists)
	sc.Step(`^the file "([^"]*)" contains "([^"]*)"$`, w.assertFileContains)
	sc.Step(`^the file "([^"]*)" does not contain "([^"]*)"$`, w.assertFileDoesNotContain)
}

// --- Given steps — set up git state ---

// aRepoExists seeds an origin repo and a clone of it under srcDir.
// The layout mirrors the workspace_test.go helper: origin at tmpDir/origins/<name>,
// clone at srcDir/github.com/test/<name>. CLAUDE.md and a sample skill are
// included so AGENTS.md generation has something to route to.
func (w *World) aRepoExists(name string) error {
	origin := filepath.Join(w.tmpDir, "origins", name)
	if err := os.MkdirAll(origin, 0o755); err != nil {
		return err
	}
	for _, cmd := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@t.co"},
		{"config", "user.name", "t"},
	} {
		if _, err := w.gitRun(origin, cmd...); err != nil {
			return err
		}
	}
	files := map[string]string{
		"README.md":                    "# " + name + "\n",
		"CLAUDE.md":                    "instructions for " + name + "\n",
		".claude/skills/test/SKILL.md": "# test\n",
	}
	for rel, content := range files {
		full := filepath.Join(origin, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return err
		}
	}
	for _, cmd := range [][]string{
		{"add", "-A"},
		{"commit", "-qm", "init"},
	} {
		if _, err := w.gitRun(origin, cmd...); err != nil {
			return err
		}
	}

	dest := w.repoClone(name)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if _, err := w.gitRun(w.tmpDir, "clone", "-q", origin, dest); err != nil {
		return err
	}
	for _, cmd := range [][]string{
		{"config", "user.email", "t@t.co"},
		{"config", "user.name", "t"},
	} {
		if _, err := w.gitRun(dest, cmd...); err != nil {
			return err
		}
	}
	return nil
}

func (w *World) aGroveExistsWithWrite(slug, repo string) error {
	if err := w.run("new", slug, "-w", repo, "--no-fetch"); err != nil {
		return err
	}
	if w.lastResult.exitCode != 0 {
		return fmt.Errorf("grove new %s -w %s failed (exit %d): %s", slug, repo, w.lastResult.exitCode, w.lastResult.stderr)
	}
	return nil
}

func (w *World) aGroveExistsWithWriteAndRead(slug, writeRepo, readRepo string) error {
	if err := w.run("new", slug, "-w", writeRepo, "-r", readRepo, "--no-fetch"); err != nil {
		return err
	}
	if w.lastResult.exitCode != 0 {
		return fmt.Errorf("grove new %s failed (exit %d): %s", slug, w.lastResult.exitCode, w.lastResult.stderr)
	}
	return nil
}

func (w *World) worktreeHasUncommittedFile(slugRepo, filename string) error {
	p := filepath.Join(w.worktreePath(slugRepo), filename)
	return os.WriteFile(p, []byte("wip\n"), 0o644)
}

func (w *World) fileHasBeenDeleted(relPath string) error {
	full := filepath.Join(w.groveRoot, filepath.FromSlash(relPath))
	return os.Remove(full)
}

// aProfileWithWrite writes a profile.toml directly, bypassing grove new.
// The canonical path is github.com/test/<repo> to match how aRepoExists seeds.
func (w *World) aProfileWithWrite(name, repo string) error {
	dir := w.profileDir(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	canonical := "github.com/test/" + repo
	content := fmt.Sprintf("version = 1\nrepos = [\"%s:write\"]\n", canonical)
	return os.WriteFile(filepath.Join(dir, "profile.toml"), []byte(content), 0o644)
}

func (w *World) aProfileWithWriteAndRead(name, writeRepo, readRepo string) error {
	dir := w.profileDir(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	wc := "github.com/test/" + writeRepo
	rc := "github.com/test/" + readRepo
	content := fmt.Sprintf("version = 1\nrepos = [\"%s:write\", \"%s:read\"]\n", wc, rc)
	return os.WriteFile(filepath.Join(dir, "profile.toml"), []byte(content), 0o644)
}

func (w *World) profileHasLocalMd(name, content string) error {
	dir := w.profileDir(name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "LOCAL.md"), []byte(content+"\n"), 0o644)
}

// --- Then steps — assert on git state ---

func (w *World) worktreeIsOnBranch(slugRepo, branch string) error {
	wt := w.worktreePath(slugRepo)
	got, err := w.gitRun(wt, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	if got != branch {
		return fmt.Errorf("worktree %q: branch = %q, want %q", slugRepo, got, branch)
	}
	return nil
}

func (w *World) worktreeIsDetached(slugRepo string) error {
	wt := w.worktreePath(slugRepo)
	got, err := w.gitRun(wt, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	if got != "HEAD" {
		return fmt.Errorf("worktree %q: expected detached HEAD, got branch %q", slugRepo, got)
	}
	return nil
}

func (w *World) repoHasBranch(repoName, branch string) error {
	clone := w.repoClone(repoName)
	got, err := w.gitRun(clone, "branch", "--list", branch)
	if err != nil {
		return err
	}
	if !strings.Contains(got, strings.TrimPrefix(branch, "refs/heads/")) {
		return fmt.Errorf("repo %q: branch %q not found (output: %q)", repoName, branch, got)
	}
	return nil
}

func (w *World) repoHasNoBranchesMatching(repoName, pattern string) error {
	clone := w.repoClone(repoName)
	got, err := w.gitRun(clone, "branch", "--list", pattern)
	if err != nil {
		return err
	}
	if strings.TrimSpace(got) != "" {
		return fmt.Errorf("repo %q: unexpected branches matching %q: %q", repoName, pattern, got)
	}
	return nil
}

func (w *World) repoHasExactlyNWorktreeEntries(repoName string, n int) error {
	clone := w.repoClone(repoName)
	got, err := w.gitRun(clone, "worktree", "list")
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != n {
		return fmt.Errorf("repo %q: %d worktree entries, want %d:\n%s", repoName, len(lines), n, got)
	}
	return nil
}

func (w *World) groveStillExists(slug string) error {
	dir := w.groveDir(slug)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("grove %q should still exist but: %v", slug, err)
	}
	return nil
}

func (w *World) groveDoesNotExist(slug string) error {
	dir := w.groveDir(slug)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return fmt.Errorf("grove %q should not exist", slug)
	}
	return nil
}

func (w *World) assertFileExists(relPath string) error {
	full := filepath.Join(w.groveRoot, filepath.FromSlash(relPath))
	if _, err := os.Stat(full); err != nil {
		return fmt.Errorf("file %q: %v", relPath, err)
	}
	return nil
}

func (w *World) assertFileContains(relPath, substr string) error {
	full := filepath.Join(w.groveRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), substr) {
		return fmt.Errorf("file %q does not contain %q", relPath, substr)
	}
	return nil
}

func (w *World) assertFileDoesNotContain(relPath, substr string) error {
	full := filepath.Join(w.groveRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	if strings.Contains(string(data), substr) {
		return fmt.Errorf("file %q should not contain %q", relPath, substr)
	}
	return nil
}
