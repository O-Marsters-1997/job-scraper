import assert from "node:assert/strict";
import { assignStatusKeys } from "./statusKeys";

const named = (...names: string[]) => names.map((Name) => ({ ID: Name, Name }));

assert.deepEqual(
	assignStatusKeys(
		named("Draft", "Applied", "Phone Screen", "Interview", "Offer", "Rejected"),
	),
	{
		Draft: "d",
		Applied: "a",
		"Phone Screen": "p",
		Interview: "i",
		Offer: "o",
		Rejected: "r",
	},
);

assert.deepEqual(assignStatusKeys(named("Hired", "Kept")), {
	Hired: "i",
	Kept: "e",
});
assert.deepEqual(assignStatusKeys(named("Draft", "Dead")), {
	Draft: "d",
	Dead: "e",
});
assert.deepEqual(assignStatusKeys(named("Hk", "Jl", "Dd", "D")), {
	Dd: "d",
});
