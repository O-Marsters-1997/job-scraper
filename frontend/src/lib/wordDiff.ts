export type WordOp = { op: "same" | "add" | "del"; text: string };

const tokens = (s: string) => s.match(/\s+|\S+/g) ?? [];

export function wordDiff(a: string, b: string): WordOp[] {
	const x = tokens(a);
	const y = tokens(b);
	const lcs = Array.from({ length: x.length + 1 }, () =>
		new Array<number>(y.length + 1).fill(0),
	);
	for (let i = x.length - 1; i >= 0; i--) {
		for (let j = y.length - 1; j >= 0; j--) {
			lcs[i]![j] =
				x[i] === y[j]
					? lcs[i + 1]![j + 1]! + 1
					: Math.max(lcs[i + 1]![j]!, lcs[i]![j + 1]!);
		}
	}

	const out: WordOp[] = [];
	const push = (op: WordOp["op"], text: string) => {
		const last = out[out.length - 1];
		if (last?.op === op) last.text += text;
		else out.push({ op, text });
	};
	let i = 0;
	let j = 0;
	while (i < x.length && j < y.length) {
		if (x[i] === y[j]) {
			push("same", x[i]!);
			i++;
			j++;
		} else if (lcs[i + 1]![j]! >= lcs[i]![j + 1]!) {
			push("del", x[i++]!);
		} else {
			push("add", y[j++]!);
		}
	}
	while (i < x.length) push("del", x[i++]!);
	while (j < y.length) push("add", y[j++]!);
	return out;
}

export function listDiff(a: string[], b: string[]): WordOp[] {
	return [
		...a
			.filter((s) => !b.includes(s))
			.map((text): WordOp => ({ op: "del", text })),
		...b.map(
			(text): WordOp => ({ op: a.includes(text) ? "same" : "add", text }),
		),
	];
}
