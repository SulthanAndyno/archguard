package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/SulthanAndyno/archguard/internal/arch"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModelTabSwitching(t *testing.T) {
	m := NewModel()

	// Initial tab should be 0 (tabNews)
	if m.activeTab != tabNews {
		t.Errorf("Expected initial tab %d, got %d", tabNews, m.activeTab)
	}

	// Press Tab to cycle forward
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newM.(Model)
	if m.activeTab != tabUpdates {
		t.Errorf("Expected tab %d after Tab, got %d", tabUpdates, m.activeTab)
	}

	// Jump to tab 3 directly
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = newM.(Model)
	if m.activeTab != tabOrphans {
		t.Errorf("Expected tab %d after pressing '3', got %d", tabOrphans, m.activeTab)
	}

	// Shift+Tab backward
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = newM.(Model)
	if m.activeTab != tabUpdates {
		t.Errorf("Expected tab %d after Shift+Tab, got %d", tabUpdates, m.activeTab)
	}
}

func TestModelOrphanToggleAndSelectAll(t *testing.T) {
	m := NewModel()
	m.activeTab = tabOrphans
	m.orphans = []arch.OrphanPackage{
		{Name: "pkg-a", Selected: false},
		{Name: "pkg-b", Selected: false},
	}

	// Press Space to toggle first orphan
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m = newM.(Model)

	if !m.orphans[0].Selected {
		t.Errorf("Expected orphans[0] to be selected after spacebar")
	}
	if m.orphans[1].Selected {
		t.Errorf("Expected orphans[1] to remain unselected")
	}

	// Press 'a' to select all
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newM.(Model)

	if !m.orphans[0].Selected || !m.orphans[1].Selected {
		t.Errorf("Expected all orphans to be selected after pressing 'a'")
	}

	// Press 'a' again to deselect all
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newM.(Model)

	if m.orphans[0].Selected || m.orphans[1].Selected {
		t.Errorf("Expected all orphans to be deselected after second 'a'")
	}
}

func TestRenderUpdatesView(t *testing.T) {
	m := NewModel()
	m.activeTab = tabUpdates
	m.localLoading = false
	m.newsLoading = false
	m.updates = []arch.PackageUpdate{
		{Name: "ark", OldVersion: "26.08.2-1", NewVersion: "26.08.3-1"},
		{Name: "ca-certificates-mozilla", OldVersion: "3.132-1", NewVersion: "3.133-1"},
		{Name: "docker", OldVersion: "1:29.9.2-1", NewVersion: "1:29.9.3-1"},
	}
	output := m.View()
	if !strings.Contains(output, "Pending Packages: 3") {
		t.Errorf("Expected output to contain 'Pending Packages: 3'")
	}
	if !strings.Contains(output, "ca-certificates-mozilla") {
		t.Errorf("Expected output to contain package name")
	}
}

func TestModelDataIngestion(t *testing.T) {
	m := NewModel()

	// Ingest local data
	localMsg := LocalDataMsg{
		Updates: []arch.PackageUpdate{
			{Name: "linux", OldVersion: "6.12.1", NewVersion: "6.12.2", IsCritical: true},
		},
		Orphans: []arch.OrphanPackage{
			{Name: "unused-lib", Size: "10 MiB"},
		},
		Installed: map[string]bool{
			"linux": true,
		},
		CacheSize: "2.5G",
	}

	newM, _ := m.Update(localMsg)
	m = newM.(Model)

	if m.localLoading {
		t.Errorf("Expected localLoading to be false after LocalDataMsg")
	}
	if len(m.updates) != 1 || m.updates[0].Name != "linux" {
		t.Errorf("Expected updates to be populated")
	}
	if m.cacheSize != "2.5G" {
		t.Errorf("Expected cacheSize '2.5G', got %q", m.cacheSize)
	}

	// Ingest news data
	newsMsg := NewsDataMsg{
		News: []arch.NewsItem{
			{
				Title:                "Manual intervention required for linux",
				PubDate:              time.Now(),
				RequiresIntervention: true,
			},
		},
	}

	newM, _ = m.Update(newsMsg)
	m = newM.(Model)

	if m.newsLoading {
		t.Errorf("Expected newsLoading to be false after NewsDataMsg")
	}
	if len(m.news) != 1 {
		t.Fatalf("Expected 1 news item")
	}
	// Check intervention correlation with installed map
	if m.news[0].MatchedPackage != "linux" {
		t.Errorf("Expected matched package 'linux', got %q", m.news[0].MatchedPackage)
	}

	// Test View rendering does not crash
	viewOutput := m.View()
	if viewOutput == "" {
		t.Errorf("Expected non-empty View output")
	}
}

func TestModelActionConfirmationFlow(t *testing.T) {
	m := NewModel()
	m.activeTab = tabOrphans
	m.orphans = []arch.OrphanPackage{
		{Name: "orphan-lib", Selected: false},
	}

	// Press 'd' without selection -> should not confirm
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newM.(Model)
	if m.confirming != confirmNone {
		t.Errorf("Expected confirming to be none when no orphans selected")
	}

	// Select orphan and press 'd' -> should activate confirmDeleteOrphans
	m.orphans[0].Selected = true
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newM.(Model)
	if m.confirming != confirmDeleteOrphans {
		t.Errorf("Expected confirmDeleteOrphans, got %d", m.confirming)
	}

	// Press 'n' to cancel
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newM.(Model)
	if m.confirming != confirmNone {
		t.Errorf("Expected confirming to be reset to none after 'n'")
	}

	// In tabAudit, press 'c' -> confirmCleanCache
	m.activeTab = tabAudit
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newM.(Model)
	if m.confirming != confirmCleanCache {
		t.Errorf("Expected confirmCleanCache, got %d", m.confirming)
	}

	// Cancel cache confirmation with escape
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = newM.(Model)
	if m.confirming != confirmNone {
		t.Errorf("Expected confirming to be reset after escape")
	}

	// In tabUpdates, press 'u' -> confirmUpgradeSystem
	m.activeTab = tabUpdates
	m.updates = []arch.PackageUpdate{{Name: "glibc"}}
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = newM.(Model)
	if m.confirming != confirmUpgradeSystem {
		t.Errorf("Expected confirmUpgradeSystem, got %d", m.confirming)
	}

	// Cancel upgrade confirmation
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = newM.(Model)

	// Verify actionFinishedMsg re-triggers data refresh
	newM, cmd := m.Update(actionFinishedMsg{action: "test_action", err: nil})
	m = newM.(Model)
	if !m.localLoading {
		t.Errorf("Expected localLoading to be true after actionFinishedMsg")
	}
	if cmd == nil {
		t.Errorf("Expected tea.Cmd to be returned for data refresh")
	}
}
