import { format } from "date-fns";
import { For } from "solid-js";
import type { ScrapeRunSummary } from "@/types/scrapeRun";

export function RunHistogram(props: { summary: ScrapeRunSummary }) {
	const peak = () =>
		Math.max(
			1,
			...props.summary.Histogram.map((b) => b.OK + b.Retried + b.Failed),
		);
	const shape = (n: number) => `${(n / peak()) * 100}%`;
	const stamp = (iso: string | undefined) =>
		iso ? format(new Date(iso), "d MMM HH:mm") : "";
	return (
		<figure class="mb-3 rounded-xl border border-border bg-surface px-4 pt-3 pb-2">
			<div
				role="img"
				aria-label={`${props.summary.Total} runs over time, with retried and failed runs highlighted`}
				class="flex h-16 items-end gap-0.5"
			>
				<For each={props.summary.Histogram}>
					{(bucket) => (
						<div
							class="flex h-full min-w-0 flex-1 flex-col justify-end"
							title={`${stamp(bucket.Start)}: ${bucket.OK} ok, ${bucket.Retried} retried, ${bucket.Failed} failed`}
						>
							<div
								class="bg-destructive"
								style={{ height: shape(bucket.Failed) }}
							/>
							<div
								class="bg-status-interview"
								style={{ height: shape(bucket.Retried) }}
							/>
							<div
								class="bg-border-strong"
								style={{ height: shape(bucket.OK) }}
							/>
						</div>
					)}
				</For>
			</div>
			<figcaption class="mt-1 flex justify-between font-mono text-xs text-faint">
				<span>{stamp(props.summary.Histogram[0]?.Start)}</span>
				<span>{stamp(props.summary.Histogram.at(-1)?.Start)}</span>
			</figcaption>
		</figure>
	);
}
