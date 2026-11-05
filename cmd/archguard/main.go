package main

import (
	"fmt"
	"os"

	"github.com/SulthanAndyno/archguard/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	Version   = "0.1.0"
	BuildDate = "2026-11-05"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Printf("archguard v%s (%s)\n", Version, BuildDate)
			os.Exit(0)
		case "-h", "--help", "help":
			printHelp()
			os.Exit(0)
		}
	}

	p := tea.NewProgram(tui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running archguard: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`archguard - Arch Linux System Health, News & Maintenance Companion

USAGE:
  archguard [FLAGS]

FLAGS:
  -v, --version    Show version information
  -h, --help       Show this help message

KEYBINDINGS:
  Tab / Shift+Tab  Switch between tabs
  1 - 4            Jump directly to tab 1, 2, 3, or 4
  j / k / ↑ / ↓    Navigate through items
  Space            Select / toggle item (in Orphan Packages tab)
  r                Refresh system data
  q / Ctrl+C       Quit archguard

AUTHOR:
  Sulthan Andyno <sulthandyno@gmail.com>
  https://github.com/SulthanAndyno/archguard`)
}
