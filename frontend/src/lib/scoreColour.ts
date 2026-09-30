export const MATCHED_COLOUR = "#059669";
export const MISSING_COLOUR = "#dc2626";
const WARNING_COLOUR = "#d97706";

export function scoreColour(n: number): string {
	if (n >= 80) return MATCHED_COLOUR;
	if (n >= 65) return "#10b981";
	if (n >= 50) return WARNING_COLOUR;
	return MISSING_COLOUR;
}
