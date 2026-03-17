package scholar

import (
	scholarlib "github.com/compscidr/scholar"
	"testing"
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
