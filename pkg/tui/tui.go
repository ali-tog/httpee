package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"httpee/pkg/config"
	"httpee/pkg/executor"
	"httpee/pkg/exporter"
	"httpee/pkg/history"
	"httpee/pkg/parser"
	"httpee/pkg/variables"
)

type requestItem struct {
	req parser.Request
}

func (i requestItem) Title() string       { return i.req.Name }
func (i requestItem) FilterValue() string { return i.req.Name + " " + i.req.URL + " " + i.req.Method }

type executionResultMsg struct {
	req  parser.Request
	resp *executor.Response
	err  error
}

// focusPane tracks which panel the keyboard is focused on.
type focusPane int

const (
	focusLeft  focusPane = 0
	focusRight focusPane = 1
)

type model struct {
	client     *executor.Client
	list       list.Model
	viewport   viewport.Model
	searchInput textinput.Model
	requests   []parser.Request
	variables  map[string]string
	historyEntries []history.HistoryEntry
	historyList  list.Model

	focus      focusPane
	ready      bool

	loading    bool
	spinner    spinner.Model

	editor     textarea.Model
	editingRequest bool

	lastResp   *executor.Response
	lastErr    error

	showingRequestPreview bool
	showVarsPanel         bool
	showHistoryPanel      bool
	showResponsePanel     bool // right panel visible

	actionMsg  string
	showHeaders bool

	cfg        config.Config

	width      int
	height     int
}

func matchKey(msg tea.KeyMsg, keys []string) bool {
	str := msg.String()
	for _, k := range keys {
		if str == k {
			return true
		}
	}
	return false
}

func formatKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

