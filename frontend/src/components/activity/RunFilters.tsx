import { For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import {
	type ActivitySearch,
	clearFilters,
	facetValues,
	onlyFacetValue,
	resetFacet,
	toggleFacetValue,
} from "@/lib/scrapeRuns";
import { cn } from "@/lib/utils";
import type { ScrapeRunFacet, ScrapeRunSummary } from "@/types/scrapeRun";

const OUTCOME_DOTS: Record<string, string> = {
	running: "bg-primary",
	succeeded: "bg-status-offer",
	retried: "bg-status-interview",
	failed: "bg-destructive",
};

interface Option {
	value: string;
	label: string;
}

const FACET_TITLES: Record<ScrapeRunFacet, string> = {
	outcome: "Outcome",
	kind: "Type",
	source: "Source",
	trigger: "Trigger",
};

const STATIC_OPTIONS: Record<Exclude<ScrapeRunFacet, "source">, Option[]> = {
	outcome: [
		{ value: "failed", label: "Failed" },
		{ value: "retried", label: "Retried" },
		{ value: "succeeded", label: "Succeeded" },
		{ value: "running", label: "Running" },
	],
	kind: [{ value: "search", label: "Searches" }],
	trigger: [
		{ value: "scheduled", label: "Scheduled" },
		{ value: "manual", label: "Manual" },
	],
};

const FACETS: ScrapeRunFacet[] = ["outcome", "kind", "source", "trigger"];

function FacetGroup(props: {
	facet: ScrapeRunFacet;
	options: Option[];
	counts: Record<string, number>;
	search: ActivitySearch;
	onChange: (next: ActivitySearch) => void;
}) {
	const selected = () => facetValues(props.search, props.facet);
	return (
		<fieldset class="mb-5">
			<legend class="mb-1.5 flex w-full items-center justify-between text-sm font-semibold text-foreground">
				{FACET_TITLES[props.facet]}
				<Show when={selected().length > 0}>
					<button
						type="button"
						class="text-xs font-normal text-primary hover:underline"
						onClick={() =>
							props.onChange(resetFacet(props.search, props.facet))
						}
					>
						Reset
					</button>
				</Show>
			</legend>
			<For each={props.options}>
				{(option) => (
					<div class="group flex items-center gap-2 rounded px-1 py-1 hover:bg-surface-muted">
						<label class="flex min-w-0 flex-1 cursor-pointer items-center gap-2 text-sm text-foreground">
							<input
								type="checkbox"
								class="size-4 accent-primary"
								checked={selected().includes(option.value)}
								onChange={() =>
									props.onChange(
										toggleFacetValue(props.search, props.facet, option.value),
									)
								}
							/>
							<Show when={props.facet === "outcome"}>
								<span
									class={cn("size-2 rounded-full", OUTCOME_DOTS[option.value])}
								/>
							</Show>
							<span class="truncate">{option.label}</span>
						</label>
						<button
							type="button"
							class="text-xs text-muted opacity-0 hover:text-foreground focus-visible:opacity-100 group-hover:opacity-100"
							aria-label={`Only ${option.label}`}
							onClick={() =>
								props.onChange(
									onlyFacetValue(props.search, props.facet, option.value),
								)
							}
						>
							Only
						</button>
						<span class="w-10 text-right font-mono text-xs tabular-nums text-faint">
							{(props.counts[option.value] ?? 0).toLocaleString()}
						</span>
					</div>
				)}
			</For>
		</fieldset>
	);
}

export function FilterSidebar(props: {
	search: ActivitySearch;
	summary: ScrapeRunSummary | undefined;
	sourceLabel: (name: string) => string;
	onChange: (next: ActivitySearch) => void;
}) {
	const countsFor = (facet: ScrapeRunFacet) =>
		props.summary?.Facets[facet] ?? {};
	const sourceOptions = (): Option[] => {
		const names = new Set([
			...Object.keys(countsFor("source")),
			...facetValues(props.search, "source"),
		]);
		return [...names]
			.map((value) => ({ value, label: props.sourceLabel(value) }))
			.sort((a, b) => a.label.localeCompare(b.label));
	};
	const optionsFor = (facet: ScrapeRunFacet) =>
		facet === "source" ? sourceOptions() : STATIC_OPTIONS[facet];
	return (
		<aside aria-label="Filters" class="w-52 shrink-0">
			<For each={FACETS}>
				{(facet) => (
					<FacetGroup
						facet={facet}
						options={optionsFor(facet)}
						counts={countsFor(facet)}
						search={props.search}
						onChange={props.onChange}
					/>
				)}
			</For>
		</aside>
	);
}

export function FilterPills(props: {
	search: ActivitySearch;
	sourceLabel: (name: string) => string;
	onChange: (next: ActivitySearch) => void;
}) {
	const label = (facet: ScrapeRunFacet, value: string) =>
		facet === "source"
			? props.sourceLabel(value)
			: (STATIC_OPTIONS[facet as Exclude<ScrapeRunFacet, "source">].find(
					(o) => o.value === value,
				)?.label ?? value);
	return (
		<div class="mb-3 flex flex-wrap items-center gap-2">
			<For each={FACETS}>
				{(facet) => (
					<For each={facetValues(props.search, facet)}>
						{(value) => (
							<span class="inline-flex items-center gap-1 rounded-md border border-accent-border bg-accent-subtle px-2 py-0.5 text-xs text-accent-text">
								{FACET_TITLES[facet]} is{" "}
								<span class="font-medium">{label(facet, value)}</span>
								<button
									type="button"
									aria-label={`Remove ${FACET_TITLES[facet]} ${label(facet, value)}`}
									class="rounded hover:text-foreground"
									onClick={() =>
										props.onChange(toggleFacetValue(props.search, facet, value))
									}
								>
									<Icon name="x" class="size-3" />
								</button>
							</span>
						)}
					</For>
				)}
			</For>
			<Button
				variant="link"
				size="sm"
				class="h-auto px-1 text-xs text-muted"
				onClick={() => props.onChange(clearFilters(props.search))}
			>
				Clear all
			</Button>
		</div>
	);
}
