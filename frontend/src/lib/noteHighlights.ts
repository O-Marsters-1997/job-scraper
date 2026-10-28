const ALL = "job-comment";
const ACTIVE = "job-comment-active";
const ranges = new Map<string, Range>();

function paint(name: string, list: Range[]) {
	if (!("highlights" in CSS)) return;
	if (list.length === 0) CSS.highlights.delete(name);
	else CSS.highlights.set(name, new Highlight(...list));
}

export function setCommentHighlights(next: Map<string, Range>) {
	ranges.clear();
	for (const [id, r] of next) ranges.set(id, r);
	paint(ALL, [...next.values()]);
}

export function setActiveComment(id: string | null) {
	const r = id ? ranges.get(id) : undefined;
	paint(ACTIVE, r ? [r] : []);
}

export function clearCommentHighlights() {
	ranges.clear();
	paint(ALL, []);
	paint(ACTIVE, []);
}

export function commentRange(id: string): Range | undefined {
	return ranges.get(id);
}

export function commentIdAtPoint(x: number, y: number): string | null {
	const pos = document.caretPositionFromPoint?.(x, y);
	const fallback = pos ? null : document.caretRangeFromPoint?.(x, y);
	const node = pos?.offsetNode ?? fallback?.startContainer;
	const offset = pos?.offset ?? fallback?.startOffset ?? 0;
	if (!node) return null;
	for (const [id, r] of ranges) {
		if (r.isPointInRange(node, offset)) return id;
	}
	return null;
}
