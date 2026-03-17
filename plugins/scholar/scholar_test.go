package scholar

import (
	"strings"
	"testing"

	scholarlib "github.com/compscidr/scholar"
)

func TestSortArticlesByDateDesc(t *testing.T) {
	articles := []*scholarlib.Article{
		{Title: "Old", Year: 2020, Month: 3, Day: 15},
		{Title: "Newest", Year: 2025, Month: 1, Day: 10},
		{Title: "SameYearLater", Year: 2023, Month: 11, Day: 5},
		{Title: "SameYearEarlier", Year: 2023, Month: 2, Day: 20},
		{Title: "SameYearMonth", Year: 2023, Month: 11, Day: 1},
	}

	sortArticlesByDateDesc(articles)

	expected := []string{"Newest", "SameYearLater", "SameYearMonth", "SameYearEarlier", "Old"}
	for i, title := range expected {
		if articles[i].Title != title {
			t.Errorf("position %d: expected %q, got %q", i, title, articles[i].Title)
		}
	}
}

func TestRenderArticlesHTML_Empty(t *testing.T) {
	result := renderArticlesHTML(nil)
	if !strings.Contains(result, "No publications found") {
		t.Errorf("expected 'No publications found' for empty list, got %q", result)
	}
}

func TestRenderArticlesHTML_WithArticles(t *testing.T) {
	articles := []*scholarlib.Article{
		{
			Title:        "Test Paper",
			Authors:      "Alice, Bob",
			ScholarURL:   "https://scholar.google.com/test",
			Year:         2024,
			Journal:      "Test Journal",
			NumCitations: 42,
		},
	}
	result := renderArticlesHTML(articles)

	if !strings.Contains(result, "Test Paper") {
		t.Error("expected title in output")
	}
	if !strings.Contains(result, "Alice, Bob") {
		t.Error("expected authors in output")
	}
	if !strings.Contains(result, "2024") {
		t.Error("expected year in output")
	}
	if !strings.Contains(result, "Test Journal") {
		t.Error("expected journal in output")
	}
	if !strings.Contains(result, "42 citations") {
		t.Error("expected citation count in output")
	}
	if !strings.Contains(result, `href="https://scholar.google.com/test"`) {
		t.Error("expected scholar URL in href")
	}
}

func TestRenderArticlesHTML_XSSEscaping(t *testing.T) {
	articles := []*scholarlib.Article{
		{
			Title:      `<script>alert("xss")</script>`,
			Authors:    `Bob "the hacker"`,
			ScholarURL: "https://scholar.google.com/safe",
		},
	}
	result := renderArticlesHTML(articles)

	if strings.Contains(result, "<script>") {
		t.Error("title should be HTML-escaped")
	}
	if strings.Contains(result, `"the hacker"`) {
		t.Error("authors should be HTML-escaped")
	}
}

func TestSafeHref(t *testing.T) {
	tests := []struct {
		input string
		safe  bool
	}{
		{"https://scholar.google.com/test", true},
		{"http://example.com", true},
		{"javascript:alert(1)", false},
		{"data:text/html,<h1>hi</h1>", false},
		{"ftp://files.example.com", false},
		{"", false},
	}
	for _, tt := range tests {
		result := safeHref(tt.input)
		if tt.safe && result == "" {
			t.Errorf("expected %q to be safe, got empty", tt.input)
		}
		if !tt.safe && result != "" {
			t.Errorf("expected %q to be blocked, got %q", tt.input, result)
		}
	}
}
