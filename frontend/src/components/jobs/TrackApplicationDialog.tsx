import { createEffect, createSignal, For } from "solid-js";
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
	const statusesQuery = useApplicationStatuses();
	const createMutation = useCreateApplication();
	const updateMutation = useUpdateApplication();

	const [statusId, setStatusId] = createSignal("");
	const [notes, setNotes] = createSignal("");
	const [appliedAt, setAppliedAt] = createSignal("");
	const [salary, setSalary] = createSignal("");

	const isEdit = () => !!props.existingApp;

	createEffect(() => {
		if (props.open) {
			setStatusId(props.existingApp?.statusId ?? "");
			setNotes(props.existingApp?.notes ?? "");
			setAppliedAt(props.existingApp?.appliedAt ?? "");
			setSalary(props.existingApp?.salaryInfo ?? "");
		}
	});

	const handleSubmit = async () => {
		const jobId = props.job?.ID;
		if (!jobId) return;
		if (!isEdit()) {
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
		props.onOpenChange(false);
	};

	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>
						{isEdit() ? "Edit application" : "Track application"}
					</DialogTitle>
					<p class="text-sm text-muted">{props.job?.Title}</p>
				</DialogHeader>

				<div class="flex flex-col gap-4">
					<div>
						<Label>Status</Label>
						<select
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
							<Label>Applied date</Label>
							<Input
								type="date"
								value={appliedAt()}
								onInput={(e) => setAppliedAt(e.currentTarget.value)}
							/>
						</div>
						<div>
							<Label>Salary / comp</Label>
							<Input
								type="text"
								placeholder="e.g. £80,000"
								value={salary()}
								onInput={(e) => setSalary(e.currentTarget.value)}
							/>
						</div>
					</div>

					<div>
						<Label>Notes</Label>
						<textarea
							class="field resize-y"
							rows={3}
							placeholder="Any notes…"
							value={notes()}
							onInput={(e) => setNotes(e.currentTarget.value)}
						/>
					</div>
				</div>

				<DialogFooter class="border-t border-border pt-4">
					<Button variant="outline" onClick={() => props.onOpenChange(false)}>
						Cancel
					</Button>
					<Button
						onClick={handleSubmit}
						disabled={createMutation.isPending || updateMutation.isPending}
					>
						{createMutation.isPending || updateMutation.isPending
							? isEdit() ? "Updating…" : "Saving…"
							: isEdit() ? "Update" : "Save"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
