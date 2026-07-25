package tui

import "testing"

// These tests exercise the pure selection logic. Like the rest of the package
// they compile only where the Bubble Tea dependency tree is available; run them
// locally with `go test ./internal/tui/`.

func TestPickStateCyclesThroughThreeStates(t *testing.T) {
	// The whole reason the picker is a TUI and not an fzf multi-select:
	// selection is ternary, not binary.
	s := stateNone
	if s = s.next(); s != stateWrite {
		t.Fatalf("none.next() = %v, want write", s)
	}
	if s = s.next(); s != stateRead {
		t.Fatalf("write.next() = %v, want read", s)
	}
	if s = s.next(); s != stateNone {
		t.Fatalf("read.next() = %v, want none (must wrap)", s)
	}
}

func TestPickStateCyclesBackward(t *testing.T) {
	if got := stateNone.prev(); got != stateRead {
		t.Fatalf("none.prev() = %v, want read", got)
	}
	if got := stateRead.prev(); got != stateWrite {
		t.Fatalf("read.prev() = %v, want write", got)
	}
}

func TestSelectionSurvivesRefilter(t *testing.T) {
	// Mark a repo, then filter it out of view and back. The pick must persist,
	// because states are indexed against entries, not against visible rows.
	m := newPicker(entryStub{}.entries())
	if len(m.states) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(m.states))
	}
	m.states[1] = stateWrite // mark the middle repo

	// Filter to something matching only entry 0, then clear.
	m.filter.SetValue("zzz-no-match")
	m.refilter()
	m.filter.SetValue("")
	m.refilter()

	if m.states[1] != stateWrite {
		t.Error("selection lost across refilter")
	}
	if w, _ := m.counts(); w != 1 {
		t.Errorf("write count = %d, want 1", w)
	}
}

func TestResultOrdersWritesFirst(t *testing.T) {
	m := newPicker(entryStub{}.entries())
	m.states[0] = stateRead
	m.states[1] = stateWrite
	got := m.result()
	if len(got) != 2 {
		t.Fatalf("result len = %d, want 2", len(got))
	}
	if got[0].Role != "write" {
		t.Errorf("first result role = %v, want write", got[0].Role)
	}
}
