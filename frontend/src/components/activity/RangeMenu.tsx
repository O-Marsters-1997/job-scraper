import { createEffect, createSignal, For, onCleanup, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import {
	type ActivitySearch,
	customRangeError,
	DEFAULT_RANGE,
	RANGE_PRESETS,
	setCustomRange,
	setRange,
} from "@/lib/scrapeRuns";
import { cn } from "@/lib/utils";

function rangeLabel(search: ActivitySearch): string {
	if (search.from && search.to) return "Custom range";
	const preset = search.range ?? DEFAULT_RANGE;
	return RANGE_PRESETS[preset as keyof typeof RANGE_PRESETS];
}

export function RangeMenu(props: {
	search: ActivitySearch;
	onChange: (next: ActivitySearch) => void;
}) {
	const [open, setOpen] = createSignal(false);
	const [from, setFrom] = createSignal("");
	const [to, setTo] = createSignal("");
	const iso = (local: string) => (local ? new Date(local).toISOString() : "");
	const error = () => customRangeError(iso(from()), iso(to()));
	const choose = (next: ActivitySearch) => {
		props.onChange(next);
		setOpen(false);
	};
	createEffect(() => {
		if (!open()) return;
		const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
		document.addEventListener("keydown", onKey);
		onCleanup(() => document.removeEventListener("keydown", onKey));
	});
	return (
		<div class="relative">
			<Button
				variant="outline"
				size="sm"
				aria-expanded={open()}
				onClick={() => setOpen((o) => !o)}
			>
				{rangeLabel(props.search)}
				<Icon name="chevronDown" class="size-3.5" />
			</Button>
			<Show when={open()}>
				<div class="absolute right-0 z-20 mt-1 w-72 rounded-lg border border-border bg-surface p-2 shadow-lg">
					<For each={Object.entries(RANGE_PRESETS)}>
						{([value, label]) => (
							<button
								type="button"
								class={cn(
									"flex w-full items-center justify-between rounded px-2 py-1.5 text-left text-sm hover:bg-surface-muted",
									!props.search.from &&
										(props.search.range ?? DEFAULT_RANGE) === value &&
										"text-primary",
								)}
								onClick={() => choose(setRange(props.search, value))}
							>
								{label}
								<Show
									when={
										!props.search.from &&
										(props.search.range ?? DEFAULT_RANGE) === value
									}
								>
									<Icon name="check" class="size-4" />
								</Show>
							</button>
						)}
					</For>
					<div class="mt-2 border-t border-border pt-2">
						<p class="mb-2 px-2 text-sm font-semibold">Custom range</p>
						<label class="mb-2 flex items-center justify-between gap-2 px-2 text-sm text-muted">
							From
							<input
								type="datetime-local"
								class="field h-8 w-48 text-sm"
								value={from()}
								onInput={(e) => setFrom(e.currentTarget.value)}
							/>
						</label>
						<label class="mb-2 flex items-center justify-between gap-2 px-2 text-sm text-muted">
							To
							<input
								type="datetime-local"
								class="field h-8 w-48 text-sm"
								value={to()}
								onInput={(e) => setTo(e.currentTarget.value)}
							/>
						</label>
						<div class="flex items-center justify-between gap-2 px-2">
							<output class="text-xs text-faint">
								{from() && to() ? (error() ?? "") : "Runs are kept for 30 days"}
							</output>
							<Button
								size="sm"
								disabled={error() !== null}
								onClick={() =>
									choose(setCustomRange(props.search, iso(from()), iso(to())))
								}
							>
								Apply
							</Button>
						</div>
					</div>
				</div>
			</Show>
		</div>
	);
}
