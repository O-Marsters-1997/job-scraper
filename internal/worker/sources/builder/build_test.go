package builder_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
)

func TestBuildSource_EveryRegisteredSourceInstantiates(t *testing.T) {
	for _, info := range sourcespec.Sources() {
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

		src, ok := builder.BuildSource(target)
		if !ok || src.Cfg().Name != info.Name {
			t.Errorf("source %q (kind %q) is registered but not wired into BuildSource", info.Name, info.Kind)
		}
	}
}

func TestBuildSource_Wis(t *testing.T) {
	target := dto.SourceTarget{Source: "wis", Value: "product engineer", Filters: map[string]string{"region": "uk"}, Enabled: true}
	src, ok := builder.BuildSource(target)
	if !ok || src.Cfg().Name != "wis" {
		t.Error("expected a wis source")
	}
}

func TestBuildSource_BoardSource(t *testing.T) {
	target := dto.SourceTarget{Source: "greenhouse", Value: "acme", Enabled: true}
	src, ok := builder.BuildSource(target)
	if !ok || src.Cfg().Name != "greenhouse" {
		t.Error("expected a greenhouse source")
	}
}

func TestBuildSource_DisabledTargetExcluded(t *testing.T) {
	target := dto.SourceTarget{Source: "greenhouse", Value: "acme", Enabled: false}
	if _, ok := builder.BuildSource(target); ok {
		t.Error("disabled target should not produce a source")
	}
}

func TestBuildSource_URLSource(t *testing.T) {
	target := dto.SourceTarget{Source: "indeed", Value: "https://www.indeed.com/jobs?q=engineer", Enabled: true}
	src, ok := builder.BuildSource(target)
	if !ok || src.Cfg().Name != "indeed" {
		t.Error("expected an indeed source")
	}
}

func TestBuildSource_LinkedIn(t *testing.T) {
	target := dto.SourceTarget{Source: "linkedin", Value: "product engineer", Filters: map[string]string{"location": "London"}, Enabled: true}
	src, ok := builder.BuildSource(target)
	if !ok || src.Cfg().Name != "linkedin" {
		t.Error("expected a linkedin source")
	}
}

func TestBuildSource_UnknownSourceIgnored(t *testing.T) {
	target := dto.SourceTarget{Source: "unknown-ats", Value: "sometoken", Enabled: true}
	if _, ok := builder.BuildSource(target); ok {
		t.Error("unknown source should not produce a source")
	}
}
