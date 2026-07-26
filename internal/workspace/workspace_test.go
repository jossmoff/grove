// Package workspace_test drives the real lifecycle against real git
// repositories. The things most likely to break are the git interactions, and
// mocking git would test the mock.
package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jossmoff/grove/internal/config"
	"github.com/jossmoff/grove/internal/gitx"
	"github.com/jossmoff/grove/internal/index"
	"github.com/jossmoff/grove/internal/manifest"
	"github.com/jossmoff/grove/internal/workspace"
)

// env builds a scratch world: bare-ish origin repos, canonical clones, and a
// config pointing at them.
type env struct {
	t   *testing.T
	cfg config.Config
}

func newEnv(t *testing.T, repos ...string) *env {
	t.Helper()
	base := t.TempDir()

	cfg := config.Config{
		Root:         filepath.Join(base, "src"),
		GroveRoot:    filepath.Join(base, "groves"),
		BranchPrefix: "grove",
		IndexTTLSecs: 0,
		AgentFile:    "AGENTS.md",
		SkillsDir:    filepath.Join(".claude", "skills"),
	}

	for _, r := range repos {
		origin := filepath.Join(base, "origins", r)
		mustMkdir(t, origin)
		mustGit(t, origin, "init", "-q", "-b", "main")
		mustGit(t, origin, "config", "user.email", "t@t.co")
		mustGit(t, origin, "config", "user.name", "t")
		mustWrite(t, filepath.Join(origin, "README.md"), "# "+r+"\n")
		mustWrite(t, filepath.Join(origin, "CLAUDE.md"), "instructions for "+r+"\n")
		mustWrite(t, filepath.Join(origin, ".claude", "skills", "test", "SKILL.md"), "# test\n")
		mustGit(t, origin, "add", "-A")
		mustGit(t, origin, "commit", "-qm", "init")

		dest := filepath.Join(cfg.Root, "github.com", "joss", r)
		mustMkdir(t, filepath.Dir(dest))
		mustGit(t, base, "clone", "-q", origin, dest)
		mustGit(t, dest, "config", "user.email", "t@t.co")
		mustGit(t, dest, "config", "user.name", "t")
	}
	return &env{t: t, cfg: cfg}
}

func (e *env) entry(name string) index.Entry {
	e.t.Helper()
	p := filepath.Join(e.cfg.Root, "github.com", "joss", name)
	return index.Entry{
		Canonical: "github.com/joss/" + name,
		Name:      "joss/" + name,
		Path:      p,
	}
}

func (e *env) mainClone(name string) string {
	return filepath.Join(e.cfg.Root, "github.com", "joss", name)
}

func (e *env) create(slug string, sel []workspace.Selection) (string, error) {
	return workspace.Create(e.cfg, sel, workspace.CreateOpts{Slug: slug})
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(p))
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitx.Run(dir, args...)
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return out
}

