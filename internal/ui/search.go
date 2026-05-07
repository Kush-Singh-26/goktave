package ui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type SearchBar struct {
	input    textinput.Model
	focused  bool
}

func NewSearchBar() SearchBar {
	ti := textinput.New()
	ti.Placeholder = "Search YouTube Music..."
	ti.Focus()
	ti.CharLimit = 100
	ti.SetWidth(50)
	return SearchBar{
		input:   ti,
		focused: true,
	}
}

func (s *SearchBar) Focus() {
	s.focused = true
	s.input.Focus()
}

func (s *SearchBar) Blur() {
	s.focused = false
	s.input.Blur()
}

func (s *SearchBar) Value() string {
	return s.input.Value()
}

func (s *SearchBar) Update(msg tea.Msg) (SearchBar, tea.Cmd) {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return *s, cmd
}

func (s SearchBar) View() string {
	style := StyleInput.Copy()
	if s.focused {
		style = style.BorderForeground(Terracotta)
	} else {
		style = style.BorderForeground(FgGhost)
	}
	return style.Render(s.input.View())
}
