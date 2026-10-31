import type { Contact } from "@/types/contact";

export type ContactGroupKey =
	| "in_conversation"
	| "to_contact"
	| "waiting"
	| "revisit"
	| "closed";

export const contactGroups: { key: ContactGroupKey; label: string }[] = [
	{ key: "in_conversation", label: "In conversation" },
	{ key: "to_contact", label: "To contact" },
	{ key: "waiting", label: "Waiting for a reply" },
	{ key: "revisit", label: "Revisit later" },
	{ key: "closed", label: "Closed" },
];

export function contactGroup(c: Contact): ContactGroupKey {
	switch (c.stage) {
		case "to_contact":
			return "to_contact";
		case "messaged":
			return "waiting";
		case "closed":
			return "closed";
		case "replied":
			return c.outcome === "not_now" ? "revisit" : "in_conversation";
	}
}

export function groupContacts(
	contacts: Contact[],
): Record<ContactGroupKey, Contact[]> {
	const out: Record<ContactGroupKey, Contact[]> = {
		in_conversation: [],
		to_contact: [],
		waiting: [],
		revisit: [],
		closed: [],
	};
	for (const c of contacts) out[contactGroup(c)].push(c);
	return out;
}
