import { For } from "solid-js";
import type { ScoreRow } from "@/types/job";
import { SignalRow } from "./SignalRow";

const GROUPS: { label: string; keep: (r: ScoreRow) => boolean }[] = [
	{ label: "Corrected", keep: (r) => r.corrected === true },
	{ label: "Gates", keep: (r) => !r.corrected && r.effect === "gated" },
	{ label: "Boosts", keep: (r) => !r.corrected && r.effect === "favourite" },
	{
		label: "Met",
		keep: (r) => !r.corrected && (r.effect === "meets" || r.effect === "level"),
	},
	{ label: "Missed", keep: (r) => !r.corrected && r.effect === "misses" },
	{ label: "Unknown", keep: (r) => !r.corrected && r.effect === "unknown" },
];

export function Signals(props: {
	jobId: string;
	rows: ScoreRow[];
	notes: Record<string, string>;
	onNote: (key: string, note: string) => void;
}) {
	const groups = () =>
		GROUPS.map((g) => ({
			...g,
			rows: props.rows.filter(g.keep),
		})).filter((g) => g.rows.length > 0);
	return (
		<div class="flex flex-col gap-3">
			<For each={groups()}>
				{(g) => (
					<div class="flex flex-col gap-0.5">
						<span class="text-2xs font-medium uppercase tracking-wide text-faint">
							{g.label}
						</span>
						<For each={g.rows}>
							{(r) => (
								<SignalRow
									jobId={props.jobId}
									row={r}
									note={props.notes[r.key] ?? ""}
									onNote={(note) => props.onNote(r.key, note)}
								/>
							)}
						</For>
					</div>
				)}
			</For>
		</div>
	);
}
