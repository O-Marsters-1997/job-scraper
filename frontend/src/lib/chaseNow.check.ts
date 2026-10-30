import assert from "node:assert/strict";
import type { ApplicationWithDetails } from "@/types/application";
import type { Contact } from "@/types/contact";
import { chaseNowRows } from "./chaseNow";

const contact = (over: Partial<Contact>): Contact => ({
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
	stage: "messaged",
	follow_up_by: "2026-10-01",
	reply: "",
	sent_since_reply: 0,
	history: [],
	created_at: "2026-10-01T00:00:00Z",
	...over,
});
const app = (
	over: Partial<ApplicationWithDetails>,
): ApplicationWithDetails => ({
	ID: "a",
	UserID: "",
	JobID: "",
	StatusID: "",
	AppliedAt: null,
	SalaryInfo: "",
	CreatedAt: "",
	UpdatedAt: "",
	ChaseBy: "2026-10-01T00:00:00Z",
	JobTitle: "Dev",
	JobCompanySlug: "acme-ltd",
	JobLocation: "",
	JobURL: "",
	JobClosedAt: null,
	StatusName: "Applied",
	StatusColour: "",
	Draft: null,
	...over,
});
const now = new Date(2026, 9, 15);

const rows = chaseNowRows(
	[
		contact({ id: "due", application_id: "a" }),
		contact({ id: "later", follow_up_by: "2026-11-01" }),
		contact({ id: "none", follow_up_by: null }),
	],
	[app({ ID: "a" }), app({ ID: "b" })],
	now,
);
assert.deepEqual(
	rows.map((r) => (r.kind === "contact" ? r.contact.id : r.app.ID)),
	["due", "b"],
);
assert.equal(
	rows[0]?.kind === "contact" && rows[0].applicationStatus,
	"Applied",
);

const via = chaseNowRows(
	[
		contact({
			id: "y",
			follow_up_by: null,
			company_name: "Acme Ltd",
			stage: "closed",
		}),
		contact({ id: "x", follow_up_by: null, company_name: "Acme Ltd" }),
	],
	[app({})],
	now,
)[0];
assert.equal(via?.kind === "application" && via.via?.id, "x");
