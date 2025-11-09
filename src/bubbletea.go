package src

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var seraphBanner = `
 ____                       _     
/ ___|  ___ _ __ __ _ _ __ | |__  
\___ \ / _ \ '__/ _' | '_ \| '_ \ 
 ___) |  __/ | | (_| | |_) | | | |
|____/ \___|_|  \__,_| .__/|_| |_| 
                     |_|          
`

var (
	primary    = lipgloss.Color("#6C8EEF")
	secondary  = lipgloss.Color("#9ECBFF")
	accent     = lipgloss.Color("#FFD787")
	successCol = lipgloss.Color("#A6E3A1")
	errorCol   = lipgloss.Color("#F38BA8")
	textCol    = lipgloss.Color("#CDD6F4")
	muted      = lipgloss.Color("#7F849C")

	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(primary)
	subtitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(secondary)
	normalStyle    = lipgloss.NewStyle().Foreground(textCol)
	mutedStyle     = lipgloss.NewStyle().Foreground(muted).Italic(true)
	highlightStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	successStyle   = lipgloss.NewStyle().Foreground(successCol)
	errorStyle     = lipgloss.NewStyle().Foreground(errorCol)
	infoStyle      = lipgloss.NewStyle().Foreground(secondary)
)

func InitialMenuModel() MenuModel {
	return MenuModel{
		Choices:      []string{"Add API", "View APIs", "Delete API", "Exit"},
		URLCopied:    false,
		ClipboardErr: "",
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ClipboardMsg:
		m.URLCopied = msg.Success
		if msg.Err != nil {
			m.ClipboardErr = msg.Err.Error()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "c", "C":
			return m, CopyToClipboard(mergeURL)
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Choices)-1 {
				m.Cursor++
			}
		case "enter":
			switch m.Cursor {
			case 0:
				return initialFormModel(), nil
			case 1:
				return initialSelectionModel("view"), nil
			case 2:
				return initialSelectionModel("delete"), nil
			case 3:
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	var b strings.Builder
	b.WriteString(highlightStyle.Render(seraphBanner) + "\n")
	b.WriteString(titleStyle.Render("API Merger") + "\n\n")

	if m.URLCopied {
		b.WriteString(successStyle.Render("URL copied to clipboard") + "\n\n")
	} else if m.ClipboardErr != "" {
		b.WriteString(errorStyle.Render("Copy error: "+m.ClipboardErr) + "\n\n")
	}

	for i, choice := range m.Choices {
		prefix := "  "
		style := normalStyle
		if m.Cursor == i {
			prefix = "› "
			style = highlightStyle
		}
		b.WriteString(style.Render(prefix+choice) + "\n")
	}

	b.WriteString("\n" + mutedStyle.Render("c copy URL • ↑/↓ navigate • enter select"))
	return b.String()
}

func initialSelectionModel(mode string) SelectionModel {
	return SelectionModel{
		Items: GetAPINames(),
		Mode:  mode,
	}
}

func (m SelectionModel) Init() tea.Cmd {
	return nil
}

func (m SelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
		case "esc":
			return InitialMenuModel(), nil
		case "enter":
			if m.Mode == "delete" && len(m.Items) > 0 {
				name := m.Items[m.Cursor]
				ApiConfigsMutex.Lock()
				delete(ApiConfigs, name)
				SaveAPIConfigs(ApiConfigs)
				ApiConfigsMutex.Unlock()
				m.Deleted = name
				m.Items = GetAPINames()
				if m.Cursor >= len(m.Items) {
					m.Cursor = len(m.Items) - 1
				}
			}
		}
	}
	return m, nil
}