func TestWriteGetsBranchReadGetsDetached(t *testing.T) {
	t.Parallel()
	e := newEnv(t, "polywit", "byol")

	dir, err := e.create("fix", []workspace.Selection{
		{Entry: e.entry("polywit"), Role: manifest.RoleWrite},
		{Entry: e.entry("byol"), Role: manifest.RoleRead},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := mustGit(t, filepath.Join(dir, "polywit"), "rev-parse", "--abbrev-ref", "HEAD"); got != "grove/fix" {
		t.Errorf("write repo branch = %q, want grove/fix", got)
	}
	if got := mustGit(t, filepath.Join(dir, "byol"), "rev-parse", "--abbrev-ref", "HEAD"); got != "HEAD" {
		t.Errorf("read repo = %q, want detached HEAD", got)
	}
}

func TestReadReposLeaveNoBranchesBehind(t *testing.T) {
	// The entire point of the detached design.
	t.Parallel()
	e := newEnv(t, "polywit", "byol")

	if _, err := e.create("fix", []workspace.Selection{
		{Entry: e.entry("polywit"), Role: manifest.RoleWrite},
		{Entry: e.entry("byol"), Role: manifest.RoleRead},
	}); err != nil {
		t.Fatal(err)
	}

	if got := mustGit(t, e.mainClone("byol"), "branch", "--list", "grove/*"); got != "" {
		t.Errorf("read repo grew branches: %q", got)
	}
	if got := mustGit(t, e.mainClone("polywit"), "branch", "--list", "grove/*"); got == "" {
		t.Error("write repo should have a grove/* branch")
	}
}

func TestTwoGrovesCanShareARepo(t *testing.T) {
	// Git refuses one branch in two worktrees; the slug in the branch name is
	// what makes this legal. If this breaks, the tool is single-grove.
	t.Parallel()
	e := newEnv(t, "polywit")

	one, err := e.create("one", []workspace.Selection{{Entry: e.entry("polywit"), Role: manifest.RoleWrite}})
	if err != nil {
		t.Fatal(err)
	}
	two, err := e.create("two", []workspace.Selection{{Entry: e.entry("polywit"), Role: manifest.RoleWrite}})
	if err != nil {
		t.Fatalf("second grove on same repo must work: %v", err)
	}

	if got := mustGit(t, filepath.Join(one, "polywit"), "rev-parse", "--abbrev-ref", "HEAD"); got != "grove/one" {
		t.Errorf("grove one on %q", got)
	}
	if got := mustGit(t, filepath.Join(two, "polywit"), "rev-parse", "--abbrev-ref", "HEAD"); got != "grove/two" {
		t.Errorf("grove two on %q", got)
	}
}

func TestPartialFailureRollsBackCompletely(t *testing.T) {
	t.Parallel()
	e := newEnv(t, "polywit", "byol")
	// Pre-existing branch on the *second* repo: the first will already have
	// been materialised when validation... no — validation is up-front, so
	// nothing should be created at all.
	mustGit(t, e.mainClone("byol"), "branch", "grove/halfway")

	_, err := e.create("halfway", []workspace.Selection{
		{Entry: e.entry("polywit"), Role: manifest.RoleWrite},
		{Entry: e.entry("byol"), Role: manifest.RoleWrite},
	})
	if err == nil {
		t.Fatal("want failure on pre-existing branch")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("err = %v", err)
	}
	if _, statErr := os.Stat(e.cfg.GroveDir("halfway")); !os.IsNotExist(statErr) {
		t.Error("grove dir left behind after failed create")
	}
	if got := mustGit(t, e.mainClone("polywit"), "branch", "--list", "grove/halfway"); got != "" {
		t.Errorf("stray branch left on first repo: %q", got)
	}
}

func TestDirCollisionRejectedBeforeMutation(t *testing.T) {
	t.Parallel()
	e := newEnv(t, "polywit")
	// Second entry with the same leaf under a different owner.
	dup := filepath.Join(e.cfg.Root, "gitlab.com", "acme", "polywit")
	mustMkdir(t, filepath.Dir(dup))
	mustGit(t, e.cfg.Root, "clone", "-q", e.mainClone("polywit"), dup)

	_, err := e.create("dupe", []workspace.Selection{
		{Entry: e.entry("polywit"), Role: manifest.RoleWrite},
		{Entry: index.Entry{Canonical: "gitlab.com/acme/polywit", Name: "acme/polywit", Path: dup}, Role: manifest.RoleRead},
	})
	if err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("want dir collision error, got %v", err)
	}
	if _, statErr := os.Stat(e.cfg.GroveDir("dupe")); !os.IsNotExist(statErr) {
		t.Error("grove dir created despite collision")
	}
}

func TestFinishRefusesDirtyThenForces(t *testing.T) {
	t.Parallel()
	e := newEnv(t, "polywit")
	dir, err := e.create("fix", []workspace.Selection{{Entry: e.entry("polywit"), Role: manifest.RoleWrite}})
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "polywit", "scratch.txt"), "wip")

	err = workspace.Finish(e.cfg, dir, workspace.FinishOpts{})
	if err == nil || !strings.Contains(err.Error(), "uncommitted") {
		t.Fatalf("want uncommitted refusal, got %v", err)
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Fatal("finish must not have removed anything on refusal")
	}

	if err := workspace.Finish(e.cfg, dir, workspace.FinishOpts{Force: true}); err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Error("grove dir survives --force finish")
	}
	// Worktree must be pruned from the main clone, not merely orphaned.
	wt := mustGit(t, e.mainClone("polywit"), "worktree", "list")
	if n := len(strings.Split(wt, "\n")); n != 1 {
		t.Errorf("worktree list has %d entries after finish, want 1:\n%s", n, wt)
	}
}

