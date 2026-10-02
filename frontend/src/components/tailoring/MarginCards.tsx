import {
	createEffect,
	createSignal,
	For,
	on,
	onCleanup,
	onMount,
	Show,
} from "solid-js";
import { createStore, reconcile } from "solid-js/store";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { cardTops } from "@/lib/cardLayout";
import { charsToSave } from "@/lib/docLayout";
import { cn } from "@/lib/utils";
import { wordDiff } from "@/lib/wordDiff";
import { type DraftEditorState, SKILLS_CARD } from "../../hooks/useDraftEditor";
import type { SuggestionsState } from "../../hooks/useSuggestions";
import { isSparse, type LineFit } from "./DocPage";
import { SuggestionCard } from "./SuggestionCard";

const GAP = 10;
const DOT = 20;
const INLINE_GAP = 6;
const WAND_PX = 32;

const anchorSelector = (key: string) =>
	key === SKILLS_CARD ? '[data-section="skills"]' : `[data-slot-id="${key}"]`;

type Tone = { dot: string; text: string };

function headerOf(ed: DraftEditorState, key: string): Tone {
	if (key === SKILLS_CARD)
		return {
			dot: "bg-status-interview",
			text: `${ed.gaps().length} missing from your CV`,
		};
	if (ed.findingsFor(key).some((f) => f.severity === "block"))
		return { dot: "bg-destructive", text: "Blocking issue" };
	return ed.edited(key)
		? { dot: "bg-border-strong", text: "Your edit" }
		: { dot: "bg-status-interview", text: "Short last line" };
}

const SEVERITY_DOT = {
	block: "bg-destructive",
	warn: "bg-status-interview",
	info: "bg-border-strong",
};

function Card(props: {
	cardKey: string;
	editor: DraftEditorState;
	fit: LineFit | undefined;
	active: boolean;
	editable: boolean;
	onOpen: () => void;
	onResolve: () => void;
	onFit?: ((slotId: string) => void) | undefined;
}) {
	// eslint-disable-next-line solid/reactivity -- pedantic: the editor object is never replaced
	const ed = props.editor;
	const key = () => props.cardKey;
	const isSlot = () => key() !== SKILLS_CARD;
	const resolved = () => ed.isResolved(key());
	const header = () => headerOf(ed, key());
	const sparse = () => isSparse(props.fit);
	return (
		<div
			data-testid="margin-card"
			data-card-key={key()}
			class={cn(
				"rounded-xl bg-surface text-sm transition-shadow duration-300",
				props.active
					? "shadow-xl ring-1 ring-accent-border"
					: "ring-1 ring-border",
				resolved() && !props.active && "opacity-60",
			)}
		>
			<button
				type="button"
				aria-expanded={props.active}
				class="flex w-full items-center gap-2 px-3.5 py-2.5 text-left"
				onClick={() => props.onOpen()}
			>
				<Show
					when={!resolved()}
					fallback={
						<Icon name="check" size={12} class="shrink-0 text-primary" />
					}
				>
					<span class={cn("size-2 shrink-0 rounded-full", header().dot)} />
				</Show>
				<span class="min-w-0 flex-1 truncate text-xs font-medium text-foreground">
					{header().text}
				</span>
				<Show when={props.fit}>
					{(fit) => (
						<span
							class={cn(
								"font-mono text-2xs tabular-nums",
								sparse() ? "text-status-interview" : "text-faint",
							)}
						>
							{fit().lines} {fit().lines === 1 ? "line" : "lines"}
						</span>
					)}
				</Show>
			</button>

			<Show when={props.active}>
				<div class="flex flex-col gap-3 border-t border-border px-3.5 pt-3 pb-3.5 animate-in fade-in duration-200">
					<Show when={key() === SKILLS_CARD}>
						<p class="text-xs text-muted">
							The job asks for these and your CV never mentions them. Add one
							only if it is true.
						</p>
						<ul class="flex flex-col gap-1.5">
							<For each={ed.gaps()}>
								{(gap) => <li class="text-xs text-foreground">{gap}</li>}
							</For>
						</ul>
					</Show>

					<Show when={isSlot()}>
						<Show when={ed.edited(key())}>
							<div>
								<p class="text-xs font-semibold text-muted">Your changes</p>
								<p class="mt-1 text-xs leading-relaxed" data-testid="card-diff">
									<For each={wordDiff(ed.original(key()), ed.text(key()))}>
										{(part) => (
											<span
												class={cn(
													part.op === "del" &&
														"text-faint line-through decoration-destructive/70",
													part.op === "add" &&
														"bg-accent-subtle text-accent-text",
													part.op === "same" && "text-muted",
												)}
											>
												{part.text}
											</span>
										)}
									</For>
								</p>
							</div>
						</Show>
						<For each={ed.findingsFor(key())}>
							{(f) => (
								<p class="flex items-start gap-2 text-xs text-foreground">
									<span
										class={cn(
											"mt-1 size-1.5 shrink-0 rounded-full",
											SEVERITY_DOT[f.severity],
										)}
									/>
									{f.message}
								</p>
							)}
						</For>
						<Show when={sparse()}>
							<div class="flex items-center justify-between gap-2 rounded-lg bg-surface-muted px-2.5 py-2 ring-1 ring-border">
								<p class="text-xs text-muted" data-testid="card-fit">
									Last line is{" "}
									{Math.round((props.fit?.lastLineFill ?? 0) * 100)}% full. Cut
									about{" "}
									{charsToSave(
										ed.text(key()),
										props.fit?.lines ?? 1,
										props.fit?.lastLineFill ?? 0,
									)}{" "}
									characters to save a line.
								</p>
								<Show when={props.onFit}>
									{(fit) => (
										<Button
											variant="outline"
											size="sm"
											class="shrink-0"
											onClick={() => fit()(key())}
										>
											<Icon name="wand" size={13} />
											Fit
										</Button>
									)}
								</Show>
							</div>
						</Show>
					</Show>

					<div class="flex items-center gap-2">
						<Button size="sm" onClick={props.onResolve}>
							<Icon name="check" size={14} />
							{resolved() ? "Reopen" : "Resolve"}
						</Button>
						<Show when={isSlot() && props.editable && ed.edited(key())}>
							<Button variant="ghost" size="sm" onClick={() => ed.undo(key())}>
								<Icon name="rotateCcw" size={14} />
								Undo my edits
							</Button>
						</Show>
					</div>
				</div>
			</Show>
		</div>
	);
}

