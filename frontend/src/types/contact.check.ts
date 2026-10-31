import assert from "node:assert/strict";
import { type Contact, relationshipStatus } from "./contact";

const today = "2026-10-10";
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
const c = (change: Partial<Contact>): Contact => ({ ...base, ...change });

const interested = c({ stage: "replied", outcome: "interested" });
const referred = c({ stage: "replied", outcome: "referred" });
const overdue = c({ stage: "messaged", follow_up_by: "2026-10-10" });
const waiting = c({ stage: "messaged", follow_up_by: "2026-10-12" });
const parked = c({
	stage: "replied",
	outcome: "not_now",
	follow_up_by: "2027-01-01",
});
const toContact = c({});
const closed = c({ stage: "closed", outcome: "no_reply" });

assert.equal(relationshipStatus([], today), "no_one_yet");
assert.equal(relationshipStatus([interested], today), "in_conversation");
assert.equal(relationshipStatus([referred], today), "in_conversation");
assert.equal(relationshipStatus([overdue], today), "chase_due");
assert.equal(relationshipStatus([waiting], today), "waiting");
assert.equal(relationshipStatus([parked], today), "parked");
assert.equal(relationshipStatus([toContact], today), "not_started");
assert.equal(relationshipStatus([closed], today), "gone_cold");
assert.equal(relationshipStatus([closed, closed], today), "gone_cold");

assert.equal(
	relationshipStatus([overdue, interested, toContact], today),
	"in_conversation",
);
assert.equal(
	relationshipStatus([waiting, parked, toContact, overdue], today),
	"chase_due",
);
assert.equal(
	relationshipStatus([parked, toContact, waiting], today),
	"waiting",
);
assert.equal(relationshipStatus([toContact, parked], today), "parked");
assert.equal(relationshipStatus([toContact, closed], today), "not_started");
assert.equal(
	relationshipStatus([c({ ...parked, follow_up_by: "2026-10-09" })], today),
	"chase_due",
);
assert.equal(
	relationshipStatus([c({ ...closed, follow_up_by: "2026-10-01" })], today),
	"gone_cold",
);
