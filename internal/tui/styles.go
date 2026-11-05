package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Arch Linux Official Brand Color Palette
	ArchCyan    = lipgloss.Color("#1793D1")
	ArchDark    = lipgloss.Color("#0f557a")
	WarningRed  = lipgloss.Color("#FF5555")
	SuccessGreen = lipgloss.Color("#50FA7B")
	MutedGray   = lipgloss.Color("#6272A4")
	Highlight   = lipgloss.Color("#BD93F9")
	White       = lipgloss.Color("#F8F8F2")

	// Header Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(ArchCyan).
			Padding(0, 2)

	SubTitleStyle = lipgloss.NewStyle().
			Foreground(MutedGray).
			Italic(true)

	// Tab Styles
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(ArchCyan).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(MutedGray).
				Padding(0, 2)

	// Box / Container Styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ArchCyan).
			Padding(1, 2)

	WarningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(WarningRed).
			Foreground(WarningRed).
			Bold(true).
			Padding(0, 1)

	// Item list styles
	SelectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ArchCyan).
				Background(lipgloss.Color("#21222C"))

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(White)

	CriticalItemStyle = lipgloss.NewStyle().
				Foreground(WarningRed).
				Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedGray).
			MarginTop(1)
)
