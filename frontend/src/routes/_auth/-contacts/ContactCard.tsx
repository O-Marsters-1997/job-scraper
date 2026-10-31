import { createSignal, For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { formatChaseDate } from "@/lib/chase";
import {
	type Contact,
	type ContactEvent,
	type ContactSent,
	channelLabels,
	replyOutcomes,
} from "@/types/contact";
import { outreachKinds } from "@/types/outreach";
import { Drafter } from "./Drafter";

export function eventLabel(e: ContactEvent): string {
	const outcome = replyOutcomes.find((o) => o.key === e.outcome)?.label;
	switch (e.kind) {
		case "added":
			return "Added";
		case "sent":
			return e.template_kind && e.template_name
				? [
						e.channel ? channelLabels[e.channel] : "Sent",
						outreachKinds.find((k) => k.key === e.template_kind)?.label,
						e.template_name,
					]
						.filter(Boolean)
						.join(" · ")
				: `Sent${e.channel ? ` by ${channelLabels[e.channel]}` : ""}`;
		case "reply":
			return `Replied${outcome ? `: ${outcome}` : ""}`;
		case "closed":
			return "Closed: no reply";
	}
}

export function ContactCard(props: {
	contact: Contact;
	applicationStatus?: string | undefined;
	onSent: () => void;
	onReply: () => void;
	onDelete: () => void;
	onDraftSent: (sent: ContactSent) => void;
}) {
	const c = () => props.contact;
	const [drafting, setDrafting] = createSignal(false);
	const subtitle = () =>
		[c().role, c().company_name, c().job_title, props.applicationStatus]
			.filter(Boolean)
			.join(" · ");
	return (
		<Card class="flex flex-col gap-3 p-4">
			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0">
					<h3 class="truncate text-sm font-semibold text-foreground">
						{c().name}
					</h3>
					<Show when={subtitle()}>
						<p class="truncate text-xs text-muted">{subtitle()}</p>
					</Show>
				</div>
				<Show when={c().follow_up_by}>
					{(day) => (
						<span class="shrink-0 text-xs text-faint">
							Follow up {formatChaseDate(day())}
						</span>
					)}
				</Show>
			</div>
			<Show when={c().reply}>
				<blockquote class="border-l-2 border-border pl-3 text-sm text-muted">
					{c().reply}
				</blockquote>
			</Show>
			<ol class="flex flex-col gap-1 text-xs text-muted">
				<For each={c().history}>
					{(e) => (
						<li>
							<span class="text-faint">{formatChaseDate(e.occurred_on)}</span>{" "}
							{eventLabel(e)}
						</li>
					)}
				</For>
			</ol>
			<div class="flex flex-wrap gap-2">
				<Show when={c().stage !== "closed"}>
					<Button
						size="sm"
						aria-expanded={drafting()}
						onClick={() => setDrafting((open) => !open)}
					>
						Draft message
					</Button>
					<Button size="sm" variant="outline" onClick={props.onSent}>
						Mark sent
					</Button>
				</Show>
				<Show when={c().stage === "messaged" || c().stage === "replied"}>
					<Button size="sm" variant="outline" onClick={props.onReply}>
						They replied
					</Button>
				</Show>
				<Button size="sm" variant="ghost" onClick={props.onDelete}>
					Delete
				</Button>
			</div>
			<Show when={drafting()}>
				<Drafter contact={c()} onSent={props.onDraftSent} />
			</Show>
		</Card>
	);
}
