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
import { cardTops } from "@/lib/cardLayout";
import type { LineFit } from "@/lib/docLayout";
import { cn } from "@/lib/utils";
import type { DraftEditorState } from "../../hooks/useDraftEditor";
import type { SuggestionsState } from "../../hooks/useSuggestions";
import { cardHeader, MarginCard } from "./MarginCard";
import { SuggestionCard } from "./SuggestionCard";

const GAP = 10;
const DOT = 20;
const INLINE_GAP = 6;
const WAND_PX = 32;

const anchorSelector = (key: string) => `[data-slot-id="${key}"]`;

function CardDot(props: {
	cardKey: string;
	editor: DraftEditorState;
	top: number;
	onOpen: () => void;
}) {
	const header = () => cardHeader(props.editor, props.cardKey);
	return (
		<button
			type="button"
			aria-label={header().text}
			data-testid="card-dot"
			data-card-key={props.cardKey}
			class="pointer-events-auto absolute left-0.5 grid place-items-center rounded-full focus-visible:ring-2 focus-visible:ring-primary focus-visible:outline-none"
			style={{ top: `${props.top}px`, width: `${DOT}px`, height: `${DOT}px` }}
			onClick={() => props.onOpen()}
		>
			<span
				class={cn(
					"size-2 rounded-full ring-2 ring-surface",
					props.editor.isResolved(props.cardKey)
						? "bg-border-strong"
						: header().dot,
				)}
			/>
		</button>
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
	hovered: string | undefined;
	onHover: (key: string, over: boolean) => void;
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
		const anchor = (key: string) =>
			stage.querySelector(anchorSelector(key))?.getBoundingClientRect();
		const desired = (key: string) => (anchor(key)?.top ?? paperTop) - base;
		const keys = props.keys;
		const active = props.active;
		if (!props.narrow) {
			const slots = keys.map((key) => ({
				key,
				desired: desired(key),
				height: cards.get(key)?.offsetHeight ?? 0,
			}));
			setTops(reconcile(cardTops(slots, keys.indexOf(active ?? ""), GAP)));
		} else {
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
		}
		setReady(true);
	};

	onMount(() => {
		const ro = new ResizeObserver(layout);
		if (props.stage) ro.observe(props.stage);
		for (const el of cards.values()) ro.observe(el);
		onCleanup(() => ro.disconnect());
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
				<MarginCard
					cardKey={key}
					editor={props.editor}
					fit={props.fits[key]}
					active={props.active === key}
					hovered={props.hovered === key}
					onHover={props.onHover}
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
							<CardDot
								cardKey={key}
								editor={props.editor}
								top={tops[key] ?? 0}
								onOpen={() => open(key)}
							/>
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
