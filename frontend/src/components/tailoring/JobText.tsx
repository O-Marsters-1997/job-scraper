import { For, type JSX, Show } from "solid-js";
import { findQuoteSpans } from "@/lib/quoteMarks";
import { cn } from "@/lib/utils";
import type { DraftInsights } from "@/types/tailoring";

export type JobTextBlock = {
	label: string;
	source: DraftInsights["lines"][number]["source"];
	text: string;
};

type Mark = { slotId: string; n: number; quote: string };

function Marked(props: {
	text: string;
	marks: Mark[];
	selected: string | undefined;
	onSelect: (slotId: string) => void;
}) {
	const parts = () => {
		const byKey = new Map(props.marks.map((m) => [m.slotId, m]));
		const out: JSX.Element[] = [];
		let i = 0;
		for (const span of findQuoteSpans(
			props.text,
			props.marks.map((m) => ({ key: m.slotId, quote: m.quote })),
		)) {
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
						<p class="whitespace-pre-wrap">
							<Marked
								text={block.text}
								marks={props.marks.filter((m) => m.source === block.source)}
								selected={props.selected}
								onSelect={props.onSelect}
							/>
						</p>
					</div>
				)}
			</For>
			<Show when={props.blocks.length === 0}>
				<p class="text-muted">No job text yet.</p>
			</Show>
		</section>
	);
}
