import { Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { pageFit } from "@/lib/docLayout";
import type { SaveStatus } from "@/lib/saveLoop";
import { cn } from "@/lib/utils";
import type { PageMetrics } from "./DocPage";

const plural = (n: number, word: string) => `${n} ${word}${n === 1 ? "" : "s"}`;

export function SaveIndicator(props: {
	status: SaveStatus;
	onRetry: () => void;
}) {
	return (
		<output
			class="flex h-8 w-48 shrink-0 items-center gap-1.5 text-xs text-faint"
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
				<Button variant="ghost" size="sm" onClick={() => props.onRetry()}>
					Retry
				</Button>
			</Show>
		</output>
	);
}

export function PageMeter(props: {
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
	const label = () => {
		if (props.googlePages !== undefined)
			return `Google renders ${props.googlePages} pages`;
		const { over, lines } = fit();
		if (over) return `Spills onto page 2 by ${plural(lines, "line")}`;
		const spare =
			lines === 0 ? "no lines spare" : `${plural(lines, "line")} spare`;
		return `One page · ${spare}${props.googleAgrees ? " · Google agrees" : ""}`;
	};
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
				{label()}
			</span>
		</output>
	);
}
