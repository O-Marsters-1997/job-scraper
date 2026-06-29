import { createSignal, For, Show } from "solid-js";
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
import { useApplicationStatuses } from "@/hooks/useApplicationStatuses";
import {
	useCreateApplication,
	useUpdateApplication,
} from "@/hooks/useApplications";
import type { Job } from "@/types/job";

export interface ExistingApp {
	id: string;
	statusId: string;
	notes?: string;
	appliedAt?: string | null;
	salaryInfo?: string;
}

interface TrackApplicationDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	job: Job | undefined;
	existingApp?: ExistingApp;
}

export function TrackApplicationDialog(props: TrackApplicationDialogProps) {
	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>
						{props.existingApp ? "Edit application" : "Track application"}
					</DialogTitle>
					<p class="text-sm text-muted">{props.job?.Title}</p>
				</DialogHeader>
				{/* Mount fresh on each open so signals initialize from props.existingApp
				    once — no createEffect needed to re-seed on open. */}
				<Show when={props.open}>
					<TrackApplicationForm
						job={props.job}
						existingApp={props.existingApp}
						onClose={() => props.onOpenChange(false)}
					/>
				</Show>
			</DialogContent>
		</Dialog>
	);
}

function TrackApplicationForm(props: {
	job: Job | undefined;
	existingApp?: ExistingApp;
	onClose: () => void;
}) {
	const statusesQuery = useApplicationStatuses();
	const createMutation = useCreateApplication();
	const updateMutation = useUpdateApplication();

	const isEdit = !!props.existingApp;
	const [statusId, setStatusId] = createSignal(
		props.existingApp?.statusId ?? "",
	);
	const [notes, setNotes] = createSignal(props.existingApp?.notes ?? "");
	const [appliedAt, setAppliedAt] = createSignal(
		props.existingApp?.appliedAt ?? "",
	);
	const [salary, setSalary] = createSignal(props.existingApp?.salaryInfo ?? "");

	const handleSubmit = async () => {
		const jobId = props.job?.ID;
		if (!jobId) return;
		if (!isEdit) {
			await createMutation.mutateAsync({
				job_id: jobId,
				status_id: statusId() || undefined,
				notes: notes(),
				applied_at: appliedAt() || null,
				salary_info: salary(),
			});
		} else {
			const appId = props.existingApp?.id;
			if (!appId) return;
			await updateMutation.mutateAsync({
				id: appId,
				data: {
					status_id: statusId() || undefined,
					notes: notes(),
					applied_at: appliedAt() || null,
					salary_info: salary(),
				},
			});
		}
		props.onClose();
	};

	return (
		<>
			<div class="flex flex-col gap-4">
				<div>
					<Label for="track-status">Status</Label>
					<select
						id="track-status"
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
						<Label for="track-applied-at">Applied date</Label>
						<Input
							id="track-applied-at"
							type="date"
							value={appliedAt()}
							onInput={(e) => setAppliedAt(e.currentTarget.value)}
						/>
					</div>
					<div>
						<Label for="track-salary">Salary / comp</Label>
						<Input
							id="track-salary"
							type="text"
							placeholder="e.g. £80,000"
							value={salary()}
							onInput={(e) => setSalary(e.currentTarget.value)}
						/>
					</div>
				</div>

				<div>
					<Label for="track-notes">Notes</Label>
					<textarea
						id="track-notes"
						class="field resize-y"
						rows={3}
						placeholder="Any notes…"
						value={notes()}
						onInput={(e) => setNotes(e.currentTarget.value)}
					/>
				</div>
			</div>

			<DialogFooter class="border-t border-border pt-4">
				<Button variant="outline" onClick={props.onClose}>
					Cancel
				</Button>
				<Button
					onClick={handleSubmit}
					disabled={createMutation.isPending || updateMutation.isPending}
				>
					{createMutation.isPending || updateMutation.isPending
						? isEdit
							? "Updating…"
							: "Saving…"
						: isEdit
							? "Update"
							: "Save"}
				</Button>
			</DialogFooter>
		</>
	);
}
