export const MATCH_THRESHOLD = 0.5;
export const LOW_CONFIDENCE_THRESHOLD = 0.4;

export interface CriterionScore {
	key: string;
	probability: number;
	matched: boolean;
}

export function criterionScores(
	criteria: Record<string, number> | null | undefined,
): CriterionScore[] {
	if (!criteria) return [];
	return Object.entries(criteria).map(([key, probability]) => ({
		key,
		probability,
		matched: probability >= MATCH_THRESHOLD,
	}));
}

export function isLowConfidence(
	confidence: number | null | undefined,
): boolean {
	return confidence != null && confidence < LOW_CONFIDENCE_THRESHOLD;
}
