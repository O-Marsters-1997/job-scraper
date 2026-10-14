import assert from "node:assert/strict";
import { companySearchPatch, parseCompanySearch } from "./companyFilters";

assert.deepEqual(parseCompanySearch({}), {
	q: "",
	favourite: false,
	noBoard: false,
	sort: "relevance",
	poster: undefined,
	page: undefined,
});

const parsed = parseCompanySearch({
	q: "acme",
	favourite: "true",
	noBoard: true,
	sort: "alphabetical",
	poster: "agency",
	page: "3",
});
assert.equal(parsed.q, "acme");
assert.equal(parsed.favourite, true);
assert.equal(parsed.noBoard, true);
assert.equal(parsed.sort, "alphabetical");
assert.equal(parsed.poster, "agency");
assert.equal(parsed.page, 3);

assert.equal(parseCompanySearch({ sort: "bogus" }).sort, "relevance");
assert.equal(parseCompanySearch({ poster: "x" }).poster, undefined);
assert.equal(parseCompanySearch({ page: "1" }).page, undefined);
assert.equal(parseCompanySearch({ page: "abc" }).page, undefined);

assert.deepEqual(
	companySearchPatch({
		q: "",
		favourite: false,
		sort: "relevance",
		page: undefined,
		noBoard: true,
	}),
	{
		q: undefined,
		favourite: undefined,
		sort: undefined,
		page: undefined,
		noBoard: true,
	},
	"default values drop out of the URL",
);

console.log("✓ companyFilters checks passed");
