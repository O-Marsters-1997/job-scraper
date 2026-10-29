import assert from "node:assert/strict";
import { matchCarried } from "./carriedMatch";

const isCarried = (id: string) => id.startsWith("carried:");
const texts: Record<string, string> = {
	"carried:p:0": "  Cut  costs by 10% ",
	"carried:p:1": "Hand written",
	"carried:p:2": "Cut costs by 10%",
};
const textOf = (id: string) => texts[id] ?? "";
const bank = [{ achievementId: "a1", text: "Cut costs by 10%" }];

assert.deepEqual(
	matchCarried(["carried:p:0", "carried:p:1"], isCarried, textOf, bank),
	["a1", "carried:p:1"],
);
assert.deepEqual(
	matchCarried(["carried:p:0", "carried:p:2"], isCarried, textOf, bank),
	["a1", "carried:p:2"],
);
assert.deepEqual(matchCarried(["a1", "carried:p:0"], isCarried, textOf, bank), [
	"a1",
	"carried:p:0",
]);
