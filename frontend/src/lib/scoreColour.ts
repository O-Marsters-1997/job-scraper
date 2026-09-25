export const MATCHED_COLOUR = "#059669";
export const MISSING_COLOUR = "#dc2626";
export const WARNING_COLOUR = "#d97706";

export function scoreColour(n: number): string {
	if (n >= 80) return MATCHED_COLOUR;
	if (n >= 65) return "#10b981";
	if (n >= 50) return WARNING_COLOUR;
	return MISSING_COLOUR;
}

export function tint(
	colour: string,
	opts?: { bg?: number; fg?: number },
): { background: string; color: string } {
	const bg = opts?.bg ?? 12;
	const fg = opts?.fg ?? 80;
	return {
		background: `color-mix(in srgb, ${colour} ${bg}%, white)`,
		color: `color-mix(in srgb, ${colour} ${fg}%, black)`,
	};
}
