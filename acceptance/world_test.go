package acceptance_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// World holds the isolated state for a single scenario: temp directories,
// the test config, and the result of the most recent grove invocation.
type World struct {
	tmpDir     string // root of all per-scenario temp data
	srcDir     string // $tmpDir/src  — seeded repo clones
	groveRoot  string // $tmpDir/groves — where groves land
	configPath string // $tmpDir/config.toml — written to GROVE_CONFIG
	cacheDir   string // $tmpDir/cache — written to GROVE_CACHE_DIR

	nextCwd    string            // overrides run() cwd for the next call only
	lastResult cmdResult         // output of the most recent grove invocation
	saved      map[string]string // named captures from "and capture as"
}

type cmdResult struct {
	stdout   string
	stderr   string
	exitCode int
}

// setup allocates temp directories and writes the per-scenario config file.
func (w *World) setup() error {
	tmp, err := os.MkdirTemp("", "grove-acceptance-*")
	if err != nil {
		return err
	}
	w.tmpDir = tmp
	w.srcDir = filepath.Join(tmp, "src")
	w.groveRoot = filepath.Join(tmp, "groves")
	w.configPath = filepath.Join(tmp, "config.toml")
	w.cacheDir = filepath.Join(tmp, "cache")
	w.saved = map[string]string{}

	for _, d := range []string{w.srcDir, w.groveRoot, w.cacheDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	// A minimal .gitconfig so git never falls back to the host's global config.
	gitcfg := "[user]\n\temail = grove-test@grove.internal\n\tname = Grove Test\n[init]\n\tdefaultBranch = main\n"
	if err := os.WriteFile(filepath.Join(tmp, ".gitconfig"), []byte(gitcfg), 0o644); err != nil {
		return err
	}

	cfg := fmt.Sprintf(
		"root = %q\ngrove_root = %q\nbranch_prefix = \"grove\"\nindex_ttl_secs = 0\nagent_file = \"AGENTS.md\"\nskills_dir = \".claude/skills\"\n",
		w.srcDir, w.groveRoot,
	)
	return os.WriteFile(w.configPath, []byte(cfg), 0o644)
}

// teardown removes all per-scenario temp data.
func (w *World) teardown() {
	if w.tmpDir != "" {
		_ = os.RemoveAll(w.tmpDir)
		w.tmpDir = ""
	}
}

// run executes the grove binary, capturing stdout/stderr/exitCode into
// w.lastResult. It always returns nil — assertions on the result are the
// responsibility of Then step functions, not this helper.
func (w *World) run(args ...string) error {
	cmd := exec.Command(groveBin, args...)
	cmd.Env = append(filteredEnv(),
		"GROVE_CONFIG="+w.configPath,
		"GROVE_CACHE_DIR="+w.cacheDir,
		"HOME="+w.tmpDir,
	)
	cwd := w.nextCwd
	if cwd == "" {
		cwd = w.tmpDir
	}
	cmd.Dir = cwd
	w.nextCwd = ""

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	}
	w.lastResult = cmdResult{
		stdout:   strings.TrimSpace(stdout.String()),
		stderr:   strings.TrimSpace(stderr.String()),
		exitCode: code,
	}
	return nil
}

// filteredEnv returns the host environment with any GROVE_* vars stripped so
// the host developer's own grove config can never leak into test scenarios.
func filteredEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GROVE_") {
			continue
		}
		env = append(env, e)
	}
	return env
}

// gitRun executes git in dir with the test HOME so only our .gitconfig is read.
func (w *World) gitRun(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(filteredEnv(), "HOME="+w.tmpDir)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %v in %s: %s", args, dir, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// groveDir returns the absolute path of a grove by slug.
func (w *World) groveDir(slug string) string {
	return filepath.Join(w.groveRoot, slug)
}

// repoClone returns the absolute path of the seeded clone for name.
// Repos are seeded at srcDir/github.com/test/<name>.
func (w *World) repoClone(name string) string {
	return filepath.Join(w.srcDir, "github.com", "test", name)
}

// worktreePath resolves "slug/repo" notation to an absolute path.
func (w *World) worktreePath(slugSlashRepo string) string {
	return filepath.Join(w.groveRoot, filepath.FromSlash(slugSlashRepo))
}

// profileDir returns where grove stores profiles given GROVE_CONFIG.
// Mirrors config.ProfilesDir(): dir(GROVE_CONFIG)/profiles/<name>.
func (w *World) profileDir(name string) string {
	return filepath.Join(filepath.Dir(w.configPath), "profiles", name)
}

// stdoutJSONArray unmarshals the last stdout as a JSON array of objects.
func (w *World) stdoutJSONArray() ([]map[string]interface{}, error) {
	var v []map[string]interface{}
	if err := json.Unmarshal([]byte(w.lastResult.stdout), &v); err != nil {
		return nil, fmt.Errorf("stdout is not a JSON array: %v\nstdout: %s", err, w.lastResult.stdout)
	}
	return v, nil
}
