/**
 * Canonical fallback colour for application statuses with no explicit colour set.
 * Sourced from the --color-status-saved design token (the "saved" status colour).
 */
export const STATUS_FALLBACK_COLOUR = "#64748b";

/**
 * Default colour palette for the status colour picker.
 * These are deliberate user-facing swatches, not design tokens.
 */
export const STATUS_PALETTE = [
	{ hex: "#6366f1", label: "Indigo" },
	{ hex: "#22c55e", label: "Green" },
	{ hex: "#ef4444", label: "Red" },
	{ hex: "#f59e0b", label: "Amber" },
	{ hex: "#3b82f6", label: "Blue" },
	{ hex: "#a855f7", label: "Purple" },
	{ hex: STATUS_FALLBACK_COLOUR, label: "Slate" },
	{ hex: "#ec4899", label: "Pink" },
];
