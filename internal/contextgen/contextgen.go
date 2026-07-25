// Package contextgen renders the agent-facing context for a grove.
//
// The design rule: route, don't inline. Concatenating every repo's own agent
// file produces a doc that is mostly irrelevant and stale on arrival. The
// generated file says what is here, what depends on what, and where the
// per-repo instructions live — pointers, lazily followed.
//
// Rendered files are write-only: grove never reads them back. Anything
// hand-written belongs in CONTEXT.md / LOCAL.md, which this file routes to.
package contextgen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jossmoff/grove/internal/config"
	"github.com/jossmoff/grove/internal/manifest"
)

// Skill is a per-repo agent skill, namespaced by its owning repo — two repos
// may each define `test`, and the workspace-level namespace must not be
// ambiguous.
type Skill struct {
	Qualified string // polywit:test
	Path      string // polywit/.claude/skills/test/SKILL.md
}

// DiscoverSkills finds skills in each repo's skills dir.
func DiscoverSkills(cfg config.Config, groveDir string, m *manifest.Manifest) []Skill {
	var out []Skill
	for _, r := range m.Repos {
		dir := filepath.Join(groveDir, r.Dir, cfg.SkillsDir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skill := filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(skill); err == nil {
				out = append(out, Skill{
					Qualified: r.Dir + ":" + e.Name(),
					Path:      filepath.ToSlash(filepath.Join(r.Dir, cfg.SkillsDir, e.Name(), "SKILL.md")),
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Qualified < out[j].Qualified })
	return out
}

// agentFileFor locates a repo's own instruction file, if present. The list is
// deliberately generous: different agent CLIs read different names, and this
// lookup is the whole integration surface with them.
func agentFileFor(cfg config.Config, groveDir, dir string) (string, bool) {
	for _, name := range []string{cfg.AgentFile, "CLAUDE.md", "AGENTS.md", ".cursorrules"} {
		if _, err := os.Stat(filepath.Join(groveDir, dir, name)); err == nil {
			return filepath.ToSlash(filepath.Join(dir, name)), true
		}
	}
	return "", false
}

// Render produces the grove-level agent file.
func Render(cfg config.Config, groveDir string, m *manifest.Manifest) string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }

	w("# Grove: %s", m.Slug)
	w("")
	if m.Task != "" {
		w("**Task:** %s", m.Task)
		w("")
	}
	w("This directory is a grove: several independent repositories checked out")
	w("together so one task can span them. Each subdirectory is a git worktree of")
	w("a *separate* repo — they do not share history, and a commit in one is")
	w("unrelated to a commit in another.")
	w("")

	w("## Repositories")
	w("")
	w("| Directory | Role | Branch | Instructions |")
	w("|---|---|---|---|")
	for _, r := range m.Repos {
		branch := m.Branch
		if r.Role == manifest.RoleRead {
			branch = "detached (read-only)"
		}
		instr := "—"
		if p, ok := agentFileFor(cfg, groveDir, r.Dir); ok {
			instr = "`" + p + "`"
		}
		w("| `%s` | %s | `%s` | %s |", r.Dir, r.Role, branch, instr)
	}
	w("")
	w("Read a repo's own instructions **when you first touch that repo**, not up")
	w("front. They are not reproduced here on purpose — this file would go stale")
	w("the moment they change.")
	w("")

	var writes, reads []string
	for _, r := range m.Repos {
		if r.Role == manifest.RoleWrite {
			writes = append(writes, "`"+r.Dir+"`")
		} else {
			reads = append(reads, "`"+r.Dir+"`")
		}
	}
	w("## Rules")
	w("")
	if len(writes) > 0 {
		w("- **Write here:** %s. These are on branch `%s`. Commit as you go.", strings.Join(writes, ", "), m.Branch)
	}
	if len(reads) > 0 {
		w("- **Do not write here:** %s. Detached checkouts, present for context", strings.Join(reads, ", "))
		w("  only. Read freely. Any edit will be discarded, so do not plan work that")
		w("  depends on changing them — if you find you need to, stop and say so.")
	}
	w("- Never `git checkout` a branch that is checked out in a sibling directory; git will refuse.")
	w("")

	// Dependency direction, from the dependent (depends_on).
	var deps []manifest.Repo
	for _, r := range m.Repos {
		if len(r.DependsOn) > 0 {
			deps = append(deps, r)
		}
	}
	if len(deps) > 0 {
		w("## Dependency direction")
		w("")
		for _, r := range deps {
			quoted := make([]string, len(r.DependsOn))
			for i, d := range r.DependsOn {
				quoted[i] = "`" + d + "`"
			}
			w("- `%s` **depends on** %s.", r.Dir, strings.Join(quoted, ", "))
		}
		w("")
		w("If you change an interface in a dependency that a dependent consumes, you")
		w("cannot simply edit both and call the task done: there is a publish/version")
		w("step between them. Land the upstream change, get a version out, then")
		w("consume it. If that is impossible within this task, say so rather than")
		w("leaving the pair silently inconsistent.")
		w("")
	}

	if skills := DiscoverSkills(cfg, groveDir, m); len(skills) > 0 {
		w("## Skills")
		w("")
		w("Namespaced by owning repo (two repos may each define a skill of the same")
		w("name). Read one when relevant to the repo it belongs to.")
		w("")
		for _, s := range skills {
			w("- `%s` → `%s`", s.Qualified, s.Path)
		}
		w("")
	}

	// Route to notes when present. Notes are hand-written; this file only
	// points at them.
	var notes []string
	for _, f := range []string{"CONTEXT.md", "LOCAL.md", "PLAN.md"} {
		if _, err := os.Stat(filepath.Join(groveDir, f)); err == nil {
			notes = append(notes, f)
		}
	}
	if len(notes) > 0 {
		w("## Notes")
		w("")
		if has(notes, "CONTEXT.md") {
			w("- `CONTEXT.md` — durable knowledge about this repo combination. Read it first.")
		}
		if has(notes, "LOCAL.md") {
			w("- `LOCAL.md` — machine-specific paths and quirks.")
		}
		if has(notes, "PLAN.md") {
			w("- `PLAN.md` — the task's working memory. Keep it updated as you go; it")
			w("  survives context compaction when this conversation does not.")
		}
		w("")
	}

	w("---")
	w("<!-- generated by grove; regenerate with `grove sync` -->")
	return b.String()
}

func has(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// WriteAll renders AGENTS.md (always) and seeds PLAN.md (only if absent —
// PLAN.md is yours, not the tool's).
func WriteAll(cfg config.Config, groveDir string, m *manifest.Manifest) error {
	if err := os.WriteFile(filepath.Join(groveDir, cfg.AgentFile), []byte(Render(cfg, groveDir, m)), 0o644); err != nil {
		return err
	}
	plan := filepath.Join(groveDir, "PLAN.md")
	if _, err := os.Stat(plan); os.IsNotExist(err) {
		seed := fmt.Sprintf("# Plan: %s\n\n%s\n\n## Steps\n\n- [ ] \n\n## Notes\n\n", m.Slug, m.Task)
		return os.WriteFile(plan, []byte(seed), 0o644)
	}
	return nil
}
