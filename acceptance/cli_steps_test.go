package acceptance_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
)

func registerCLISteps(sc *godog.ScenarioContext, w *World) {
	// When
	sc.Step(`^I run "([^"]*)"$`, w.iRun)
	sc.Step(`^I run "([^"]*)" inside grove "([^"]*)"$`, w.iRunInsideGrove)
	sc.Step(`^I run "([^"]*)" and capture as "([^"]*)"$`, w.iRunAndCaptureAs)

	// Then — stdout/stderr/exit code
	sc.Step(`^the exit code is 0$`, w.exitCodeIs0)
	sc.Step(`^the exit code is non-zero$`, w.exitCodeIsNonZero)
	sc.Step(`^stdout is empty$`, w.stdoutIsEmpty)
	sc.Step(`^stderr is empty$`, w.stderrIsEmpty)
	sc.Step(`^stdout contains "([^"]*)"$`, w.stdoutContains)
	sc.Step(`^stderr contains "([^"]*)"$`, w.stderrContains)
	sc.Step(`^stdout does not contain "([^"]*)"$`, w.stdoutDoesNotContain)
	sc.Step(`^stdout is exactly the grove path for "([^"]*)"$`, w.stdoutIsGrovePathFor)
	sc.Step(`^stdout is valid JSON$`, w.stdoutIsValidJSON)
	sc.Step(`^the JSON contains slug "([^"]*)"$`, w.jsonContainsSlug)
	sc.Step(`^stdout is the grove root$`, w.stdoutIsGroveRoot)
	sc.Step(`^stdout is a valid absolute path$`, w.stdoutIsValidAbsPath)
}

// registerSteps wires all step definitions to the scenario context.
func registerSteps(sc *godog.ScenarioContext, w *World) {
	registerGitSteps(sc, w)
	registerCLISteps(sc, w)
}

// --- When steps ---

// iRun parses a command string like "grove new fix -w backend" and executes
// it. The leading "grove" is stripped and the grove binary replaces it.
func (w *World) iRun(cmdStr string) error {
	args := strings.Fields(cmdStr)
	if len(args) > 0 && args[0] == "grove" {
		args = args[1:]
	}
	return w.run(args...)
}

// iRunInsideGrove sets the working directory to the grove directory before
// invoking the command. grove commands with no slug argument discover the
// grove by walking up from cwd, the same way git walks up for .git.
func (w *World) iRunInsideGrove(cmdStr, slug string) error {
	w.nextCwd = w.groveDir(slug)
	return w.iRun(cmdStr)
}

// iRunAndCaptureAs runs a command and writes its stdout to a file in tmpDir
// under the given name. Subsequent iRun calls that reference that filename
// will find the file because tmpDir is the default working directory.
func (w *World) iRunAndCaptureAs(cmdStr, saveName string) error {
	if err := w.iRun(cmdStr); err != nil {
		return err
	}
	if w.lastResult.exitCode != 0 {
		return fmt.Errorf("command %q failed (exit %d): %s", cmdStr, w.lastResult.exitCode, w.lastResult.stderr)
	}
	w.saved[saveName] = w.lastResult.stdout
	return os.WriteFile(filepath.Join(w.tmpDir, saveName), []byte(w.lastResult.stdout), 0o644)
}

// --- Then steps ---

func (w *World) exitCodeIs0() error {
	if w.lastResult.exitCode != 0 {
		return fmt.Errorf("exit code = %d, want 0\nstderr: %s\nstdout: %s",
			w.lastResult.exitCode, w.lastResult.stderr, w.lastResult.stdout)
	}
	return nil
}

func (w *World) exitCodeIsNonZero() error {
	if w.lastResult.exitCode == 0 {
		return fmt.Errorf("exit code = 0, want non-zero\nstdout: %s\nstderr: %s",
			w.lastResult.stdout, w.lastResult.stderr)
	}
	return nil
}

func (w *World) stdoutIsEmpty() error {
	if w.lastResult.stdout != "" {
		return fmt.Errorf("stdout not empty: %q", w.lastResult.stdout)
	}
	return nil
}

func (w *World) stderrIsEmpty() error {
	if w.lastResult.stderr != "" {
		return fmt.Errorf("stderr not empty: %q", w.lastResult.stderr)
	}
	return nil
}

func (w *World) stdoutContains(substr string) error {
	if !strings.Contains(w.lastResult.stdout, substr) {
		return fmt.Errorf("stdout does not contain %q\nstdout: %s", substr, w.lastResult.stdout)
	}
	return nil
}

func (w *World) stderrContains(substr string) error {
	if !strings.Contains(w.lastResult.stderr, substr) {
		return fmt.Errorf("stderr does not contain %q\nstderr: %s", substr, w.lastResult.stderr)
	}
	return nil
}

func (w *World) stdoutDoesNotContain(substr string) error {
	if strings.Contains(w.lastResult.stdout, substr) {
		return fmt.Errorf("stdout should not contain %q\nstdout: %s", substr, w.lastResult.stdout)
	}
	return nil
}

// stdoutIsGrovePathFor checks that stdout is exactly the expected grove path
// and nothing else — validating the "stdout is the answer" invariant.
func (w *World) stdoutIsGrovePathFor(slug string) error {
	want := w.groveDir(slug)
	if w.lastResult.stdout != want {
		return fmt.Errorf("stdout = %q, want %q", w.lastResult.stdout, want)
	}
	return nil
}

func (w *World) stdoutIsValidJSON() error {
	if _, err := w.stdoutJSONArray(); err != nil {
		return err
	}
	return nil
}

func (w *World) jsonContainsSlug(slug string) error {
	items, err := w.stdoutJSONArray()
	if err != nil {
		return err
	}
	for _, item := range items {
		if s, ok := item["slug"].(string); ok && s == slug {
			return nil
		}
	}
	return fmt.Errorf("JSON array does not contain an object with slug %q\nstdout: %s", slug, w.lastResult.stdout)
}

// stdoutIsGroveRoot checks that stdout is exactly the grove root directory —
// the value grove finish prints so a shell wrapper can escape the deleted dir.
func (w *World) stdoutIsGroveRoot() error {
	if w.lastResult.stdout != w.groveRoot {
		return fmt.Errorf("stdout = %q, want grove root %q", w.lastResult.stdout, w.groveRoot)
	}
	return nil
}

func (w *World) stdoutIsValidAbsPath() error {
	p := w.lastResult.stdout
	if !filepath.IsAbs(p) {
		return fmt.Errorf("stdout %q is not an absolute path", p)
	}
	return nil
}
