import assert from "node:assert/strict";
import {
	excludeCompanyName,
	getScoringConfig,
	isCompanyExcluded,
	unexcludeCompanyName,
} from "../mocks/scoring";

assert.equal(excludeCompanyName("Acme Corp"), true, "first exclusion adds");
assert.equal(excludeCompanyName("acme corp"), false, "repeating adds nothing");
assert.deepEqual(getScoringConfig().excludedCompanies, ["acme corp"]);
assert.equal(isCompanyExcluded("acme-corp"), true, "matches by slug");
assert.equal(isCompanyExcluded("globex"), false);

unexcludeCompanyName("Acme Corp");
assert.equal(
	isCompanyExcluded("acme-corp"),
	false,
	"undo restores the company",
);
