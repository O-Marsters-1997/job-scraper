import { createSignal, For, Show, untrack } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { useApplicationStatuses } from "../../../hooks/useApplicationStatuses";
import { useUpdateApplication } from "../../../hooks/useApplications";
import type { ApplicationWithDetails } from "../../../types/application";

export function EditApplicationDialog(props: {
	app: ApplicationWithDetails | null;
	open: boolean;
	onOpenChange: (open: boolean) => void;
}) {
	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent>
				<Show when={props.app}>
					{(app) => (
						<EditApplicationForm
							app={app()}
							onClose={() => props.onOpenChange(false)}
						/>
					)}
				</Show>
			</DialogContent>
		</Dialog>
	);
}

function EditApplicationForm(props: {
	app: ApplicationWithDetails;
	onClose: () => void;
}) {
	const statusesQuery = useApplicationStatuses();
	const updateMutation = useUpdateApplication();

	const initial = untrack(() => props.app);
	const [statusId, setStatusId] = createSignal(initial.StatusID);
	const [notes, setNotes] = createSignal(initial.Notes);
	const [appliedAt, setAppliedAt] = createSignal(initial.AppliedAt ?? "");
	const [salary, setSalary] = createSignal(initial.SalaryInfo);

	const saveForm = useFormSubmit(async () => {
		await updateMutation.mutateAsync({
			id: props.app.ID,
			data: {
				status_id: statusId() || undefined,
				notes: notes(),
				applied_at: appliedAt() || null,
				salary_info: salary(),
			},
		});
		props.onClose();
	});

	return (
		<>
			<DialogHeader>
				<DialogTitle>Edit application</DialogTitle>
				<p class="text-sm text-faint">{props.app.JobTitle}</p>
			</DialogHeader>

			<form onSubmit={saveForm.submit} class="flex flex-col gap-4">
				<FormFeedback error={saveForm.error()} />
				<div>
					<Label for="edit-app-status">Status</Label>
					<select
						id="edit-app-status"
						class="field"
						value={statusId()}
						onChange={(e) => setStatusId(e.currentTarget.value)}
					>
						<option value="">— No status —</option>
						<For each={statusesQuery.data}>
							{(s) => <option value={s.ID}>{s.Name}</option>}
						</For>
					</select>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<Label for="edit-app-applied-at">Applied date</Label>
						<Input
							id="edit-app-applied-at"
							type="date"
							value={appliedAt()}
							onInput={(e) => setAppliedAt(e.currentTarget.value)}
						/>
					</div>

					<div>
						<Label for="edit-app-salary">Salary / comp</Label>
						<Input
							id="edit-app-salary"
							type="text"
							placeholder="e.g. £80,000"
							value={salary()}
							onInput={(e) => setSalary(e.currentTarget.value)}
						/>
					</div>
				</div>

				<div>
					<Label for="edit-app-notes">Notes</Label>
					<textarea
						id="edit-app-notes"
						class="field resize-y"
						rows={3}
						placeholder="Any notes about this application…"
						value={notes()}
						onInput={(e) => setNotes(e.currentTarget.value)}
					/>
				</div>
				<DialogFooter class="border-t border-border pt-4">
					<Button type="button" variant="outline" onClick={props.onClose}>
						Cancel
					</Button>
					<Button type="submit" disabled={saveForm.pending()}>
						Save
					</Button>
				</DialogFooter>
			</form>
		</>
	);
}
