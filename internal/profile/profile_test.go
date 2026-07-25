package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/google/go-cmp/cmp"
	"github.com/jossmoff/grove/internal/manifest"
)

// decode is a test helper for one repos list.
func decode(t *testing.T, src string) (*Profile, error) {
	t.Helper()
	var p Profile
	_, err := toml.Decode(src, &p)
	return &p, err
}

func TestEntryShorthandLonghand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		src     string
		want    []Entry
		wantErr string
	}{
		{
			name: "shorthand defaults to read",
			src:  `version = 1` + "\n" + `repos = ["byol"]`,
			want: []Entry{{At: "byol", Role: manifest.RoleRead}},
		},
		{
			name: "shorthand with role",
			src:  `version = 1` + "\n" + `repos = ["polywit:write", "byol:read"]`,
			want: []Entry{
				{At: "polywit", Role: manifest.RoleWrite},
				{At: "byol", Role: manifest.RoleRead},
			},
		},
		{
			name: "longhand with pinning and deps",
			src: `version = 1
repos = [
  { at = "gitlab.com/acme/schemas", role = "read", base = "v2.1", subtree = "libs/schemas", depends_on = ["byol"] },
]`,
			want: []Entry{{
				At: "gitlab.com/acme/schemas", Role: manifest.RoleRead,
				Base: "v2.1", Subtree: "libs/schemas", DependsOn: []string{"byol"},
			}},
		},
		{
			name: "mixed shorthand and longhand",
			src: `version = 1
repos = ["byol", { at = "polywit", role = "write" }]`,
			want: []Entry{
				{At: "byol", Role: manifest.RoleRead},
				{At: "polywit", Role: manifest.RoleWrite},
			},
		},
		{
			name:    "unknown role rejected",
			src:     `version = 1` + "\n" + `repos = ["byol:reference"]`,
			wantErr: `unknown role "reference"`,
		},
		{
			name:    "unknown key rejected — typos must not be silence",
			src:     `version = 1` + "\n" + `repos = [{ at = "byol", rolle = "read" }]`,
			wantErr: `unknown key "rolle"`,
		},
		{
			name:    "table without at rejected",
			src:     `version = 1` + "\n" + `repos = [{ role = "read" }]`,
			wantErr: `missing required key "at"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := decode(t, tt.src)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.want, got.Repos); diff != "" {
				t.Errorf("repos mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	in := &Profile{
		Description: "verification cluster",
		Repos: []Entry{
			{At: "github.com/joss/polywit", Role: manifest.RoleWrite},
			{At: "github.com/joss/byol", Role: manifest.RoleRead, Base: "v2.1"},
		},
	}
	if err := in.Save(root, "verification"); err != nil {
		t.Fatal(err)
	}

	out, err := Load(root, "verification")
	if err != nil {
		t.Fatal(err)
	}
	if out.Version != Version {
		t.Errorf("Version = %d, want %d", out.Version, Version)
	}
	if diff := cmp.Diff(in.Repos, out.Repos); diff != "" {
		t.Errorf("round trip mismatch (-want +got):\n%s", diff)
	}
}

func TestSaveCreatesNotesButNeverOverwrites(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	p := &Profile{Repos: []Entry{{At: "byol", Role: manifest.RoleRead}}}

	if err := p.Save(root, "v"); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{ContextFile, LocalFile} {
		if _, err := os.Stat(filepath.Join(root, "v", f)); err != nil {
			t.Errorf("%s not created: %v", f, err)
		}
	}

	// Notes are yours, not the tool's.
	marker := "my hand-written knowledge"
	ctx := filepath.Join(root, "v", ContextFile)
	if err := os.WriteFile(ctx, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := p.Save(root, "v"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(ctx)
	if string(data) != marker {
		t.Error("Save overwrote CONTEXT.md — notes must never be clobbered")
	}
}

func TestVersionMismatchRejected(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	dir := filepath.Join(root, "old")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "version = 99\nrepos = [\"byol\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "profile.toml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, "old"); err == nil {
		t.Fatal("want version mismatch error, got nil — silent misparse of durable data")
	}
}
