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
import { useDeleteApplications } from "../../../hooks/useApplications";
import type { ApplicationWithDetails } from "../../../types/application";

export function DeleteApplicationDialog(props: {
	apps: ApplicationWithDetails[];
	onClose: () => void;
	onDeleted: (ids: string[]) => void;
}) {
	const deleteMutation = useDeleteApplications();
	const many = () => props.apps.length > 1;
	const count = () => `${props.apps.length} applications`;
	const label = () => (many() ? count() : "application");

	const deleteForm = useFormSubmit(async () => {
		const ids = props.apps.map((a) => a.ID);
		if (ids.length === 0) return;
		await deleteMutation.mutateAsync(ids);
		props.onDeleted(ids);
		props.onClose();
	});

	return (
		<Dialog
			open={props.apps.length > 0}
			onOpenChange={(open) => {
				if (open) return;
				props.onClose();
				deleteForm.setError(null);
			}}
		>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Delete {label()}?</DialogTitle>
					<p class="text-sm text-muted">
						This removes tracking for{" "}
						<span class="font-medium text-foreground">
							{many() ? count() : props.apps[0]?.JobTitle}
						</span>
						. {many() ? "The jobs stay" : "The job stays"} on your Jobs list.
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
							loading={deleteForm.pending()}
						>
							Delete {label()}
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
