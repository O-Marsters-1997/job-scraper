import { createEffect, createSignal, Show } from "solid-js";
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
import { useAddTrackedDoc } from "../../hooks/useCVTemplates";

export function AddDocDialog(props: {
	open: boolean;
	onOpenChange: (v: boolean) => void;
}) {
	const addMutation = useAddTrackedDoc();
	const [docUrl, setDocUrl] = createSignal("");
	const [urlError, setUrlError] = createSignal<string | null>(null);

	createEffect(() => {
		if (!props.open) {
			setDocUrl("");
			setUrlError(null);
		}
	});

	const handleAdd = async () => {
		setUrlError(null);
		try {
			await addMutation.mutateAsync(docUrl().trim());
			props.onOpenChange(false);
		} catch (err) {
			if (err instanceof Error && err.message === "invalid-url") {
				setUrlError("Invalid Google Docs URL or ID");
			} else if (err instanceof Error && err.message === "access-denied") {
				setUrlError(
					"Cannot access this document. Make sure it's shared with your Google account.",
				);
			} else {
				setUrlError("Something went wrong. Please try again.");
			}
		}
	};

	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Add Google Doc</DialogTitle>
					<p class="text-sm text-faint">
						Paste the URL of a Google Doc to track it as a CV template.
					</p>
				</DialogHeader>

				<div class="flex flex-col gap-3">
					<div>
						<Label for="add-doc-url">Google Docs URL</Label>
						<Input
							id="add-doc-url"
							type="url"
							placeholder="https://docs.google.com/document/d/…"
							value={docUrl()}
							onInput={(e) => {
								setDocUrl(e.currentTarget.value);
								setUrlError(null);
							}}
							onKeyDown={(e) => e.key === "Enter" && handleAdd()}
							aria-invalid={!!urlError()}
							aria-describedby={urlError() ? "add-doc-url-error" : undefined}
						/>
						<Show when={urlError()}>
							<p
								id="add-doc-url-error"
								class="mt-1.5 text-xs text-destructive-strong"
							>
								{urlError()}
							</p>
						</Show>
					</div>
				</div>

				<DialogFooter class="border-t border-border pt-4">
					<Button variant="outline" onClick={() => props.onOpenChange(false)}>
						Cancel
					</Button>
					<Button
						onClick={handleAdd}
						disabled={addMutation.isPending || !docUrl().trim()}
					>
						Add doc
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