// New configures and initializes the root Terminal User Interface application state model.
func New(reqs []parser.Request, vars map[string]string, cfg config.Config) model {
	if vars == nil {
		vars = make(map[string]string)
	}
	items := make([]list.Item, len(reqs))
	for i, r := range reqs {
		items[i] = requestItem{req: r}
	}

	l := list.New(items, requestDelegate{}, 0, 0)
	l.Title = "Requests"
	l.SetShowFilter(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.KeyMap.CursorUp.SetKeys(cfg.Keys.MoveUp...)
	l.KeyMap.CursorDown.SetKeys(cfg.Keys.MoveDown...)

	hl := list.New(nil, historyDelegate{}, 0, 0)
	hl.Title = "History"
	hl.SetShowFilter(false)
	hl.SetShowTitle(false)
	hl.SetShowHelp(false)
	hl.SetShowStatusBar(false)
	hl.KeyMap.CursorUp.SetKeys(cfg.Keys.MoveUp...)
	hl.KeyMap.CursorDown.SetKeys(cfg.Keys.MoveDown...)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Search requests..."
	ti.Focus()
	ti.PromptStyle = searchInputStyle
	ti.TextStyle = searchInputStyle

	ta := textarea.New()
	ta.Placeholder = "Edit request (Method URL\\nHeaders...) "
	ta.ShowLineNumbers = true

	return model{
		client:      executor.NewClient(),
		requests:    reqs,
		variables:   vars,
		list:        l,
		historyList: hl,
		spinner:     s,
		searchInput: ti,
		editor:      ta,
		focus:       focusLeft,
		cfg:         cfg,
	}
}

// Init serves as the initialization hook for Bubbletea.
func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// Update handles the event loop.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case matchKey(msg, m.cfg.Keys.Quit):
			return m, tea.Quit
		case matchKey(msg, m.cfg.Keys.SwitchFocus):
			// Switch focus only when right panel is visible
			if m.showResponsePanel {
				if m.focus == focusLeft {
					m.focus = focusRight
					m.searchInput.Blur()
				} else {
					m.focus = focusLeft
					m.searchInput.Focus()
				}
			}
			return m, nil
		}

		if m.focus == focusLeft {
			switch {
			case matchKey(msg, m.cfg.Keys.Execute):
				if m.showHistoryPanel {
					if i, ok := m.historyList.SelectedItem().(historyItem); ok {
						m.lastResp = &executor.Response{
							StatusCode: i.entry.StatusCode,
							Status:     fmt.Sprintf("%d", i.entry.StatusCode),
							Headers:    i.entry.Headers,
							Body:       []byte(i.entry.Body),
							Duration:   time.Duration(i.entry.DurationMs) * time.Millisecond,
						}
						m.lastErr = nil
						m.actionMsg = ""
						m.showingRequestPreview = false
						m.showVarsPanel = false
						m.showResponsePanel = true
						m.resize()
						m.focus = focusRight
						m.searchInput.Blur()
						m.updateViewportContent()
						return m, nil
					}
				} else {
					if i, ok := m.list.SelectedItem().(requestItem); ok {
						m.loading = true
						m.lastResp = nil
						m.lastErr = nil
						m.actionMsg = ""
						m.showingRequestPreview = false
						m.showVarsPanel = false
						m.showHistoryPanel = false
						m.showResponsePanel = true
						m.resize()

						resolved := applyVariables(i.req, m.variables)

						m.focus = focusRight
						m.searchInput.Blur()

						return m, tea.Batch(
							m.spinner.Tick,
							executeRequest(m.client, resolved, i.req),
						)
					}
				}
			case matchKey(msg, m.cfg.Keys.MoveUp), matchKey(msg, m.cfg.Keys.MoveDown), msg.String() == "pgup", msg.String() == "pgdown":
				if m.showHistoryPanel {
					m.historyList, cmd = m.historyList.Update(msg)
				} else {
					m.list, cmd = m.list.Update(msg)
				}
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			case matchKey(msg, m.cfg.Keys.Preview):
				if !m.showHistoryPanel {
					if i, ok := m.list.SelectedItem().(requestItem); ok {
						m.showingRequestPreview = true
						m.showVarsPanel = false
						m.showHistoryPanel = false
						m.showResponsePanel = true
						m.resize()
						m.focus = focusRight
						m.searchInput.Blur()
						m.viewport.SetContent(renderRequestPreview(i.req, m.variables))
						m.viewport.GotoTop()
					}
				}
			case matchKey(msg, m.cfg.Keys.ToggleHistory):
				m.showHistoryPanel = !m.showHistoryPanel
				if m.showHistoryPanel {
					m.historyEntries, _ = history.ReadLog(100)
					m.filterHistoryList(m.searchInput.Value())
				}
			case matchKey(msg, m.cfg.Keys.ToggleVariables):
				m.showVarsPanel = !m.showVarsPanel
				m.showingRequestPreview = false
				m.showHistoryPanel = false
				m.showResponsePanel = true
				m.resize()
				m.focus = focusRight
				m.searchInput.Blur()
				if m.showVarsPanel {
					m.viewport.SetContent(renderVarsPanel(m.variables))
					m.viewport.GotoTop()
				} else {
					m.updateViewportContent()
				}
			}
		} else if m.focus == focusRight {
			if m.editingRequest {
				switch {
				case matchKey(msg, m.cfg.Keys.EditSave):
					rawReq := m.editor.Value()
					fakeFile := "### Edited Request\n" + rawReq
					parsed, _, err := parser.Parse(strings.NewReader(fakeFile), ".")
					if err == nil && len(parsed) > 0 {
						newReq := parsed[0]
						if i, ok := m.list.SelectedItem().(requestItem); ok {
							for j, r := range m.requests {
								if r.Name == i.req.Name && r.URL == i.req.URL {
									m.requests[j] = newReq
									break
								}
							}
						}
						m.editingRequest = false
						m.filterList(m.searchInput.Value())
						m.viewport.SetContent(renderRequestPreview(newReq, m.variables))
						m.actionMsg = "Request updated in memory."
					} else {
						m.actionMsg = "Failed to parse edited request!"
					}
					return m, nil
				case matchKey(msg, m.cfg.Keys.EditCancel):
					m.editingRequest = false
					return m, nil
				}
				m.editor, cmd = m.editor.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}

			switch {
			case matchKey(msg, m.cfg.Keys.Execute):
				if m.showingRequestPreview {
					m.editingRequest = true
					if i, ok := m.list.SelectedItem().(requestItem); ok {
						var b strings.Builder
						b.WriteString(fmt.Sprintf("%s %s\n", i.req.Method, i.req.URL))
						for k, v := range i.req.Headers {
							b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
						}
						b.WriteString("\n")
						if i.req.Body != "" {
							b.WriteString(strings.TrimRight(i.req.Body, "\n"))
						}
						m.editor.SetValue(b.String())
						m.editor.Focus()
						m.editor.CursorEnd()
					}
				}
				return m, nil
			case matchKey(msg, m.cfg.Keys.ToggleHeaders):
				if !m.showingRequestPreview {
					m.showHeaders = !m.showHeaders
					m.updateViewportContent()
				}
				return m, nil
			case matchKey(msg, m.cfg.Keys.CopyBody):
				if m.lastResp != nil && !m.showingRequestPreview {
					if err := exporter.CopyToClipboard(string(m.lastResp.Body)); err != nil {
						m.actionMsg = "Error: " + err.Error()
					} else {
						m.actionMsg = "Copied body to clipboard."
					}
				}
				return m, nil
			case matchKey(msg, m.cfg.Keys.CopyCurl):
				if m.lastResp != nil && !m.showingRequestPreview {
					if i, ok := m.list.SelectedItem().(requestItem); ok {
						curlStr := exporter.GenerateCurl(i.req)
						if err := exporter.CopyToClipboard(curlStr); err != nil {
							m.actionMsg = "Error: " + err.Error()
						} else {
							m.actionMsg = "Copied code to clipboard."
						}
					}
				}
				return m, nil
			case matchKey(msg, m.cfg.Keys.CloseRightPanel):
				// Close the right panel
				m.showResponsePanel = false
				m.showingRequestPreview = false
				m.showVarsPanel = false
				m.resize()
				m.focus = focusLeft
				m.searchInput.Focus()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(0, 0)
			m.ready = true
		}
		m.resize()

	case executionResultMsg:
		m.loading = false
		m.lastResp = msg.resp
		m.lastErr = msg.err
		if msg.err == nil && msg.resp != nil {
			_ = history.Save(msg.req, msg.resp)
		}
		m.updateViewportContent()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var spCmd tea.Cmd
			m.spinner, spCmd = m.spinner.Update(msg)
			return m, spCmd
		}
	}

	if m.focus == focusLeft {
		oldVal := m.searchInput.Value()
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)

		if m.searchInput.Value() != oldVal {
			if m.showHistoryPanel {
				m.filterHistoryList(m.searchInput.Value())
			} else {
				m.filterList(m.searchInput.Value())
			}
		}
	} else if m.focus == focusRight {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// leftPanelWidth returns the width for the left panel.
// When split, it's 40% of total width; when solo it's the full width.
func (m model) leftPanelWidth() int {
	if m.showResponsePanel && m.width > 0 {
		return m.width * 40 / 100
	}
	return m.width
}

// rightPanelWidth returns the width for the right panel.
func (m model) rightPanelWidth() int {
	if m.width > 0 {
		return m.width - m.leftPanelWidth()
	}
	return 0
}

// contentHeight returns the usable inner height for panel content.
func (m model) contentHeight() int {
	if m.height == 0 {
		return 0
	}
	// Fixed height rows: 
	// Header (1)
	// Outer Border top+bottom (2)
	// Status Line (1)
	// Help Menu (1 to 2 depending on wrap, let's reserve 2)
	// Margin (1)
	return m.height - 7
}

// resize recalculates and applies dimensions to all child components.
// Must be called on WindowSizeMsg and whenever showResponsePanel changes.
func (m *model) resize() {
	leftW := m.leftPanelWidth()
	rightW := m.rightPanelWidth()
	contentH := m.contentHeight()
	searchH := 2 // search box takes 1 line + 1 line padding/margin

	// constrain search input width so it doesn't force the left panel to expand
	m.searchInput.Width = leftW - 4

	// left list: subtract search box height
	m.list.SetSize(leftW-4, contentH-searchH)
	m.historyList.SetSize(leftW-4, contentH-searchH)

	// right viewport / editor
	if m.showResponsePanel {
		m.viewport.Width = rightW - 4
		m.viewport.Height = contentH
		m.editor.SetWidth(rightW - 4)
		m.editor.SetHeight(contentH)
	} else {
		// keep viewport sane for when we open it later
		m.viewport.Width = m.width - 4
		m.viewport.Height = contentH
		m.editor.SetWidth(m.width - 4)
		m.editor.SetHeight(contentH)
	}
}

func (m *model) filterList(query string) {
	query = strings.ToLower(query)
	var filtered []list.Item
	for _, req := range m.requests {
		if query == "" ||
		   strings.Contains(strings.ToLower(req.Name), query) ||
		   strings.Contains(strings.ToLower(req.URL), query) ||
		   strings.Contains(strings.ToLower(req.Method), query) {
			filtered = append(filtered, requestItem{req: req})
		}
	}
	m.list.SetItems(filtered)
}

func (m *model) filterHistoryList(query string) {
	query = strings.ToLower(query)
	var filtered []list.Item
	for _, entry := range m.historyEntries {
		if query == "" ||
		   strings.Contains(strings.ToLower(entry.RequestName), query) ||
		   strings.Contains(strings.ToLower(entry.URL), query) ||
		   strings.Contains(strings.ToLower(entry.Method), query) {
			filtered = append(filtered, historyItem{entry: entry})
		}
	}
	m.historyList.SetItems(filtered)
}

func (m *model) updateViewportContent() {
	if m.lastErr != nil {
		m.viewport.SetContent(fmt.Sprintf("Error: %v", m.lastErr))
		return
	}

	if m.lastResp == nil {
		m.viewport.SetContent("Select a request and press Enter to execute.")
		return
	}

	var sb strings.Builder

	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true).Render(fmt.Sprintf("Duration: %s", m.lastResp.Duration.String())))
	sb.WriteString("\n\n")

	if m.showHeaders {
		for k, v := range m.lastResp.Headers {
			for _, val := range v {
				sb.WriteString(fmt.Sprintf("%s: %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(k), val))
			}
		}
		sb.WriteString("\n")
	}

	bodyStr := string(m.lastResp.Body)
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(bodyStr), "", "  "); err == nil {
		highlighted := HighlightJSON(prettyJSON.String())
		sb.WriteString(highlighted)
	} else {
		sb.WriteString(bodyStr)
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoTop()
}

func statusStyle(code int) lipgloss.Style {
	if code >= 200 && code < 300 {
		return statusOKStyle
	}
	return statusErrStyle
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// ── Left panel ──────────────────────────────────────────────────────────
	leftWidth := m.leftPanelWidth()
	leftHeaderTitle := "REQUESTS"
	if m.showHistoryPanel {
		leftHeaderTitle = "HISTORY"
	}

	leftFocused := m.focus == focusLeft
	leftBorder := inactiveBorderStyle
	if leftFocused {
		leftBorder = activeBorderStyle
	}

	leftHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("86")).
		Bold(true).
		Padding(0, 2).
		Render(leftHeaderTitle)
	leftHeader = lipgloss.NewStyle().MarginLeft(1).Render(leftHeader)

	searchView := lipgloss.NewStyle().Padding(0, 1).Render(m.searchInput.View())
	var listView string
	if m.showHistoryPanel {
		listView = m.historyList.View()
	} else {
		listView = m.list.View()
	}

	panelH := m.contentHeight()
	leftContent := lipgloss.JoinVertical(lipgloss.Left, searchView, listView)
	leftBox := leftBorder.Copy().Width(leftWidth - 2).Height(panelH).Render(leftContent)
	leftPanel := lipgloss.JoinVertical(lipgloss.Left, leftHeader, leftBox)

	// ── Right panel (only when showResponsePanel) ────────────────────────────
	var fullView string
	if m.showResponsePanel {
		rightWidth := m.rightPanelWidth()

		// Determine right header
		rightHeaderBg := "86"
		var rightHeaderTitle string
		if m.editingRequest {
			rightHeaderTitle = "EDIT REQUEST"
		} else if m.showingRequestPreview {
			rightHeaderTitle = "PREVIEW"
		} else if m.showVarsPanel {
			rightHeaderTitle = "VARIABLES"
		} else if m.loading {
			rightHeaderTitle = "RESPONSE  " + m.spinner.View()
		} else if m.lastResp != nil {
			rightHeaderTitle = fmt.Sprintf("RESPONSE %s", m.lastResp.Status)
			if m.lastResp.StatusCode >= 200 && m.lastResp.StatusCode < 300 {
				rightHeaderBg = "42"
			} else if m.lastResp.StatusCode >= 400 {
				rightHeaderBg = "196"
			}
		} else {
			rightHeaderTitle = "RESPONSE"
		}

		closeHint := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(" [x] close")
		rightHeaderRendered := lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color(rightHeaderBg)).
			Bold(true).
			Padding(0, 2).
			Render(rightHeaderTitle)
		rightHeader := lipgloss.NewStyle().MarginLeft(1).Render(
			lipgloss.JoinHorizontal(lipgloss.Top, rightHeaderRendered, closeHint),
		)

		rightFocused := m.focus == focusRight
		rightBorder := inactiveBorderStyle
		if rightFocused {
			rightBorder = activeBorderStyle
		}

		var rightContent string
		if m.editingRequest {
			rightContent = m.editor.View()
		} else {
			rightContent = m.viewport.View()
		}

		rightBox := rightBorder.Copy().Width(rightWidth - 2).Height(panelH).Render(rightContent)
		rightPanel := lipgloss.JoinVertical(lipgloss.Left, rightHeader, rightBox)

		fullView = lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	} else {
		// Single panel — resize list to full width
		fullView = leftPanel
	}

	// ── Status line ──────────────────────────────────────────────────────────
	statusStr := ""
	if m.loading {
		statusStr = m.spinner.View() + " Executing..."
	} else if m.actionMsg != "" {
		statusStr = m.actionMsg
	}
	statusLine := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Padding(0, 1).Render(statusStr)

	// ── Help bar ─────────────────────────────────────────────────────────────
	var keys []string
	keys = append(keys, formatKeys(m.cfg.Keys.Quit), "quit")
	if m.showResponsePanel {
		keys = append(keys, formatKeys(m.cfg.Keys.SwitchFocus), "switch focus")
	}
	if m.focus == focusLeft {
		keys = append(keys, formatKeys(m.cfg.Keys.Execute), "execute", "↑/↓", "move", formatKeys(m.cfg.Keys.Preview), "preview", formatKeys(m.cfg.Keys.ToggleVariables), "variables", formatKeys(m.cfg.Keys.ToggleHistory), "history")
	} else {
		if m.editingRequest {
			keys = append(keys, formatKeys(m.cfg.Keys.EditSave), "save", formatKeys(m.cfg.Keys.EditCancel), "cancel")
		} else if m.showingRequestPreview {
			keys = append(keys, formatKeys(m.cfg.Keys.CloseRightPanel), "close", formatKeys(m.cfg.Keys.Execute), "edit", "↑/↓", "scroll")
		} else if m.showVarsPanel {
			keys = append(keys, formatKeys(m.cfg.Keys.CloseRightPanel), "close", "↑/↓", "scroll")
		} else {
			keys = append(keys, formatKeys(m.cfg.Keys.CloseRightPanel), "close", "↑/↓", "scroll", formatKeys(m.cfg.Keys.ToggleHeaders), "headers", formatKeys(m.cfg.Keys.CopyBody), "copy", formatKeys(m.cfg.Keys.CopyCurl), "curl")
		}
	}

	var pairs []string
	for i := 0; i < len(keys); i += 2 {
		pairs = append(pairs, keyStyle.Render(keys[i])+" "+descStyle.Render(keys[i+1]))
	}
	helpView := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(pairs, separatorStyle.Render("•")))

	footer := lipgloss.JoinVertical(lipgloss.Left, statusLine, helpView)

	return lipgloss.JoinVertical(lipgloss.Left, fullView, footer)
}

