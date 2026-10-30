import { createSignal, For, Show } from "solid-js";
import { Input } from "@/components/ui/input";
import { rowLabel } from "@/lib/scoreRows";
import type { ScoreRow } from "@/types/job";

const GROUPS: { label: string; effects: ScoreRow["effect"][] }[] = [
	{ label: "Gates", effects: ["gated"] },
	{ label: "Boosts", effects: ["favourite"] },
	{ label: "Met", effects: ["meets", "level"] },
	{ label: "Missed", effects: ["misses"] },
	{ label: "Unknown", effects: ["unknown"] },
];

export function Signals(props: {
	rows: ScoreRow[];
	notes: Record<string, string>;
	onNote: (key: string, note: string) => void;
}) {
	const groups = () =>
		GROUPS.map((g) => ({
			...g,
			rows: props.rows.filter((r) => g.effects.includes(r.effect)),
		})).filter((g) => g.rows.length > 0);
	return (
		<div class="flex flex-col gap-3">
			<For each={groups()}>
				{(g) => (
					<div class="flex flex-col gap-0.5">
						<span class="text-2xs font-medium uppercase tracking-wide text-faint">
							{g.label}
						</span>
						<For each={g.rows}>{(r) => <Row row={r} {...props} />}</For>
					</div>
				)}
			</For>
		</div>
	);
}

function Row(props: {
	row: ScoreRow;
	notes: Record<string, string>;
	onNote: (key: string, note: string) => void;
}) {
	const [open, setOpen] = createSignal(false);
	return (
		<div class="flex flex-col gap-1">
			<div class="flex items-center gap-1">
				<span
					class="min-w-0 flex-1 truncate px-2 py-1 text-sm text-foreground"
					title={rowLabel(props.row)}
				>
					{rowLabel(props.row)}
				</span>
				<button
					type="button"
					class="cursor-pointer px-1 text-sm text-faint hover:text-foreground aria-pressed:text-primary"
					aria-label={`Note on ${rowLabel(props.row)}`}
					aria-pressed={open() || !!props.notes[props.row.key]}
					onClick={() => setOpen(!open())}
				>
					✎
				</button>
			</div>
			<Show when={open()}>
				<Input
					class="h-8 text-sm"
					placeholder="One-line note"
					aria-label={`Note on ${rowLabel(props.row)}`}
					value={props.notes[props.row.key] ?? ""}
					onInput={(e) => props.onNote(props.row.key, e.currentTarget.value)}
				/>
			</Show>
		</div>
	);
}
