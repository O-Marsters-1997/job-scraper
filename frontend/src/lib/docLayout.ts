import type { LayoutBlock, LayoutBorder, LayoutRun } from "@/types/tailoring";

export const SPARSE_LAST_LINE = 0.3;

export type LineFit = { lines: number; lastLineFill: number };

export const isSparse = (fit: LineFit | undefined) =>
	!!fit && fit.lines > 1 && fit.lastLineFill < SPARSE_LAST_LINE;

const OVERFLOW_TOLERANCE_PT = 0.5;
const FEWER_LINES_SLACK = 0.97;
const FULL_LINE_SPACING = 100;

export const linePt = (size: number, ratio: number, spacing: number) =>
	size *
	ratio *
	((spacing > 0 ? spacing : FULL_LINE_SPACING) / FULL_LINE_SPACING);

const sameBorder = (a: LayoutBorder | null, b: LayoutBorder | null) =>
	a !== null &&
	b !== null &&
	a.width === b.width &&
	a.color === b.color &&
	a.padding === b.padding &&
	a.dash === b.dash;

export type BorderDraw = { top: boolean; bottom: boolean };

export function borderDraws(blocks: LayoutBlock[]): BorderDraw[] {
	return blocks.map((b, i) => ({
		top: !sameBorder(blocks[i - 1]?.borderTop ?? null, b.borderTop),
		bottom: !sameBorder(blocks[i + 1]?.borderBottom ?? null, b.borderBottom),
	}));
}

export const hasEndTab = (b: LayoutBlock) =>
	b.tabStops.some((t) => t.alignment === "end");

export function splitAtTab(runs: LayoutRun[]): [LayoutRun[], LayoutRun[]] {
	const left: LayoutRun[] = [];
	const right: LayoutRun[] = [];
	let afterTab = false;
	for (const r of runs) {
		if (afterTab) {
			right.push(r);
			continue;
		}
		const [pre = "", ...rest] = r.text.split("\t");
		if (pre) left.push({ ...r, text: pre });
		if (rest.length) {
			afterTab = true;
			const post = rest.join("\t").trimStart();
			if (post) right.push({ ...r, text: post });
		}
	}
	return [left, right];
}

export const blockText = (b: LayoutBlock) =>
	trimParagraphEnd(b.runs)
		.map((r) => r.text)
		.join("");

export type PageFit = { over: boolean; lines: number };

export function pageFit(
	contentPt: number,
	availablePt: number,
	bodyLinePt: number,
): PageFit {
	const over = contentPt - availablePt;
	return over > OVERFLOW_TOLERANCE_PT
		? { over: true, lines: Math.ceil(over / bodyLinePt) }
		: { over: false, lines: Math.max(0, Math.floor(-over / bodyLinePt)) };
}

export const isSpacer = (b: LayoutBlock) =>
	b.runs.every((r) => r.text.replace(/\n$/, "") === "");

export function trimParagraphEnd(runs: LayoutRun[]): LayoutRun[] {
	const last = runs.at(-1);
	if (!last?.text.endsWith("\n")) return runs;
	return [...runs.slice(0, -1), { ...last, text: last.text.slice(0, -1) }];
}

export function charsToSave(
	text: string,
	lines: number,
	lastLineFill: number,
): number {
	if (lines < 2 || lastLineFill <= 0 || lastLineFill >= SPARSE_LAST_LINE)
		return 0;
	const perLine = text.length / (lines - 1 + lastLineFill);
	return Math.ceil(lastLineFill * perLine);
}

export function maxCharsForFewerLines(
	text: string,
	lines: number,
	lastLineFill: number,
): number {
	if (lines < 2) return text.length;
	const perLine = text.length / (lines - 1 + lastLineFill);
	return Math.floor(perLine * (lines - 1) * FEWER_LINES_SLACK);
}