export function MarginCards(props: {
	editor: DraftEditorState;
	keys: string[];
	suggestions: SuggestionsState;
	measureLines: (slotId: string, text: string) => number;
	stage: HTMLElement | undefined;
	fits: Record<string, LineFit>;
	active: string | undefined;
	editable: boolean;
	narrow: boolean;
	onActive: (key: string | undefined) => void;
	onFit?: ((slotId: string) => void) | undefined;
}) {
	const [tops, setTops] = createStore<Record<string, number>>({});
	const [ready, setReady] = createSignal(false);
	const cards = new Map<string, HTMLElement>();
	let rail: HTMLElement | undefined;

	const layout = () => {
		const stage = props.stage;
		if (!rail || !stage) return;
		const base = rail.getBoundingClientRect().top;
		const paperTop = stage.getBoundingClientRect().top;
		const keys = props.keys;
		const anchor = (key: string) =>
			stage.querySelector(anchorSelector(key))?.getBoundingClientRect();
		const desired = (key: string) => (anchor(key)?.top ?? paperTop) - base;
		const active = props.active;
		if (props.narrow) {
			const dots = keys
				.filter((key) => key !== active)
				.map((key) => ({ key, desired: desired(key), height: DOT }));
			const next = cardTops(dots, -1, 0);
			if (active && keys.includes(active)) {
				const a = anchor(active);
				next[active] = a
					? Math.max(a.bottom, a.top + WAND_PX) - base + INLINE_GAP
					: desired(active);
			}
			setTops(reconcile(next));
		} else {
			const slots = keys.map((key) => ({
				key,
				desired: desired(key),
				height: cards.get(key)?.offsetHeight ?? 0,
			}));
			setTops(reconcile(cardTops(slots, keys.indexOf(active ?? ""), GAP)));
		}
		setReady(true);
	};

	onMount(() => {
		const ro = new ResizeObserver(layout);
		if (props.stage) ro.observe(props.stage);
		for (const el of cards.values()) ro.observe(el);
		onCleanup(() => ro.disconnect());
		layout();
	});
	createEffect(
		on(
			[
				() => props.active,
				() => props.narrow,
				() => props.keys.join(),
				() => JSON.stringify(props.fits),
				() => props.stage,
			],
			layout,
		),
	);

	const open = (key: string) => {
		props.onActive(key);
		const anchor = props.stage?.querySelector(anchorSelector(key));
		anchor?.scrollIntoView({ behavior: "smooth", block: "center" });
		anchor
			?.querySelector<HTMLElement>('[role="textbox"]')
			?.focus({ preventScroll: true });
	};

	const card = (key: string) => (
		<Show
			when={props.suggestions.get(key)}
			fallback={
				<Card
					cardKey={key}
					editor={props.editor}
					fit={props.fits[key]}
					active={props.active === key}
					editable={props.editable}
					onOpen={() => open(key)}
					onResolve={() => {
						props.editor.setResolved(key, !props.editor.isResolved(key));
						props.onActive(undefined);
					}}
					onFit={props.onFit}
				/>
			}
		>
			{(sug) => (
				<SuggestionCard
					suggestion={sug()}
					linesBefore={
						props.fits[key]?.lines ?? props.measureLines(key, sug().before)
					}
					linesAfter={
						sug().done ? props.measureLines(key, sug().text) : undefined
					}
					onAccept={() => props.suggestions.accept(key)}
					onReject={() => props.suggestions.reject(key)}
					onRetry={() => props.suggestions.retry(key)}
				/>
			)}
		</Show>
	);

	return (
		<aside
			ref={rail}
			aria-label="Margin notes"
			class={cn(
				"relative",
				props.narrow
					? "pointer-events-none absolute inset-0"
					: "w-80 shrink-0 self-stretch",
			)}
		>
			<For each={props.keys}>
				{(key) => (
					<Show
						when={!props.narrow || props.active === key}
						fallback={
							<button
								type="button"
								tabIndex={-1}
								aria-label={headerOf(props.editor, key).text}
								data-testid="card-dot"
								data-card-key={key}
								class="pointer-events-auto absolute left-0.5 grid place-items-center"
								style={{
									top: `${tops[key] ?? 0}px`,
									width: `${DOT}px`,
									height: `${DOT}px`,
								}}
								onClick={() => open(key)}
							>
								<span
									class={cn(
										"size-2 rounded-full ring-2 ring-surface",
										props.editor.isResolved(key)
											? "bg-border-strong"
											: headerOf(props.editor, key).dot,
									)}
								/>
							</button>
						}
					>
						<div
							ref={(el) => {
								cards.set(key, el);
							}}
							class={cn(
								"absolute transition-[top,transform] duration-300 ease-out",
								props.narrow
									? "pointer-events-auto inset-x-2 z-20"
									: "inset-x-0",
								!ready() && "invisible",
								!props.narrow && props.active === key && "-translate-x-2",
							)}
							style={{ top: `${tops[key] ?? 0}px` }}
						>
							{card(key)}
						</div>
					</Show>
				)}
			</For>
		</aside>
	);
}
