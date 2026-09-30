import { createSignal } from "solid-js";
import { FONT_KEYS, THEME_KEYS } from "./tweaks.themes";

export type ThemeKey =
	| "violet"
	| "midnight"
	| "ember"
	| "forest"
	| "plum"
	| "graphite"
	| "custom";
export type FontKey = "jakarta" | "dm" | "sora" | "outfit" | "ibm";
export type SizeKey = "xs" | "sm" | "md" | "lg";
export type SidebarWidthKey = "narrow" | "default" | "wide";
export type DensityKey = "compact" | "default" | "spacious";
export type RadiusKey = "sharp" | "default" | "round";

export interface Tweaks {
	theme: ThemeKey;
	font: FontKey;
	size: SizeKey;
	sidebarWidth: SidebarWidthKey;
	density: DensityKey;
	radius: RadiusKey;
	customColors: Record<string, string>;
}

export const CUSTOM_DEFAULTS: Record<string, string> = {
	"--color-sidebar": "oklch(0.23 0.055 285)",
	"--color-sidebar-hover": "oklch(0.29 0.06 285)",
	"--color-sidebar-active": "oklch(0.62 0.20 300 / 0.16)",
	"--color-sidebar-active-foreground": "oklch(0.84 0.13 295)",
	"--color-sidebar-foreground": "oklch(0.70 0.04 285)",
	"--color-sidebar-foreground-strong": "oklch(0.95 0.02 290)",
	"--color-sidebar-border": "oklch(0.31 0.06 285)",
	"--color-background": "oklch(0.9842 0.0034 247.86)",
	"--color-surface": "oklch(1 0 0)",
	"--color-surface-muted": "oklch(0.9876 0.0017 247.84)",
	"--color-foreground": "oklch(0.2077 0.0398 265.75)",
	"--color-muted": "oklch(0.4455 0.0374 257.28)",
	"--color-faint": "oklch(0.5544 0.0407 256.79)",
	"--color-border": "oklch(0.9288 0.0126 255.51)",
	"--color-border-strong": "oklch(0.869 0.0198 252.89)",
	"--color-primary": "oklch(0.55 0.18 285)",
	"--color-primary-hover": "oklch(0.47 0.17 285)",
	"--color-accent-subtle": "oklch(0.97 0.015 290)",
	"--color-accent-border": "oklch(0.90 0.06 288)",
	"--color-accent-text": "oklch(0.40 0.13 285)",
};

export const DEFAULTS: Tweaks = {
	theme: "violet",
	font: "jakarta",
	size: "sm",
	sidebarWidth: "default",
	density: "default",
	radius: "default",
	customColors: CUSTOM_DEFAULTS,
};

export const STORAGE_KEY = "job-scraper-tweaks";

export const THEME_VAR_NAMES = [
	"--color-sidebar",
	"--color-sidebar-hover",
	"--color-sidebar-active",
	"--color-sidebar-active-foreground",
	"--color-sidebar-foreground",
	"--color-sidebar-foreground-strong",
	"--color-sidebar-border",
	"--color-background",
	"--color-surface",
	"--color-surface-muted",
	"--color-foreground",
	"--color-muted",
	"--color-faint",
	"--color-border",
	"--color-border-strong",
	"--color-primary",
	"--color-primary-hover",
	"--color-accent-subtle",
	"--color-accent-border",
	"--color-accent-text",
];

interface ThemeVarGroup {
	label: string;
	vars: { key: string; label: string }[];
}

export const THEME_VAR_GROUPS: ThemeVarGroup[] = [
	{
		label: "Sidebar",
		vars: [
			{ key: "--color-sidebar", label: "Background" },
			{ key: "--color-sidebar-hover", label: "Hover" },
			{ key: "--color-sidebar-active", label: "Active bg" },
			{ key: "--color-sidebar-active-foreground", label: "Active text" },
			{ key: "--color-sidebar-foreground", label: "Text" },
			{ key: "--color-sidebar-foreground-strong", label: "Text strong" },
			{ key: "--color-sidebar-border", label: "Border" },
		],
	},
	{
		label: "Surfaces",
		vars: [
			{ key: "--color-background", label: "Canvas" },
			{ key: "--color-surface", label: "Surface" },
			{ key: "--color-surface-muted", label: "Surface muted" },
		],
	},
	{
		label: "Text",
		vars: [
			{ key: "--color-foreground", label: "Foreground" },
			{ key: "--color-muted", label: "Muted" },
			{ key: "--color-faint", label: "Faint" },
		],
	},
	{
		label: "Borders",
		vars: [
			{ key: "--color-border", label: "Border" },
			{ key: "--color-border-strong", label: "Border strong" },
		],
	},
	{
		label: "Accent",
		vars: [
			{ key: "--color-primary", label: "Primary" },
			{ key: "--color-primary-hover", label: "Primary hover" },
			{ key: "--color-accent-subtle", label: "Subtle bg" },
			{ key: "--color-accent-border", label: "Border" },
			{ key: "--color-accent-text", label: "Text" },
		],
	},
];

function savedChoice<T extends string>(
	value: unknown,
	choices: readonly T[],
	fallback: T,
): T {
	return typeof value === "string" && choices.includes(value as T)
		? (value as T)
		: fallback;
}

export function loadTweaks(): Tweaks {
	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			const parsed: unknown = JSON.parse(stored);
			if (!parsed || typeof parsed !== "object" || Array.isArray(parsed))
				return DEFAULTS;
			const value = parsed as Record<string, unknown>;
			const savedColors = value.customColors;
			const colors =
				savedColors &&
				typeof savedColors === "object" &&
				!Array.isArray(savedColors)
					? (savedColors as Record<string, unknown>)
					: {};
			return {
				theme: savedChoice(value.theme, THEME_KEYS, DEFAULTS.theme),
				font: savedChoice(value.font, FONT_KEYS, DEFAULTS.font),
				size: savedChoice(value.size, ["xs", "sm", "md", "lg"], DEFAULTS.size),
				sidebarWidth: savedChoice(
					value.sidebarWidth,
					["narrow", "default", "wide"],
					DEFAULTS.sidebarWidth,
				),
				density: savedChoice(
					value.density,
					["compact", "default", "spacious"],
					DEFAULTS.density,
				),
				radius: savedChoice(
					value.radius,
					["sharp", "default", "round"],
					DEFAULTS.radius,
				),
				customColors: Object.fromEntries(
					Object.entries(CUSTOM_DEFAULTS).map(([key, fallback]) => [
						key,
						typeof colors[key] === "string" &&
						(typeof CSS === "undefined" || CSS.supports("color", colors[key]))
							? colors[key]
							: fallback,
					]),
				),
			};
		}
	} catch {}
	return { ...DEFAULTS };
}

export function saveTweaks(t: Tweaks): void {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(t));
}

// Charts derive their colours from CSS vars at render time (Canvas can't read
// them live), so anything reading a theme-derived colour in a memo must also
// read this to recompute when the applied theme changes.
const [themeVersion, bumpThemeVersion] = createSignal(0);
export { themeVersion };
export function markThemeApplied(): void {
	bumpThemeVersion((v) => v + 1);
}
