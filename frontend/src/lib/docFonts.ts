export type DocFont = { css: string; ratio: number; known: boolean };

const ARIMO = { css: "Arimo, Arial, sans-serif", ratio: 1.1499 };
const TINOS = { css: "Tinos, 'Times New Roman', serif", ratio: 1.1499 };

// ratio is (hhea ascender + descender + lineGap) / unitsPerEm of the
// metric-compatible font that stands in for each Docs font.
const KNOWN: Record<string, { css: string; ratio: number }> = {
	calibri: { css: "Carlito, Calibri, sans-serif", ratio: 1.2207 },
	carlito: { css: "Carlito, Calibri, sans-serif", ratio: 1.2207 },
	arial: ARIMO,
	arimo: ARIMO,
	"times new roman": TINOS,
	tinos: TINOS,
	cambria: { css: "Caladea, Cambria, serif", ratio: 1.15 },
	caladea: { css: "Caladea, Cambria, serif", ratio: 1.15 },
};

const SERIF_HINT = /serif|times|georgia|garamond|palatino|book|roman/i;
const SANS_HINT = /sans/i;

export function resolveFont(name: string): DocFont {
	const known = KNOWN[name.trim().toLowerCase()];
	if (known) return { ...known, known: true };
	const serif = SERIF_HINT.test(name) && !SANS_HINT.test(name);
	return { ...(serif ? TINOS : ARIMO), known: false };
}
