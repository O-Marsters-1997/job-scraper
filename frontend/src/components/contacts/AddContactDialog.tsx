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
import { useCreateContact } from "@/hooks/useContacts";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import type { Job } from "@/types/job";

interface AddContactDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	job?: Pick<Job, "ID" | "Title"> | undefined;
}

export function AddContactDialog(props: AddContactDialogProps) {
	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Add contact</DialogTitle>
					<Show when={props.job}>
						{(job) => <p class="text-sm text-muted">{job().Title}</p>}
					</Show>
				</DialogHeader>
				<Show when={props.open}>
					<AddContactForm
						job={props.job}
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
	return "Couldn't add the contact. Try again.";
}

function AddContactForm(props: {
	job: Pick<Job, "ID" | "Title"> | undefined;
	onClose: () => void;
}) {
	const create = useCreateContact();
	const [name, setName] = createSignal("");
	const [role, setRole] = createSignal("");
	const [companyName, setCompanyName] = createSignal("");
	const [linkedin, setLinkedin] = createSignal("");
	const [email, setEmail] = createSignal("");

	const form = useFormSubmit(async () => {
		await create.mutateAsync({
			name: name().trim(),
			role: role().trim(),
			company_id: null,
			company_name: companyName().trim(),
			job_id: props.job?.ID ?? null,
			linkedin_url: linkedin().trim(),
			email: email().trim(),
		});
		props.onClose();
	}, errorMessage);

	return (
		<form onSubmit={form.submit} class="flex flex-col gap-4">
			<FormFeedback error={form.error()} />
			<div>
				<Label for="contact-name">Name</Label>
				<Input
					id="contact-name"
					required
					maxLength={200}
					value={name()}
					onInput={(e) => setName(e.currentTarget.value)}
				/>
			</div>
			<div>
				<Label for="contact-role">Role</Label>
				<Input
					id="contact-role"
					placeholder="Recruiter, Director of Engineering…"
					maxLength={200}
					value={role()}
					onInput={(e) => setRole(e.currentTarget.value)}
				/>
			</div>
			<Show when={!props.job}>
				<div>
					<Label for="contact-company">Company</Label>
					<Input
						id="contact-company"
						maxLength={200}
						value={companyName()}
						onInput={(e) => setCompanyName(e.currentTarget.value)}
					/>
				</div>
			</Show>
			<div>
				<Label for="contact-linkedin">LinkedIn</Label>
				<Input
					id="contact-linkedin"
					type="url"
					placeholder="https://www.linkedin.com/in/…"
					value={linkedin()}
					onInput={(e) => setLinkedin(e.currentTarget.value)}
				/>
			</div>
			<div>
				<Label for="contact-email">Email</Label>
				<Input
					id="contact-email"
					type="email"
					value={email()}
					onInput={(e) => setEmail(e.currentTarget.value)}
				/>
			</div>
			<DialogFooter class="border-t border-border pt-4">
				<Button type="button" variant="outline" onClick={props.onClose}>
					Cancel
				</Button>
				<Button
					type="submit"
					loading={form.pending()}
					disabled={name().trim() === ""}
				>
					Add contact
				</Button>
			</DialogFooter>
		</form>
	);
}
