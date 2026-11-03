import assert from "node:assert/strict";
import type { Contact } from "@/types/contact";
import { contactSections } from "./contactStatus";

const base: Contact = {
	id: "c",
	name: "Sam",
	role: "",
	company_id: null,
	company_name: "",
	company_kind: "",
	job_id: null,
	job_title: "",
	application_id: null,
	linkedin_url: "",
	email: "",
	stage: "to_contact",
	follow_up_by: null,
	reply: "",
	sent_since_reply: 0,
	history: [],
	reach_outs: [],
	created_at: "2026-10-01T00:00:00Z",
};
const sent = (day: string) => ({
	id: day,
	kind: "sent" as const,
	occurred_on: day,
	job_id: null,
});
const open = { task_id: "t", job_id: "j", job_title: "Dev", done_at: null };

const toContact = {
	...base,
	id: "a",
	stage: "messaged" as const,
	reach_outs: [open],
};
const oldWait = {
	...base,
	id: "b",
	stage: "messaged" as const,
	history: [sent("2026-10-02")],
};
const newWait = {
	...base,
	id: "c",
	stage: "messaged" as const,
	history: [sent("2026-10-09")],
};
const olderRest = {
	...base,
	id: "d",
	stage: "closed" as const,
	created_at: "2026-09-01T00:00:00Z",
};
const newerRest = {
	...base,
	id: "e",
	stage: "replied" as const,
	history: [sent("2026-10-12")],
};

const s = contactSections([newWait, olderRest, toContact, newerRest, oldWait]);
assert.deepEqual(
	s.toContact.map((c) => c.id),
	["a"],
);
assert.deepEqual(
	s.waiting.map((c) => c.id),
	["b", "c"],
);
assert.deepEqual(
	s.rest.map((c) => c.id),
	["e", "d"],
);
