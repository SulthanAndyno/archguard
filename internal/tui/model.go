package tui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/SulthanAndyno/archguard/internal/arch"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tabIndex int

const (
	tabNews tabIndex = iota
	tabUpdates
	tabOrphans
	tabAudit
)

var tabNames = []string{
	"1. Arch News",
	"2. Pending Updates",
	"3. Orphan Packages",
	"4. System & Cache Audit",
}

type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmDeleteOrphans
	confirmCleanCache
	confirmUpgradeSystem
)

// LocalDataMsg holds fast, local pacman telemetry
type LocalDataMsg struct {
	Updates     []arch.PackageUpdate
	Orphans     []arch.OrphanPackage
	Installed   map[string]bool
	CacheSize   string
	PacnewFiles []string
	Err         error
}

// NewsDataMsg holds asynchronous news feed results
type NewsDataMsg struct {
	News []arch.NewsItem
	Err  error
}

// actionFinishedMsg is returned after an interactive tea.ExecProcess completes
type actionFinishedMsg struct {
	action string
	err    error
}

type Model struct {
	activeTab  tabIndex
	confirming confirmAction
	width      int
	height     int

	localLoading bool
	newsLoading  bool
	spinner      spinner.Model
	statusMsg    string

	newsCursor    int
	updatesCursor int
	orphansCursor int

	news          []arch.NewsItem
	updates       []arch.PackageUpdate
	orphans       []arch.OrphanPackage
	installedPkgs map[string]bool
	cacheSize     string
	pacnewFiles   []string
	newsErr       error
}

func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ArchCyan)

	return Model{
		activeTab:    tabNews,
		confirming:   confirmNone,
		localLoading: true,
		newsLoading:  true,
		spinner:      s,
		statusMsg:    "Scanning Arch Linux environment...",
		width:        92,
		height:       24,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchLocalDataCmd(),
		fetchNewsCmd(),
	)
}

func fetchLocalDataCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		updates, _ := arch.GetPendingUpdates(ctx)
		orphans, _ := arch.GetOrphans(ctx)
		installed, _ := arch.GetInstalledPackages(ctx)
		cacheSize, _ := arch.GetCacheSize(ctx)
		pacnewFiles, _ := arch.GetPacnewFiles()

		return LocalDataMsg{
			Updates:     updates,
			Orphans:     orphans,
			Installed:   installed,
			CacheSize:   cacheSize,
			PacnewFiles: pacnewFiles,
		}
	}
}

func fetchNewsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		items, err := arch.FetchNews(ctx, 10)
		return NewsDataMsg{
			News: items,
			Err:  err,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.localLoading || m.newsLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case LocalDataMsg:
		m.localLoading = false
		m.updates = msg.Updates
		m.orphans = msg.Orphans
		m.installedPkgs = msg.Installed
		m.cacheSize = msg.CacheSize
		m.pacnewFiles = msg.PacnewFiles

		// If news was already loaded, correlate interventions
		if len(m.news) > 0 && m.installedPkgs != nil {
			m.news = arch.CheckInterventions(m.news, m.installedPkgs)
		}
		m.updateStatus()
		return m, nil

	case NewsDataMsg:
		m.newsLoading = false
		m.newsErr = msg.Err
		if msg.Err == nil {
			if m.installedPkgs != nil {
				m.news = arch.CheckInterventions(msg.News, m.installedPkgs)
			} else {
				m.news = msg.News
			}
		}
		m.updateStatus()
		return m, nil

	case actionFinishedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Action failed or cancelled: %v", msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("Action '%s' completed successfully!", msg.action)
		}
		// Refresh local data immediately
		m.localLoading = true
		return m, tea.Batch(m.spinner.Tick, fetchLocalDataCmd())

	case tea.KeyMsg:
		// Handle Active Confirmation Dialog
		if m.confirming != confirmNone {
			switch msg.String() {
			case "y", "Y", "enter":
				action := m.confirming
				m.confirming = confirmNone

				switch action {
				case confirmDeleteOrphans:
					var selected []string
					for _, o := range m.orphans {
						if o.Selected {
							selected = append(selected, o.Name)
						}
					}
					if len(selected) == 0 {
						m.statusMsg = "No packages selected."
						return m, nil
					}
					c := exec.Command("sudo", append([]string{"pacman", "-Rns", "--"}, selected...)...)
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return actionFinishedMsg{action: "remove_orphans", err: err}
					})

				case confirmCleanCache:
					c := exec.Command("sudo", "paccache", "-rk2")
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return actionFinishedMsg{action: "clean_cache", err: err}
					})

				case confirmUpgradeSystem:
					c := exec.Command("sudo", "pacman", "-Syu")
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return actionFinishedMsg{action: "system_upgrade", err: err}
					})
				}

			case "n", "N", "esc":
				m.confirming = confirmNone
				m.statusMsg = "Action cancelled."
				return m, nil
			}
			return m, nil
		}

		// Standard Keybindings
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		// Tab Switching
		case "tab", "l", "right":
			m.activeTab = (m.activeTab + 1) % tabIndex(len(tabNames))
			return m, nil
		case "shift+tab", "h", "left":
			if m.activeTab == 0 {
				m.activeTab = tabIndex(len(tabNames) - 1)
			} else {
				m.activeTab--
			}
			return m, nil
		case "1":
			m.activeTab = tabNews
			return m, nil
		case "2":
			m.activeTab = tabUpdates
			return m, nil
		case "3":
			m.activeTab = tabOrphans
			return m, nil
		case "4":
			m.activeTab = tabAudit
			return m, nil

		// Navigation
		case "up", "k":
			switch m.activeTab {
			case tabNews:
				if m.newsCursor > 0 {
					m.newsCursor--
				}
			case tabUpdates:
				if m.updatesCursor > 0 {
					m.updatesCursor--
				}
			case tabOrphans:
				if m.orphansCursor > 0 {
					m.orphansCursor--
				}
			}
			return m, nil

		case "down", "j":
			switch m.activeTab {
			case tabNews:
				if m.newsCursor < len(m.news)-1 {
					m.newsCursor++
				}
			case tabUpdates:
				if m.updatesCursor < len(m.updates)-1 {
					m.updatesCursor++
				}
			case tabOrphans:
				if m.orphansCursor < len(m.orphans)-1 {
					m.orphansCursor++
				}
			}
			return m, nil

		// Toggle item in Orphans
		case " ":
			if m.activeTab == tabOrphans && len(m.orphans) > 0 {
				m.orphans[m.orphansCursor].Selected = !m.orphans[m.orphansCursor].Selected
				m.updateStatus()
				return m, nil
			}

		// Select all / deselect all in Orphans
		case "a":
			if m.activeTab == tabOrphans && len(m.orphans) > 0 {
				allSelected := true
				for _, o := range m.orphans {
					if !o.Selected {
						allSelected = false
						break
					}
				}
				for i := range m.orphans {
					m.orphans[i].Selected = !allSelected
				}
				m.updateStatus()
				return m, nil
			}

		// Delete selected orphans
		case "d":
			if m.activeTab == tabOrphans {
				count := 0
				for _, o := range m.orphans {
					if o.Selected {
						count++
					}
				}
				if count == 0 {
					m.statusMsg = "No orphans selected. Press [Space] or [a] first."
				} else {
					m.confirming = confirmDeleteOrphans
				}
				return m, nil
			}

		// Clean cache
		case "c":
			if m.activeTab == tabAudit {
				m.confirming = confirmCleanCache
				return m, nil
			}

		// Safe system upgrade
		case "u":
			if m.activeTab == tabUpdates {
				if len(m.updates) == 0 {
					m.statusMsg = "No pending updates available."
				} else {
					m.confirming = confirmUpgradeSystem
				}
				return m, nil
			}

		// Open news link in browser
		case "o", "enter":
			if m.activeTab == tabNews && len(m.news) > 0 && m.newsCursor < len(m.news) {
				link := m.news[m.newsCursor].Link
				if link != "" {
					_ = exec.Command("xdg-open", link).Start()
					m.statusMsg = "Opening advisory link in default browser..."
				}
				return m, nil
			}

		case "r":
			// Refresh data asynchronously
			m.localLoading = true
			m.newsLoading = true
			m.statusMsg = "Refreshing system data..."
			return m, tea.Batch(
				m.spinner.Tick,
				fetchLocalDataCmd(),
				fetchNewsCmd(),
			)
		}
	}

	return m, nil
}

