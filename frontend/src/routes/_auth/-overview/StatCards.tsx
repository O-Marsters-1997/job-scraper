import { Link } from "@tanstack/solid-router";
import type { JSX } from "solid-js";
import { Card } from "@/components/ui/card";
import { cn, plural } from "@/lib/utils";

interface FeatureStatProps {
	label: string;
	value: string;
	delta: string;
	deltaUp?: boolean;
	cta?: JSX.Element;
}

export function FeatureStat(props: FeatureStatProps) {
	return (
		<Card class="bg-accent-subtle/30 ring-1 ring-accent-border/40">
			<div class="flex h-full flex-col p-5">
				<div class="flex items-center justify-between gap-2">
					<p class="text-xs font-medium text-muted">{props.label}</p>
					{props.cta}
				</div>
				<p class="mt-3 font-mono text-4xl font-medium tabular-nums text-foreground">
					{props.value}
				</p>
				<p
					class={cn(
						"mt-auto pt-2 text-xs",
						props.deltaUp ? "text-primary" : "text-muted",
					)}
				>
					{props.delta}
				</p>
			</div>
		</Card>
	);
}

interface MiniStatProps {
	label: string;
	value: string;
	hint: string;
}

function MiniStat(props: MiniStatProps) {
	return (
		<div class="flex items-center justify-between gap-3 px-5 py-3.5">
			<div class="min-w-0">
				<p class="text-xs font-medium text-muted">{props.label}</p>
				<p class="mt-0.5 truncate text-xs text-faint">{props.hint}</p>
			</div>
			<p class="shrink-0 font-mono text-lg font-medium tabular-nums text-foreground">
				{props.value}
			</p>
		</div>
	);
}

const ctaClass =
	"shrink-0 text-xs font-medium text-primary transition-colors hover:text-primary-hover";

export function StatCards(props: {
	jobs: {
		total: number;
		sources: number;
		newToday: number;
		todaySources: number;
	};
	apps: {
		total: number;
		awaiting: number;
		responded: number;
		responseRate: number;
	};
}) {
	return (
		<div class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-[1.4fr_1.4fr_1.2fr]">
			<FeatureStat
				label="New today"
				value={props.jobs.newToday.toString()}
				delta={
					props.jobs.todaySources > 0
						? `across ${plural(props.jobs.todaySources, "source")}`
						: "none scraped today"
				}
				cta={
					<Link to="/jobs" class={ctaClass}>
						Review jobs →
					</Link>
				}
			/>
			<FeatureStat
				label="Active applications"
				value={props.apps.total.toString()}
				delta={
					props.apps.awaiting > 0
						? `${props.apps.awaiting} awaiting response`
						: "all responded to"
				}
				deltaUp={props.apps.awaiting > 0}
				cta={
					<Link
						to="/applications"
						search={{ status: undefined, chase: undefined }}
						class={ctaClass}
					>
						Manage →
					</Link>
				}
			/>
			<Card class="sm:col-span-2 lg:col-span-1">
				<div class="grid h-full grid-rows-2 divide-y divide-border">
					<MiniStat
						label="Total jobs"
						value={props.jobs.total.toString()}
						hint={`from ${plural(props.jobs.sources, "source")}`}
					/>
					<MiniStat
						label="Response rate"
						value={`${props.apps.responseRate}%`}
						hint={`${props.apps.responded} of ${props.apps.total} responded`}
					/>
				</div>
			</Card>
		</div>
	);
}
