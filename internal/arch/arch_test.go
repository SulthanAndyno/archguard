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
			Title:   "Update to ca-certificates-mozilla >= 3.131.1-1 to prevent MITM",
			PubDate: time.Now(),
		},
		{
			Title:   "Standard kernel release notes",
			PubDate: time.Now(),
		},
	}

	installed := map[string]bool{
		"glibc":                  true,
		"ca-certificates-mozilla": true,
		"ca":                     true,
		"firefox":                true,
	}

	result := CheckInterventions(items, installed)

	if result[0].MatchedPackage != "glibc" {
		t.Errorf("Expected matched package 'glibc', got %q", result[0].MatchedPackage)
	}

	// Should match ca-certificates-mozilla, NOT just 'ca'
	if result[1].MatchedPackage != "ca-certificates-mozilla" {
		t.Errorf("Expected 'ca-certificates-mozilla' (longest match), got %q", result[1].MatchedPackage)
	}

	if result[2].MatchedPackage != "" {
		t.Errorf("Expected no match for item 2, got %q", result[2].MatchedPackage)
	}
}

func TestParseCheckupdatesOutput(t *testing.T) {
	sampleOutput := []byte(`
linux 6.12.1-arch1 -> 6.12.2-arch1
firefox 132.0.1-1 -> 132.0.2-1
nvidia 565.77-1 -> 565.77-2
invalid_line_without_arrow
`)

	updates := parseCheckupdatesOutput(sampleOutput)
	if len(updates) != 3 {
		t.Fatalf("Expected 3 valid updates, got %d", len(updates))
	}

	if updates[0].Name != "linux" || !updates[0].IsCritical {
		t.Errorf("Expected critical package 'linux', got %+v", updates[0])
	}
	if updates[1].Name != "firefox" || updates[1].IsCritical {
		t.Errorf("Expected non-critical package 'firefox', got %+v", updates[1])
	}
	if updates[2].Name != "nvidia" || !updates[2].IsCritical {
		t.Errorf("Expected critical package 'nvidia', got %+v", updates[2])
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
