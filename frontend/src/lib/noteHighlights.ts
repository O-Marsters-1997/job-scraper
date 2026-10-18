import { createSignal } from "solid-js";

const NAME = "job-note";
const ranges = new Map<string, Range>();
const [highlightedIds, setHighlightedIds] = createSignal<Set<string> | null>(
	null,
);

export function setNoteHighlights(next: Map<string, Range>) {
	ranges.clear();
	for (const [id, r] of next) ranges.set(id, r);
	setHighlightedIds(new Set(next.keys()));
	if ("highlights" in CSS) {
		CSS.highlights.set(NAME, new Highlight(...next.values()));
	}
}

export function clearNoteHighlights() {
	ranges.clear();
	setHighlightedIds(null);
	if ("highlights" in CSS) CSS.highlights.delete(NAME);
}

export function isOrphanedNote(id: string): boolean {
	const ids = highlightedIds();
	return ids !== null && !ids.has(id);
}

export function noteIdAtCaret(): string | null {
	const sel = document.getSelection();
	if (!sel || !sel.isCollapsed || !sel.anchorNode) return null;
	for (const [id, r] of ranges) {
		if (r.isPointInRange(sel.anchorNode, sel.anchorOffset)) return id;
	}
	return null;
}

export function scrollToHighlight(id: string) {
	ranges.get(id)?.startContainer.parentElement?.scrollIntoView({
		block: "center",
		behavior: "smooth",
	});
}

export function scrollToNote(id: string) {
	document
		.getElementById(`job-note-${id}`)
		?.scrollIntoView({ block: "nearest", behavior: "smooth" });
}
