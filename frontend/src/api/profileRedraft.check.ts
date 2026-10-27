import assert from "node:assert/strict";
import { setDemoData } from "../lib/demoData";
import {
	createDraft,
	fetchDraft,
	redraftProfile,
	saveDraftSlots,
} from "./tailoring";

setDemoData(true);
const { id } = await createDraft({
	jobId: "j",
	docId: "d",
	tabId: "t",
	mode: "generate",
});
let draft = await fetchDraft(id);
while (draft.status !== "ready") draft = await fetchDraft(id);
const base = draft.base?.profile;
assert.ok(base, "mock draft has a base Profile");

const redrafted = await redraftProfile(id, { steer: "lead with platform" });
assert.match(redrafted.content?.profile ?? "", /lead with platform/);

await assert.rejects(
	redraftProfile(id, { steer: "reject" }),
	/rejected by the guardrails/,
);
assert.match(
	(await fetchDraft(id)).content?.profile ?? "",
	/lead with platform/,
	"a rejected redraft leaves the Profile unchanged",
);

const reverted = await saveDraftSlots(id, [{ slotId: "profile", text: base }]);
assert.equal(reverted.content?.profile, base);

console.log("✓ profile redraft checks passed");
