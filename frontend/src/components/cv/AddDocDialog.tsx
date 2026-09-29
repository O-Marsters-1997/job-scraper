import { createSignal } from "solid-js";
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
import { useAddTrackedDoc } from "../../hooks/useCVTemplates";

export function AddDocDialog(props: {
	open: boolean;
	onOpenChange: (v: boolean) => void;
}) {
	const addMutation = useAddTrackedDoc();
	const [docUrl, setDocUrl] = createSignal("");

	const form = useFormSubmit(
		async () => {
			await addMutation.mutateAsync(docUrl().trim());
			handleOpenChange(false);
		},
		(err) => {
			if (err instanceof Error && err.message === "invalid-url") {
				return "Invalid Google Docs URL or ID";
			}
			if (err instanceof Error && err.message === "access-denied") {
				return "Cannot access this document. Make sure it's shared with your Google account.";
			}
			return "Something went wrong. Please try again.";
		},
	);

	const handleOpenChange = (open: boolean) => {
		if (!open) {
			setDocUrl("");
			form.setError(null);
		}
		props.onOpenChange(open);
	};

	return (
		<Dialog open={props.open} onOpenChange={handleOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Add Google Doc</DialogTitle>
					<p class="text-sm text-faint">
						Paste the URL of a Google Doc to track it as a CV template.
					</p>
				</DialogHeader>

				<form onSubmit={form.submit} class="flex flex-col gap-3">
					<FormFeedback error={form.error()} />
					<div>
						<Label for="add-doc-url">Google Docs URL</Label>
						<Input
							id="add-doc-url"
							type="url"
							placeholder="https://docs.google.com/document/d/…"
							value={docUrl()}
							onInput={(e) => {
								setDocUrl(e.currentTarget.value);
								form.setError(null);
							}}
							aria-invalid={!!form.error()}
						/>
					</div>
					<DialogFooter class="border-t border-border pt-4">
						<Button
							type="button"
							variant="outline"
							onClick={() => handleOpenChange(false)}
						>
							Cancel
						</Button>
						<Button type="submit" disabled={form.pending() || !docUrl().trim()}>
							Add doc
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
