import {
	createEffect,
	createSignal,
	For,
	onCleanup,
	onMount,
	Show,
	untrack,
} from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import JobDescription from "@/components/jobs/JobDescription";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useCreateJobComment, useJobComments } from "@/hooks/useJobComments";
import { findQuote, quoteFromRange } from "@/lib/noteAnchor";
import {
	clearCommentHighlights,
	commentIdAtPoint,
	commentRange,
	setActiveComment,
	setCommentHighlights,
} from "@/lib/noteHighlights";
import { loadedData } from "@/lib/queryData";
import {
	type CommentQuote,
	JOB_COMMENT_MAX_LENGTH,
	type JobComment,
} from "@/types/jobComment";
import { CommentItem } from "./CommentItem";

type Anchor = { top: number; left: number };
type Draft = { quote: CommentQuote; at: Anchor };

const CLOSE_DELAY_MS = 250;
const PANEL =
	"absolute z-10 flex w-72 max-w-full flex-col gap-2 rounded-md border border-border bg-surface p-3 shadow-md";

export function CommentableDescription(props: { html: string; jobId: string }) {
	let root!: HTMLDivElement;
	const comments = useJobComments(() => props.jobId);
	const create = useCreateJobComment(() => props.jobId);
	const [selection, setSelection] = createSignal<{
		range: Range;
		at: Anchor;
	} | null>(null);
	const [draft, setDraft] = createSignal<Draft | null>(null);
	const [text, setText] = createSignal("");
	const [openId, setOpenId] = createSignal<string | null>(null);
	const [openAt, setOpenAt] = createSignal<Anchor>({ top: 0, left: 0 });
	const [orphans, setOrphans] = createSignal<JobComment[]>([]);
	let closeTimer: ReturnType<typeof setTimeout> | undefined;

	const anchorBelow = (r: Range): Anchor => {
		const box = root.getBoundingClientRect();
		const rect = r.getBoundingClientRect();
		return {
			top: rect.bottom - box.top + 4,
			left: Math.max(0, Math.min(rect.left - box.left, box.width - 288)),
		};
	};

	const onSelectionChange = () => {
		if (draft()) return;
		const sel = document.getSelection();
		const r = sel && sel.rangeCount > 0 ? sel.getRangeAt(0) : null;
		if (
			!r ||
			r.collapsed ||
			r.toString().trim() === "" ||
			!root.contains(r.commonAncestorContainer)
		) {
			setSelection(null);
			return;
		}
		const box = root.getBoundingClientRect();
		const end = r.getBoundingClientRect();
		setSelection({
			range: r.cloneRange(),
			at: { top: end.bottom - box.top + 4, left: end.right - box.left },
		});
	};

	const onKeyDown = (e: KeyboardEvent) => {
		if (e.key !== "Escape") return;
		setDraft(null);
		setOpenId(null);
	};

	onMount(() => {
		document.addEventListener("keydown", onKeyDown);
		document.addEventListener("selectionchange", onSelectionChange);
		root.addEventListener("mousemove", onPointerMove);
		root.addEventListener("click", onClick);
	});
	onCleanup(() => {
		document.removeEventListener("keydown", onKeyDown);
		document.removeEventListener("selectionchange", onSelectionChange);
		clearTimeout(closeTimer);
		clearCommentHighlights();
	});

	createEffect(() => {
		void props.html;
		const list = loadedData(comments) ?? [];
		const found = new Map<string, Range>();
		const lost: JobComment[] = [];
		for (const c of list) {
			const r = findQuote(root, c.Quote);
			if (r) found.set(c.ID, r);
			else lost.push(c);
		}
		setCommentHighlights(found);
		setOrphans(lost);
		const id = untrack(openId);
		if (id && !found.has(id)) setOpenId(null);
	});

	createEffect(() => setActiveComment(openId()));

	const open = (id: string) => {
		clearTimeout(closeTimer);
		const r = commentRange(id);
		if (!r || openId() === id) return;
		setOpenAt(anchorBelow(r));
		setOpenId(id);
	};
	const scheduleClose = () => {
		clearTimeout(closeTimer);
		closeTimer = setTimeout(() => {
			const active = document.activeElement;
			const editing = active?.closest("[data-comment-panel]");
			if (!editing) setOpenId(null);
		}, CLOSE_DELAY_MS);
	};

	const idUnderPointer = (e: MouseEvent): string | null | undefined => {
		const inPanel =
			e.target instanceof Element && e.target.closest("[data-comment-panel]");
		if (draft() || inPanel) return undefined;
		return commentIdAtPoint(e.clientX, e.clientY);
	};
	const onPointerMove = (e: MouseEvent) => {
		const id = idUnderPointer(e);
		if (id) open(id);
		else if (id === null && openId()) scheduleClose();
	};
	const onClick = (e: MouseEvent) => {
		if (!document.getSelection()?.isCollapsed) return;
		const id = idUnderPointer(e);
		if (id) open(id);
	};

	const startComment = () => {
		const s = selection();
		if (!s) return;
		const left = Math.max(0, Math.min(s.at.left - 288, root.clientWidth - 288));
		setDraft({
			quote: quoteFromRange(root, s.range),
			at: { top: s.at.top, left },
		});
		setText("");
		setOpenId(null);
		setSelection(null);
		document.getSelection()?.removeAllRanges();
	};
	const save = (e: SubmitEvent) => {
		e.preventDefault();
		const d = draft();
		if (!d) return;
		create.mutate(
			{ body: text(), quote: d.quote },
			{ onSuccess: () => setDraft(null) },
		);
	};

	const openComment = () =>
		(loadedData(comments) ?? []).find((c) => c.ID === openId());

	return (
		<div class="flex flex-col gap-4">
			<div ref={root} class="relative">
				<JobDescription html={props.html} />
				<Show when={!draft() && selection()}>
					{(s) => (
						<Button
							type="button"
							size="icon-sm"
							class="absolute z-10 -translate-x-full shadow-md"
							style={{ top: `${s().at.top}px`, left: `${s().at.left}px` }}
							aria-label="Add comment"
							onMouseDown={(e) => e.preventDefault()}
							onClick={startComment}
						>
							<Icon name="messageSquare" />
						</Button>
					)}
				</Show>
				<Show when={draft()}>
					{(d) => (
						<form
							data-comment-panel
							class={PANEL}
							style={{ top: `${d().at.top}px`, left: `${d().at.left}px` }}
							onSubmit={save}
						>
							<FormFeedback
								error={create.isError ? "Could not save the comment." : null}
							/>
							<Label for="job-comment-body">Add comment</Label>
							<Textarea
								id="job-comment-body"
								ref={(el: HTMLTextAreaElement) =>
									queueMicrotask(() => el.focus())
								}
								rows={3}
								maxLength={JOB_COMMENT_MAX_LENGTH}
								value={text()}
								onInput={(e) => setText(e.currentTarget.value)}
							/>
							<div class="flex gap-2">
								<Button
									type="submit"
									size="sm"
									disabled={create.isPending || text().trim().length === 0}
								>
									Save
								</Button>
								<Button
									type="button"
									size="sm"
									variant="ghost"
									onClick={() => setDraft(null)}
								>
									Cancel
								</Button>
							</div>
						</form>
					)}
				</Show>
				<Show when={openComment()}>
					{(c) => (
						<div
							data-comment-panel
							role="dialog"
							aria-label="Comment"
							class={PANEL}
							style={{ top: `${openAt().top}px`, left: `${openAt().left}px` }}
							onMouseEnter={() => clearTimeout(closeTimer)}
							onMouseLeave={scheduleClose}
						>
							<CommentItem
								comment={c()}
								jobId={props.jobId}
								onRemoved={() => setOpenId(null)}
							/>
						</div>
					)}
				</Show>
			</div>
			<Show when={orphans().length > 0}>
				<section class="flex flex-col gap-3 border-t border-border pt-3">
					<h3 class="text-sm font-semibold text-foreground">
						Comments on text no longer in the description
					</h3>
					<For each={orphans()}>
						{(c) => (
							<div class="flex flex-col gap-1">
								<blockquote class="line-clamp-4 whitespace-pre-wrap break-words border-l-2 border-border pl-3 text-sm text-muted">
									{c.Quote.exact}
								</blockquote>
								<CommentItem comment={c} jobId={props.jobId} />
							</div>
						)}
					</For>
				</section>
			</Show>
		</div>
	);
}
