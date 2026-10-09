import { createSignal, untrack } from "solid-js";
import { Button } from "@/components/ui/button";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import {
	draftError,
	draftFrom,
	effectiveDraft,
	toRunWindow,
} from "@/lib/runWindow";
import { runWindowErrorMessage } from "@/lib/runWindowError";
import { useUpdateSourceTarget } from "../../../../hooks/useSourceTargets";
import type { SourceTarget } from "../../../../types/sourceTarget";
import { RunWindowFields } from "./RunWindowFields";

export function ScheduleEditor(props: {
	target: SourceTarget;
	incremental: boolean;
	onDone: () => void;
}) {
	const updateMutation = useUpdateSourceTarget();
	const [draft, setDraft] = createSignal(
		untrack(() => draftFrom(props.target.RunWindow)),
	);
	const invalid = () => draftError(effectiveDraft(draft(), props.incremental));

	const form = useFormSubmit(
		async () => {
			if (invalid()) return;
			const effective = {
				...draft(),
				automatic: draft().automatic && props.incremental,
			};
			await updateMutation.mutateAsync({
				id: props.target.ID,
				run_window: toRunWindow(effective),
			});
			props.onDone();
		},
		(err) =>
			runWindowErrorMessage(err) ?? "Could not save the schedule. Try again.",
	);

	return (
		<form onSubmit={form.submit} class="flex flex-col gap-3 py-1">
			<RunWindowFields
				idPrefix={`schedule-${props.target.ID}`}
				value={draft()}
				onChange={setDraft}
				incremental={props.incremental}
				error={invalid() ?? form.error()}
			/>
			<div class="flex items-center gap-2">
				<Button
					type="submit"
					size="sm"
					disabled={form.pending() || !!invalid()}
				>
					{form.pending() ? "Saving…" : "Save schedule"}
				</Button>
				<Button type="button" variant="ghost" size="sm" onClick={props.onDone}>
					Cancel
				</Button>
			</div>
		</form>
	);
}
