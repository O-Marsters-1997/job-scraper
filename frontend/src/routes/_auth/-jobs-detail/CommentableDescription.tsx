import { createEffect, createSignal, onCleanup, onMount, Show } from "solid-js";
import JobDescription from "@/components/jobs/JobDescription";
import { Button } from "@/components/ui/button";
import { useJobNotes } from "@/hooks/useJobNotes";
import { findQuote, quoteFromRange } from "@/lib/noteAnchor";
import {
	clearNoteHighlights,
	noteIdAtCaret,
	scrollToNote,
	setNoteHighlights,
} from "@/lib/noteHighlights";
import type { NoteQuote } from "@/types/jobNote";

export function CommentableDescription(props: {
	html: string;
	jobId: string;
	onComment: (quote: NoteQuote) => void;
}) {
	let root!: HTMLDivElement;
	const notes = useJobNotes(() => props.jobId);
	const [range, setRange] = createSignal<Range | null>(null);
	const [pos, setPos] = createSignal({ top: 0, left: 0 });

	const onSelectionChange = () => {
		const sel = document.getSelection();
		const r = sel && sel.rangeCount > 0 ? sel.getRangeAt(0) : null;
		if (
			!r ||
			r.collapsed ||
			r.toString().trim() === "" ||
			!root.contains(r.commonAncestorContainer)
		) {
			setRange(null);
			return;
		}
		const box = root.getBoundingClientRect();
		const end = r.getBoundingClientRect();
		setPos({ top: end.bottom - box.top + 4, left: end.right - box.left });
		setRange(r.cloneRange());
	};

	onMount(() =>
		document.addEventListener("selectionchange", onSelectionChange),
	);
	onCleanup(() =>
		document.removeEventListener("selectionchange", onSelectionChange),
	);

	createEffect(() => {
		void props.html;
		const found = new Map<string, Range>();
		for (const n of notes.data ?? []) {
			const r = n.Quote && findQuote(root, n.Quote);
			if (r) found.set(n.ID, r);
		}
		setNoteHighlights(found);
	});
	onCleanup(clearNoteHighlights);

	onMount(() => {
		const onClick = () => {
			const id = noteIdAtCaret();
			if (id) scrollToNote(id);
		};
		root.addEventListener("click", onClick);
		onCleanup(() => root.removeEventListener("click", onClick));
	});

	const comment = () => {
		const r = range();
		if (!r) return;
		props.onComment(quoteFromRange(root, r));
		document.getSelection()?.removeAllRanges();
		setRange(null);
	};

	return (
		<div ref={root} class="relative">
			<JobDescription html={props.html} />
			<Show when={range()}>
				<Button
					type="button"
					size="sm"
					class="absolute z-10 -translate-x-full shadow-md"
					style={{ top: `${pos().top}px`, left: `${pos().left}px` }}
					onMouseDown={(e) => e.preventDefault()}
					onClick={comment}
				>
					Comment
				</Button>
			</Show>
		</div>
	);
}
