package api

import "testing"

func TestSeededWorldRegions(t *testing.T) {
	store := NewMarketStore()
	regions := store.Regions()
	expected := []string{
		"Northern Alliance",
		"Eastern Coalition",
		"Southern Resource Pact",
		"Oceanic Tech Arch",
	}
	if len(regions) < len(expected) {
		t.Fatalf("expected at least %d regions, got %d", len(expected), len(regions))
	}
	seen := make(map[string]bool)
	for _, region := range regions {
		seen[region.Name] = true
	}
	for _, name := range expected {
		if !seen[name] {
			t.Fatalf("expected region %q to be seeded", name)
		}
	}
}

func TestSeededWorldEvents(t *testing.T) {
	store := NewMarketStore()
	events := store.WorldEvents()
	expected := []string{
		"Tech Bubble Burst",
		"Resource War",
		"Digital Currency Crisis",
		"Boros Election",
		"Arcadia Privacy Act",
		"El Dorado Succession",
	}
	if len(events) < len(expected) {
		t.Fatalf("expected at least %d world events, got %d", len(expected), len(events))
	}
	seen := make(map[string]bool)
	for _, event := range events {
		if event.StartsAt == 0 || event.EndsAt == 0 || event.EndsAt <= event.StartsAt {
			t.Fatalf("invalid event timing for %q: %d -> %d", event.Name, event.StartsAt, event.EndsAt)
		}
		seen[event.Name] = true
	}
	for _, name := range expected {
		if !seen[name] {
			t.Fatalf("expected world event %q to be seeded", name)
		}
	}
}
