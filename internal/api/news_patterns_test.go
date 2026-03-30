package api

import (
	"strings"
	"testing"
	"time"
)

func TestLoadNewsPatterns(t *testing.T) {
	library, err := loadNewsPatterns()
	if err != nil {
		t.Fatalf("expected news patterns to load: %v", err)
	}
	if library == nil || len(library.Categories) == 0 {
		t.Fatalf("expected categories in news patterns")
	}
	if _, ok := library.Categories["EARNINGS"]; !ok {
		t.Fatalf("expected EARNINGS category")
	}
}

func TestGeneratePatternNews(t *testing.T) {
	store := NewMarketStore()
	now := time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)
	items := store.generatePatternNews(now)
	if len(items) == 0 {
		t.Fatalf("expected seeded news items from patterns")
	}
	for _, item := range items {
		if item.Headline == "" {
			t.Errorf("expected headline for %+v", item)
		}
		if strings.Contains(item.Headline, "{") || strings.Contains(item.Body, "{") {
			t.Errorf("expected templates to be filled for %+v", item)
		}
		for _, scope := range item.ImpactScope {
			if strings.Contains(scope, "{") {
				t.Errorf("expected impact scope to be filled for %+v", item)
			}
		}
		if item.Category == "" {
			t.Errorf("expected category for %+v", item)
		}
	}
}
