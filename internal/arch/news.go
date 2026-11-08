package arch

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// wordRegex matches individual words/package identifiers, ignoring punctuation
var wordRegex = regexp.MustCompile(`[a-zA-Z0-9][a-zA-Z0-9_\.\-]+`)

// NewsItem represents an official Arch Linux news announcement.
type NewsItem struct {
	Title                string
	Link                 string
	PubDate              time.Time
	Snippet              string
	RequiresIntervention bool
	MatchedPackage       string
}

const ArchNewsFeedURL = "https://archlinux.org/feeds/news/"

// FetchNews downloads and parses the latest announcements from the official Arch Linux RSS feed.
func FetchNews(ctx context.Context, limit int) ([]NewsItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(ArchNewsFeedURL, ctx)
	if err != nil {
		return nil, err
	}

	var items []NewsItem
	count := 0
	for _, entry := range feed.Items {
		if limit > 0 && count >= limit {
			break
		}

		var pubDate time.Time
		if entry.PublishedParsed != nil {
			pubDate = *entry.PublishedParsed
		}

		// Clean up snippet/description (strip HTML-ish simple tags for terminal display)
		snippet := cleanHTML(entry.Description)
		if len(snippet) > 220 {
			snippet = snippet[:217] + "..."
		}

		titleLower := strings.ToLower(entry.Title)
		isIntervention := strings.Contains(titleLower, "manual intervention") ||
			strings.Contains(titleLower, "remedy") ||
			strings.Contains(titleLower, "requires intervention")

		item := NewsItem{
			Title:                entry.Title,
			Link:                 entry.Link,
			PubDate:              pubDate,
			Snippet:              snippet,
			RequiresIntervention: isIntervention,
		}

		items = append(items, item)
		count++
	}

	return items, nil
}

// CheckInterventions correlates news items with the list of locally installed packages.
// Uses word boundary tokenization to eliminate false positives and non-deterministic map order bugs.
func CheckInterventions(items []NewsItem, installedPackages map[string]bool) []NewsItem {
	for i := range items {
		titleLower := strings.ToLower(items[i].Title)
		tokens := wordRegex.FindAllString(titleLower, -1)

		var bestMatch string
		for _, token := range tokens {
			if len(token) < 2 {
				continue
			}
			if installedPackages[token] {
				// Prioritize the longest matching package name (e.g. ca-certificates-mozilla over ca)
				if len(token) > len(bestMatch) {
					bestMatch = token
				}
			}
		}
		items[i].MatchedPackage = bestMatch
	}
	return items
}

// cleanHTML does a lightweight strip of basic HTML tags for terminal preview.
func cleanHTML(input string) string {
	var b strings.Builder
	inTag := false
	for _, r := range input {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteRune(r)
		}
	}
	res := strings.ReplaceAll(b.String(), "&lt;", "<")
	res = strings.ReplaceAll(res, "&gt;", ">")
	res = strings.ReplaceAll(res, "&amp;", "&")
	res = strings.ReplaceAll(res, "&quot;", "\"")
	res = strings.ReplaceAll(res, "&#39;", "'")
	return strings.TrimSpace(res)
}
