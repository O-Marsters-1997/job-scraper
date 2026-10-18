import type { NoteQuote } from "../types/jobNote";

const CONTEXT_CHARS = 30;
const MAX_EXACT = 2000;
const SHOW_TEXT = 4;

function docOf(root: Node): Document {
	return root.ownerDocument ?? (root as Document);
}

function textNodes(root: Node): Text[] {
	const walker = docOf(root).createTreeWalker(root, SHOW_TEXT);
	const nodes: Text[] = [];
	for (let n = walker.nextNode(); n; n = walker.nextNode()) {
		nodes.push(n as Text);
	}
	return nodes;
}

function textOf(nodes: Text[]): string {
	return nodes.map((n) => n.data).join("");
}

function textOffsetOf(root: Node, container: Node, offset: number): number {
	const head = docOf(root).createRange();
	head.selectNodeContents(root);
	head.setEnd(container, offset);
	return head.toString().length;
}

function pointAt(nodes: Text[], target: number, isEnd: boolean) {
	let seen = 0;
	for (const node of nodes) {
		const end = seen + node.data.length;
		if (isEnd ? target <= end : target < end) {
			return { node, offset: target - seen };
		}
		seen = end;
	}
	const last = nodes[nodes.length - 1] as Text;
	return { node: last, offset: last.data.length };
}

function commonSuffix(a: string, b: string): number {
	let n = 0;
	while (
		n < a.length &&
		n < b.length &&
		a[a.length - 1 - n] === b[b.length - 1 - n]
	)
		n++;
	return n;
}

function commonPrefix(a: string, b: string): number {
	let n = 0;
	while (n < a.length && n < b.length && a[n] === b[n]) n++;
	return n;
}

export function quoteFromRange(root: Node, range: Range): NoteQuote {
	const text = textOf(textNodes(root));
	const start = textOffsetOf(root, range.startContainer, range.startOffset);
	const end = Math.min(
		textOffsetOf(root, range.endContainer, range.endOffset),
		start + MAX_EXACT,
	);
	return {
		exact: text.slice(start, end),
		prefix: text.slice(Math.max(0, start - CONTEXT_CHARS), start),
		suffix: text.slice(end, end + CONTEXT_CHARS),
	};
}

export function findQuote(root: Node, quote: NoteQuote): Range | null {
	const nodes = textNodes(root);
	const text = textOf(nodes);
	if (!quote.exact) return null;

	let best = -1;
	let bestScore = -1;
	for (
		let i = text.indexOf(quote.exact);
		i !== -1;
		i = text.indexOf(quote.exact, i + 1)
	) {
		const score =
			commonSuffix(
				text.slice(Math.max(0, i - quote.prefix.length), i),
				quote.prefix,
			) + commonPrefix(text.slice(i + quote.exact.length), quote.suffix);
		if (score > bestScore) {
			best = i;
			bestScore = score;
		}
	}
	if (best === -1) return null;

	const range = docOf(root).createRange();
	const from = pointAt(nodes, best, false);
	const to = pointAt(nodes, best + quote.exact.length, true);
	range.setStart(from.node, from.offset);
	range.setEnd(to.node, to.offset);
	return range;
}
