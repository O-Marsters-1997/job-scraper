import { parseColor } from "@kobalte/core/colors";

// ── sRGB ↔ oklch (no external deps) ─────────────────────────────────────────
// Standard OKLab/OKLCH matrices. Validated against the project's @theme tokens:
//   oklch(0.55 0.18 285)        → #6645D9  (violet primary)
//   oklch(0.23 0.055 285)       → #2A1F57  (indigo sidebar)
//   oklch(0.9842 0.0034 247.86) → #F8FAFC  (canvas)

function srgbToLinear(c: number): number {
	return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
}
function linearToSrgb(c: number): number {
	return c <= 0.0031308 ? 12.92 * c : 1.055 * c ** (1 / 2.4) - 0.055;
}
const clamp01 = (x: number) => Math.min(1, Math.max(0, x));

function rgbToOklch(r: number, g: number, b: number): [number, number, number] {
	const lr = srgbToLinear(r / 255);
	const lg = srgbToLinear(g / 255);
	const lb = srgbToLinear(b / 255);

	const l = 0.4122214708 * lr + 0.5363325363 * lg + 0.0514459929 * lb;
	const m = 0.2119034982 * lr + 0.6806995451 * lg + 0.1073969566 * lb;
	const s = 0.0883024619 * lr + 0.2817188376 * lg + 0.6299787005 * lb;

	const l_ = Math.cbrt(l);
	const m_ = Math.cbrt(m);
	const s_ = Math.cbrt(s);

	const L = 0.210454256 * l_ + 0.793617785 * m_ - 0.004072047 * s_;
	const a = 1.977998495 * l_ - 2.428592205 * m_ + 0.45059371 * s_;
	const bb = 0.025904037 * l_ + 0.782771766 * m_ - 0.808675766 * s_;

	const C = Math.sqrt(a * a + bb * bb);
	let H = (Math.atan2(bb, a) * 180) / Math.PI;
	if (H < 0) H += 360;
	return [L, C, H];
}

function oklchToRgb(L: number, C: number, H: number): [number, number, number] {
	const hr = (H * Math.PI) / 180;
	const a = C * Math.cos(hr);
	const b = C * Math.sin(hr);

	const l_ = L + 0.3963377774 * a + 0.2158037573 * b;
	const m_ = L - 0.1055613458 * a - 0.0638541728 * b;
	const s_ = L - 0.0894841775 * a - 1.291485548 * b;

	const lc = l_ ** 3;
	const mc = m_ ** 3;
	const sc = s_ ** 3;

	const lr = 4.0767416621 * lc - 3.3077115913 * mc + 0.2309699292 * sc;
	const lg = -1.2684380046 * lc + 2.6097574011 * mc - 0.3413193965 * sc;
	const lb = -0.0041960863 * lc - 0.7034186147 * mc + 1.707614701 * sc;

	return [
		Math.round(clamp01(linearToSrgb(lr)) * 255),
		Math.round(clamp01(linearToSrgb(lg)) * 255),
		Math.round(clamp01(linearToSrgb(lb)) * 255),
	];
}

// ── Parse oklch CSS string ───────────────────────────────────────────────────
// Accepts: oklch(L C H) or oklch(L C H / a)
// Returns null when not an oklch string (caller falls through).
function parseOklch(
	css: string,
): { L: number; C: number; H: number; alpha: number } | null {
	const m = css
		.trim()
		.match(
			/^oklch\(\s*([\d.]+)\s+([\d.]+)\s+([\d.]+)(?:\s*\/\s*([\d.]+))?\s*\)$/i,
		);
	if (!m) return null;
	return {
		L: Number(m[1]),
		C: Number(m[2]),
		H: Number(m[3]),
		alpha: m[4] !== undefined ? Number(m[4]) : 1,
	};
}

// ── Public API ────────────────────────────────────────────────────────────────

/**
 * Return the L (lightness) component of an oklch CSS string, or null if the
 * string isn't oklch. Used to decide the auth brand panel's accessible ink tone.
 */
export function oklchLightness(css: string): number | null {
	return parseOklch(css)?.L ?? null;
}

/**
 * Read a CSS custom property off :root and return it as a hex string.
 * Used by Chart.js utilities — Canvas can't consume CSS vars or oklch directly.
 * Reuses oklchToHex, so oklch values are converted and hex/rgb values pass through.
 */
export function cssVarHex(name: string, fallback = "#000000"): string {
	if (typeof document === "undefined") return fallback;
	const raw = getComputedStyle(document.documentElement)
		.getPropertyValue(name)
		.trim();
	return raw ? oklchToHex(raw) : fallback;
}

/**
 * Convert an oklch CSS string to a hex string suitable for Kobalte's ColorPicker.
 * Opaque → #rrggbb; alpha < 1 → #rrggbbaa (Kobalte parses 8-digit hex).
 * Non-oklch values (hex, rgb, rgba) are returned unchanged for safe migration.
 */
export function oklchToHex(css: string): string {
	const p = parseOklch(css);
	if (!p) return css; // already hex / rgb — pass through
	const [r, g, b] = oklchToRgb(p.L, p.C, p.H);
	const hex2 = (n: number) => n.toString(16).padStart(2, "0");
	const base = `#${hex2(r)}${hex2(g)}${hex2(b)}`;
	if (p.alpha < 1) {
		const a = Math.round(clamp01(p.alpha) * 255);
		return `${base}${hex2(a)}`;
	}
	return base;
}

/**
 * Convert a hex/rgba CSS string (as emitted by ColorPicker) to an oklch string.
 * Uses Kobalte's parseColor for reliable channel extraction.
 * Result: oklch(L C H) or oklch(L C H / a) when alpha < 1.
 */
export function toOklch(css: string): string {
	// If it's already oklch, nothing to do.
	if (parseOklch(css)) return css;

	let r: number, g: number, b: number, alpha: number;
	try {
		const c = parseColor(css).toFormat("rgba");
		r = c.getChannelValue("red");
		g = c.getChannelValue("green");
		b = c.getChannelValue("blue");
		alpha = c.getChannelValue("alpha");
	} catch {
		return css; // unparseable — leave as-is
	}

	const [L, C, H] = rgbToOklch(r, g, b);
	const Ls = L.toFixed(4);
	const Cs = C.toFixed(4);
	const Hs = H.toFixed(2);
	if (alpha < 1) {
		return `oklch(${Ls} ${Cs} ${Hs} / ${alpha.toFixed(3)})`;
	}
	return `oklch(${Ls} ${Cs} ${Hs})`;
}
