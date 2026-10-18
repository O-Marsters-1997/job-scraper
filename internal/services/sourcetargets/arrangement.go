package sourcetargets

import (
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

type verdict int

const (
	verdictReject verdict = iota
	verdictVerify
	verdictAccept
)

func arrangementVerdict(card dto.Job, filters map[string]string) verdict {
	switch filters["arrangement"] {
	case sourcespec.LinkedInRemote:
		switch {
		case card.WorkArrangement == "remote":
			return verdictAccept
		case filters["city_cards"] == sourcespec.LinkedInCityCardsFullPage && card.RemoteCheckURL != "":
			return verdictVerify
		}
		return verdictReject
	case sourcespec.LinkedInOnsite, sourcespec.LinkedInHybrid:
		if card.WorkArrangement == "remote" {
			return verdictReject
		}
	}
	return verdictAccept
}

func detailCard(card dto.Job, targetFilters []map[string]string) (dto.Job, bool) {
	if card.Source != "linkedin" || len(targetFilters) == 0 {
		return card, true
	}
	best := verdictReject
	for _, filters := range targetFilters {
		best = max(best, arrangementVerdict(card, filters))
	}
	if best == verdictAccept {
		card.RemoteCheckURL = ""
	}
	return card, best != verdictReject
}
