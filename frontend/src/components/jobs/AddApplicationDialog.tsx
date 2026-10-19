import { createEffect, createSignal, Show } from "solid-js";
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
import { useAddManualJob } from "@/hooks/useManualJob";
import { useResolveUrl } from "@/hooks/useSources";
import type { Job } from "@/types/job";

interface AddApplicationDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	onAdded: (job: Pick<Job, "ID" | "Title">) => void;
}

export function AddApplicationDialog(props: AddApplicationDialogProps) {
	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Add application</DialogTitle>
				</DialogHeader>
				<Show when={props.open}>
					<AddApplicationForm
						onAdded={props.onAdded}
						onClose={() => props.onOpenChange(false)}
					/>
				</Show>
			</DialogContent>
		</Dialog>
	);
}

function isHttpUrl(value: string) {
	return URL.canParse(value) && /^https?:/.test(value);
}

function AddApplicationForm(props: {
	onAdded: (job: Pick<Job, "ID" | "Title">) => void;
	onClose: () => void;
}) {
	const addJob = useAddManualJob();
	const [url, setUrl] = createSignal("");
	const [title, setTitle] = createSignal("");
	const [company, setCompany] = createSignal("");
	const [description, setDescription] = createSignal("");
	const [found, setFound] = createSignal<Pick<Job, "ID" | "Title"> | null>(
		null,
	);

	const resolved = useResolveUrl(
		() => url().trim(),
		() => isHttpUrl(url().trim()),
	);
	createEffect(() => {
		const hit = resolved.data;
		if (hit?.kind === "ats" && company() === "") setCompany(hit.value);
	});

	const form = useFormSubmit(async () => {
		const result = await addJob.mutateAsync({
			url: url().trim(),
			title: title().trim(),
			company: company().trim(),
			description: description().trim(),
		});
		if (result.existing) {
			setFound(result.job);
			return;
		}
		props.onAdded(result.job);
	});

	return (
		<Show
			when={found()}
			fallback={
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
					<div class="grid grid-cols-2 gap-3">
						<div>
							<Label for="add-job-title">Title</Label>
							<Input
								id="add-job-title"
								type="text"
								required
								value={title()}
								onInput={(e) => setTitle(e.currentTarget.value)}
							/>
						</div>
						<div>
							<Label for="add-job-company">Company</Label>
							<Input
								id="add-job-company"
								type="text"
								required={resolved.data?.kind !== "ats"}
								value={company()}
								onInput={(e) => setCompany(e.currentTarget.value)}
							/>
						</div>
					</div>
					<div>
						<Label for="add-job-description">Description (optional)</Label>
						<textarea
							id="add-job-description"
							class="field min-h-24"
							value={description()}
							onInput={(e) => setDescription(e.currentTarget.value)}
						/>
					</div>
					<DialogFooter class="border-t border-border pt-4">
						<Button type="button" variant="outline" onClick={props.onClose}>
							Cancel
						</Button>
						<Button type="submit" disabled={form.pending()}>
							{form.pending() ? "Adding…" : "Add"}
						</Button>
					</DialogFooter>
				</form>
			}
		>
			{(job) => (
				<div class="flex flex-col gap-4">
					<p class="text-sm text-foreground">
						Already in your jobs: {job().Title}
					</p>
					<DialogFooter class="border-t border-border pt-4">
						<Button type="button" variant="outline" onClick={props.onClose}>
							Cancel
						</Button>
						<Button type="button" onClick={() => props.onAdded(job())}>
							Continue
						</Button>
					</DialogFooter>
				</div>
			)}
		</Show>
	);
}
