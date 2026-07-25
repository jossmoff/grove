package tui

import "github.com/jossmoff/grove/internal/index"

type entryStub struct{}

func (entryStub) entries() []index.Entry {
	return []index.Entry{
		{Canonical: "github.com/joss/polywit", Name: "joss/polywit"},
		{Canonical: "github.com/joss/byol", Name: "joss/byol"},
		{Canonical: "github.com/joss/jvmv", Name: "joss/jvmv"},
	}
}
