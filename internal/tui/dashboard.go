package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jossmoff/grove/internal/status"
)

// dashboardModel is a read-only browser over groves: a left list, a right
// detail pane. Like the picker it targets bubbletea v1 and is verified by
// signature, not compiled here. See BUILD_NOTES.md.
type dashboardModel struct {
	groves []status.GroveStatus
	cursor int
	quit   bool
}

var (
	dashDirty = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dashClean = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dashHead  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	dashLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	dashPanel = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dashMiss  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func (m dashboardModel) Init() tea.Cmd { return nil }

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "q", "esc", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.groves)-1 {
			m.cursor++
		}
	}
	return m, nil
}

func (m dashboardModel) View() tea.View {
	if len(m.groves) == 0 {
		return tea.NewView(dashLabel.Render("No groves. Run `grove new <slug>` to create one.") + "\n")
	}

	var left strings.Builder
	left.WriteString(dashHead.Render("groves") + "\n\n")
	for i, g := range m.groves {
		dot := dashClean.Render("○")
		if g.DirtyRepos() > 0 {
			dot = dashDirty.Render("●")
		}
		name := g.Slug
		if i == m.cursor {
			name = lipgloss.NewStyle().Background(lipgloss.Color("236")).Bold(true).Render(" " + name + " ")
		}
		left.WriteString(fmt.Sprintf("%s %s\n", dot, name))
		left.WriteString(dashLabel.Render(fmt.Sprintf("  %d repos", len(g.Repos))) + "\n")
	}

	g := m.groves[m.cursor]
	var right strings.Builder
	right.WriteString(dashHead.Render(g.Slug) + "\n\n")
	right.WriteString(dashLabel.Render("branch  ") + dashPanel.Render(g.Branch) + "\n")
	if g.Task != "" {
		right.WriteString(dashLabel.Render("task    ") + g.Task + "\n")
	}
	right.WriteString(dashLabel.Render("path    ") + dashLabel.Render(g.Dir) + "\n\n")
	for _, r := range g.Repos {
		roleStyle := dashClean
		if r.Role == "read" {
			roleStyle = dashPanel
		}
		sum := r.Summary()
		sumStyle := dashLabel
		switch {
		case sum == "missing":
			sumStyle = dashMiss
		case sum != "clean":
			sumStyle = dashDirty
		}
		right.WriteString(fmt.Sprintf("%-28s %s  %s\n",
			r.At, roleStyle.Render(string(r.Role)), sumStyle.Render(sum)))
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(30).Render(left.String()),
		right.String(),
	)
	return tea.NewView(body + "\n" + help("↑↓", "move", "q", "quit") + "\n")
}

// ShowDashboard runs the interactive grove browser.
func ShowDashboard(groves []status.GroveStatus) error {
	_, err := tea.NewProgram(dashboardModel{groves: groves}).Run()
	return err
}
