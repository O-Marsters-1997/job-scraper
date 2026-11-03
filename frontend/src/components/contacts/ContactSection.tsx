import { For, type JSX, Show } from "solid-js";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { chaseOverdueDays } from "@/lib/chase";
import { firstName } from "@/lib/contactCompanies";
import { lastSentOn, openReachOut, stageLabel } from "@/lib/contactStatus";
import { cn } from "@/lib/utils";
import type { Contact } from "@/types/contact";

type SectionKind = "toContact" | "waiting" | "rest";

function sentAgo(c: Contact): string {
	const sent = lastSentOn(c);
	if (!sent) return "";
	const days = chaseOverdueDays(sent, new Date());
	if (days === 0) return "Sent today";
	return `Sent ${days} ${days === 1 ? "day" : "days"} ago`;
}

export function ContactSection(props: {
	title: string;
	kind: SectionKind;
	contacts: Contact[];
	highlightId: string | undefined;
	onOpen: (c: Contact) => void;
	onReply: (c: Contact) => void;
}) {
	return (
		<Show when={props.contacts.length > 0}>
			<section>
				<h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-faint">
					{props.title} ({props.contacts.length})
				</h2>
				<div class="grid gap-3 md:grid-cols-2">
					<For each={props.contacts}>
						{(c) => (
							<Card
								class={cn(
									"flex items-center justify-between gap-3 p-4",
									props.highlightId === c.id &&
										"outline-2 outline-primary motion-safe:animate-pulse",
								)}
							>
								<button
									type="button"
									class="min-w-0 text-left"
									onClick={() => props.onOpen(c)}
								>
									<span class="block truncate text-sm font-semibold text-foreground">
										{c.name}
									</span>
									<span class="block truncate text-xs text-muted">
										{props.kind === "toContact"
											? openReachOut(c)?.job_title
											: props.kind === "waiting"
												? sentAgo(c)
												: [c.role, c.company_name].filter(Boolean).join(" · ")}
									</span>
								</button>
								{rowAction(props, c)}
							</Card>
						)}
					</For>
				</div>
			</section>
		</Show>
	);
}

function rowAction(
	props: {
		kind: SectionKind;
		onOpen: (c: Contact) => void;
		onReply: (c: Contact) => void;
	},
	c: Contact,
): JSX.Element {
	switch (props.kind) {
		case "toContact":
			return (
				<Button size="sm" onClick={() => props.onOpen(c)}>
					Write to {firstName(c.name)}
				</Button>
			);
		case "waiting":
			return (
				<Button size="sm" variant="outline" onClick={() => props.onReply(c)}>
					They replied
				</Button>
			);
		case "rest":
			return <Badge variant="secondary">{stageLabel(c)}</Badge>;
	}
}
