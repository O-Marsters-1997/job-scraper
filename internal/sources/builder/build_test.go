package builder_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources/builder"
)

func TestBuildSources_WisAlwaysIncluded(t *testing.T) {
	srcs := builder.BuildSources(nil)
	if len(srcs) == 0 {
		t.Fatal("expected at least wis source")
	}
	if srcs[0].Cfg().Name != "wis" {
		t.Errorf("want first source to be wis, got %s", srcs[0].Cfg().Name)
	}
}

func TestBuildSources_BoardSourcesGrouped(t *testing.T) {
	targets := []dto.SourceTarget{
		{Source: "greenhouse", Value: "acme", Enabled: true},
		{Source: "greenhouse", Value: "widgetco", Enabled: true},
		{Source: "lever", Value: "startup", Enabled: true},
	}
	srcs := builder.BuildSources(targets)

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
	srcs := builder.BuildSources(targets)

	for _, s := range srcs {
		if s.Cfg().Name == "greenhouse" {
			t.Error("disabled greenhouse target should not produce a source")
		}
	}
}

func TestBuildSources_URLSourcesIncludedWhenTargetPresent(t *testing.T) {
	targets := []dto.SourceTarget{
		{Source: "linkedin", Value: "https://www.linkedin.com/jobs/search/?keywords=engineer", Enabled: true},
	}
	srcs := builder.BuildSources(targets)

	found := false
	for _, s := range srcs {
		if s.Cfg().Name == "linkedin" {
			found = true
		}
	}
	if !found {
		t.Error("expected linkedin source when URL target is present")
	}
}

func TestBuildSources_URLSourcesExcludedWhenNoTarget(t *testing.T) {
	srcs := builder.BuildSources(nil)
	for _, s := range srcs {
		if s.Cfg().Name == "linkedin" || s.Cfg().Name == "indeed" {
			t.Errorf("expected %s to be absent when no URL target configured", s.Cfg().Name)
		}
	}
}

func TestBuildSources_UnknownSourceIgnored(t *testing.T) {
	targets := []dto.SourceTarget{
		{Source: "unknown-ats", Value: "sometoken", Enabled: true},
	}
	srcs := builder.BuildSources(targets)
	if len(srcs) == 0 {
		t.Fatal("expected at least wis")
	}
}
