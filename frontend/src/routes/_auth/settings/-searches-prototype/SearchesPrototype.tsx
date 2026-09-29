import { createSignal, onCleanup, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SourceBadge } from "@/components/SourceBadge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { buildBoardUrl, draftFromTarget, pagingNote } from "./boardUrl";
import {
	DraftFields,
	ParseNotes,
	SegmentedTabs,
	SourcePicker,
	useBoardDraft,
} from "./parts";
import { BoardSearchesTable, CompanyBoardsTable } from "./tables";
import type { ProtoSearches, SourceTab } from "./useProtoSearches";

const TAB_BLURB: Record<SourceTab, string> = {
	boards: "Result pages on job boards. Each run starts at page 1 and pages on.",
	ats: "Company careers boards, re-checked every few hours.",
};

export function SearchesPrototype(props: {
	s: ProtoSearches;
	tab: SourceTab;
	onTab: (t: SourceTab) => void;
}) {
	const [added, setAdded] = createSignal<string | null>(null);
	let timer: ReturnType<typeof setTimeout> | undefined;
	onCleanup(() => clearTimeout(timer));

	const announce = (message: string) => {
		setAdded(message);
		clearTimeout(timer);
		timer = setTimeout(() => setAdded(null), 6000);
	};

	return (
		<>
			<Omnibox
				s={props.s}
				onTab={props.onTab}
				onAdded={announce}
				onStart={() => setAdded(null)}
			/>
			<FormFeedback success={added() ?? false} />

			<div class="mb-3 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
				<SegmentedTabs
					value={props.tab}
					onChange={props.onTab}
					counts={{
						ats: props.s.list("ats").length,
						boards: props.s.list("boards").length,
					}}
				/>
				<p class="text-xs text-faint">{TAB_BLURB[props.tab]}</p>
			</div>

			<QueryBoundary query={props.s.query} fallbackRows={4}>
				{() => (
					<>
						<div hidden={props.tab !== "boards"}>
							<BoardSearchesTable s={props.s} />
						</div>
						<div hidden={props.tab !== "ats"}>
							<CompanyBoardsTable s={props.s} />
						</div>
					</>
				)}
			</QueryBoundary>
		</>
	);
}

type Detected =
	| { kind: "none" }
	| { kind: "checking" }
	| { kind: "board" }
	| { kind: "manual" }
	| { kind: "ats"; source: string; value: string }
	| { kind: "miss"; reason: string };

const SPECIFIC_MISS = /not a search results page|isn't a full URL/;

