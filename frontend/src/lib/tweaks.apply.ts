import {
	CUSTOM_DEFAULTS,
	type DensityKey,
	type FontKey,
	markThemeApplied,
	type RadiusKey,
	type SidebarWidthKey,
	type SizeKey,
	THEME_VAR_NAMES,
	type ThemeKey,
	type Tweaks,
} from "./tweaks";
import { FONTS, THEMES } from "./tweaks.themes";

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

export function applyTheme(
	key: ThemeKey,
	customColors?: Record<string, string>,
): void {
	const root = document.documentElement;
	for (const v of THEME_VAR_NAMES) root.style.removeProperty(v);
	if (key === "custom") {
		const colors = customColors ?? CUSTOM_DEFAULTS;
		for (const [v, val] of Object.entries(colors)) {
			root.style.setProperty(v, val);
		}
	} else {
		for (const [v, val] of Object.entries(THEMES[key].vars)) {
			root.style.setProperty(v, val);
		}
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
	applyTheme(t.theme, t.customColors);
	applyFont(t.font);
	applyRadius(t.radius);
	applySidebarWidth(t.sidebarWidth);
	applySize(t.size);
	applyDensity(t.density);
}

export function applyTweaks(t: Tweaks): void {
	applyAll(t);
	markThemeApplied();
}
