import assert from "node:assert/strict";
import type { ApplicationWithDetails } from "../types/application";
import type { ApplicationStatus } from "../types/applicationStatus";
import {
	needsCv,
	nextSelection,
	selectedDraftApps,
} from "./applicationSelection";

const ids = ["a", "b", "c", "d"];

assert.deepEqual(nextSelection(ids, {}, undefined, "b", false), { b: true });
assert.deepEqual(nextSelection(ids, { b: true }, "b", "b", false), {});
assert.deepEqual(nextSelection(ids, { a: true }, "a", "c", true), {
	a: true,
	b: true,
	c: true,
});
assert.deepEqual(nextSelection(ids, { d: true }, "d", "b", true), {
	b: true,
	c: true,
	d: true,
});

const app = (
	ID: string,
	StatusID: string,
	Draft: ApplicationWithDetails["Draft"] = null,
) => ({ ID, StatusID, Draft }) as ApplicationWithDetails;
const statuses = [
	{ ID: "s1", SystemKey: "draft" },
	{ ID: "s2", SystemKey: "applied" },
] as ApplicationStatus[];

assert.deepEqual(
	selectedDraftApps(
		[app("a", "s1"), app("b", "s2"), app("c", "s1")],
		{ a: true, b: true },
		statuses,
	).map((a) => a.ID),
	["a"],
);

assert.equal(needsCv(app("a", "s1")), true);
assert.equal(
	needsCv(
		app("a", "s1", { ID: "d", Status: "failed", Outcome: null, LastError: "" }),
	),
	true,
);
assert.equal(
	needsCv(
		app("a", "s1", {
			ID: "d",
			Status: "running",
			Outcome: null,
			LastError: "",
		}),
	),
	false,
);
