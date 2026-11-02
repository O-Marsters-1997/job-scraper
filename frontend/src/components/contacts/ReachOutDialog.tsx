import { createMemo, createSignal, For, Show } from "solid-js";
import { ApiError } from "@/api/client";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useContacts, useReachOut } from "@/hooks/useContacts";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { cn } from "@/lib/utils";

const NEW = "new";

interface ReachOutDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	jobId: string;
	companyId: string | undefined;
	company: string;
}

function errorMessage(err: unknown): string {
	if (err instanceof ApiError) {
		const body = err.body as { error?: unknown } | undefined;
		if (typeof body?.error === "string") return body.error;
	}
	return "Couldn't set up the reach out. Try again.";
}

function ReachOutForm(props: ReachOutDialogProps) {
	const contacts = useContacts();
	const reachOut = useReachOut();
	const [choice, setChoice] = createSignal(NEW);
	const [name, setName] = createSignal("");
	const [role, setRole] = createSignal("");
	const [linkedin, setLinkedin] = createSignal("");
	const [email, setEmail] = createSignal("");

	const atCompany = createMemo(() =>
		props.companyId
			? (contacts.data ?? []).filter((c) => c.company_id === props.companyId)
			: [],
	);

	const form = useFormSubmit(async () => {
		const existing = choice();
		await reachOut.mutateAsync(
			existing === NEW
				? {
						job_id: props.jobId,
						contact_id: null,
						contact: {
							name: name().trim(),
							role: role().trim(),
							company_id: props.companyId ?? null,
							company_name: props.company,
							job_id: props.jobId,
							linkedin_url: linkedin().trim(),
							email: email().trim(),
						},
					}
				: { job_id: props.jobId, contact_id: existing, contact: null },
		);
		props.onOpenChange(false);
	}, errorMessage);

	const optionClass = (selected: boolean) =>
		cn(
			"flex cursor-pointer items-center gap-3 rounded-lg border px-3 py-2 focus-within:ring-2 focus-within:ring-primary",
			selected
				? "border-accent-border bg-accent-subtle"
				: "border-transparent hover:bg-surface-muted",
		);

	return (
		<form onSubmit={form.submit} class="flex flex-col gap-4">
			<FormFeedback error={form.error()} />
			<div
				role="radiogroup"
				aria-label="Who to message"
				class="flex flex-col gap-1"
			>
				<For each={atCompany()}>
					{(c) => (
						<label class={optionClass(choice() === c.id)}>
							<input
								type="radio"
								class="sr-only"
								name="who"
								checked={choice() === c.id}
								onChange={() => setChoice(c.id)}
							/>
							<span class="min-w-0 flex-1">
								<span class="block truncate text-sm font-medium text-foreground">
									{c.name}
								</span>
								<span class="block truncate text-xs text-muted">{c.role}</span>
							</span>
						</label>
					)}
				</For>
				<label class={optionClass(choice() === NEW)}>
					<input
						type="radio"
						class="sr-only"
						name="who"
						checked={choice() === NEW}
						onChange={() => setChoice(NEW)}
					/>
					<span class="flex size-8 items-center justify-center rounded-full border border-dashed border-border-strong text-faint">
						<Icon name="plus" size={14} />
					</span>
					<span class="text-sm font-medium text-foreground">Someone new</span>
				</label>
			</div>
			<Show when={choice() === NEW}>
				<div class="grid gap-2 sm:grid-cols-2">
					<Input
						class="sm:col-span-2"
						aria-label="Name"
						placeholder="Name"
						maxLength={200}
						value={name()}
						onInput={(e) => setName(e.currentTarget.value)}
					/>
					<Input
						aria-label="Role"
						placeholder="Role, e.g. Talent Partner"
						maxLength={200}
						value={role()}
						onInput={(e) => setRole(e.currentTarget.value)}
					/>
					<Input
						aria-label="Email"
						type="email"
						placeholder="Email"
						value={email()}
						onInput={(e) => setEmail(e.currentTarget.value)}
					/>
					<Input
						class="sm:col-span-2"
						aria-label="LinkedIn"
						type="url"
						placeholder="LinkedIn URL"
						value={linkedin()}
						onInput={(e) => setLinkedin(e.currentTarget.value)}
					/>
				</div>
			</Show>
			<DialogFooter class="border-t border-border pt-4">
				<Button
					type="button"
					variant="outline"
					onClick={() => props.onOpenChange(false)}
				>
					Cancel
				</Button>
				<Button
					type="submit"
					loading={form.pending()}
					disabled={choice() === NEW && name().trim() === ""}
				>
					Reach out
				</Button>
			</DialogFooter>
		</form>
	);
}

export function ReachOutDialog(props: ReachOutDialogProps) {
	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<DialogContent class="max-w-lg">
				<DialogHeader>
					<DialogTitle>Reach out at {props.company}</DialogTitle>
					<p class="text-sm text-muted">
						Who are you messaging? They'll wait on your Contacts page until it's
						sent.
					</p>
				</DialogHeader>
				<Show when={props.open}>
					<ReachOutForm {...props} />
				</Show>
			</DialogContent>
		</Dialog>
	);
}
