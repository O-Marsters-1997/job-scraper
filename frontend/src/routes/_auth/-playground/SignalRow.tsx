import { createSignal, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/ui/input";
import { useRevertCorrection, useSetCorrection } from "@/hooks/useCorrections";
import { rowLabel } from "@/lib/scoreRows";
import { cn } from "@/lib/utils";
import type { ScoreRow } from "@/types/job";

const canCorrect = (r: ScoreRow) =>
	r.corrected === true || (r.effect !== "retired" && r.key.includes(":"));

const btn =
	"rounded px-1.5 text-xs text-muted enabled:cursor-pointer hover:text-foreground";

export function SignalRow(props: {
	jobId: string;
	row: ScoreRow;
	note: string;
	onNote: (note: string) => void;
}) {
	const [open, setOpen] = createSignal(false);
	const set = useSetCorrection();
	const revert = useRevertCorrection();
	const target = () => ({ jobId: props.jobId, optionId: props.row.key });
	const busy = () => set.isPending || revert.isPending;
	const answer = (value: "yes" | "no") => set.mutate({ ...target(), value });
	return (
		<div class="flex flex-col gap-1">
			<div
				class={cn(
					"flex items-center gap-1 px-2 py-1 text-sm text-foreground",
					props.row.corrected && "bg-accent-subtle text-accent-text",
				)}
			>
				<span class="min-w-0 flex-1 truncate" title={rowLabel(props.row)}>
					{rowLabel(props.row)}
				</span>
				<Show when={props.row.corrected}>
					<span class="text-xs">corrected</span>
				</Show>
				<Show when={canCorrect(props.row)}>
					<Show
						when={props.row.resolved === "unknown" && !props.row.corrected}
						fallback={
							<button
								type="button"
								class={btn}
								disabled={busy()}
								aria-label={
									props.row.corrected
										? `Revert correction of ${props.row.label}`
										: `Mark ${props.row.label} as wrong`
								}
								onClick={() =>
									props.row.corrected
										? revert.mutate(target())
										: answer(props.row.resolved === "yes" ? "no" : "yes")
								}
							>
								✗
							</button>
						}
					>
						<button
							type="button"
							class={btn}
							disabled={busy()}
							aria-label={`Answer ${props.row.label}: yes`}
							onClick={() => answer("yes")}
						>
							yes
						</button>
						<button
							type="button"
							class={btn}
							disabled={busy()}
							aria-label={`Answer ${props.row.label}: no`}
							onClick={() => answer("no")}
						>
							no
						</button>
					</Show>
				</Show>
				<button
					type="button"
					class={btn}
					aria-label={`Note on ${rowLabel(props.row)}`}
					aria-pressed={open() || !!props.note}
					onClick={() => setOpen(!open())}
				>
					<Icon name="pen" size={14} />
				</button>
			</div>
			<Show when={open()}>
				<Input
					class="h-8 text-sm"
					placeholder="One-line note"
					aria-label={`Note text for ${rowLabel(props.row)}`}
					value={props.note}
					onInput={(e) => props.onNote(e.currentTarget.value)}
				/>
			</Show>
		</div>
	);
}
