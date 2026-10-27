import assert from "node:assert/strict";
import type { ApplicationWithDetails } from "../types/application";
import type { ApplicationStatus } from "../types/applicationStatus";
import {
	completeApps,
	needsCv,
	nextSelection,
	runNeighbour,
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

assert.deepEqual(
	completeApps([
		app("a", "s1"),
		app("b", "s1", { Outcome: "kept", Status: "ready" } as never),
	]).map((a) => a.ID),
	["b"],
);

assert.equal(runNeighbour(["a", "b", "c"], "b", 1), "c");
assert.equal(runNeighbour(["a", "b", "c"], "b", -1), "a");
assert.equal(runNeighbour(["a", "b", "c"], "c", 1), undefined);
assert.equal(runNeighbour(["a", "b", "c"], "a", -1), undefined);
assert.equal(runNeighbour(["a", "b", "c"], "x", 1), undefined);
