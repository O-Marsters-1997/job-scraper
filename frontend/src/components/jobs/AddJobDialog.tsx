import { createSignal, Show } from "solid-js";
import { ApiError } from "@/api/client";
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
import { useSubmitJobUrl } from "@/hooks/useSubmitJobUrl";
import type { SubmitJobUrlResult } from "@/types/submitJobUrl";

interface AddJobDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onSubmitted: (result: SubmitJobUrlResult) => void;
}

export function AddJobDialog(props: AddJobDialogProps) {
	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Add job</DialogTitle>
				</DialogHeader>
				<Show when={props.open}>
					<AddJobForm
						onSubmitted={props.onSubmitted}
						onClose={() => props.onOpenChange(false)}
					/>
				</Show>
			</DialogContent>
		</Dialog>
	);
}

function errorMessage(err: unknown): string {
	if (err instanceof ApiError) {
		const body = err.body as { error?: unknown } | undefined;
		if (typeof body?.error === "string") return body.error;
	}
	return "Couldn't add the job. Try again.";
}

function AddJobForm(props: {
	onSubmitted: (result: SubmitJobUrlResult) => void;
	onClose: () => void;
}) {
	const submit = useSubmitJobUrl();
	const [url, setUrl] = createSignal("");

	const form = useFormSubmit(async () => {
		props.onSubmitted(await submit.mutateAsync(url().trim()));
	}, errorMessage);

	return (
		<form onSubmit={form.submit} class="flex flex-col gap-4">
			<FormFeedback error={form.error()} />
			<div>
				<Label for="add-job-url">Job URL</Label>
				<Input
					id="add-job-url"
					type="url"
					required
					placeholder="https://"
					value={url()}
					onInput={(e) => setUrl(e.currentTarget.value)}
				/>
			</div>
			<DialogFooter class="border-t border-border pt-4">
				<Button type="button" variant="outline" onClick={props.onClose}>
					Cancel
				</Button>
				<Button type="submit" loading={form.pending()}>
					Add job
				</Button>
			</DialogFooter>
		</form>
	);
}
