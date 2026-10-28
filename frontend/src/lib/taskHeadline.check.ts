import assert from "node:assert/strict";
import type { Task } from "@/types/task";
import { taskHeadline } from "./taskHeadline";

const base: Task = {
	ID: "t1",
	TypeID: "ty1",
	TypeName: "Apply",
	TypeColour: "#6366f1",
	TypeSystemKey: "apply",
	JobID: null,
	JobTitle: "",
	JobCompanySlug: "",
	JobCompanyID: null,
	ApplicationID: null,
	JobPostedAt: null,
	Link: "",
	Notes: "",
	CompletedAt: null,
	CreatedAt: "2026-10-17T09:00:00Z",
	UpdatedAt: "2026-10-17T09:00:00Z",
};

assert.equal(
	taskHeadline({
		...base,
		JobID: "j1",
		JobTitle: "Engineer",
		Link: "https://a.dev",
	}),
	"Apply · Engineer",
);
assert.equal(
	taskHeadline({ ...base, Link: "https://jobs.acme.dev/x/1", Notes: "n" }),
	"Apply · jobs.acme.dev",
);
assert.equal(
	taskHeadline({ ...base, Notes: "call Sam\nthen email" }),
	"Apply · call Sam",
);
assert.equal(taskHeadline(base), "Apply");
assert.equal(
	taskHeadline({ ...base, JobID: "j1", JobTitle: "", Notes: "x" }),
	"Apply · x",
);
