import { Show } from "solid-js";
import { formatDate } from "../lib/datetime";
import type { Quota, UsageLevel } from "../types/quota";

const BAR_TONE: Record<UsageLevel, string> = {
	ok: "bg-primary",
	warn: "bg-status-interview",
	critical: "bg-destructive",
};

function amount(n: number, unit: string): string {
	return unit === "USD" ? `$${n.toFixed(2)}` : `${n.toFixed(1)} ${unit}`;
}

export function QuotaBar(props: { quota: Quota }) {
	const q = () => props.quota;
	return (
		<Show when={q().status === "ok" && q().used !== null}>
			<Show
				when={q().limit !== null && q().percent !== null}
				fallback={
					<p class="text-sm text-foreground">
						{amount(q().used ?? 0, q().unit)} this month
					</p>
				}
			>
				<div class="max-w-lg">
					<div class="flex items-baseline justify-between text-sm text-foreground">
						<span>
							{amount(q().used ?? 0, q().unit)} of{" "}
							{amount(q().limit ?? 0, q().unit)}
						</span>
						<span>{Math.round(q().percent ?? 0)}%</span>
					</div>
					<div
						role="progressbar"
						aria-label={`${q().provider} usage`}
						aria-valuemin={0}
						aria-valuemax={100}
						aria-valuenow={Math.max(
							0,
							Math.min(100, Math.round(q().percent ?? 0)),
						)}
						class="mt-1.5 h-2 overflow-hidden rounded-full bg-border"
					>
						<div
							class={`h-full rounded-full ${BAR_TONE[q().level]}`}
							style={{
								width: `${Math.max(0, Math.min(100, q().percent ?? 0))}%`,
							}}
						/>
					</div>
					<Show when={q().resetsAt}>
						{(resetsAt) => (
							<p class="mt-1 text-xs text-faint">
								Resets {formatDate(resetsAt())}
							</p>
						)}
					</Show>
				</div>
			</Show>
		</Show>
	);
}
