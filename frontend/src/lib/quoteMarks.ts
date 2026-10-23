export type QuoteSpan = { start: number; end: number; key: string };

const escapeRegExp = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

export function findQuoteSpans(
	text: string,
	quotes: { key: string; quote: string }[],
): QuoteSpan[] {
	const spans: QuoteSpan[] = [];
	for (const { key, quote } of quotes) {
		const words = quote.split(/\s+/).filter(Boolean);
		if (words.length === 0) continue;
		const hit = new RegExp(words.map(escapeRegExp).join("\\s+")).exec(text);
		if (hit)
			spans.push({ start: hit.index, end: hit.index + hit[0].length, key });
	}
	spans.sort((a, b) => a.start - b.start);
	const kept: QuoteSpan[] = [];
	for (const s of spans) {
		const last = kept.at(-1);
		if (!last || s.start >= last.end) kept.push(s);
	}
	return kept;
}