func executeRequest(client *executor.Client, req parser.Request, originalReq parser.Request) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.Execute(req)
		return executionResultMsg{req: originalReq, resp: resp, err: err}
	}
}

func getHistoryLoader() variables.HistoryLoader {
	return func(reqName string) ([]byte, map[string][]string, error) {
		entry, err := history.GetLatest(reqName)
		if err != nil {
			return nil, nil, err
		}
		return []byte(entry.Body), entry.Headers, nil
	}
}

// applyVariables substitutes all {{token}} placeholders in a request's URL,
// headers and body using the provided variable map and history logs.
func applyVariables(req parser.Request, vars map[string]string) parser.Request {
	loader := getHistoryLoader()
	req.URL = variables.Substitute(req.URL, vars, loader)
	req.Body = variables.Substitute(req.Body, vars, loader)
	newHeaders := make(map[string]string, len(req.Headers))
	for k, v := range req.Headers {
		newHeaders[k] = variables.Substitute(v, vars, loader)
	}
	req.Headers = newHeaders
	return req
}

func renderRequestPreview(req parser.Request, vars map[string]string) string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	methodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("251"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	tokenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	b.WriteString(titleStyle.Render("Request Preview") + "\n")
	b.WriteString(hintStyle.Render("Variables are shown resolved below • press 'x' to close") + "\n\n")

	b.WriteString(fmt.Sprintf("%s %s\n", methodStyle.Render("Name:  "), valStyle.Render(req.Name)))

	methodColor := "205"
	switch req.Method {
	case "GET":    methodColor = "42"
	case "POST":   methodColor = "39"
	case "PUT":    methodColor = "214"
	case "DELETE": methodColor = "196"
	case "PATCH":  methodColor = "226"
	}
	coloredMethod := lipgloss.NewStyle().Foreground(lipgloss.Color(methodColor)).Render(req.Method)
	b.WriteString(fmt.Sprintf("%s %s\n", methodStyle.Render("Method:"), coloredMethod))

	loader := getHistoryLoader()
	resolvedURL := variables.Substitute(req.URL, vars, loader)
	urlDisplay := resolvedURL
	if resolvedURL != req.URL {
		urlDisplay = valStyle.Render(resolvedURL) + "  " + tokenStyle.Render("(was: "+req.URL+")")
	} else {
		urlDisplay = valStyle.Render(resolvedURL)
	}
	b.WriteString(fmt.Sprintf("%s %s\n\n", methodStyle.Render("URL:   "), urlDisplay))

	if len(req.Headers) > 0 {
		b.WriteString(titleStyle.Render("Headers:") + "\n")
		for k, v := range req.Headers {
			resolvedV := variables.Substitute(v, vars, loader)
			b.WriteString(fmt.Sprintf("  %s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(k+":"),
				valStyle.Render(resolvedV)))
		}
		b.WriteString("\n")
	}

	if req.Body != "" {
		resolvedBody := variables.Substitute(req.Body, vars, loader)
		b.WriteString(titleStyle.Render("Body:") + "\n")
		if strings.HasPrefix(strings.TrimSpace(resolvedBody), "{") || strings.HasPrefix(strings.TrimSpace(resolvedBody), "[") {
			resolvedBody = HighlightJSON(resolvedBody)
		}
		b.WriteString(resolvedBody)
	}

	return b.String()
}