func (m *Model) updateStatus() {
	selectedOrphans := 0
	for _, o := range m.orphans {
		if o.Selected {
			selectedOrphans++
		}
	}

	newsCount := len(m.news)
	if m.newsLoading {
		m.statusMsg = fmt.Sprintf("Ready (Local) | %d Updates, %d Orphans (%d selected) | Loading News...",
			len(m.updates), len(m.orphans), selectedOrphans)
	} else if m.newsErr != nil {
		m.statusMsg = fmt.Sprintf("Ready | %d Updates, %d Orphans (%d selected) | News Offline",
			len(m.updates), len(m.orphans), selectedOrphans)
	} else {
		m.statusMsg = fmt.Sprintf("Ready | %d News, %d Updates, %d Orphans (%d selected)",
			newsCount, len(m.updates), len(m.orphans), selectedOrphans)
	}
}

func (m Model) View() string {
	if m.localLoading {
		return fmt.Sprintf("\n\n   %s  %s\n\n", m.spinner.View(), m.statusMsg)
	}

	var b strings.Builder

	// 1. Header
	title := TitleStyle.Render(" ARCHGUARD ") + "  " + SubTitleStyle.Render("System Health, News & Maintenance Companion")
	b.WriteString("\n" + title + "\n\n")

	// 2. Tabs
	var renderedTabs []string
	for i, name := range tabNames {
		tabLabel := name
		if i == 0 && m.newsLoading {
			tabLabel += " " + m.spinner.View()
		}
		if tabIndex(i) == m.activeTab {
			renderedTabs = append(renderedTabs, ActiveTabStyle.Render(tabLabel))
		} else {
			renderedTabs = append(renderedTabs, InactiveTabStyle.Render(tabLabel))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...) + "\n\n")

	// 3. Tab Contents
	var content string
	switch m.activeTab {
	case tabNews:
		content = m.renderNewsTab()
	case tabUpdates:
		content = m.renderUpdatesTab()
	case tabOrphans:
		content = m.renderOrphansTab()
	case tabAudit:
		content = m.renderAuditTab()
	}

	// Dynamic panel width based on window size
	panelWidth := m.width - 4
	if panelWidth < 65 {
		panelWidth = 65
	} else if panelWidth > 115 {
		panelWidth = 115
	}

	b.WriteString(PanelStyle.Width(panelWidth).Render(content))

	// 4. Confirmation Banner if pending
	if m.confirming != confirmNone {
		var confirmText string
		switch m.confirming {
		case confirmDeleteOrphans:
			count := 0
			for _, o := range m.orphans {
				if o.Selected {
					count++
				}
			}
			confirmText = fmt.Sprintf("CONFIRM REMOVAL: Run 'sudo pacman -Rns' to delete %d package(s)? [y] Confirm • [n/Esc] Cancel", count)
		case confirmCleanCache:
			confirmText = "CONFIRM CACHE TRIM: Run 'sudo paccache -rk2' to clean old packages? [y] Confirm • [n/Esc] Cancel"
		case confirmUpgradeSystem:
			confirmText = fmt.Sprintf("CONFIRM UPGRADE: Run 'sudo pacman -Syu' for %d pending package(s)? [y] Confirm • [n/Esc] Cancel", len(m.updates))
		}
		b.WriteString("\n" + WarningBoxStyle.Render(" "+confirmText+" "))
	}

	// 5. Footer & Keybindings Help
	var helpText string
	if m.confirming != confirmNone {
		helpText = "[y / Enter] Proceed with Sudo • [n / Esc] Cancel"
	} else {
		switch m.activeTab {
		case tabNews:
			helpText = "[Tab/1-4] Switch • [j/k] Navigate • [o/Enter] Open in Browser • [r] Refresh • [q] Quit"
		case tabUpdates:
			helpText = "[Tab/1-4] Switch • [j/k] Navigate • [u] Safe Upgrade (pacman -Syu) • [r] Refresh • [q] Quit"
		case tabOrphans:
			helpText = "[Tab/1-4] Switch • [Space] Toggle • [a] All • [d] Delete Selected • [r] Refresh • [q] Quit"
		case tabAudit:
			helpText = "[Tab/1-4] Switch • [j/k] Navigate • [c] Clean Cache (paccache -rk2) • [r] Refresh • [q] Quit"
		}
	}

	help := HelpStyle.Render(helpText + "   (" + m.statusMsg + ")")
	b.WriteString("\n" + help + "\n")

	return b.String()
}

