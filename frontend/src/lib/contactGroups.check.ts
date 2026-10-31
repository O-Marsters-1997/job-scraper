import assert from "node:assert/strict";
import type { Contact } from "@/types/contact";
import { contactGroup, groupContacts } from "./contactGroups";

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
	created_at: "2026-10-01T00:00:00Z",
};

assert.equal(contactGroup(base), "to_contact");
assert.equal(contactGroup({ ...base, stage: "messaged" }), "waiting");
assert.equal(contactGroup({ ...base, stage: "closed" }), "closed");
assert.equal(
	contactGroup({ ...base, stage: "replied", outcome: "interested" }),
	"in_conversation",
);
assert.equal(
	contactGroup({ ...base, stage: "replied", outcome: "referred" }),
	"in_conversation",
);
assert.equal(
	contactGroup({ ...base, stage: "replied", outcome: "not_now" }),
	"revisit",
);
assert.equal(
	groupContacts([base, { ...base, id: "d", stage: "messaged" }]).waiting.length,
	1,
);
