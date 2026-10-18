import type { NoteQuote } from "../types/jobNote";

const CONTEXT = 30;
const MAX_EXACT = 2000;

function textNodes(root: Node): Text[] {
	const doc = root.ownerDocument ?? (root as Document);
	const walker = doc.createTreeWalker(root, NodeFilter.SHOW_TEXT);
	const nodes: Text[] = [];
	for (let n = walker.nextNode(); n; n = walker.nextNode()) {
		nodes.push(n as Text);
	}
	return nodes;
}

function offsetOf(root: Node, container: Node, offset: number): number {
	const doc = root.ownerDocument ?? (root as Document);
	const head = doc.createRange();
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
	const text = textNodes(root)
		.map((n) => n.data)
		.join("");
	const start = offsetOf(root, range.startContainer, range.startOffset);
	const end = offsetOf(root, range.endContainer, range.endOffset);
	return {
		exact: text.slice(start, Math.min(end, start + MAX_EXACT)),
		prefix: text.slice(Math.max(0, start - CONTEXT), start),
		suffix: text.slice(end, end + CONTEXT),
	};
}

export function findQuote(root: Node, quote: NoteQuote): Range | null {
	const nodes = textNodes(root);
	const text = nodes.map((n) => n.data).join("");
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

	const doc = root.ownerDocument ?? (root as Document);
	const range = doc.createRange();
	const from = pointAt(nodes, best, false);
	const to = pointAt(nodes, best + quote.exact.length, true);
	range.setStart(from.node, from.offset);
	range.setEnd(to.node, to.offset);
	return range;
}
