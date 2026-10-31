import { createSignal, For, Show, untrack } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { useMarkContactSent, useRecordContactReply } from "@/hooks/useContacts";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import {
	type Contact,
	type ContactChannel,
	channelLabels,
	type ReplyOutcome,
	replyOutcomes,
} from "@/types/contact";

const sendChannels: ContactChannel[] = ["linkedin", "email", "call"];

interface DialogProps {
	contact: Contact | undefined;
	onClose: () => void;
}

export function MarkSentDialog(props: DialogProps) {
	return (
		<Dialog open={!!props.contact} onOpenChange={(o) => !o && props.onClose()}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Mark sent</DialogTitle>
					<Show when={props.contact}>
						{(c) => <p class="text-sm text-muted">{c().name}</p>}
					</Show>
				</DialogHeader>
				<Show when={props.contact}>
					{(c) => <MarkSentForm contact={c()} onClose={props.onClose} />}
				</Show>
			</DialogContent>
		</Dialog>
	);
}

function MarkSentForm(props: { contact: Contact; onClose: () => void }) {
	const mark = useMarkContactSent();
	const [channel, setChannel] = createSignal<ContactChannel>(
		untrack(() => (props.contact.email ? "email" : "linkedin")),
	);
	const form = useFormSubmit(async () => {
		await mark.mutateAsync({
			id: props.contact.id,
			channel: channel(),
			template_kind: "",
			template_name: "",
		});
		props.onClose();
	});
	return (
		<form onSubmit={form.submit} class="flex flex-col gap-4">
			<FormFeedback error={form.error()} />
			<div>
				<Label for="sent-channel">Sent by</Label>
				<select
					id="sent-channel"
					class="field"
					value={channel()}
					onChange={(e) => setChannel(e.currentTarget.value as ContactChannel)}
				>
					<For each={sendChannels}>
						{(c) => <option value={c}>{channelLabels[c]}</option>}
					</For>
				</select>
			</div>
			<DialogFooter class="border-t border-border pt-4">
				<Button type="button" variant="outline" onClick={props.onClose}>
					Cancel
				</Button>
				<Button type="submit" loading={form.pending()}>
					Mark sent
				</Button>
			</DialogFooter>
		</form>
	);
}

export function ReplyDialog(props: DialogProps) {
	return (
		<Dialog open={!!props.contact} onOpenChange={(o) => !o && props.onClose()}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>They replied</DialogTitle>
					<Show when={props.contact}>
						{(c) => <p class="text-sm text-muted">{c().name}</p>}
					</Show>
				</DialogHeader>
				<Show when={props.contact}>
					{(c) => <ReplyForm contact={c()} onClose={props.onClose} />}
				</Show>
			</DialogContent>
		</Dialog>
	);
}

function ReplyForm(props: { contact: Contact; onClose: () => void }) {
	const record = useRecordContactReply();
	const [outcome, setOutcome] = createSignal<ReplyOutcome>("interested");
	const [reply, setReply] = createSignal("");
	const form = useFormSubmit(async () => {
		await record.mutateAsync({
			id: props.contact.id,
			outcome: outcome(),
			reply: reply().trim(),
		});
		props.onClose();
	});
	return (
		<form onSubmit={form.submit} class="flex flex-col gap-4">
			<FormFeedback error={form.error()} />
			<fieldset class="flex flex-col gap-2">
				<legend class="mb-1 text-sm font-medium text-foreground">
					What did they say?
				</legend>
				<For each={replyOutcomes}>
					{(o) => (
						<label class="flex cursor-pointer items-start gap-2 text-sm">
							<input
								type="radio"
								name="reply-outcome"
								class="mt-1 accent-primary"
								checked={outcome() === o.key}
								onChange={() => setOutcome(o.key)}
							/>
							<span>
								<span class="font-medium text-foreground">{o.label}</span>
								<span class="block text-xs text-faint">{o.effect}</span>
							</span>
						</label>
					)}
				</For>
			</fieldset>
			<div>
				<Label for="reply-text">Their reply (optional)</Label>
				<textarea
					id="reply-text"
					class="field resize-y"
					rows={3}
					maxLength={4000}
					value={reply()}
					onInput={(e) => setReply(e.currentTarget.value)}
				/>
			</div>
			<DialogFooter class="border-t border-border pt-4">
				<Button type="button" variant="outline" onClick={props.onClose}>
					Cancel
				</Button>
				<Button type="submit" loading={form.pending()}>
					Save
				</Button>
			</DialogFooter>
		</form>
	);
}
