export type DocFont = { css: string; ratio: number };

// ratio is (hhea ascender + descender + lineGap) / unitsPerEm of the
// metric-compatible font that stands in for each Docs font.
const CARLITO: DocFont = { css: "Carlito, Calibri, sans-serif", ratio: 1.2207 };
const ARIMO: DocFont = { css: "Arimo, Arial, sans-serif", ratio: 1.1499 };
const TINOS: DocFont = {
	css: "Tinos, 'Times New Roman', serif",
	ratio: 1.1499,
};
const CALADEA: DocFont = { css: "Caladea, Cambria, serif", ratio: 1.15 };

const KNOWN: Record<string, DocFont> = {
	calibri: CARLITO,
	carlito: CARLITO,
	arial: ARIMO,
	arimo: ARIMO,
	"times new roman": TINOS,
	tinos: TINOS,
	cambria: CALADEA,
	caladea: CALADEA,
};

const SERIF_HINT = /serif|times|georgia|garamond|palatino|book|roman/i;
const SANS_HINT = /sans/i;

export function resolveFont(name: string): DocFont {
	const known = KNOWN[name.trim().toLowerCase()];
	if (known) return known;
	return SERIF_HINT.test(name) && !SANS_HINT.test(name) ? TINOS : ARIMO;
}
