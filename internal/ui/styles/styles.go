package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSecondary = lipgloss.Color("#666666")
	ColorSuccess   = lipgloss.Color("#04B575")
	ColorWarning   = lipgloss.Color("#FFA500")
	ColorError     = lipgloss.Color("#FF4C4C")
	ColorText      = lipgloss.Color("#EEEEEE")
	ColorSubtext   = lipgloss.Color("#999999")
	ColorHighlight = lipgloss.Color("#2D2D2D")

	// Base styles
	Base = lipgloss.NewStyle().Foreground(ColorText)

	// Panes
	Pane = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 1)

	ActivePane = Pane.Copy().
			BorderForeground(ColorPrimary)

	// Watchlist
	ListItem = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1)

	SelectedItem = ListItem.Copy().
			Background(ColorHighlight).
			Foreground(ColorPrimary).
			Bold(true)

	PositiveChange = lipgloss.NewStyle().Foreground(ColorSuccess)
	NegativeChange = lipgloss.NewStyle().Foreground(ColorError)

	// Chart
	ChartLabel = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			Width(8).
			Align(lipgloss.Right)
)
