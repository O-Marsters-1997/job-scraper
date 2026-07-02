package builder_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/builder"
)

// TestBuildSources_EveryRegisteredSourceInstantiates guards against the silent-drop
// trap: a source added to the registry but never wired into BuildSources compiles fine
// yet is dropped at runtime with no error. For each registered source we feed one
// enabled target of the right kind and assert BuildSources produces it.
func TestBuildSources_EveryRegisteredSourceInstantiates(t *testing.T) {
	for _, info := range sources.Sources() {
		target := dto.SourceTarget{Source: info.Name, Enabled: true}
		switch info.Kind {
		case "board":
			target.Value = "acme"
		case "filter":
			target.Value = "engineer"
		case "url":
			target.Value = "https://example.com/jobs?q=engineer"
		default:
			t.Fatalf("source %q has unknown kind %q", info.Name, info.Kind)
		}

		found := false
		for _, s := range builder.BuildSources([]dto.SourceTarget{target}, nil) {
			if s.Cfg().Name == info.Name {
				found = true
			}
		}
		if !found {
			t.Errorf("source %q (kind %q) is registered but not wired into BuildSources", info.Name, info.Kind)
		}
	}
}

func TestBuildSources_WisIncludedWhenTargetPresent(t *testing.T) {
	targets := []dto.SourceTarget{
		{Source: "wis", Value: "product engineer", Filters: map[string]string{"region": "uk"}, Enabled: true},
	}
	srcs := builder.BuildSources(targets, nil)
	found := false
	for _, s := range srcs {
		if s.Cfg().Name == "wis" {
			found = true
		}
	}
	if !found {
		t.Error("expected wis source when wis target is present")
	}
}

func TestBuildSources_WisExcludedWhenNoTarget(t *testing.T) {
	srcs := builder.BuildSources(nil, nil)
	for _, s := range srcs {
		if s.Cfg().Name == "wis" {
			t.Error("expected wis to be absent when no wis targets configured")
		}
	}
}

func TestBuildSources_BoardSourcesGrouped(t *testing.T) {
	targets := []dto.SourceTarget{
		{Source: "greenhouse", Value: "acme", Enabled: true},
		{Source: "greenhouse", Value: "widgetco", Enabled: true},
		{Source: "lever", Value: "startup", Enabled: true},
	}
	srcs := builder.BuildSources(targets, nil)

	sourceNames := make(map[string]bool)
	for _, s := range srcs {
		sourceNames[s.Cfg().Name] = true
	}

	if !sourceNames["greenhouse"] {
		t.Error("expected greenhouse source")
	}
	if !sourceNames["lever"] {
		t.Error("expected lever source")
	}

	count := 0
	for _, s := range srcs {
		if s.Cfg().Name == "greenhouse" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want 1 greenhouse source, got %d", count)
	}
}

func TestBuildSources_DisabledTargetsExcluded(t *testing.T) {
	targets := []dto.SourceTarget{
		{Source: "greenhouse", Value: "acme", Enabled: false},
	}
	srcs := builder.BuildSources(targets, nil)

	for _, s := range srcs {
		if s.Cfg().Name == "greenhouse" {
			t.Error("disabled greenhouse target should not produce a source")
		}
	}
}

func TestBuildSources_URLSourcesIncludedWhenTargetPresent(t *testing.T) {
	targets := []dto.SourceTarget{
		{Source: "indeed", Value: "https://www.indeed.com/jobs?q=engineer", Enabled: true},
	}
	srcs := builder.BuildSources(targets, nil)

	found := false
	for _, s := range srcs {
		if s.Cfg().Name == "indeed" {
			found = true
		}
	}
	if !found {
		t.Error("expected indeed source when URL target is present")
	}
}

func TestBuildSources_URLSourcesExcludedWhenNoTarget(t *testing.T) {
	srcs := builder.BuildSources(nil, nil)
	for _, s := range srcs {
		if s.Cfg().Name == "linkedin" || s.Cfg().Name == "indeed" {
			t.Errorf("expected %s to be absent when no URL target configured", s.Cfg().Name)
		}
	}
}

func TestBuildSources_LinkedInIncludedWhenTargetPresent(t *testing.T) {
	targets := []dto.SourceTarget{
		{Source: "linkedin", Value: "product engineer", Filters: map[string]string{"location": "London"}, Enabled: true},
	}
	srcs := builder.BuildSources(targets, nil)

	found := false
	for _, s := range srcs {
		if s.Cfg().Name == "linkedin" {
			found = true
		}
	}
	if !found {
		t.Error("expected linkedin source when linkedin target is present")
	}
}

func TestBuildSources_UnknownSourceIgnored(t *testing.T) {
	targets := []dto.SourceTarget{
		{Source: "unknown-ats", Value: "sometoken", Enabled: true},
	}
	srcs := builder.BuildSources(targets, nil)
	// Unknown source produces no output; result is an empty slice, not a panic.
	for _, s := range srcs {
		if s.Cfg().Name == "unknown-ats" {
			t.Errorf("unexpected source for unknown-ats")
		}
	}
}
