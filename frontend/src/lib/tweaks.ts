export type ThemeKey = "teal" | "midnight" | "ember" | "forest" | "plum" | "graphite";
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
}

export const DEFAULTS: Tweaks = {
	theme: "teal",
	font: "jakarta",
	size: "sm",
	sidebarWidth: "default",
	density: "default",
	radius: "default",
};

export const STORAGE_KEY = "job-scraper-tweaks";

// CSS variable names that theme switching can override
const THEME_VAR_NAMES = [
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

interface ThemeEntry {
	name: string;
	// Preview swatch colours (inline styles only, independent of applied theme)
	swatch: { sb: string; cv: string; ac: string };
	// CSS variable overrides — empty means "use pristine @theme values"
	vars: Record<string, string>;
}

export const THEMES: Record<ThemeKey, ThemeEntry> = {
	teal: {
		name: "Teal",
		swatch: { sb: "#111921", cv: "#f8fafc", ac: "#0f9d92" },
		vars: {},
	},
	midnight: {
		name: "Midnight",
		swatch: { sb: "#0e1117", cv: "#f8f9ff", ac: "#2563eb" },
		vars: {
			"--color-sidebar": "#0e1117",
			"--color-sidebar-hover": "#171e2e",
			"--color-sidebar-active": "rgba(59,130,246,.13)",
			"--color-sidebar-active-foreground": "#93c5fd",
			"--color-sidebar-foreground": "#7a8fab",
			"--color-sidebar-foreground-strong": "#e0e8f5",
			"--color-sidebar-border": "#171e2e",
			"--color-background": "#f8f9ff",
			"--color-surface": "#ffffff",
			"--color-surface-muted": "#f8f9ff",
			"--color-foreground": "#0f1628",
			"--color-muted": "#3d4e6b",
			"--color-faint": "#8fa3c1",
			"--color-border": "#e2e6f0",
			"--color-border-strong": "#c8d0e4",
			"--color-primary": "#2563eb",
			"--color-primary-hover": "#1d4ed8",
			"--color-accent-subtle": "#eff6ff",
			"--color-accent-border": "#bfdbfe",
			"--color-accent-text": "#1e3a5f",
		},
	},
	ember: {
		name: "Ember",
		swatch: { sb: "#18110a", cv: "#fdfaf6", ac: "#d97706" },
		vars: {
			"--color-sidebar": "#18110a",
			"--color-sidebar-hover": "#251a0e",
			"--color-sidebar-active": "rgba(245,158,11,.12)",
			"--color-sidebar-active-foreground": "#fcd34d",
			"--color-sidebar-foreground": "#9a836a",
			"--color-sidebar-foreground-strong": "#f0e0cc",
			"--color-sidebar-border": "#251a0e",
			"--color-background": "#fdfaf6",
			"--color-surface": "#ffffff",
			"--color-surface-muted": "#fdfaf6",
			"--color-foreground": "#1c1208",
			"--color-muted": "#6b5240",
			"--color-faint": "#a8917c",
			"--color-border": "#ede8dc",
			"--color-border-strong": "#ddd0bb",
			"--color-primary": "#d97706",
			"--color-primary-hover": "#b45309",
			"--color-accent-subtle": "#fffbeb",
			"--color-accent-border": "#fde68a",
			"--color-accent-text": "#78350f",
		},
	},
	forest: {
		name: "Forest",
		swatch: { sb: "#0d1a0f", cv: "#f6fdf8", ac: "#059669" },
		vars: {
			"--color-sidebar": "#0d1a0f",
			"--color-sidebar-hover": "#142218",
			"--color-sidebar-active": "rgba(16,185,129,.12)",
			"--color-sidebar-active-foreground": "#6ee7b7",
			"--color-sidebar-foreground": "#728f77",
			"--color-sidebar-foreground-strong": "#d0f0d8",
			"--color-sidebar-border": "#142218",
			"--color-background": "#f6fdf8",
			"--color-surface": "#ffffff",
			"--color-surface-muted": "#f6fdf8",
			"--color-foreground": "#061a09",
			"--color-muted": "#2d5b35",
			"--color-faint": "#80a688",
			"--color-border": "#d8f0dc",
			"--color-border-strong": "#b8e0c0",
			"--color-primary": "#059669",
			"--color-primary-hover": "#047857",
			"--color-accent-subtle": "#ecfdf5",
			"--color-accent-border": "#6ee7b7",
			"--color-accent-text": "#064e3b",
		},
	},
	plum: {
		name: "Plum",
		swatch: { sb: "#120d1e", cv: "#faf8ff", ac: "#7c3aed" },
		vars: {
			"--color-sidebar": "#120d1e",
			"--color-sidebar-hover": "#1e1530",
			"--color-sidebar-active": "rgba(139,92,246,.13)",
			"--color-sidebar-active-foreground": "#c4b5fd",
			"--color-sidebar-foreground": "#8878a8",
			"--color-sidebar-foreground-strong": "#e8e0f8",
			"--color-sidebar-border": "#1e1530",
			"--color-background": "#faf8ff",
			"--color-surface": "#ffffff",
			"--color-surface-muted": "#faf8ff",
			"--color-foreground": "#14082a",
			"--color-muted": "#483870",
			"--color-faint": "#9585b8",
			"--color-border": "#ede8f8",
			"--color-border-strong": "#d8d0f0",
			"--color-primary": "#7c3aed",
			"--color-primary-hover": "#6d28d9",
			"--color-accent-subtle": "#f5f3ff",
			"--color-accent-border": "#ddd6fe",
			"--color-accent-text": "#4c1d95",
		},
	},
	graphite: {
		name: "Graphite",
		swatch: { sb: "#111111", cv: "#f7f7f7", ac: "#404040" },
		vars: {
			"--color-sidebar": "#111111",
			"--color-sidebar-hover": "#1c1c1c",
			"--color-sidebar-active": "rgba(255,255,255,.08)",
			"--color-sidebar-active-foreground": "#e8e8e8",
			"--color-sidebar-foreground": "#888888",
			"--color-sidebar-foreground-strong": "#e0e0e0",
			"--color-sidebar-border": "#1c1c1c",
			"--color-background": "#f7f7f7",
			"--color-surface": "#ffffff",
			"--color-surface-muted": "#f7f7f7",
			"--color-foreground": "#111111",
			"--color-muted": "#555555",
			"--color-faint": "#999999",
			"--color-border": "#e4e4e4",
			"--color-border-strong": "#cccccc",
			"--color-primary": "#404040",
			"--color-primary-hover": "#2a2a2a",
			"--color-accent-subtle": "#f0f0f0",
			"--color-accent-border": "#cccccc",
			"--color-accent-text": "#111111",
		},
	},
};

export const THEME_KEYS: ThemeKey[] = [
	"teal",
	"midnight",
	"ember",
	"forest",
	"plum",
	"graphite",
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

const FONT_VAR_NAMES = ["--font-sans", "--font-mono"];

const RADIUS_OVERRIDES: Record<RadiusKey, Record<string, string>> = {
	sharp: {
		"--radius-sm": "0.0625rem",
		"--radius-md": "0.125rem",
		"--radius-lg": "0.1875rem",
		"--radius-xl": "0.25rem",
		"--radius-2xl": "0.375rem",
		"--radius-3xl": "0.5rem",
	},
	default: {},
	round: {
		"--radius-sm": "0.375rem",
		"--radius-md": "0.625rem",
		"--radius-lg": "0.875rem",
		"--radius-xl": "1.25rem",
		"--radius-2xl": "1.5rem",
		"--radius-3xl": "2rem",
	},
};

const ALL_RADIUS_VARS = new Set([
	...Object.keys(RADIUS_OVERRIDES.sharp),
	...Object.keys(RADIUS_OVERRIDES.round),
]);

const SIDEBAR_W: Record<SidebarWidthKey, string | null> = {
	narrow: "12.5rem",
	default: null,
	wide: "16.25rem",
};

// ── Persistence ────────────────────────────────────────────────────────────

export function loadTweaks(): Tweaks {
	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) return { ...DEFAULTS, ...JSON.parse(stored) };
	} catch { }
	return { ...DEFAULTS };
}

