import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { useDeleteTaskType } from "@/hooks/useTasks";
import type { TaskType } from "@/types/task";

export function DeleteTaskTypeDialog(props: {
	type: TaskType | null;
	onClose: () => void;
}) {
	const deleteMutation = useDeleteTaskType();

	const deleteForm = useFormSubmit(async () => {
		if (!props.type) return;
		await deleteMutation.mutateAsync(props.type.ID);
		props.onClose();
	});

	const count = () => props.type?.TaskCount ?? 0;

	return (
		<Dialog
			open={props.type !== null}
			onOpenChange={(open) => {
				if (open) return;
				props.onClose();
				deleteForm.setError(null);
			}}
		>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Delete task type?</DialogTitle>
					<p class="text-sm text-muted">
						Deleting{" "}
						<span class="font-medium text-foreground">{props.type?.Name}</span>{" "}
						deletes its tasks too.{" "}
						<span class="font-medium text-foreground">
							{count()} task{count() === 1 ? "" : "s"} will be deleted.
						</span>{" "}
						This can't be undone.
					</p>
				</DialogHeader>
				<form onSubmit={deleteForm.submit} class="flex flex-col gap-4">
					<FormFeedback error={deleteForm.error()} />
					<DialogFooter class="border-t border-border pt-4">
						<Button type="button" variant="outline" onClick={props.onClose}>
							Cancel
						</Button>
						<Button
							type="submit"
							variant="destructive"
							disabled={deleteForm.pending()}
						>
							{deleteForm.pending() ? "Deleting…" : "Delete task type"}
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
