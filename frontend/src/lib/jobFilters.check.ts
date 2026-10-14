import assert from "node:assert/strict";
import {
	activeFilterCount,
	DEFAULT_FILTERS,
	filterCompanyJobs,
	parseSearch,
} from "./jobFilters";

assert.equal(parseSearch({}).graded, "all");
assert.equal(parseSearch({ graded: "ungraded" }).graded, "ungraded");
assert.equal(parseSearch({ graded: "nonsense" }).graded, "all");
assert.equal(activeFilterCount({ ...DEFAULT_FILTERS, graded: "ungraded" }), 1);

assert.ok(activeFilterCount(DEFAULT_FILTERS) === 0);
assert.ok(
	activeFilterCount({
		...DEFAULT_FILTERS,
		suit: 50,
		src: ["LinkedIn"],
		sal: true,
	}) === 3,
);

assert.equal(parseSearch({}).scored, true, "Jobs page defaults scored on");
assert.equal(parseSearch({ scored: false }).scored, false);
assert.equal(parseSearch({ scored: "1" }).scored, true);

assert.equal(parseSearch({}).sort, "relevant");
assert.equal(parseSearch({ sort: "newest" }).sort, "newest");
assert.equal(parseSearch({ sort: "best" }).sort, "best");
assert.equal(parseSearch({ sort: "bogus" }).sort, "relevant");

assert.equal(parseSearch({}).fav, false);
assert.equal(parseSearch({ fav: "1" }).fav, true);
assert.equal(parseSearch({ fav: true }).fav, true);
assert.equal(parseSearch({ fav: "nonsense" }).fav, false);
assert.equal(activeFilterCount({ ...DEFAULT_FILTERS, fav: true }), 1);

assert.equal(parseSearch({}).poster, "all");
assert.equal(parseSearch({ poster: "agency" }).poster, "agency");
assert.equal(parseSearch({ poster: "nonsense" }).poster, "all");
assert.equal(activeFilterCount({ ...DEFAULT_FILTERS, poster: "agency" }), 1);

assert.equal(parseSearch({ page: "3" }).page, 3);
assert.equal(parseSearch({ page: 3 }).page, 3);
assert.equal(parseSearch({ page: "1" }).page, undefined);
assert.equal(parseSearch({ page: "0" }).page, undefined);
assert.equal(parseSearch({ page: "2.5" }).page, undefined);
assert.equal(parseSearch({ page: "abc" }).page, undefined);
assert.equal(parseSearch({}).page, undefined);
assert.equal(parseSearch({ page: "1e3" }).page, undefined);
assert.equal(parseSearch({ page: "0x10" }).page, undefined);

console.log("✓ jobFilters checks passed");

assert.deepEqual(
	filterCompanyJobs(
		[
			{ ID: "1", CompanyID: "company-a", CompanySlug: "other" },
			{ ID: "2", CompanySlug: "acme" },
			{ ID: "3", CompanyID: "company-b", CompanySlug: "acme" },
		] as never[],
		{ ID: "company-a", Slug: "acme" },
	).map((job) => job.ID),
	["1", "2"],
);
