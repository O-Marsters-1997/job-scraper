import { Link } from "@tanstack/solid-router";
import {
	createEffect,
	createSignal,
	type JSX,
	onCleanup,
	onMount,
	Show,
} from "solid-js";
import { googleWriteHref } from "@/components/GoogleWriteConsent";
import { Icon } from "@/components/Icon";
import { PdfPreview } from "@/components/PdfPreview";
import { Button } from "@/components/ui/button";
import { maxCharsForFewerLines, pageFit } from "@/lib/docLayout";
import type { SaveStatus } from "@/lib/saveLoop";
import { KeptDraftExistsError, keptDraft } from "@/lib/tailoring";
import { cn } from "@/lib/utils";
import { wordDiff } from "@/lib/wordDiff";
import type { Draft, DraftLayout } from "@/types/tailoring";
import { fetchDraftPdf } from "../../api/tailoring";
import { createDraftEditor } from "../../hooks/useDraftEditor";
import { useGoogleStatus } from "../../hooks/useGoogle";
import { useJob } from "../../hooks/useJobs";
import { usePdfUrl } from "../../hooks/usePdfUrl";
import { createSuggestions } from "../../hooks/useSuggestions";
import {
	useDiscardDraft,
	useJobDrafts,
	useKeepDraft,
} from "../../hooks/useTailoring";
import { ChangesDiff } from "./ChangesDiff";
import {
	DocPage,
	isSparse,
	type PageEditor,
	type PageMetrics,
	PX_PER_PT,
} from "./DocPage";
import { MarginCards } from "./MarginCards";
import { WandMenu } from "./WandMenu";

const NO_METRICS: PageMetrics = {
	contentPt: 0,
	availablePt: 1,
	bodyLinePt: 15,
	fits: {},
};

