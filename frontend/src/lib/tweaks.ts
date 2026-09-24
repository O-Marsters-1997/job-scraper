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

// OKLCH defaults mirroring the violet @theme block in src/styles.css.
// Keeping oklch so custom palette storage stays consistent with the design system.
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

// CSS variable names that theme switching can override
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

export interface ThemeVarGroup {
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

interface ThemeEntry {
	name: string;
	// Preview swatch colours (inline styles only, independent of applied theme).
	// sb=sidebar, cv=canvas, ac=accent, su=surface card, bd=border.
	swatch: { sb: string; cv: string; ac: string; su: string; bd: string };
	// CSS variable overrides — empty means "use pristine @theme values"
	vars: Record<string, string>;
}

export const THEMES: Record<ThemeKey, ThemeEntry> = {
	violet: {
		name: "Violet",
		swatch: {
			sb: "oklch(0.23 0.055 285)",
			cv: "oklch(0.98 0.006 285)",
			ac: "oklch(0.55 0.18 285)",
			su: "oklch(1 0 0)",
			bd: "oklch(0.93 0.01 290)",
		},
		// empty = use pristine @theme values (violet is the system default)
		vars: {},
	},
	midnight: {
		name: "Midnight",
		swatch: {
			sb: "oklch(0.13 0.02 258)",
			cv: "oklch(0.984 0.005 280)",
			ac: "oklch(0.546 0.224 264)",
			su: "oklch(1 0 0)",
			bd: "oklch(0.92 0.02 252)",
		},
		vars: {
			"--color-sidebar": "oklch(0.13 0.02 258)",
			"--color-sidebar-hover": "oklch(0.18 0.03 258)",
			"--color-sidebar-active": "oklch(0.62 0.19 259 / 0.13)",
			"--color-sidebar-active-foreground": "oklch(0.81 0.10 252)",
			"--color-sidebar-foreground": "oklch(0.60 0.04 250)",
			"--color-sidebar-foreground-strong": "oklch(0.93 0.02 250)",
			"--color-sidebar-border": "oklch(0.18 0.03 258)",
			"--color-background": "oklch(0.984 0.005 280)",
			"--color-surface": "oklch(1 0 0)",
			"--color-surface-muted": "oklch(0.984 0.005 280)",
			"--color-foreground": "oklch(0.15 0.03 258)",
			"--color-muted": "oklch(0.40 0.04 252)",
			"--color-faint": "oklch(0.68 0.04 252)",
			"--color-border": "oklch(0.92 0.02 252)",
			"--color-border-strong": "oklch(0.85 0.03 252)",
			"--color-primary": "oklch(0.546 0.224 264)",
			"--color-primary-hover": "oklch(0.488 0.198 264)",
			"--color-accent-subtle": "oklch(0.976 0.014 252)",
			"--color-accent-border": "oklch(0.875 0.05 252)",
			"--color-accent-text": "oklch(0.29 0.07 252)",
		},
	},
	ember: {
		name: "Ember",
		swatch: {
			sb: "oklch(0.14 0.02 57)",
			cv: "oklch(0.988 0.006 80)",
			ac: "oklch(0.666 0.179 58)",
			su: "oklch(1 0 0)",
			bd: "oklch(0.93 0.014 82)",
		},
		vars: {
			"--color-sidebar": "oklch(0.14 0.02 57)",
			"--color-sidebar-hover": "oklch(0.19 0.025 57)",
			"--color-sidebar-active": "oklch(0.766 0.175 72 / 0.12)",
			"--color-sidebar-active-foreground": "oklch(0.888 0.141 82)",
			"--color-sidebar-foreground": "oklch(0.60 0.035 57)",
			"--color-sidebar-foreground-strong": "oklch(0.90 0.03 70)",
			"--color-sidebar-border": "oklch(0.19 0.025 57)",
			"--color-background": "oklch(0.988 0.006 80)",
			"--color-surface": "oklch(1 0 0)",
			"--color-surface-muted": "oklch(0.988 0.006 80)",
			"--color-foreground": "oklch(0.16 0.025 55)",
			"--color-muted": "oklch(0.44 0.04 55)",
			"--color-faint": "oklch(0.64 0.04 64)",
			"--color-border": "oklch(0.93 0.014 82)",
			"--color-border-strong": "oklch(0.86 0.02 82)",
			"--color-primary": "oklch(0.666 0.179 58)",
			"--color-primary-hover": "oklch(0.591 0.158 59)",
			"--color-accent-subtle": "oklch(0.988 0.022 95)",
			"--color-accent-border": "oklch(0.924 0.121 95)",
			"--color-accent-text": "oklch(0.41 0.1 58)",
		},
	},
	forest: {
		name: "Forest",
		swatch: {
			sb: "oklch(0.14 0.02 148)",
			cv: "oklch(0.988 0.008 155)",
			ac: "oklch(0.596 0.127 163)",
			su: "oklch(1 0 0)",
			bd: "oklch(0.95 0.025 155)",
		},
		vars: {
			"--color-sidebar": "oklch(0.14 0.02 148)",
			"--color-sidebar-hover": "oklch(0.19 0.025 148)",
			"--color-sidebar-active": "oklch(0.696 0.166 163 / 0.12)",
			"--color-sidebar-active-foreground": "oklch(0.871 0.093 160)",
			"--color-sidebar-foreground": "oklch(0.60 0.04 148)",
			"--color-sidebar-foreground-strong": "oklch(0.94 0.03 155)",
			"--color-sidebar-border": "oklch(0.19 0.025 148)",
			"--color-background": "oklch(0.988 0.008 155)",
			"--color-surface": "oklch(1 0 0)",
			"--color-surface-muted": "oklch(0.988 0.008 155)",
			"--color-foreground": "oklch(0.13 0.03 155)",
			"--color-muted": "oklch(0.40 0.05 152)",
			"--color-faint": "oklch(0.66 0.05 148)",
			"--color-border": "oklch(0.95 0.025 155)",
			"--color-border-strong": "oklch(0.89 0.04 155)",
			"--color-primary": "oklch(0.596 0.127 163)",
			"--color-primary-hover": "oklch(0.534 0.113 163)",
			"--color-accent-subtle": "oklch(0.984 0.018 163)",
			"--color-accent-border": "oklch(0.871 0.093 160)",
			"--color-accent-text": "oklch(0.31 0.06 163)",
		},
	},
	plum: {
		name: "Plum",
		swatch: {
			sb: "oklch(0.12 0.03 287)",
			cv: "oklch(0.985 0.006 290)",
			ac: "oklch(0.541 0.281 293)",
			su: "oklch(1 0 0)",
			bd: "oklch(0.95 0.02 290)",
		},
		vars: {
			"--color-sidebar": "oklch(0.12 0.03 287)",
			"--color-sidebar-hover": "oklch(0.18 0.04 285)",
			"--color-sidebar-active": "oklch(0.659 0.252 293 / 0.13)",
			"--color-sidebar-active-foreground": "oklch(0.807 0.121 292)",
			"--color-sidebar-foreground": "oklch(0.60 0.06 290)",
			"--color-sidebar-foreground-strong": "oklch(0.93 0.03 292)",
			"--color-sidebar-border": "oklch(0.18 0.04 285)",
			"--color-background": "oklch(0.985 0.006 290)",
			"--color-surface": "oklch(1 0 0)",
			"--color-surface-muted": "oklch(0.985 0.006 290)",
			"--color-foreground": "oklch(0.12 0.04 285)",
			"--color-muted": "oklch(0.34 0.08 288)",
			"--color-faint": "oklch(0.64 0.07 290)",
			"--color-border": "oklch(0.95 0.02 290)",
			"--color-border-strong": "oklch(0.87 0.04 290)",
			"--color-primary": "oklch(0.541 0.281 293)",
			"--color-primary-hover": "oklch(0.488 0.261 290)",
			"--color-accent-subtle": "oklch(0.984 0.007 290)",
			"--color-accent-border": "oklch(0.88 0.065 290)",
			"--color-accent-text": "oklch(0.31 0.1 285)",
		},
	},
	graphite: {
		name: "Graphite",
		swatch: {
			sb: "oklch(0.14 0 0)",
			cv: "oklch(0.97 0 0)",
			ac: "oklch(0.30 0 0)",
			su: "oklch(1 0 0)",
			bd: "oklch(0.93 0 0)",
		},
		vars: {
			"--color-sidebar": "oklch(0.14 0 0)",
			"--color-sidebar-hover": "oklch(0.18 0 0)",
			"--color-sidebar-active": "oklch(1 0 0 / 0.08)",
			"--color-sidebar-active-foreground": "oklch(0.93 0 0)",
			"--color-sidebar-foreground": "oklch(0.60 0 0)",
			"--color-sidebar-foreground-strong": "oklch(0.92 0 0)",
			"--color-sidebar-border": "oklch(0.18 0 0)",
			"--color-background": "oklch(0.97 0 0)",
			"--color-surface": "oklch(1 0 0)",
			"--color-surface-muted": "oklch(0.97 0 0)",
			"--color-foreground": "oklch(0.14 0 0)",
			"--color-muted": "oklch(0.42 0 0)",
			"--color-faint": "oklch(0.67 0 0)",
			"--color-border": "oklch(0.93 0 0)",
			"--color-border-strong": "oklch(0.85 0 0)",
			"--color-primary": "oklch(0.30 0 0)",
			"--color-primary-hover": "oklch(0.22 0 0)",
			"--color-accent-subtle": "oklch(0.95 0 0)",
			"--color-accent-border": "oklch(0.85 0 0)",
			"--color-accent-text": "oklch(0.14 0 0)",
		},
	},
	// Custom theme: swatch is a placeholder; actual colours come from customColors at runtime
	custom: {
		name: "Custom",
		swatch: {
			sb: "oklch(0.23 0.055 285)",
			cv: "oklch(0.98 0.006 285)",
			ac: "oklch(0.55 0.18 285)",
			su: "oklch(1 0 0)",
			bd: "oklch(0.93 0.01 290)",
		},
		vars: {},
	},
};

export const THEME_KEYS: ThemeKey[] = [
	"violet",
	"midnight",
	"ember",
	"forest",
	"plum",
	"graphite",
	"custom",
];

interface FontEntry {
	name: string;
	ui: string;
	mono: string;
}

export const FONTS: Record<FontKey, FontEntry> = {
	jakarta: {
		name: "Jakarta",
		ui: "'Plus Jakarta Sans', ui-sans-serif, system-ui, sans-serif",
		mono: "'JetBrains Mono', ui-monospace, Menlo, monospace",
	},
	dm: {
		name: "DM Sans",
		ui: "'DM Sans', ui-sans-serif, system-ui, sans-serif",
		mono: "'IBM Plex Mono', ui-monospace, Menlo, monospace",
	},
	sora: {
		name: "Sora",
		ui: "'Sora', ui-sans-serif, system-ui, sans-serif",
		mono: "'Fira Code', ui-monospace, Menlo, monospace",
	},
	outfit: {
		name: "Outfit",
		ui: "'Outfit', ui-sans-serif, system-ui, sans-serif",
		mono: "'Space Mono', ui-monospace, Menlo, monospace",
	},
	ibm: {
		name: "IBM Plex",
		ui: "'IBM Plex Sans', ui-sans-serif, system-ui, sans-serif",
		mono: "'IBM Plex Mono', ui-monospace, Menlo, monospace",
	},
};

export const FONT_KEYS: FontKey[] = ["jakarta", "dm", "sora", "outfit", "ibm"];

// ── Persistence ────────────────────────────────────────────────────────────

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