func TestFinishKeepsBranches(t *testing.T) {
	// Work may be pushed but unmerged; deleting the branch would lose it.
	t.Parallel()
	e := newEnv(t, "polywit")
	dir, err := e.create("fix", []workspace.Selection{{Entry: e.entry("polywit"), Role: manifest.RoleWrite}})
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.Finish(e.cfg, dir, workspace.FinishOpts{Force: true}); err != nil {
		t.Fatal(err)
	}
	if got := mustGit(t, e.mainClone("polywit"), "branch", "--list", "grove/fix"); !strings.Contains(got, "grove/fix") {
		t.Errorf("branch deleted by finish: %q", got)
	}
}

func TestContextRoutesAndNamespacesSkills(t *testing.T) {
	t.Parallel()
	e := newEnv(t, "polywit", "byol")
	dir, err := e.create("fix", []workspace.Selection{
		{Entry: e.entry("polywit"), Role: manifest.RoleWrite},
		{Entry: e.entry("byol"), Role: manifest.RoleRead},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	md := string(data)

	if !strings.Contains(md, "polywit/CLAUDE.md") {
		t.Error("should route to the per-repo file")
	}
	if strings.Contains(md, "instructions for polywit") {
		t.Error("must not inline repo instructions")
	}
	for _, s := range []string{"polywit:test", "byol:test"} {
		if !strings.Contains(md, s) {
			t.Errorf("missing namespaced skill %s", s)
		}
	}
}

func TestManifestHoldsNoDerivedState(t *testing.T) {
	// Guards the central design rule: if someone adds dirty/status fields to
	// the manifest, this fails.
	t.Parallel()
	e := newEnv(t, "polywit")
	dir, err := e.create("fix", []workspace.Selection{{Entry: e.entry("polywit"), Role: manifest.RoleWrite}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, manifest.FileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"dirty", "ahead", "behind", "status", "last_commit", "pushed"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("manifest persists derived state %q:\n%s", forbidden, data)
		}
	}
}

func TestBasePinsAReadRepo(t *testing.T) {
	t.Parallel()
	e := newEnv(t, "byol")
	// Tag the first commit, then advance main.
	main := e.mainClone("byol")
	mustGit(t, main, "tag", "v1")
	mustWrite(t, filepath.Join(main, "new.txt"), "later work\n")
	mustGit(t, main, "add", "-A")
	mustGit(t, main, "commit", "-qm", "later")

	dir, err := e.create("pinned", []workspace.Selection{
		{Entry: e.entry("byol"), Role: manifest.RoleRead, Base: "v1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The pinned checkout must not contain the later file.
	if _, err := os.Stat(filepath.Join(dir, "byol", "new.txt")); !os.IsNotExist(err) {
		t.Error("base pin ignored: checkout contains post-tag work")
	}
	// And the manifest stores what was asked for, not a SHA.
	m, err := manifest.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Repos[0].Base != "v1" {
		t.Errorf("manifest base = %q, want the requested ref v1", m.Repos[0].Base)
	}
}

func TestDiscoverWalksUp(t *testing.T) {
	t.Parallel()
	e := newEnv(t, "polywit")
	dir, err := e.create("fix", []workspace.Selection{{Entry: e.entry("polywit"), Role: manifest.RoleWrite}})
	if err != nil {
		t.Fatal(err)
	}
	// From deep inside a repo inside the grove.
	deep := filepath.Join(dir, "polywit", ".claude", "skills")
	got, ok := manifest.Discover(deep)
	if !ok {
		t.Fatal("Discover failed from inside the grove")
	}
	if got != dir {
		t.Errorf("Discover = %q, want %q", got, dir)
	}
	if _, ok := manifest.Discover(t.TempDir()); ok {
		t.Error("Discover found a grove where there is none")
	}
}