func (m Model) renderNewsTab() string {
	if m.newsLoading {
		return fmt.Sprintf("\n  %s Fetching latest official Arch Linux announcements...", m.spinner.View())
	}
	if m.newsErr != nil {
		return lipgloss.NewStyle().Foreground(WarningRed).Render(fmt.Sprintf("Failed to fetch RSS news: %v (Press 'r' to retry)", m.newsErr))
	}
	if len(m.news) == 0 {
		return "No news items available."
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ArchCyan).Render("Official Arch Linux News & Advisories:") + "\n\n")

	for i, item := range m.news {
		cursor := "  "
		if i == m.newsCursor {
			cursor = "▶ "
		}

		itemTitle := item.Title
		if item.RequiresIntervention {
			itemTitle = WarningBoxStyle.Render("MANUAL ACTION REQUIRED") + " " + itemTitle
		}

		if item.MatchedPackage != "" {
			itemTitle += " " + lipgloss.NewStyle().Foreground(Highlight).Render(fmt.Sprintf("[Installed: %s]", item.MatchedPackage))
		}

		dateStr := item.PubDate.Format("2006-01-02")
		line := fmt.Sprintf("%s%s  (%s)", cursor, itemTitle, lipgloss.NewStyle().Foreground(MutedGray).Render(dateStr))

		if i == m.newsCursor {
			b.WriteString(SelectedItemStyle.Render(line) + "\n")
			if item.Snippet != "" {
				snippetBox := lipgloss.NewStyle().Foreground(White).PaddingLeft(4).Render(item.Snippet)
				b.WriteString(snippetBox + "\n")
			}
		} else {
			b.WriteString(NormalItemStyle.Render(line) + "\n")
		}
	}

	return b.String()
}

func (m Model) renderUpdatesTab() string {
	if len(m.updates) == 0 {
		return lipgloss.NewStyle().Foreground(SuccessGreen).Render("✔ System is completely up to date! No pending updates found.")
	}

	var b strings.Builder
	header := fmt.Sprintf("Pending Packages: %d", len(m.updates))
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ArchCyan).Render(header) + "\n\n")

	// Calculate maximum package name width for uniform column alignment
	maxNameLen := 26
	for _, u := range m.updates {
		if len(u.Name) > maxNameLen && len(u.Name) <= 34 {
			maxNameLen = len(u.Name)
		}
	}

	// Simple scroll window if list is very large
	maxVisible := 14
	startIdx := 0
	if m.updatesCursor >= maxVisible {
		startIdx = m.updatesCursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(m.updates) {
		endIdx = len(m.updates)
	}

	if startIdx > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(MutedGray).Render(fmt.Sprintf("  ▲ ... %d more packages above ...", startIdx)) + "\n")
	}

	for i := startIdx; i < endIdx; i++ {
		u := m.updates[i]
		cursor := "  "
		if i == m.updatesCursor {
			cursor = "▶ "
		}

		badge := ""
		if u.IsCritical {
			badge = lipgloss.NewStyle().Background(WarningRed).Foreground(White).Bold(true).Render(" CRITICAL ") + " "
		}

		line := fmt.Sprintf("%s%s%-*s  %-14s -> %s", cursor, badge, maxNameLen, u.Name, u.OldVersion, lipgloss.NewStyle().Foreground(SuccessGreen).Render(u.NewVersion))
		if i == m.updatesCursor {
			b.WriteString(SelectedItemStyle.Render(line) + "\n")
		} else if u.IsCritical {
			b.WriteString(CriticalItemStyle.Render(line) + "\n")
		} else {
			b.WriteString(NormalItemStyle.Render(line) + "\n")
		}
	}

	if endIdx < len(m.updates) {
		b.WriteString(lipgloss.NewStyle().Foreground(MutedGray).Render(fmt.Sprintf("  ▼ ... %d more packages below ...", len(m.updates)-endIdx)) + "\n")
	}

	return b.String()
}