const RAIL_PX = 320;
const RAIL_GAP_PX = 48;
const DESKTOP_PAD_PX = 64;
const NARROW_PAD_PX = 32;

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? "" : "s"}`;

function SaveIndicator(props: { status: SaveStatus; onRetry: () => void }) {
	return (
		<output
			class="flex items-center gap-1.5 text-xs text-faint"
			data-testid="save-status"
		>
			<span
				aria-hidden="true"
				class={cn(
					"size-1.5 rounded-full transition-colors",
					props.status === "saving" && "animate-pulse bg-status-interview",
					props.status === "saved" && "bg-status-offer",
					props.status === "failed" && "bg-destructive",
				)}
			/>
			<Show
				when={props.status === "failed"}
				fallback={props.status === "saving" ? "Saving…" : "Saved"}
			>
				<span class="text-destructive-strong">Save failed</span>
				<Button variant="ghost" size="sm" onClick={props.onRetry}>
					Retry
				</Button>
			</Show>
		</output>
	);
}

function PageMeter(props: {
	metrics: PageMetrics;
	googlePages: number | undefined;
	googleAgrees: boolean;
}) {
	const fit = () =>
		pageFit(
			props.metrics.contentPt,
			props.metrics.availablePt,
			props.metrics.bodyLinePt,
		);
	const over = () => props.googlePages !== undefined || fit().over;
	const pct = () =>
		Math.min(100, (props.metrics.contentPt / props.metrics.availablePt) * 100);
	return (
		<output class="flex items-center gap-2.5" data-testid="page-meter">
			<div class="h-1.5 w-20 overflow-hidden rounded-full bg-border">
				<div
					class={cn(
						"h-full rounded-full transition-[width,background-color] duration-500 ease-out",
						over()
							? "bg-destructive"
							: fit().lines === 0
								? "bg-status-interview"
								: "bg-status-offer",
					)}
					style={{ width: `${pct()}%` }}
				/>
			</div>
			<span
				class={cn(
					"text-xs font-medium whitespace-nowrap",
					over() ? "text-destructive-strong" : "text-foreground",
				)}
			>
				<Show
					when={props.googlePages}
					fallback={
						<Show
							when={fit().over}
							fallback={
								<>
									One page ·{" "}
									{fit().lines === 0
										? "no lines spare"
										: `${plural(fit().lines, "line")} spare`}
									{props.googleAgrees ? " · Google agrees" : ""}
								</>
							}
						>
							Spills onto page 2 by {plural(fit().lines, "line")}
						</Show>
					}
				>
					{(n) => `Google renders ${n()} pages`}
				</Show>
			</span>
		</output>
	);
}

function PrintPreview(props: { draftId: string; onClose: () => void }) {
	const url = usePdfUrl(
		() => ({ id: props.draftId }),
		(s) => fetchDraftPdf(s.id),
	);
	onMount(() => {
		const onKey = (e: KeyboardEvent) => e.key === "Escape" && props.onClose();
		window.addEventListener("keydown", onKey);
		onCleanup(() => window.removeEventListener("keydown", onKey));
	});
	return (
		<div class="fixed inset-0 z-[55] flex justify-end bg-black/30 backdrop-blur-[2px]">
			<button
				type="button"
				aria-label="Close preview"
				class="flex-1 cursor-default"
				onClick={() => props.onClose()}
			/>
			<aside
				aria-label="Print preview"
				class="scroll-slim flex h-full w-full max-w-[52rem] flex-col overflow-y-auto bg-background shadow-2xl"
			>
				<div class="sticky top-0 z-10 flex h-14 items-center justify-between border-b border-border bg-surface px-5">
					<div>
						<p class="text-sm font-semibold text-foreground">Print preview</p>
						<p class="text-xs text-faint">
							Google's own export of the Draft, the final word on page breaks.
						</p>
					</div>
					<Button
						variant="ghost"
						size="icon"
						aria-label="Close preview"
						onClick={() => props.onClose()}
					>
						<Icon name="x" size={16} />
					</Button>
				</div>
				<div class="p-6">
					<PdfPreview url={url} title="Draft CV PDF" />
				</div>
			</aside>
		</div>
	);
}

function Notice(props: { children: JSX.Element; alert?: boolean }) {
	return (
		<div
			role={props.alert ? "alert" : undefined}
			class="border-b border-border bg-surface px-4 py-3 text-sm"
		>
			{props.children}
		</div>
	);
}

export function DraftEditor(props: {
	draft: Draft;
	layout: DraftLayout | null;
	layoutError?: string | undefined;
}) {
	// eslint-disable-next-line solid/reactivity -- pedantic: the layout is read once on purpose
	const ed = createDraftEditor(() => props.draft, props.layout);
	const keep = useKeepDraft();
	const discard = useDiscardDraft();
	const google = useGoogleStatus();
	const job = useJob(() => props.draft.jobId);
	const jobDrafts = useJobDrafts(() => props.draft.jobId);

	const [active, setActive] = createSignal<string>();
	const [metrics, setMetrics] = createSignal(NO_METRICS);
	const [showCounts, setShowCounts] = createSignal(true);
	const [showChanges, setShowChanges] = createSignal(false);
	const [preview, setPreview] = createSignal(false);
	const [blockedByKept, setBlockedByKept] = createSignal(false);
	const [stage, setStage] = createSignal<HTMLDivElement>();
	const [askOpen, setAskOpen] = createSignal(false);
	const [mainWidth, setMainWidth] = createSignal(0);
	const [innerHeight, setInnerHeight] = createSignal(0);
	const [wandTop, setWandTop] = createSignal(0);
	let measureLines: (slotId: string, text: string) => number = () => 1;

	const pagePx = () => (props.layout?.page.width ?? 0) * PX_PER_PT;
	const narrow = () =>
		mainWidth() > 0 &&
		mainWidth() < pagePx() + RAIL_PX + RAIL_GAP_PX + DESKTOP_PAD_PX;
	const scale = () =>
		narrow() ? Math.min(1, (mainWidth() - NARROW_PAD_PX) / pagePx()) : 1;

	const suggestions = createSuggestions({
		// eslint-disable-next-line solid/reactivity -- pedantic: the Draft id is fixed for this editor
		draftId: props.draft.id,
		text: ed.text,
		maxChars: (slotId) => {
			const fit = metrics().fits[slotId];
			const text = ed.text(slotId);
			return fit
				? maxCharsForFewerLines(text, fit.lines, fit.lastLineFill)
				: text.length;
		},
		apply: ed.setText,
	});
	const cardKeys = () =>
		ed.cardKeys(
			(slotId) =>
				ed.edited(slotId) ||
				!!suggestions.get(slotId) ||
				isSparse(metrics().fits[slotId]),
		);
	const activeSlot = () => {
		const key = active();
		return key && ed.slotIds.includes(key) ? key : undefined;
	};
	const focusLine = (slotId: string) =>
		queueMicrotask(() =>
			stage()
				?.querySelector<HTMLElement>(
					`[data-slot-id="${slotId}"] [role="textbox"]`,
				)
				?.focus({ preventScroll: true }),
		);
	const showWand = () => {
		const key = activeSlot();
		return !!key && editable() && !suggestions.get(key);
	};
	const fitSlot = (slotId: string) => {
		setActive(slotId);
		void suggestions.ask(slotId, "fit");
	};

	const editable = () =>
		props.draft.status === "ready" && props.draft.outcome === null;
	const changes = () => {
		const { base, content } = props.draft;
		return base && content ? { base, content } : null;
	};
	const otherKept = () => {
		const kept = keptDraft(jobDrafts.data ?? []);
		return kept?.id === props.draft.id ? undefined : kept;
	};

	const pageEditor: PageEditor = {
		get editable() {
			return editable();
		},
		get showCounts() {
			return showCounts();
		},
		get active() {
			return active();
		},
		text: ed.text,
		label: ed.label,
		diffFor: (slotId) => {
			const sug = suggestions.get(slotId);
			if (!sug || sug.error !== undefined) return null;
			return sug.done
				? wordDiff(sug.before, sug.text)
				: [
						{ op: "del", text: `${sug.before} ` },
						{ op: "add", text: sug.text },
					];
		},
		onInput: ed.setText,
		onFocus: setActive,
		onBlur: () => void ed.flush(),
	};

	createEffect(() => {
		active();
		setAskOpen(false);
	});
	createEffect(() => {
		const key = activeSlot();
		const root = stage();
		metrics();
		scale();
		void suggestions.get(key ?? "")?.done;
		const line = key && root?.querySelector(`[data-slot-id="${key}"]`);
		if (line && root)
			setWandTop(
				line.getBoundingClientRect().top - root.getBoundingClientRect().top,
			);
	});
	onMount(() => {
		const onKey = (e: KeyboardEvent) => {
			const key = active();
			if (!key) return;
			const sug = suggestions.get(key);
			if (e.key === "Escape" && sug) {
				suggestions.reject(key);
				focusLine(key);
			} else if ((e.metaKey || e.ctrlKey) && e.key === "k" && showWand()) {
				e.preventDefault();
				setAskOpen(true);
			} else if ((e.metaKey || e.ctrlKey) && e.key === "Enter" && sug?.done) {
				e.preventDefault();
				suggestions.accept(key);
				focusLine(key);
			}
		};
		window.addEventListener("keydown", onKey);
		onCleanup(() => window.removeEventListener("keydown", onKey));
	});

	const marginCards = (narrowRail: boolean) => (
		<MarginCards
			narrow={narrowRail}
			editor={ed}
			keys={cardKeys()}
			suggestions={suggestions}
			measureLines={(id, text) => measureLines(id, text)}
			stage={stage()}
			fits={metrics().fits}
			active={active()}
			editable={editable()}
			onActive={setActive}
			onFit={editable() ? fitSlot : undefined}
		/>
	);

	const doKeep = async () => {
		if (!(await ed.flush())) return;
		keep.mutate(props.draft.id, {
			onSuccess: () => setBlockedByKept(false),
			onError: (err) => setBlockedByKept(err instanceof KeptDraftExistsError),
		});
	};
	const replaceKept = () => {
		const kept = otherKept();
		if (!kept) return;
		discard.mutate(kept.id, { onSuccess: doKeep });
	};
	const openPreview = async () => {
		if (await ed.flush()) setPreview(true);
	};

	return (
		<div class="fixed inset-0 z-50 flex flex-col bg-background">
			<header class="flex min-h-14 shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b border-border bg-surface px-4 py-1.5">
				<Link
					to="/jobs/$id"
					params={{ id: props.draft.jobId }}
					aria-label="Back to job"
					class="grid size-8 place-items-center rounded-md text-muted transition-colors hover:bg-accent-subtle hover:text-foreground"
				>
					<Icon name="chevronLeft" size={16} />
				</Link>
				<div class="w-32 shrink-0">
					<h1 class="truncate text-sm font-semibold text-foreground">
						Draft CV
					</h1>
					<p class="truncate text-xs text-faint">
						{job.data?.Title ?? "Loading job…"}
					</p>
				</div>
				<Show when={editable()}>
					<SaveIndicator status={ed.status()} onRetry={ed.retry} />
				</Show>
				<div class="order-last flex w-full flex-wrap items-center justify-center gap-x-4 gap-y-1 lg:order-none lg:w-auto lg:flex-1">
					<Show when={props.layout}>
						<PageMeter
							metrics={metrics()}
							googlePages={ed.googlePages()}
							googleAgrees={ed.googleAgrees()}
						/>
						<span class="h-4 w-px bg-border" />
						<span
							class="font-mono text-xs whitespace-nowrap text-faint tabular-nums"
							data-testid="resolved-count"
						>
							{cardKeys().filter((k) => ed.isResolved(k)).length}/
							{cardKeys().length} resolved
						</span>
						<Button
							variant="ghost"
							size="sm"
							aria-pressed={showCounts()}
							onClick={() => setShowCounts((v) => !v)}
						>
							Line counts
						</Button>
					</Show>
				</div>
				<div class="ml-auto flex flex-wrap items-center justify-end gap-1.5 lg:ml-0">
					<Show when={changes()}>
						<Button
							variant="ghost"
							size="sm"
							aria-pressed={showChanges()}
							onClick={() => setShowChanges((v) => !v)}
						>
							Changes
						</Button>
					</Show>
					<Button
						variant="ghost"
						size="sm"
						aria-label="Print preview"
						onClick={openPreview}
					>
						<Icon name="fileText" size={14} />
						<span class="hidden 2xl:inline">Print preview</span>
					</Button>
					<Show when={props.draft.draftDocUrl}>
						{(url) => (
							<Button
								as="a"
								href={url()}
								target="_blank"
								rel="noopener noreferrer"
								variant="ghost"
								size="sm"
								aria-label="Open in Google Docs"
							>
								<Icon name="externalLink" size={14} />
								<span class="hidden 2xl:inline">Google Docs</span>
							</Button>
						)}
					</Show>
					<span class="mx-1 h-5 w-px bg-border" />
					<Show
						when={props.draft.status === "keeping"}
						fallback={
							<Show
								when={props.draft.outcome === null}
								fallback={
									<span class="px-2 text-sm text-muted">
										{props.draft.outcome === "kept" ? "Kept" : "Discarded"}
									</span>
								}
							>
								<Button
									variant="outline"
									size="sm"
									disabled={discard.isPending}
									onClick={() => discard.mutate(props.draft.id)}
								>
									Discard
								</Button>
								<Button size="sm" disabled={keep.isPending} onClick={doKeep}>
									Keep draft
								</Button>
							</Show>
						}
					>
						<span class="px-2 text-sm text-muted" aria-live="polite">
							Keeping…
						</span>
					</Show>
				</div>
			</header>

			<Show
				when={editable() && google.data?.connected && !google.data.canEditDocs}
			>
				<Notice>
					<p class="mb-3 text-foreground">
						Allow FastTrack to edit your CV Doc so a kept draft is added as a
						Tab. Without it, kept drafts land as separate Docs.
					</p>
					<Button
						as="a"
						href={googleWriteHref(`/tailoring/drafts/${props.draft.id}`)}
						variant="outline"
						size="sm"
					>
						Reconnect Google
					</Button>
				</Notice>
			</Show>
			<Show when={blockedByKept()}>
				<Notice alert>
					<p class="mb-3 text-foreground">
						This job already has a kept draft. A job keeps one draft at a time.
					</p>
					<div class="flex flex-wrap gap-2">
						<Show when={otherKept()?.id}>
							{(id) => (
								<Link
									to="/tailoring/drafts/$id"
									params={{ id: id() }}
									class="inline-flex h-8 items-center rounded-md border border-border bg-surface px-3 text-xs font-medium text-foreground transition-colors hover:border-border-strong hover:bg-surface-muted"
								>
									View kept draft
								</Link>
							)}
						</Show>
						<Button
							size="sm"
							disabled={!otherKept() || discard.isPending || keep.isPending}
							onClick={replaceKept}
						>
							Discard the kept draft and keep this one
						</Button>
					</div>
				</Notice>
			</Show>
			<Show when={keep.isError && !blockedByKept()}>
				<Notice alert>
					<p class="text-destructive-strong">
						Could not keep the draft. Try again.
					</p>
				</Notice>
			</Show>
			<Show when={editable() && props.draft.lastError}>
				<Notice alert>
					<p class="text-destructive-strong">
						Keeping the draft failed: {props.draft.lastError}
					</p>
				</Notice>
			</Show>
			<Show when={discard.isError}>
				<Notice alert>
					<p class="text-destructive-strong">
						Could not discard the draft. Try again.
					</p>
				</Notice>
			</Show>

			<main
				ref={(el) => {
					const ro = new ResizeObserver(() => setMainWidth(el.clientWidth));
					ro.observe(el);
					onCleanup(() => ro.disconnect());
				}}
				class="scroll-slim flex-1 overflow-auto bg-border/45"
				onMouseDown={(e) => {
					if (e.target === e.currentTarget) setActive(undefined);
				}}
			>
				<Show
					when={!showChanges() && props.layout}
					fallback={
						<div class="mx-auto max-w-3xl px-6 py-10">
							<Show
								when={showChanges() ? changes() : null}
								fallback={
									<div
										role="alert"
										class="rounded-xl border border-border bg-surface p-6 text-sm"
									>
										<p class="font-medium text-foreground">
											This CV cannot be shown as a page here.
										</p>
										<p class="mt-1 text-muted">
											{props.layoutError ?? "Its layout is not available."} Use
											Print preview to read Google's own export, or open the Doc
											in Google Docs.
										</p>
									</div>
								}
							>
								{(c) => (
									<section
										aria-label="Changes from your base CV"
										class="rounded-xl border border-border bg-surface p-6"
									>
										<h2 class="mb-3 text-sm font-semibold">
											Changes from your base CV
										</h2>
										<ChangesDiff
											base={c().base}
											content={c().content}
											provenance={props.draft.provenance}
											onUndo={editable() ? ed.setText : undefined}
										/>
									</section>
								)}
							</Show>
						</div>
					}
				>
					{(layout) => (
						<div
							class={cn(
								"mx-auto flex w-max items-start",
								narrow() ? "px-4 pt-4 pb-40" : "gap-12 px-8 pt-16 pb-40",
							)}
						>
							<div
								ref={setStage}
								class="relative"
								style={
									scale() < 1
										? {
												width: `${pagePx() * scale()}px`,
												height: `${innerHeight() * scale()}px`,
											}
										: {}
								}
							>
								<div
									ref={(el) => {
										const ro = new ResizeObserver(() =>
											setInnerHeight(el.offsetHeight),
										);
										ro.observe(el);
										onCleanup(() => ro.disconnect());
									}}
									style={
										scale() < 1
											? {
													width: `${pagePx()}px`,
													transform: `scale(${scale()})`,
													"transform-origin": "top left",
												}
											: {}
									}
								>
									<DocPage
										layout={layout()}
										editor={pageEditor}
										onMetrics={setMetrics}
										onMeasurer={(fn) => {
											measureLines = fn;
										}}
									/>
								</div>
								<Show when={showWand() && activeSlot()}>
									{(key) => (
										<WandMenu
											top={wandTop()}
											canFit={(metrics().fits[key()]?.lines ?? 1) > 1}
											canGround={ed.segments(key()).some((seg) => seg.novel)}
											askOpen={askOpen()}
											onAskOpen={setAskOpen}
											onAction={(action, prompt) =>
												void suggestions.ask(key(), action, prompt)
											}
										/>
									)}
								</Show>
								<Show when={narrow()}>{marginCards(true)}</Show>
							</div>
							<Show when={!narrow()}>{marginCards(false)}</Show>
						</div>
					)}
				</Show>
			</main>
			<div class="sr-only" aria-live="polite">
				{suggestions.announcement()}
			</div>
			<Show when={preview()}>
				<PrintPreview
					draftId={props.draft.id}
					onClose={() => setPreview(false)}
				/>
			</Show>
		</div>
	);
}
