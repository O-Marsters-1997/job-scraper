import type {
	Contact,
	ContactCompany,
	ContactEvent,
	ContactInput,
	ContactSent,
	MarkSentInput,
	RecordReplyInput,
} from "@/types/contact";

let contacts: Contact[] = [];
let nextId = 1;

const today = () => new Date().toISOString().slice(0, 10);
const inDays = (days: number) => {
	const d = new Date();
	d.setDate(d.getDate() + days);
	return d.toISOString().slice(0, 10);
};

function event(kind: ContactEvent["kind"], rest: Partial<ContactEvent> = {}) {
	return {
		id: `event-${nextId++}`,
		kind,
		occurred_on: today(),
		job_id: null,
		...rest,
	} satisfies ContactEvent;
}

function sentSinceReply(history: ContactEvent[]): number {
	const lastReply = history.map((e) => e.kind).lastIndexOf("reply");
	return history.slice(lastReply + 1).filter((e) => e.kind === "sent").length;
}

function replace(id: string, change: (c: Contact) => Contact): Contact {
	const found = contacts.find((c) => c.id === id);
	if (!found) throw new Error(`contact ${id} not found`);
	const updated = change(found);
	contacts = contacts.map((c) => (c.id === id ? updated : c));
	return updated;
}

export function getContacts(): Contact[] {
	return [...contacts].sort((a, b) => b.created_at.localeCompare(a.created_at));
}

export function getContactCompanies(): ContactCompany[] {
	const tracked: ContactCompany = {
		id: "mock-company-northwind",
		name: "Northwind",
		kind: "direct",
		tracked: true,
		open_jobs: [
			{ job_id: "mock-job-1", title: "Platform Engineer", has_contact: false },
		],
		application_ids: [],
	};
	const linked = new Map<string, ContactCompany>();
	for (const c of contacts) {
		if (!c.company_id) continue;
		linked.set(c.company_id, {
			id: c.company_id,
			name: c.company_name,
			kind: "direct",
			tracked: false,
			open_jobs: [],
			application_ids: [],
		});
	}
	return [tracked, ...linked.values()];
}

export function createContact(input: ContactInput): Contact {
	const contact: Contact = {
		id: `contact-${nextId++}`,
		...input,
		company_kind: "",
		job_title: "",
		application_id: null,
		stage: "to_contact",
		follow_up_by: null,
		reply: "",
		sent_since_reply: 0,
		history: [event("added", { job_id: input.job_id })],
		created_at: new Date().toISOString(),
	};
	contacts = [...contacts, contact];
	return contact;
}

export function markContactSent(id: string, input: MarkSentInput): ContactSent {
	const contact = replace(id, (c) => {
		const sent = sentSinceReply(c.history);
		const history = [
			...c.history,
			event("sent", {
				channel: input.channel,
				template_name: input.template_name,
				job_id: c.job_id,
				...(input.template_kind ? { template_kind: input.template_kind } : {}),
			}),
		];
		if (sent + 1 >= 3) {
			history.push(event("closed", { outcome: "no_reply" }));
			return {
				...c,
				history,
				stage: "closed",
				outcome: "no_reply",
				follow_up_by: null,
				sent_since_reply: sent + 1,
			};
		}
		const { outcome: _outcome, ...rest } = c;
		return {
			...rest,
			history,
			stage: "messaged",
			follow_up_by: inDays(sent === 0 ? 5 : 7),
			sent_since_reply: sent + 1,
		};
	});
	return { contact, moved_application_chase: false };
}

export function recordContactReply(
	id: string,
	input: RecordReplyInput,
): Contact {
	return replace(id, (c) => ({
		...c,
		history: [...c.history, event("reply", { outcome: input.outcome })],
		stage: input.outcome === "no_thanks" ? "closed" : "replied",
		outcome: input.outcome,
		follow_up_by: input.outcome === "not_now" ? inDays(90) : null,
		reply: input.reply || c.reply,
		sent_since_reply: 0,
	}));
}

export function deleteContact(id: string): void {
	contacts = contacts.filter((c) => c.id !== id);
}
