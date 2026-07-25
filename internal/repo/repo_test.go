package repo

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		remote  string
		want    Canonical
		wantErr bool
	}{
		{
			// The case a naive url.Parse silently mangles.
			name:   "scp style",
			remote: "git@github.com:joss/polywit.git",
			want:   Canonical{Host: "github.com", Owner: "joss", Name: "polywit"},
		},
		{
			name:   "https",
			remote: "https://github.com/joss/byol.git",
			want:   Canonical{Host: "github.com", Owner: "joss", Name: "byol"},
		},
		{
			name:   "no .git suffix",
			remote: "https://github.com/joss/jvmv",
			want:   Canonical{Host: "github.com", Owner: "joss", Name: "jvmv"},
		},
		{
			name:   "ssh scheme with port",
			remote: "ssh://git@gitlab.com:2222/joss/yamber.git",
			want:   Canonical{Host: "gitlab.com", Owner: "joss", Name: "yamber"},
		},
		{
			name:   "nested gitlab groups preserved",
			remote: "git@gitlab.com:acme/platform/terraform-networking.git",
			want:   Canonical{Host: "gitlab.com", Owner: "acme/platform", Name: "terraform-networking"},
		},
		{
			name:    "pathless rejected",
			remote:  "git@github.com:polywit.git",
			wantErr: true,
		},
		{
			name:    "local path rejected — no canonical position",
			remote:  "/tmp/origins/polywit",
			wantErr: true,
		},
		{
			name:    "empty rejected",
			remote:  "  ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(tt.remote)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %+v, want error", tt.remote, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.remote, err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Parse(%q) mismatch (-want +got):\n%s", tt.remote, diff)
			}
		})
	}
}

func TestCanonicalPaths(t *testing.T) {
	t.Parallel()
	c := Canonical{Host: "github.com", Owner: "joss", Name: "polywit"}
	if got, want := c.Rel(), "github.com/joss/polywit"; got != want {
		t.Errorf("Rel() = %q, want %q", got, want)
	}
	if got, want := c.Short(), "joss/polywit"; got != want {
		t.Errorf("Short() = %q, want %q", got, want)
	}
}
