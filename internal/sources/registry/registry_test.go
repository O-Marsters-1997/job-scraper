package registry_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources/registry"
)

func TestGreenhouseRegistrationMatchesSourceInfo(t *testing.T) {
	entry, err := registry.Open(dto.SourceTarget{Source: "greenhouse", Value: "acme", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range registry.Sources() {
		if info.Name == entry.Name {
			if info.Role != entry.Role || info.Kind != "board" || entry.RequestGap <= 0 {
				t.Fatalf("source info = %+v, entry = %+v", info, entry)
			}
			return
		}
	}
	t.Fatal("greenhouse missing from registry")
}
