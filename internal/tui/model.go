package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
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

// InitialDataMsg contains the result of the parallel scan on startup
type InitialDataMsg struct {
	News        []arch.NewsItem
	Updates     []arch.PackageUpdate
	Orphans     []arch.OrphanPackage
	CacheSize   string
	PacnewFiles []string
	Err         error
}

type Model struct {
	activeTab   tabIndex
	width       int
	height      int
	loading     bool
	spinner     spinner.Model
	statusMsg   string

	newsCursor    int
	updatesCursor int
	orphansCursor int

	news        []arch.NewsItem
	updates     []arch.PackageUpdate
	orphans     []arch.OrphanPackage
	cacheSize   string
	pacnewFiles []string
}

func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ArchCyan)

	return Model{
		activeTab: tabNews,
		loading:   true,
		spinner:   s,
		statusMsg: "Scanning Arch Linux environment...",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchSystemDataCmd(),
	)
}

func fetchSystemDataCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		msg := InitialDataMsg{}

		var installedPkgs map[string]bool
		var rawNews []arch.NewsItem
		var newsErr error

		wg.Add(4)

		// 1. Fetch News
		go func() {
			defer wg.Done()
			rawNews, newsErr = arch.FetchNews(ctx, 10)
		}()

		// 2. Fetch Installed Packages
		go func() {
			defer wg.Done()
			installed, _ := arch.GetInstalledPackages(ctx)
			installedPkgs = installed
		}()

		// 3. Fetch Updates
		go func() {
			defer wg.Done()
			updates, _ := arch.GetPendingUpdates(ctx)
			msg.Updates = updates
		}()

		// 4. Fetch Orphans & Cache & Pacnew
		go func() {
			defer wg.Done()
			msg.Orphans, _ = arch.GetOrphans(ctx)
			msg.CacheSize, _ = arch.GetCacheSize(ctx)
			msg.PacnewFiles, _ = arch.GetPacnewFiles()
		}()

		wg.Wait()

		if newsErr == nil && installedPkgs != nil {
			msg.News = arch.CheckInterventions(rawNews, installedPkgs)
		} else {
			msg.News = rawNews
		}

		return msg
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case InitialDataMsg:
		m.loading = false
		m.news = msg.News
		m.updates = msg.Updates
		m.orphans = msg.Orphans
		m.cacheSize = msg.CacheSize
		m.pacnewFiles = msg.PacnewFiles
		m.statusMsg = fmt.Sprintf("Ready | %d News, %d Updates, %d Orphans",
			len(m.news), len(m.updates), len(m.orphans))
		return m, nil

	case tea.KeyMsg:
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

		// Navigation within tabs
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
				return m, nil
			}

		case "r":
			// Refresh data
			m.loading = true
			m.statusMsg = "Refreshing data..."
			return m, tea.Batch(m.spinner.Tick, fetchSystemDataCmd())
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return fmt.Sprintf("\n\n   %s  %s\n\n", m.spinner.View(), m.statusMsg)
	}

	var b strings.Builder

	// 1. Header
	title := TitleStyle.Render(" ARCHGUARD ") + "  " + SubTitleStyle.Render("System Health, News & Maintenance Companion")
	b.WriteString("\n" + title + "\n\n")

	// 2. Tabs
	var renderedTabs []string
	for i, name := range tabNames {
		if tabIndex(i) == m.activeTab {
			renderedTabs = append(renderedTabs, ActiveTabStyle.Render(name))
		} else {
			renderedTabs = append(renderedTabs, InactiveTabStyle.Render(name))
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

	b.WriteString(PanelStyle.Width(92).Render(content))

	// 4. Footer & Keybindings Help
	help := HelpStyle.Render("[Tab/1-4] Switch Tab • [j/k/↑/↓] Navigate • [Space] Toggle • [r] Refresh • [q] Quit")
	b.WriteString("\n" + help + "\n")

	return b.String()
}

func (m Model) renderNewsTab() string {
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
		line := fmt.Sprintf("%s%s  (%s)\n", cursor, itemTitle, lipgloss.NewStyle().Foreground(MutedGray).Render(dateStr))

		if i == m.newsCursor {
			b.WriteString(SelectedItemStyle.Render(line))
			if item.Snippet != "" {
				snippetBox := lipgloss.NewStyle().Foreground(White).PaddingLeft(4).Render(item.Snippet) + "\n"
				b.WriteString(snippetBox)
			}
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
	}

	return b.String()
}

func (m Model) renderUpdatesTab() string {
	if len(m.updates) == 0 {
		return lipgloss.NewStyle().Foreground(SuccessGreen).Render("✔ System is completely up to date! No pending updates found.")
	}

	var b strings.Builder
	header := fmt.Sprintf("Pending Packages: %d\n\n", len(m.updates))
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ArchCyan).Render(header))

	for i, u := range m.updates {
		cursor := "  "
		if i == m.updatesCursor {
			cursor = "▶ "
		}

		badge := ""
		if u.IsCritical {
			badge = lipgloss.NewStyle().Background(WarningRed).Foreground(White).Bold(true).Render(" CRITICAL ") + " "
		}

		line := fmt.Sprintf("%s%s%-24s %s -> %s\n", cursor, badge, u.Name, u.OldVersion, lipgloss.NewStyle().Foreground(SuccessGreen).Render(u.NewVersion))
		if i == m.updatesCursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else if u.IsCritical {
			b.WriteString(CriticalItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
	}

	return b.String()
}

func (m Model) renderOrphansTab() string {
	if len(m.orphans) == 0 {
		return lipgloss.NewStyle().Foreground(SuccessGreen).Render("✔ No orphan packages found. Your system dependencies are clean!")
	}

	var b strings.Builder
	header := fmt.Sprintf("Orphan Packages Detected: %d (Unneeded Dependencies)\nPress [Space] to toggle, remove with: sudo pacman -Rns <selected>\n\n", len(m.orphans))
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ArchCyan).Render(header))

	for i, o := range m.orphans {
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

		line := fmt.Sprintf("%s%s%-22s %-12s %s\n", cursor, checkbox, o.Name, lipgloss.NewStyle().Foreground(Highlight).Render(sizeStr), lipgloss.NewStyle().Foreground(MutedGray).Render(o.Description))

		if i == m.orphansCursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
	}

	return b.String()
}

func (m Model) renderAuditTab() string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ArchCyan).Render("System & Cache Health Overview:") + "\n\n")

	// Cache info
	b.WriteString(fmt.Sprintf("• Pacman Cache Size (/var/cache/pacman/pkg): %s\n",
		lipgloss.NewStyle().Bold(true).Foreground(Highlight).Render(m.cacheSize)))
	b.WriteString("  Tip: Clean old cache versions using `sudo paccache -rk2`\n\n")

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
	b.WriteString("  1. Review any 'MANUAL ACTION REQUIRED' in Tab 1 before upgrading\n")
	b.WriteString("  2. Run `sudo pacman -Syu` to apply pending updates\n")
	b.WriteString("  3. Check Tab 3 to trim any orphaned dependencies (`sudo pacman -Rns ...`)\n")
	b.WriteString("  4. If kernel or systemd was upgraded, reboot when convenient\n")

	return b.String()
}