function Omnibox(props: {
	s: ProtoSearches;
	onTab: (t: SourceTab) => void;
	onAdded: (message: string) => void;
	onStart: () => void;
}) {
	const board = useBoardDraft();
	const [detected, setDetected] = createSignal<Detected>({ kind: "none" });
	let input!: HTMLInputElement;
	let pending: ReturnType<typeof setTimeout> | undefined;
	let generation = 0;
	onCleanup(() => clearTimeout(pending));

	const reset = () => {
		clearTimeout(pending);
		generation++;
		board.reset();
		setDetected({ kind: "none" });
	};

	const detect = (text: string) => {
		props.onStart();
		clearTimeout(pending);
		const gen = ++generation;
		board.pasteUrl(text);
		if (!text.trim()) return setDetected({ kind: "none" });
		if (board.parsed()) {
			props.onTab("boards");
			return setDetected({ kind: "board" });
		}
		setDetected({ kind: "checking" });
		pending = setTimeout(async () => {
			const r = await props.s.resolveBoard(text.trim()).catch(() => null);
			if (gen !== generation) return;
			if (r) {
				props.onTab("ats");
				return setDetected({ kind: "ats", ...r });
			}
			const reason = board.error() ?? "";
			setDetected({
				kind: "miss",
				reason: SPECIFIC_MISS.test(reason)
					? reason
					: "We don't recognise that URL. Paste a search from LinkedIn, Indeed or Work in Startups, or a company's Greenhouse, Lever, Ashby or Workable board.",
			});
		}, 350);
	};

	const buildByHand = () => {
		reset();
		setDetected({ kind: "manual" });
		props.onTab("boards");
	};

	const duplicate = () =>
		props.s
			.list("boards")
			.some((t) => buildBoardUrl(draftFromTarget(t)) === board.baseUrl());

	const addSearch = (e: Event) => {
		e.preventDefault();
		if (!board.baseUrl() || duplicate()) return;
		const d = board.draft();
		props.s.add(d.source, board.baseUrl());
		props.onAdded(
			`Added “${d.keywords || "search"}” on ${props.s.label(d.source)}. First run queued.`,
		);
		reset();
	};

	const trackCompany = (source: string, value: string) => {
		props.s.add(source, value);
		props.onAdded(
			`Now tracking ${value} on ${props.s.label(source)}. New roles appear on the next check.`,
		);
		reset();
	};

	const ats = () => {
		const d = detected();
		return d.kind === "ats" ? d : null;
	};
	const miss = () => {
		const d = detected();
		return d.kind === "miss" ? d : null;
	};

	const open = () => {
		const k = detected().kind;
		return k === "board" || k === "manual" || k === "ats" || k === "miss";
	};

	return (
		<div
			class={cn(
				"mb-4 overflow-hidden rounded-lg border bg-surface transition-colors",
				open()
					? "border-border-strong"
					: "border-border focus-within:border-primary",
			)}
		>
			<div class="flex items-center gap-3 px-3.5 py-2.5">
				<Show
					when={detected().kind === "checking"}
					fallback={<Icon name="link" size={15} class="shrink-0 text-faint" />}
				>
					<span class="size-3.5 shrink-0 rounded-full border-2 border-border-strong border-t-primary motion-safe:animate-spin" />
				</Show>
				<label for="omnibox" class="sr-only">
					Paste a job board search or careers page URL
				</label>
				<input
					id="omnibox"
					ref={input}
					type="url"
					autocomplete="off"
					spellcheck={false}
					class="min-w-0 flex-1 bg-transparent py-1 font-mono text-data text-foreground outline-none placeholder:font-sans placeholder:text-sm placeholder:text-faint"
					placeholder="Paste a job board search or a company careers page"
					value={detected().kind === "manual" ? "" : board.urlText()}
					readOnly={detected().kind === "manual"}
					onInput={(e) => detect(e.currentTarget.value)}
					onKeyDown={(e) => {
						if (e.key === "Escape") {
							reset();
							input.blur();
						}
					}}
				/>
				<Show
					when={detected().kind !== "none"}
					fallback={
						<button
							type="button"
							onClick={buildByHand}
							class="hidden shrink-0 rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary sm:block"
						>
							Build from fields
						</button>
					}
				>
					<button
						type="button"
						onClick={reset}
						aria-label="Clear"
						class="grid size-7 shrink-0 place-items-center rounded text-faint transition hover:bg-surface-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
					>
						<Icon name="x" size={12} />
					</button>
				</Show>
			</div>

			<Show when={detected().kind === "none"}>
				<button
					type="button"
					onClick={buildByHand}
					class="-mt-1 mb-2 ml-10 text-xs font-medium text-primary sm:hidden"
				>
					No URL? Build from fields
				</button>
			</Show>

			<Show when={miss()}>
				{(d) => (
					<div class="flex items-start justify-between gap-4 border-t border-border bg-surface-muted px-4 py-3">
						<p role="alert" class="text-xs text-destructive-strong">
							{d().reason}
						</p>
						<button
							type="button"
							onClick={buildByHand}
							class="shrink-0 text-xs font-medium text-primary hover:underline"
						>
							Build from fields
						</button>
					</div>
				)}
			</Show>

			<Show when={ats()}>
				{(d) => (
					<div class="flex flex-wrap items-center justify-between gap-3 border-t border-border bg-surface-muted px-4 py-3">
						<div class="flex min-w-0 items-center gap-2">
							<SourceBadge source={props.s.label(d().source)} />
							<span class="truncate font-mono text-data text-foreground">
								{d().value}
							</span>
							<span class="text-xs text-faint">company careers board</span>
						</div>
						<Show
							when={!props.s.exists(d().source, d().value)}
							fallback={
								<span class="flex items-center gap-1.5 text-xs text-muted">
									<Icon name="check" size={13} class="text-status-offer" />
									Already tracking
								</span>
							}
						>
							<Button
								size="sm"
								onClick={() => trackCompany(d().source, d().value)}
							>
								Track company
							</Button>
						</Show>
					</div>
				)}
			</Show>

			<Show when={detected().kind === "board" || detected().kind === "manual"}>
				<form
					onSubmit={addSearch}
					class="flex flex-col gap-4 border-t border-border bg-surface-muted px-4 py-4"
				>
					<div class="flex flex-wrap items-center justify-between gap-2">
						<Show
							when={detected().kind === "board"}
							fallback={
								<>
									<p class="text-xs font-medium text-foreground">Job board</p>
									<SourcePicker board={board} />
								</>
							}
						>
							<p class="flex items-center gap-2 text-xs text-muted">
								<SourceBadge source={props.s.label(board.draft().source)} />
								search detected. Adjust anything below.
							</p>
						</Show>
					</div>

					<DraftFields board={board} />

					<div class="flex flex-wrap items-end justify-between gap-3 border-t border-border pt-3">
						<div class="flex min-w-0 flex-col gap-1">
							<Show when={board.baseUrl()}>
								<p class="text-xs text-faint">
									Starts from{" "}
									<a
										href={board.baseUrl()}
										target="_blank"
										rel="noreferrer"
										class="text-muted underline decoration-border-strong underline-offset-2 hover:text-primary"
									>
										page 1
									</a>
									, {pagingNote(board.draft().source)}
								</p>
							</Show>
							<ParseNotes board={board} />
						</div>
						<div class="flex items-center gap-2">
							<Button type="button" variant="ghost" size="sm" onClick={reset}>
								Cancel
							</Button>
							<Button
								type="submit"
								size="sm"
								disabled={!board.baseUrl() || duplicate()}
							>
								{duplicate() ? "Already added" : "Add search"}
							</Button>
						</div>
					</div>
				</form>
			</Show>
		</div>
	);
}
