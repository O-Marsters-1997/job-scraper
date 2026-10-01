import { createSignal } from "solid-js";
import type { ScoringConfig } from "../../../../types/scoringConfig";

export function useExclusionFilters(initial: ScoringConfig) {
	const [titleKeywords, setTitleKeywords] = createSignal(
		initial.excludedTitleKeywords,
	);
	const [companies, setCompanies] = createSignal(initial.excludedCompanies);
	const [locations, setLocations] = createSignal(initial.excludedLocations);
	const [requiredLocations, setRequiredLocations] = createSignal(
		initial.requiredLocations,
	);
	const [requiredTitleKeywords, setRequiredTitleKeywords] = createSignal(
		initial.requiredTitleKeywords,
	);
	const [threshold, setThreshold] = createSignal(initial.notifyThreshold);
	return {
		titleKeywords,
		setTitleKeywords,
		companies,
		setCompanies,
		locations,
		setLocations,
		requiredLocations,
		setRequiredLocations,
		requiredTitleKeywords,
		setRequiredTitleKeywords,
		threshold,
		setThreshold,
	};
}

export type ExclusionFilters = ReturnType<typeof useExclusionFilters>;
