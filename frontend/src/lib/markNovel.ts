import type { Segment } from "@/types/tailoring";

const WORD = /[\p{L}\p{N}]+/gu;

const STOPWORDS = new Set([
	"and",
	"the",
	"for",
	"with",
	"from",
	"that",
	"this",
	"into",
	"over",
	"across",
	"using",
	"through",
]);

export function markNovel(text: string, sources: string[]): Segment[] {
	const known = new Set(
		sources.flatMap((s) => s.toLowerCase().match(WORD) ?? []),
	);
	const out: Segment[] = [];
	const emit = (t: string, novel: boolean) => {
		if (t === "") return;
		const last = out.at(-1);
		if (last?.novel === novel) last.text += t;
		else out.push({ text: t, novel });
	};
	let end = 0;
	for (const m of text.matchAll(WORD)) {
		emit(text.slice(end, m.index), false);
		const word = m[0];
		const lower = word.toLowerCase();
		emit(
			word,
			[...word].length > 2 && !STOPWORDS.has(lower) && !known.has(lower),
		);
		end = m.index + word.length;
	}
	emit(text.slice(end), false);
	return out;
}
