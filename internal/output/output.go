package output

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipglosstable "github.com/charmbracelet/lipgloss/table"
	zlog "github.com/rs/zerolog/log"
	"golang.org/x/term"

	"github.com/steved/alertreplay/internal/alert"
)

func init() {
	// Disable mouse support to allow text selection in terminal.
	_ = os.Setenv("BUBBLETEA_DISABLE_MOUSE", "1")
}

const outputTimeFormat = "2006-01-02 15:04 MST"

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type tableModel struct {
	table    table.Model
	links    []string
	viewport viewport.Model
	width    int
	height   int
}

func (m tableModel) Init() tea.Cmd { return nil }

func (m tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			idx := m.table.Cursor()
			if idx >= 0 && idx < len(m.links) {
				openBrowser(m.links[idx])
			}
		case "left", "h":
			m.viewport.ScrollLeft(max(m.viewport.Width/4, 4))
		case "right", "l":
			m.viewport.ScrollRight(max(m.viewport.Width/4, 4))
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.reflow()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	m.refreshContent()

	return m, cmd
}

func (m *tableModel) reflow() {
	m.table.SetHeight(calcTableHeight(m.height))
	m.viewport.Width = m.width
	m.viewport.Height = m.height
	m.refreshContent()
}

func (m *tableModel) refreshContent() {
	content := baseStyle.Render(m.table.View()) +
		"\n  Press enter to open UI, ←/→ to scroll, q to quit\n"
	m.viewport.SetContent(content)
}

func (m tableModel) View() string {
	return m.viewport.View()
}

func newTableModel(
	alerts []alert.Alert,
	termWidth int,
	termHeight int,
) tableModel {
	hasSourceCol := slices.ContainsFunc(alerts, func(ar alert.Alert) bool { return ar.Source != "" })

	rows := make([]table.Row, 0, len(alerts))
	links := make([]string, 0, len(alerts))
	for _, ar := range alerts {
		resolvedStr := ""
		durationStr := ""
		if ar.ResolvedAt != nil {
			resolvedStr = ar.ResolvedAt.UTC().Format(outputTimeFormat)
			durationStr = ar.ResolvedAt.Sub(ar.OpenedAt).Round(time.Second).String()
		} else {
			resolvedStr = "UNRESOLVED"
			durationStr = "--"
		}

		var row table.Row
		if hasSourceCol {
			row = table.Row{
				ar.Source,
				ar.OpenedAt.UTC().Format(outputTimeFormat),
				resolvedStr,
				durationStr,
				alert.FormatLabels(ar.Labels),
			}
		} else {
			row = table.Row{
				ar.OpenedAt.UTC().Format(outputTimeFormat),
				resolvedStr,
				durationStr,
				alert.FormatLabels(ar.Labels),
			}
		}

		rows = append(rows, row)
		links = append(links, ar.URL)
	}

	columns := buildColumns(termWidth, hasSourceCol)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(calcTableHeight(termHeight)),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	vp := viewport.New(termWidth, termHeight)
	vp.SetHorizontalStep(8)

	m := tableModel{
		table:    t,
		links:    links,
		viewport: vp,
		width:    termWidth,
		height:   termHeight,
	}
	m.refreshContent()

	return m
}

func PrintEvents(alerts []alert.Alert) error {
	if len(alerts) == 0 {
		zlog.Info().Msg("No alert events found.")
		return nil
	}

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return printEventsMarkdown(alerts)
	}

	termWidth := 140
	termHeight := 40
	if w, h, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 && h > 0 {
		termWidth = w
		termHeight = h
	}

	m := newTableModel(alerts, termWidth, termHeight)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running table: %w", err)
	}

	return nil
}

func printEventsMarkdown(alerts []alert.Alert) error {
	headers := []string{"Opened", "Resolved", "Duration", "Labels", "URL"}

	hasSourceCol := slices.ContainsFunc(alerts, func(alert alert.Alert) bool { return alert.Source != "" })
	if hasSourceCol {
		headers = append([]string{"Source"}, headers...)
	}

	re := lipgloss.NewRenderer(os.Stdout)
	cellStyle := re.NewStyle().Padding(0, 1)
	t := lipglosstable.New().
		Headers(headers...).
		Border(lipgloss.MarkdownBorder()).
		BorderTop(false).
		BorderBottom(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			return cellStyle
		})

	for _, ar := range alerts {
		resolvedStr := "UNRESOLVED"
		durationStr := "--"
		if ar.ResolvedAt != nil {
			resolvedStr = ar.ResolvedAt.UTC().Format(outputTimeFormat)
			durationStr = ar.ResolvedAt.Sub(ar.OpenedAt).Round(time.Second).String()
		}

		if hasSourceCol {
			t.Row(ar.Source, ar.OpenedAt.UTC().Format(outputTimeFormat), resolvedStr, durationStr, alert.FormatLabels(ar.Labels), ar.URL)
		} else {
			t.Row(ar.OpenedAt.UTC().Format(outputTimeFormat), resolvedStr, durationStr, alert.FormatLabels(ar.Labels), ar.URL)
		}
	}

	fmt.Println(t.Render())

	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}

func calcTableHeight(termHeight int) int {
	available := termHeight - 1 - baseStyle.GetVerticalFrameSize()
	if available < 5 {
		return 5
	}
	return available
}

const (
	colWidthSource   = 30
	colWidthOpened   = 21
	colWidthResolved = 21
	colWidthDuration = 12
)

func buildColumns(termWidth int, hasSourceCol bool) []table.Column {
	var (
		fixedWidth = colWidthOpened + colWidthResolved + colWidthDuration
		numCols    = 4
	)

	if hasSourceCol {
		fixedWidth += colWidthSource
		numCols = 5
	}

	paddingWidth := numCols * 2
	labelsWidth := termWidth - baseStyle.GetHorizontalFrameSize() - paddingWidth - fixedWidth
	labelsWidth = max(labelsWidth, 20)

	if hasSourceCol {
		return []table.Column{
			{Title: "Source", Width: colWidthSource},
			{Title: "Opened", Width: colWidthOpened},
			{Title: "Resolved", Width: colWidthResolved},
			{Title: "Duration", Width: colWidthDuration},
			{Title: "Labels", Width: labelsWidth},
		}
	}

	return []table.Column{
		{Title: "Opened", Width: colWidthOpened},
		{Title: "Resolved", Width: colWidthResolved},
		{Title: "Duration", Width: colWidthDuration},
		{Title: "Labels", Width: labelsWidth},
	}
}
