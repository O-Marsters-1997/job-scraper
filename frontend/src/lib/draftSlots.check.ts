import assert from "node:assert/strict";
import { slotsToSave } from "./draftSlots";

{
	const got = slotsToSave(
		["s0", "new-a", "new-b", "s1"],
		{ s0: "one", "new-a": "added", "new-b": "  ", s1: "two" },
		{},
		[],
	);
	assert.deepEqual(got, [
		{ slotId: "s0", text: "one" },
		{ slotId: "new-a", text: "added", after: "s0" },
		{ slotId: "s1", text: "two" },
	]);
}

{
	const got = slotsToSave(
		["s0", "new-a", "new-b"],
		{ s0: "one", "new-a": "", "new-b": "tail" },
		{ "new-a": "was here", "new-gone": "x" },
		["new-gone", "new-never-saved"],
	);
	assert.deepEqual(got, [
		{ slotId: "s0", text: "one" },
		{ slotId: "new-a", text: "", remove: true },
		{ slotId: "new-b", text: "tail", after: "s0" },
		{ slotId: "new-gone", text: "", remove: true },
	]);
}
