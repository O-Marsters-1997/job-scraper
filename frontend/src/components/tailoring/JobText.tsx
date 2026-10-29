import { For, Index, type JSX, Show } from "solid-js";
import { clipSpans, type TextPart } from "@/lib/htmlText";
import { findQuoteSpans, type QuoteSpan } from "@/lib/quoteMarks";
import { cn } from "@/lib/utils";
import type { DraftInsights } from "@/types/tailoring";

export type JobTextBlock = {
	label: string;
	source: DraftInsights["lines"][number]["source"];
	text: string;
	parts?: TextPart[];
};

type Mark = { slotId: string; n: number; quote: string };

function Marked(props: {
	text: string;
	marks: Mark[];
	spans: QuoteSpan[];
	selected: string | undefined;
	onSelect: (slotId: string) => void;
}) {
	const parts = () => {
		const byKey = new Map(props.marks.map((m) => [m.slotId, m]));
		const out: JSX.Element[] = [];
		let i = 0;
		for (const span of props.spans) {
			const mark = byKey.get(span.key);
			if (!mark) continue;
			out.push(props.text.slice(i, span.start));
			out.push(
				<mark
					data-testid="quote-mark"
					class={cn(
						"rounded px-0.5 text-foreground",
						props.selected === mark.slotId
							? "bg-accent-subtle ring-1 ring-accent-border"
							: "bg-surface-muted",
					)}
				>
					<button
						type="button"
						class="text-left"
						aria-pressed={props.selected === mark.slotId}
						onClick={() => props.onSelect(mark.slotId)}
					>
						{props.text.slice(span.start, span.end)}
						<sup class="ml-0.5 font-mono text-2xs text-accent-text">
							{mark.n}
						</sup>
					</button>
				</mark>,
			);
			i = span.end;
		}
		out.push(props.text.slice(i));
		return out;
	};
	return <>{parts()}</>;
}

const PART_CLASS: Record<TextPart["kind"], string> = {
	p: "",
	h: "mt-2 text-base font-semibold",
	li: "ml-5 list-item list-disc",
};

function BlockBody(props: {
	block: JobTextBlock;
	marks: Mark[];
	selected: string | undefined;
	onSelect: (slotId: string) => void;
}) {
	const spans = () =>
		findQuoteSpans(
			props.block.text,
			props.marks.map((m) => ({ key: m.slotId, quote: m.quote })),
		);
	return (
		<Show
			when={props.block.parts}
			fallback={
				<p class="whitespace-pre-wrap">
					<Marked
						text={props.block.text}
						marks={props.marks}
						spans={spans()}
						selected={props.selected}
						onSelect={props.onSelect}
					/>
				</p>
			}
		>
			{(parts) => (
				<div class="flex flex-col gap-2">
					<Index each={clipSpans(spans(), parts())}>
						{(item) => (
							<p class={PART_CLASS[item().part.kind]}>
								<Marked
									text={item().part.text}
									marks={props.marks}
									spans={item().spans}
									selected={props.selected}
									onSelect={props.onSelect}
								/>
							</p>
						)}
					</Index>
				</div>
			)}
		</Show>
	);
}

export function JobText(props: {
	blocks: JobTextBlock[];
	marks: (Mark & { source: JobTextBlock["source"] })[];
	selected: string | undefined;
	onSelect: (slotId: string) => void;
}) {
	return (
		<section
			aria-label="Job text"
			class="flex flex-col gap-4 self-start rounded-xl border border-border bg-surface p-6 text-[14px] leading-relaxed"
		>
			<For each={props.blocks}>
				{(block) => (
					<div>
						<p class="mb-1 text-xs font-medium text-muted">{block.label}</p>
						<BlockBody
							block={block}
							marks={props.marks.filter((m) => m.source === block.source)}
							selected={props.selected}
							onSelect={props.onSelect}
						/>
					</div>
				)}
			</For>
			<Show when={props.blocks.length === 0}>
				<p class="text-muted">No job text yet.</p>
			</Show>
		</section>
	);
}
