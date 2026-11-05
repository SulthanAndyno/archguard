package arch

import (
	"testing"
	"time"
)

func TestCleanHTML(t *testing.T) {
	input := `<p>PHP versions prior to 8.5.12 are affected by &lt;a href="https://example.com"&gt;CVE-1234&lt;/a&gt;.</p>`
	expected := `PHP versions prior to 8.5.12 are affected by <a href="https://example.com">CVE-1234</a>.`
	actual := cleanHTML(input)
	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

func TestCheckInterventions(t *testing.T) {
	items := []NewsItem{
		{
			Title:   "Manual intervention required for glibc",
			PubDate: time.Now(),
		},
		{
			Title:   "Normal maintenance update",
			PubDate: time.Now(),
		},
	}

	installed := map[string]bool{
		"glibc":   true,
		"firefox": true,
	}

	result := CheckInterventions(items, installed)
	if result[0].MatchedPackage != "glibc" {
		t.Errorf("Expected matched package 'glibc', got %q", result[0].MatchedPackage)
	}
	if result[1].MatchedPackage != "" {
		t.Errorf("Expected no matched package, got %q", result[1].MatchedPackage)
	}
}

func TestParsePacmanInfo(t *testing.T) {
	sampleOutput := []byte(`
Name            : electron39
Description     : Build cross platform desktop apps
Installed Size  : 280.25 MiB

Name            : python-fasteners
Description     : Python cross-process locking
Installed Size  : 110.00 KiB
`)

	orphans := parsePacmanInfo(sampleOutput)
	if len(orphans) != 2 {
		t.Fatalf("Expected 2 orphans, got %d", len(orphans))
	}

	if orphans[0].Name != "electron39" || orphans[0].Size != "280.25 MiB" {
		t.Errorf("Unexpected orphan[0]: %+v", orphans[0])
	}

	if orphans[1].Name != "python-fasteners" || orphans[1].Size != "110.00 KiB" {
		t.Errorf("Unexpected orphan[1]: %+v", orphans[1])
	}
}
