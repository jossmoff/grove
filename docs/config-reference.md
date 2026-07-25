# Configuration reference

Grove reads three file formats. All are TOML. Unknown keys are errors, not
silently ignored — a typo that changes behaviour is worse than a loud failure.

## `config.toml`

Global configuration. Location: `$GROVE_CONFIG`, else
`$XDG_CONFIG_HOME/grove/config.toml`, else `~/.config/grove/config.toml`. An
absent file means defaults; nothing here is required.

| Key | Default | Meaning |
| --- | --- | --- |
| `root` | `~/src` | Where clones live: `<root>/<host>/<owner>/<repo>` |
| `grove_root` | `~/grove` | Where groves are created: `<grove_root>/<slug>/` |
| `branch_prefix` | `grove` | Write branches are `<prefix>/<slug>` |
| `index_ttl_secs` | `30` | How long the repo index cache stays warm |
| `agent_file` | `AGENTS.md` | Filename the generated context is written to |
| `skills_dir` | `.claude/skills` | Per-repo directory scanned for agent skills |

`agent_file` is configurable because different agent CLIs read different
filenames — that lookup is the whole integration surface with them. A leading
`~/` in `root` or `grove_root` is expanded.

```toml
root = "~/code"
grove_root = "~/groves"
index_ttl_secs = 60
```

## `profile.toml`

A durable, versioned repo-set template at `<config>/profiles/<name>/profile.toml`.
Versioned because profiles are long-lived and hand-edited; a schema bump must
fail loudly rather than silently misparse curated data.

```toml
version = 1
description = "Full-stack service cluster"

repos = [
  "client-sdk:read",                    # shorthand: NAME[:ROLE], role defaults to read
  "backend:write",
  { at = "gitlab.com/acme/schemas",     # longhand, for anything beyond name+role
    role = "read",
    base = "v2.1",                       # pin to a tag/branch/SHA
    subtree = "libs/schemas",            # (planned) sparse-checkout a subdirectory
    dir = "schemas",                     # directory name in the grove
    depends_on = ["client-sdk"] },       # declared from the dependent
]
```

**Shorthand or longhand, same key.** Use `"name:role"` for the common case; use a
table when you need `base`, `dir`, `subtree`, or `depends_on`. The two forms may
be mixed freely in one array.

**`depends_on` is declared from the dependent.** If `schemas` consumes `client-sdk`,
`schemas` lists `client-sdk` — not the other way around. A new consumer edits its own
entry, never its provider's. This drives the dependency-direction section of the
generated `AGENTS.md`.

### Profile notes

Beside `profile.toml`:

- `CONTEXT.md` — durable notes about the *combination* (e.g. "changes flow
  shared → api → web"). Copied into every grove; exported with the profile.
- `LOCAL.md` — machine-specific notes (key paths, ports, credentials). Copied
  into every grove; **never** exported.

## `grove.toml`

The per-grove manifest — grove writes this, you generally don't. It is a fully
resolved snapshot; see [Concepts](concepts.md#manifest). Fields: `slug`,
`branch`, `created`, `task`, `profile` (provenance only), and a `repos` array of
`{ at, role, dir, base }`.

Note what is *absent*: no `dirty`, no `ahead`/`behind`, no `status`. Those are
derived on read, never stored. A test fails the build if anyone adds them.

## Exporting a profile

`grove profile export` produces a single self-contained TOML document with
`CONTEXT.md` inlined as a `context` field, suitable for committing to a repo,
pasting in chat, or piping to a teammate:

```sh
grove profile export fullstack > fullstack.toml
grove profile import fullstack fullstack.toml
```

`LOCAL.md` is never included. Locals hold credentials and machine paths, and a
default that can leak is not a default.