func (m SelectionModel) View() string {
	title := "Configured APIs"
	if m.Mode == "delete" {
		title = "Delete API"
	}
	var b strings.Builder
	b.WriteString(subtitleStyle.Render(title) + "\n\n")

	if m.Mode == "delete" && m.Deleted != "" {
		b.WriteString(successStyle.Render("Deleted: "+m.Deleted) + "\n\n")
	}

	if len(m.Items) == 0 {
		empty := "No APIs configured"
		if m.Mode == "view" {
			empty = "No APIs to view"
		}
		b.WriteString(infoStyle.Render(empty) + "\n")
	} else {
		for i, name := range m.Items {
			prefix := "  "
			style := normalStyle
			if m.Cursor == i {
				prefix = "› "
				style = highlightStyle
			}
			b.WriteString(style.Render(prefix+name) + "\n")
			if m.Mode == "view" && m.Cursor == i {
				ApiConfigsMutex.RLock()
				cfg := ApiConfigs[name]
				ApiConfigsMutex.RUnlock()
				b.WriteString("    " + mutedStyle.Render(cfg.URL) + "\n")
			}
		}
	}

	help := "↑/↓ navigate • esc back"
	if m.Mode == "delete" {
		help += " • enter delete"
	}
	b.WriteString("\n" + mutedStyle.Render(help))
	return b.String()
}

func initialFormModel() FormModel {
	fields := []string{"API Name", "Full URL"}
	inputs := make([]textinput.Model, len(fields))

	for i, placeholder := range fields {
		input := textinput.New()
		input.Placeholder = placeholder
		input.Width = 40
		input.TextStyle = normalStyle
		input.Cursor.Style = highlightStyle
		input.Prompt = "> "
		inputs[i] = input
	}

	inputs[0].Focus()
	return FormModel{Inputs: inputs}
}

func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return InitialMenuModel(), nil
		case "tab", "shift+tab", "up", "down", "enter":
			if msg.String() == "enter" && m.Focus == len(m.Inputs)-1 {
				name := strings.ReplaceAll(strings.TrimSpace(m.Inputs[0].Value()), " ", "-")
				urlStr := strings.TrimSpace(m.Inputs[1].Value())

				if name == "" || urlStr == "" {
					m.ErrMsg = "Both name and URL required"
					return m, nil
				}

				parsed, err := url.Parse(urlStr)
				if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
					m.ErrMsg = "Invalid URL"
					return m, nil
				}

				queryParams := make(map[string]interface{})
				for k, vs := range parsed.Query() {
					if len(vs) > 1 {
						queryParams[k] = vs
					} else {
						queryParams[k] = vs[0]
					}
				}

				ApiConfigsMutex.Lock()
				if ApiConfigs == nil {
					ApiConfigs = make(APIConfigs)
				}

				ApiConfigs[name] = APIConfig{
					URL:         fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path),
					Querystring: queryParams,
				}
				SaveAPIConfigs(ApiConfigs)
				ApiConfigsMutex.Unlock()

				m.Success = true
				return InitialMenuModel(), nil
			}

			if msg.String() == "up" || msg.String() == "shift+tab" {
				m.Focus--
			} else {
				m.Focus++
			}

			if m.Focus < 0 {
				m.Focus = len(m.Inputs) - 1
			} else if m.Focus >= len(m.Inputs) {
				m.Focus = 0
			}

			for i := range m.Inputs {
				if i == m.Focus {
					m.Inputs[i].Focus()
				} else {
					m.Inputs[i].Blur()
				}
			}
			return m, nil
		}
	}

	cmds := make([]tea.Cmd, len(m.Inputs))
	for i := range m.Inputs {
		var cmd tea.Cmd
		m.Inputs[i], cmd = m.Inputs[i].Update(msg)
		cmds[i] = cmd
	}
	return m, tea.Batch(cmds...)
}

func (m FormModel) View() string {
	if m.Success {
		return successStyle.Render("API added successfully!") + "\n"
	}

	var b strings.Builder
	b.WriteString(subtitleStyle.Render("Add New API") + "\n\n")

	labels := []string{"Name", "URL"}
	for i, input := range m.Inputs {
		prefix := "  "
		if i == m.Focus {
			prefix = "› "
			b.WriteString(highlightStyle.Render(prefix+labels[i]) + "\n")
		} else {
			b.WriteString(normalStyle.Render(prefix+labels[i]) + "\n")
		}
		b.WriteString("    " + input.View() + "\n\n")
	}

	if m.ErrMsg != "" {
		b.WriteString(errorStyle.Render(m.ErrMsg) + "\n\n")
	}

	b.WriteString(mutedStyle.Render("tab/shift+tab move • enter save • esc back"))
	return b.String()
}
