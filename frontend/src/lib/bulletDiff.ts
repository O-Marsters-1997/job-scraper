export type BulletRow =
	| { kind: "same" | "moved" | "added" | "removed"; text: string }
	| { kind: "rewritten"; text: string; from: string };

const REWRITE_SIMILARITY = 0.3;

const words = (s: string) => new Set(s.toLowerCase().match(/[\p{L}\p{N}]+/gu));

function similarity(a: string, b: string): number {
	const x = words(a);
	const y = words(b);
	let shared = 0;
	for (const w of x) if (y.has(w)) shared++;
	const union = x.size + y.size - shared;
	return union === 0 ? 0 : shared / union;
}

function stableMatches(baseIndexOf: number[]): Set<number> {
	const best: number[][] = [];
	baseIndexOf.forEach((b, i) => {
		let seq: number[] = [];
		for (let j = 0; j < i; j++) {
			if (baseIndexOf[j]! < b && best[j]!.length > seq.length) seq = best[j]!;
		}
		best.push([...seq, i]);
	});
	const longest = best.reduce((m, s) => (s.length > m.length ? s : m), []);
	return new Set(longest);
}

export function bulletDiff(base: string[], current: string[]): BulletRow[] {
	const baseLeft = new Set(base.keys());
	const exact = new Map<number, number>();
	current.forEach((text, i) => {
		const j = base.findIndex((b, k) => baseLeft.has(k) && b === text);
		if (j >= 0) {
			baseLeft.delete(j);
			exact.set(i, j);
		}
	});

	const matched = [...exact.keys()].sort((a, b) => a - b);
	const stable = stableMatches(matched.map((i) => exact.get(i)!));
	const stableCurrent = new Set(matched.filter((_, n) => stable.has(n)));

	const rows: BulletRow[] = current.map((text, i) => {
		if (exact.has(i)) {
			return { kind: stableCurrent.has(i) ? "same" : "moved", text };
		}
		let bestJ = -1;
		let bestScore = REWRITE_SIMILARITY;
		for (const j of baseLeft) {
			const score = similarity(base[j]!, text);
			if (score >= bestScore) {
				bestJ = j;
				bestScore = score;
			}
		}
		if (bestJ < 0) return { kind: "added", text };
		baseLeft.delete(bestJ);
		return { kind: "rewritten", text, from: base[bestJ]! };
	});
	for (const j of baseLeft) rows.push({ kind: "removed", text: base[j]! });
	return rows;
}
