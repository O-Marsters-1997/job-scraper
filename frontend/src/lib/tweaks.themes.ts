import type { FontKey, ThemeKey } from "./tweaks";

interface ThemeEntry {
	name: string;
	swatch: { sb: string; cv: string; ac: string; su: string; bd: string };
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