export function saveTweaks(t: Tweaks): void {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(t));
}

// ── Apply functions ─────────────────────────────────────────────────────────

export function applyTheme(key: ThemeKey): void {
	const root = document.documentElement;
	const theme = THEMES[key];
	// Remove all theme var overrides first (restores @theme values for teal)
	for (const v of THEME_VAR_NAMES) root.style.removeProperty(v);
	// Apply overrides for non-default themes
	for (const [v, val] of Object.entries(theme.vars)) {
		root.style.setProperty(v, val);
	}
}

export function applyFont(key: FontKey): void {
	const root = document.documentElement;
	if (key === "jakarta") {
		for (const v of FONT_VAR_NAMES) root.style.removeProperty(v);
	} else {
		root.style.setProperty("--font-sans", FONTS[key].ui);
		root.style.setProperty("--font-mono", FONTS[key].mono);
	}
}

export function applyRadius(key: RadiusKey): void {
	const root = document.documentElement;
	for (const v of ALL_RADIUS_VARS) root.style.removeProperty(v);
	for (const [v, val] of Object.entries(RADIUS_OVERRIDES[key])) {
		root.style.setProperty(v, val);
	}
}

export function applySidebarWidth(key: SidebarWidthKey): void {
	const w = SIDEBAR_W[key];
	if (w) {
		document.documentElement.style.setProperty("--sidebar-w", w);
	} else {
		document.documentElement.style.removeProperty("--sidebar-w");
	}
}

export function applySize(key: SizeKey): void {
	if (key === "sm") {
		document.documentElement.removeAttribute("data-size");
	} else {
		document.documentElement.setAttribute("data-size", key);
	}
}

export function applyDensity(key: DensityKey): void {
	if (key === "default") {
		document.documentElement.removeAttribute("data-density");
	} else {
		document.documentElement.setAttribute("data-density", key);
	}
}

export function applyAll(t: Tweaks): void {
	applyTheme(t.theme);
	applyFont(t.font);
	applyRadius(t.radius);
	applySidebarWidth(t.sidebarWidth);
	applySize(t.size);
	applyDensity(t.density);
}
