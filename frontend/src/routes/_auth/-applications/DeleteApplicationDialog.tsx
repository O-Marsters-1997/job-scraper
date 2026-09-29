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
import { useDeleteApplication } from "../../../hooks/useApplications";
import type { ApplicationWithDetails } from "../../../types/application";

export function DeleteApplicationDialog(props: {
	app: ApplicationWithDetails | null;
	onClose: () => void;
}) {
	const deleteMutation = useDeleteApplication();

	const deleteForm = useFormSubmit(async () => {
		if (!props.app) return;
		await deleteMutation.mutateAsync(props.app.ID);
		props.onClose();
	});

	return (
		<Dialog
			open={props.app !== null}
			onOpenChange={(open) => {
				if (open) return;
				props.onClose();
				deleteForm.setError(null);
			}}
		>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Delete application?</DialogTitle>
					<p class="text-sm text-muted">
						This removes tracking for{" "}
						<span class="font-medium text-foreground">
							{props.app?.JobTitle}
						</span>
						. The job stays on your Jobs list. This can't be undone.
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
							{deleteForm.pending() ? "Deleting…" : "Delete application"}
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
