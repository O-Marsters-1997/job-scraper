import type { JSX } from "solid-js";
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

export function ConfirmDeleteDialog(props: {
	open: boolean;
	onClose: () => void;
	title: string;
	confirmLabel: string;
	description: JSX.Element;
	onConfirm: () => Promise<void>;
}) {
	const form = useFormSubmit(
		async () => {
			await props.onConfirm();
			props.onClose();
		},
		(err) =>
			err instanceof Error
				? err.message
				: "Could not delete. Please try again.",
	);

	return (
		<Dialog
			open={props.open}
			onOpenChange={(open) => {
				if (open) return;
				form.setError(null);
				props.onClose();
			}}
		>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{props.title}</DialogTitle>
					<p class="text-sm text-muted">{props.description}</p>
				</DialogHeader>
				<form onSubmit={form.submit} class="flex flex-col gap-4">
					<FormFeedback error={form.error()} />
					<DialogFooter class="border-t border-border pt-4">
						<Button type="button" variant="outline" onClick={props.onClose}>
							Cancel
						</Button>
						<Button
							type="submit"
							variant="destructive"
							disabled={form.pending()}
						>
							{form.pending() ? "Deleting…" : props.confirmLabel}
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
