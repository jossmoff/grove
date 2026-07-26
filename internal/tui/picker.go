// Package tui holds the interactive Bubble Tea surfaces.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"

	"github.com/jossmoff/grove/internal/index"
	"github.com/jossmoff/grove/internal/manifest"
)

// Pick is one repo chosen in the picker, with the role the user assigned.
type Pick struct {
	Entry index.Entry
	Role  manifest.Role
}

// pickState is a repo's ternary selection state. This is the whole reason a
// real TUI beats a plain fuzzy multi-select like fzf: the choice is not
// on/off, it is unselected / write / read.
type pickState int

const (
	stateNone pickState = iota
	stateWrite
	stateRead
)

func (s pickState) next() pickState {
	if s == stateRead {
		return stateNone
	}
	return s + 1
}

func (s pickState) prev() pickState {
	if s == stateNone {
		return stateRead
	}
	return s - 1
}

// --- styles ---------------------------------------------------------------

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	styleCount    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	stylePrompt   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
styleWrite    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	styleRead     = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	styleDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleMatch    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("236"))
	styleHelp     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styleKey      = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
)

// row is one visible line: an index into the entries slice plus the fuzzy
// match indexes for highlighting.
type row struct {
	entry   int
	matched []int
}

type pickerModel struct {
	entries []index.Entry
	states  []pickState // parallel to entries; survives filtering
	filter  textinput.Model
	rows    []row // current visible set, in display order
	cursor  int   // index into rows
	height  int   // visible list rows
	quit    bool
	confirm bool
}

func newPicker(entries []index.Entry) pickerModel {
	ti := textinput.New()
	ti.Placeholder = "filter…"
	ti.Prompt = ""
	ti.Focus()

	m := pickerModel{
		entries: entries,
		states:  make([]pickState, len(entries)),
		filter:  ti,
		height:  14,
	}
	m.refilter()
	return m
}

// refilter rebuilds the visible rows from the current filter text. An empty
// filter keeps the index's own recency order; a non-empty one ranks by fuzzy
// score. Crucially it operates on entry indexes, so selection state (held in
// m.states, parallel to m.entries) is never disturbed by filtering.
func (m *pickerModel) refilter() {
	q := m.filter.Value()
	if q == "" {
		m.rows = make([]row, len(m.entries))
		for i := range m.entries {
			m.rows[i] = row{entry: i}
		}
	} else {
		names := make([]string, len(m.entries))
		for i, e := range m.entries {
			names[i] = e.Canonical
		}
		matches := fuzzy.Find(q, names)
		m.rows = make([]row, len(matches))
		for i, mt := range matches {
			m.rows[i] = row{entry: mt.Index, matched: mt.MatchedIndexes}
		}
	}
	if m.cursor >= len(m.rows) {
		m.cursor = max(0, len(m.rows)-1)
	}
}

func (m pickerModel) Init() tea.Cmd { return textinput.Blink }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch km.String() {
	case "ctrl+c", "esc":
		m.quit = true
		return m, tea.Quit

	case "enter":
		// Confirm only if something is selected — an empty selection would
		// create an empty grove.
		for _, s := range m.states {
			if s != stateNone {
				m.confirm = true
				return m, tea.Quit
			}
		}
		return m, nil

	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "ctrl+n":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
		return m, nil

	case "tab":
		// Cycle the highlighted repo's role forward. Tab, not space, because
		// space is a legitimate filter character.
		if len(m.rows) > 0 {
			e := m.rows[m.cursor].entry
			m.states[e] = m.states[e].next()
		}
		return m, nil

	case "shift+tab":
		if len(m.rows) > 0 {
			e := m.rows[m.cursor].entry
			m.states[e] = m.states[e].prev()
		}
		return m, nil
	}

	// Anything else feeds the fuzzy filter.
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.refilter()
	return m, cmd
}

func (m pickerModel) counts() (w, r int) {
	for _, s := range m.states {
		switch s {
		case stateWrite:
			w++
		case stateRead:
			r++
		}
	}
	return
}

func (m pickerModel) View() tea.View {
	var b strings.Builder

	nw, nr := m.counts()
	b.WriteString(styleTitle.Render("grove") + "  ")
	b.WriteString(styleCount.Render(fmt.Sprintf("%d repos · %d write · %d read", len(m.rows), nw, nr)))
	b.WriteByte('\n')
	b.WriteString(stylePrompt.Render("› ") + m.filter.View())
	b.WriteString("\n\n")

	// Scroll window around the cursor.
	start := 0
	if m.cursor >= m.height {
		start = m.cursor - m.height + 1
	}
	end := min(start+m.height, len(m.rows))

	if len(m.rows) == 0 {
		b.WriteString(styleDim.Render("  no matches"))
		b.WriteByte('\n')
	}

	for i := start; i < end; i++ {
		r := m.rows[i]
		e := m.entries[r.entry]
		st := m.states[r.entry]

		marker, mstyle := "   ", styleDim
		roleLabel := ""
		switch st {
		case stateWrite:
			marker, mstyle, roleLabel = " ● ", styleWrite, "write"
		case stateRead:
			marker, mstyle, roleLabel = " ○ ", styleRead, "read"
		}

		name := highlight(e.Name, e.Canonical, r.matched)
		line := fmt.Sprintf("%s%-40s %s", mstyle.Render(marker), name, styleDim.Render(fmt.Sprintf("%-6s%s", roleLabel, ago(e.LastCommit))))
		if i == m.cursor {
			line = styleSelected.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(help(
		"tab", "cycle role",
		"f-type", "fuzzy filter",
		"↑↓", "move",
		"enter", "confirm",
		"esc", "cancel",
	))
	return tea.NewView(b.String())
}

// highlight bolds the fuzzy-matched characters. matched indexes are into the
// canonical string; we render the name (owner/repo), so we offset by the host
// prefix length that Name drops.
func highlight(name, canonical string, matched []int) string {
	if len(matched) == 0 {
		return name
	}
	offset := len(canonical) - len(name)
	set := make(map[int]bool, len(matched))
	for _, idx := range matched {
		set[idx-offset] = true
	}
	var b strings.Builder
	for i, ch := range name {
		if set[i] {
			b.WriteString(styleMatch.Render(string(ch)))
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func help(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, styleKey.Render(pairs[i])+" "+styleHelp.Render(pairs[i+1]))
	}
	return strings.Join(parts, styleHelp.Render("  ·  "))
}

// result collects the confirmed picks, writes first for stable ordering.
func (m pickerModel) result() []Pick {
	var picks []Pick
	for i, s := range m.states {
		switch s {
		case stateWrite:
			picks = append(picks, Pick{Entry: m.entries[i], Role: manifest.RoleWrite})
		case stateRead:
			picks = append(picks, Pick{Entry: m.entries[i], Role: manifest.RoleRead})
		}
	}
	sort.SliceStable(picks, func(i, j int) bool {
		return picks[i].Role == manifest.RoleWrite && picks[j].Role == manifest.RoleRead
	})
	return picks
}

// PickRepos runs the interactive picker. Returns (nil, nil) if the user
// cancelled — an empty, non-error result the caller treats as "abort".
func PickRepos(entries []index.Entry) ([]Pick, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no repos indexed — try `grove repo clone <url>` first")
	}
	final, err := tea.NewProgram(newPicker(entries)).Run()
	if err != nil {
		return nil, err
	}
	m := final.(pickerModel)
	if m.quit || !m.confirm {
		return nil, nil
	}
	return m.result(), nil
}
