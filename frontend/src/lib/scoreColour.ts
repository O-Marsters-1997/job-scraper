export const MATCHED_COLOUR = "var(--color-score-matched)";
export const MISSING_COLOUR = "var(--color-destructive-strong)";
export const WARNING_COLOUR = "var(--color-score-warning)";
const GOOD_COLOUR = "var(--color-score-good)";

export function scoreColour(n: number): string {
	if (n >= 80) return MATCHED_COLOUR;
	if (n >= 65) return GOOD_COLOUR;
	if (n >= 50) return WARNING_COLOUR;
	return MISSING_COLOUR;
}