func (m Model) renderOrphansTab() string {
	if len(m.orphans) == 0 {
		return lipgloss.NewStyle().Foreground(SuccessGreen).Render("✔ No orphan packages found. Your system dependencies are clean!")
	}

	var b strings.Builder
	var selectedList []string
	for _, o := range m.orphans {
		if o.Selected {
			selectedList = append(selectedList, o.Name)
		}
	}

	header := fmt.Sprintf("Orphan Packages Detected: %d (Unneeded Dependencies)", len(m.orphans))
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ArchCyan).Render(header) + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(MutedGray).Render("Press [Space] to toggle, [a] to toggle all, [d] to delete selected") + "\n\n")

	if len(selectedList) > 0 {
		cmdPreview := fmt.Sprintf("Suggested Removal: sudo pacman -Rns %s", strings.Join(selectedList, " "))
		b.WriteString(lipgloss.NewStyle().Foreground(Highlight).Render(cmdPreview) + "\n\n")
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(MutedGray).Render("Select packages below to generate removal command.") + "\n\n")
	}

	maxOrphanLen := 22
	for _, o := range m.orphans {
		if len(o.Name) > maxOrphanLen && len(o.Name) <= 30 {
			maxOrphanLen = len(o.Name)
		}
	}

	maxVisible := 14
	startIdx := 0
	if m.orphansCursor >= maxVisible {
		startIdx = m.orphansCursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(m.orphans) {
		endIdx = len(m.orphans)
	}

	if startIdx > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(MutedGray).Render(fmt.Sprintf("  ▲ ... %d more orphans above ...", startIdx)) + "\n")
	}

	for i := startIdx; i < endIdx; i++ {
		o := m.orphans[i]
		cursor := "  "
		if i == m.orphansCursor {
			cursor = "▶ "
		}

		checkbox := "[ ] "
		if o.Selected {
			checkbox = "[x] "
		}

		sizeStr := ""
		if o.Size != "" {
			sizeStr = fmt.Sprintf("(%s)", o.Size)
		}

		line := fmt.Sprintf("%s%s%-*s %-12s %s", cursor, checkbox, maxOrphanLen, o.Name, lipgloss.NewStyle().Foreground(Highlight).Render(sizeStr), lipgloss.NewStyle().Foreground(MutedGray).Render(o.Description))

		if i == m.orphansCursor {
			b.WriteString(SelectedItemStyle.Render(line) + "\n")
		} else {
			b.WriteString(NormalItemStyle.Render(line) + "\n")
		}
	}

	if endIdx < len(m.orphans) {
		b.WriteString(lipgloss.NewStyle().Foreground(MutedGray).Render(fmt.Sprintf("  ▼ ... %d more orphans below ...", len(m.orphans)-endIdx)) + "\n")
	}

	return b.String()
}

func (m Model) renderAuditTab() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ArchCyan).Render("System & Cache Health Overview:") + "\n\n")

	// Cache info
	b.WriteString(fmt.Sprintf("• Pacman Cache Size (/var/cache/pacman/pkg): %s\n",
		lipgloss.NewStyle().Bold(true).Foreground(Highlight).Render(m.cacheSize)))
	b.WriteString("  Shortcut: Press [c] to run `sudo paccache -rk2` directly\n\n")

	// Pacnew files
	b.WriteString(fmt.Sprintf("• Unmerged .pacnew files in /etc: %d found\n", len(m.pacnewFiles)))
	if len(m.pacnewFiles) > 0 {
		for _, f := range m.pacnewFiles {
			b.WriteString(fmt.Sprintf("  - %s\n", lipgloss.NewStyle().Foreground(WarningRed).Render(f)))
		}
		b.WriteString("  Tip: Merge configurations using `pacdiff`\n\n")
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(SuccessGreen).Render("  ✔ All /etc configs are cleanly merged.") + "\n\n")
	}

	// Recommended safe update flow
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ArchCyan).Render("Recommended Safe Upgrade Procedure:") + "\n")
	b.WriteString("  1. Review any 'MANUAL ACTION REQUIRED' in Tab 1 before upgrading ([o] opens link)\n")
	b.WriteString("  2. In Tab 2, press [u] to apply pending updates (`sudo pacman -Syu`)\n")
	b.WriteString("  3. In Tab 3, select unneeded orphans and press [d] to purge\n")
	b.WriteString("  4. In Tab 4, press [c] to trim old pacman cache\n")

	return b.String()
}
