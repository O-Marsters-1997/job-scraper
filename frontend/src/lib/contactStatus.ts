import type { Contact } from "@/types/contact";

export interface ContactSections {
	toContact: Contact[];
	waiting: Contact[];
	rest: Contact[];
}

export function openReachOut(c: Contact) {
	return c.reach_outs.find((r) => r.done_at === null);
}

export function lastSentOn(c: Contact): string | undefined {
	return c.history.filter((e) => e.kind === "sent").at(-1)?.occurred_on;
}

function lastActivity(c: Contact): string {
	return c.history.at(-1)?.occurred_on ?? c.created_at;
}

export function contactSections(contacts: Contact[]): ContactSections {
	const out: ContactSections = { toContact: [], waiting: [], rest: [] };
	for (const c of contacts) {
		if (openReachOut(c)) out.toContact.push(c);
		else if (c.stage === "messaged") out.waiting.push(c);
		else out.rest.push(c);
	}
	out.waiting.sort((a, b) =>
		(lastSentOn(a) ?? "").localeCompare(lastSentOn(b) ?? ""),
	);
	out.rest.sort((a, b) => lastActivity(b).localeCompare(lastActivity(a)));
	return out;
}

export function stageLabel(c: Contact): string {
	switch (c.stage) {
		case "to_contact":
			return "To contact";
		case "messaged":
			return "Messaged";
		case "closed":
			return "Closed";
		case "replied":
			return c.outcome === "not_now" ? "Parked" : "In conversation";
	}
}
