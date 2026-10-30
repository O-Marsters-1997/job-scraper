import assert from "node:assert/strict";
import type { Contact, ContactCompany } from "@/types/contact";
import { combinedHistory, companyRows, filterRows } from "./contactCompanies";

const today = "2026-10-10";
const person = (id: string, change: Partial<Contact>): Contact => ({
	id,
	name: `Person ${id}`,
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
	...change,
});
const company = (id: string, kind: string): ContactCompany => ({
	id,
	name: id,
	kind,
	tracked: true,
	open_jobs: [],
	application_ids: [],
});

const rows = companyRows(
	[
		company("acme", "direct"),
		company("bolt", "agency"),
		company("cast", "direct"),
	],
	[
		person("1", {
			company_id: "bolt",
			stage: "messaged",
			follow_up_by: "2026-10-09",
		}),
		person("2", {
			company_id: "cast",
			stage: "replied",
			outcome: "interested",
		}),
		person("3", { company_name: "Typed Co" }),
		person("4", { company_name: " typed co " }),
		person("5", {}),
	],
	today,
);

assert.deepEqual(
	rows.map((r) => [r.name, r.status]),
	[
		["bolt", "chase_due"],
		["cast", "in_conversation"],
		["acme", "no_one_yet"],
		["Typed Co", "not_started"],
	],
);
assert.equal(rows.at(-1)?.people.length, 2);
assert.deepEqual(
	filterRows(rows, "agency").map((r) => r.name),
	["bolt"],
);
assert.deepEqual(
	filterRows(rows, "employer").map((r) => r.name),
	["cast", "acme", "Typed Co"],
);

const history = combinedHistory([
	person("1", {
		history: [
			{ id: "a", kind: "added", occurred_on: "2026-10-01", job_id: null },
		],
	}),
	person("2", {
		history: [
			{ id: "b", kind: "sent", occurred_on: "2026-10-05", job_id: null },
		],
	}),
]);
const touched = companyRows(
	[company("dune", "direct")],
	[
		person("6", {
			company_id: "dune",
			history: [
				{ id: "x", kind: "added", occurred_on: "2026-10-02", job_id: null },
				{ id: "y", kind: "sent", occurred_on: "2026-10-07", job_id: null },
			],
		}),
	],
	today,
);
assert.equal(touched[0]?.lastTouch, "2026-10-07");
assert.deepEqual(
	history.map((e) => [e.id, e.who]),
	[
		["b", "Person"],
		["a", "Person"],
	],
);
