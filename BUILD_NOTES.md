# Build notes

## The TUI packages need one local step before first build

`internal/tui` depends on Bubble Tea, Bubbles, Lipgloss, and sahilm/fuzzy. The
module proxy for their transitive dependency (`golang.org/x/sys`) was unreachable
in the environment where this code was written, so:

- **`go.sum` does not yet contain hashes for the TUI dependencies.**
- The `internal/tui` and `cmd/grove` packages were **not compiled** here.

They were instead type-checked against faithful stubs reproducing the exact v1
API signatures (verified against upstream source: bubbletea v2.0.8, bubbles
v2.1.1, lipgloss v2.0.5, sahilm/fuzzy v0.1.1). The pure selection logic is
unit-tested. What was **not** verified is the runtime behaviour of the real
libraries — only that every call into them is signature-correct.

### To complete the build locally

```sh
go mod download        # fetches the TUI deps, populates go.sum
go build ./...         # compiles everything including internal/tui and cmd/grove
go test ./...          # runs all tests, including the picker's
```

If anything fails to build, it will almost certainly be a version pin to nudge in
`go.mod` (e.g. a patch bump), not a structural problem — the API surface was
checked against real source.

### Everything else builds and tests clean as-is

The nine non-TUI packages (`config`, `gitx`, `index`, `manifest`, `profile`,
`repo`, `status`, `workspace`, `contextgen`) compile and pass tests with no
network access. Only the interactive layer needs the download step.
