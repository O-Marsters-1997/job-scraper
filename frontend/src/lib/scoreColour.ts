export const MATCHED_COLOUR = "#059669";
export const MISSING_COLOUR = "#dc2626";
export const GOOD_COLOUR = "#10b981";
export const WARNING_COLOUR = "#d97706";

export const tintedChip = (colour: string) => ({
	background: `color-mix(in srgb, ${colour} 12%, white)`,
	color: `color-mix(in srgb, ${colour} 65%, black)`,
	border: `1px solid color-mix(in srgb, ${colour} 28%, white)`,
});
