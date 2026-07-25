# CLI reference

Two conventions run through the whole surface:

- **stdout is the answer; stderr is the narration.** Each command prints exactly
  one machine-readable thing to stdout (a path, a listing). Progress and
  confirmations go to stderr. This is what makes `cd "$(grove new x -w backend)"`
  work.
- **`[slug]` is optional everywhere.** Commands discover the enclosing grove by
  walking up for `grove.toml`, the way git walks up for `.git`. Naming the grove
  you are standing in is unnecessary.

The surface is flat verbs on the primary noun (grove), and nested noun-verb for
secondary nouns (`profile`, `repo`).

## Grove lifecycle

### `grove new <slug>`

Create a grove. Prints its path.

With no `-w`/`-r` flags, opens an **interactive picker**: fuzzy-filter as you
type (any character filters; `Tab` cycles a repo through unselected → write →
read; `Enter` confirms). This is the picker's reason to exist — the role choice
is ternary, which a flag list makes tedious and a plain multi-select cannot
express.

| Flag | Meaning |
| --- | --- |
| `-p, --profile NAME` | Build from a saved profile |
| `-t, --task TEXT` | One-line statement of intent |
| `-w, --write REPO` | Add REPO as write (repeatable) |
| `-r, --read REPO` | Add REPO as read (repeatable); `REPO@ref` to pin |
| `--no-fetch` | Skip the default fetch of read repos |

Read repos are fetched before checkout by default, because a stale reference
hands the agent an interface that no longer exists. Write repos are not fetched —
branching from what you last pulled is often deliberate.

```sh
grove new add-payments -w backend -r client-sdk -r infra@v1.4 -t "Extend the payments API"
grove new audit -p fullstack
```

### `grove ls`

List groves and their derived status. `--json` for machine consumption;
`-i, --interactive` opens a dashboard to browse groves and drill into per-repo
status.

### `grove path [slug]`

Print a grove's path and nothing else. Wrapped by `gcd` from the shell
integration.

### `grove sync [slug]`

Regenerate `AGENTS.md` from the manifest. Run it after hand-editing
`depends_on` or a profile's notes.

### `grove finish [slug]`

Tear down: remove worktrees, prune, delete the directory. Refuses on uncommitted
or unpushed work unless `-f, --force`. Never deletes branches. Prints the parent
directory so a shell wrapper can escape the deleted grove.

## Profiles

### `grove profile save <name>`

Save the current grove's repo set as a reusable profile. Run it from inside a
grove you liked.

### `grove profile ls`

List profiles.

### `grove profile edit <name>`

Open a profile's `profile.toml` in `$EDITOR` (prints the path if unset).

### `grove profile rm <name>`

Delete a profile.

### `grove profile export <name>`

Print a profile as one shareable TOML document, inlining `CONTEXT.md`. **Never**
inlines `LOCAL.md`.

```sh
grove profile export verification > verification.toml
```

### `grove profile import <name> <file|->`

Import a profile from an exported file, or `-` for stdin.

```sh
grove profile import verification verification.toml
curl -s https://example.com/v.toml | grove profile import verification -
```

## Repos

### `grove repo clone <url>`

Clone a remote into its canonical position `<root>/<host>/<owner>/<repo>`.
Handles scp-style (`git@host:owner/repo.git`) and https remotes. Prints the
destination.

### `grove repo ls`

List indexed repos. `--refresh` bypasses the TTL cache; `-l, --long` includes
remotes.

## Meta

### `grove init <bash|zsh|fish>`

Print shell integration. `eval "$(grove init zsh)"` defines `gcd` and `gfin`.

### `grove config`

Print the effective configuration. `--path` prints the config file location
instead. Initialise with:

```sh
grove config > "$(grove config --path)"
```
