/** Fallback colour for application statuses with no explicit colour set. */
export const STATUS_FALLBACK_COLOUR = "var(--color-status-saved)";

/** Colour swatch a new status starts on; the picker needs a literal hex. */
export const DEFAULT_STATUS_HEX = "#6366f1";

/**
 * Default colour palette for the status colour picker.
 * These are deliberate user-facing swatches, not design tokens.
 */
export const STATUS_PALETTE = [
	{ hex: DEFAULT_STATUS_HEX, label: "Indigo" },
	{ hex: "#22c55e", label: "Green" },
	{ hex: "#ef4444", label: "Red" },
	{ hex: "#f59e0b", label: "Amber" },
	{ hex: "#3b82f6", label: "Blue" },
	{ hex: "#a855f7", label: "Purple" },
	{ hex: "#64748b", label: "Slate" },
	{ hex: "#ec4899", label: "Pink" },
];