// renderVarsPanel renders the list of currently defined variables.
func renderVarsPanel(vars map[string]string) string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	keyStyle2 := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("251"))

	b.WriteString(titleStyle.Render("Variables") + "\n")
	if len(vars) == 0 {
		b.WriteString(hintStyle.Render("No variables defined. Use @name = value in your .http file.") + "\n")
		return b.String()
	}
	b.WriteString(hintStyle.Render(fmt.Sprintf("%d variable(s) in scope • press 'x' to close", len(vars))) + "\n\n")
	for k, v := range vars {
		b.WriteString(fmt.Sprintf("  %s = %s\n", keyStyle2.Render("@"+k), valStyle.Render(v)))
	}
	return b.String()
}

type historyItem struct {
	entry history.HistoryEntry
}

func (i historyItem) Title() string       { return i.entry.RequestName }
func (i historyItem) FilterValue() string { return i.entry.RequestName + " " + i.entry.URL + " " + i.entry.Method }

type historyDelegate struct{}

func (d historyDelegate) Height() int  { return 1 }
func (d historyDelegate) Spacing() int { return 1 }
func (d historyDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d historyDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(historyItem)
	if !ok {
		return
	}

	timeStr := i.entry.Timestamp.Local().Format("15:04:05")
	methodColor := "205"
	switch i.entry.Method {
	case "GET":    methodColor = "42"
	case "POST":   methodColor = "39"
	case "PUT":    methodColor = "214"
	case "DELETE": methodColor = "196"
	case "PATCH":  methodColor = "226"
	}

	methodBadge := lipgloss.NewStyle().
		Background(lipgloss.Color(methodColor)).
		Foreground(lipgloss.Color("0")).
		Bold(true).
		Width(7).
		Align(lipgloss.Center).
		Render(i.entry.Method)

	nameStr := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true).Render(i.entry.RequestName)

	statusColor := "39"
	if i.entry.StatusCode >= 200 && i.entry.StatusCode < 300 {
		statusColor = "42"
	} else if i.entry.StatusCode >= 400 {
		statusColor = "196"
	}
	statusStr := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Bold(true).Render(fmt.Sprintf("%d", i.entry.StatusCode))
	timeRender := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(timeStr)

	urlStr := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(i.entry.URL)
	str := fmt.Sprintf("%s %s %s [%s] %s", methodBadge, statusStr, nameStr, timeRender, urlStr)

	style := lipgloss.NewStyle().PaddingLeft(2).MaxWidth(m.Width() - 2)
	if index == m.Index() {
		style = lipgloss.NewStyle().PaddingLeft(0).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("205")).
			PaddingLeft(1).
			MaxWidth(m.Width() - 2)
	}

	fmt.Fprint(w, style.Render(str))
}

type requestDelegate struct{}

func (d requestDelegate) Height() int  { return 1 }
func (d requestDelegate) Spacing() int { return 1 }
func (d requestDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d requestDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(requestItem)
	if !ok {
		return
	}

	methodColor := "205"
	switch i.req.Method {
	case "GET":    methodColor = "42"
	case "POST":   methodColor = "39"
	case "PUT":    methodColor = "214"
	case "DELETE": methodColor = "196"
	case "PATCH":  methodColor = "226"
	}

	methodBadge := lipgloss.NewStyle().
		Background(lipgloss.Color(methodColor)).
		Foreground(lipgloss.Color("0")).
		Bold(true).
		Width(7).
		Align(lipgloss.Center).
		Render(i.req.Method)

	nameStr := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true).Render(i.req.Name)
	urlStr := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(i.req.URL)

	str := fmt.Sprintf("%s %s %s", methodBadge, nameStr, urlStr)

	style := lipgloss.NewStyle().PaddingLeft(2).MaxWidth(m.Width() - 2)
	if index == m.Index() {
		style = lipgloss.NewStyle().PaddingLeft(0).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("205")).
			PaddingLeft(1).
			MaxWidth(m.Width() - 2)
	}

	fmt.Fprint(w, style.Render(str))
}
