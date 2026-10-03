import assert from "node:assert/strict";
import {
	activeFilterCount,
	applyJobFilters,
	DEFAULT_FILTERS,
	filterCompanyJobs,
	isDefaultView,
	normalizeArrangement,
	parseSalary,
	parseSearch,
} from "./jobFilters";

// ponytail: inline assert so tsgo doesn't need @types/node
assert.deepEqual(parseSalary("£80,000 to £95,000"), { min: 80000, max: 95000 });
assert.deepEqual(parseSalary("£80k - £95k"), { min: 80000, max: 95000 });
assert.deepEqual(parseSalary("Up to £90k"), { min: 90000, max: 90000 });
assert.deepEqual(parseSalary("$120,000"), { min: 120000, max: 120000 });
assert.deepEqual(parseSalary("€50k–€70k"), { min: 50000, max: 70000 });
assert.ok(parseSalary("") === null);
assert.ok(parseSalary(null) === null);
assert.ok(parseSalary("competitive salary") === null);

assert.ok(
	normalizeArrangement({ WorkArrangement: "remote" } as never) === "remote",
);
assert.ok(
	normalizeArrangement({ WorkArrangement: "hybrid" } as never) === "hybrid",
);
assert.ok(
	normalizeArrangement({ WorkArrangement: "onsite" } as never) === "onsite",
);
assert.ok(normalizeArrangement({ DaysInOffice: 0 } as never) === "remote");
assert.ok(normalizeArrangement({ DaysInOffice: 3 } as never) === "hybrid");
assert.ok(normalizeArrangement({ DaysInOffice: 5 } as never) === "onsite");
assert.ok(normalizeArrangement({} as never) === "unknown");

const base = {
	ID: "1",
	URL: "",
	UpdatedAt: "",
	ScrapedAt: "",
	Location: "London",
	CompanySlug: "acme",
};

const jobs = [
	{
		...base,
		ID: "1",
		Title: "Frontend Engineer",
		Source: "LinkedIn",
		SuitabilityScore: 80,
		WorkArrangement: "remote",
		SalaryRaw: "£80k to £100k",
	},
	{
		...base,
		ID: "2",
		Title: "Backend Engineer",
		Source: "Greenhouse",
		SuitabilityScore: null,
		WorkArrangement: "hybrid",
		SalaryRaw: "£60,000",
	},
	{
		...base,
		ID: "3",
		Title: "Product Manager",
		Source: "Lever",
		SuitabilityScore: 50,
		WorkArrangement: "",
		SalaryRaw: "",
	},
] as never[];

assert.ok(applyJobFilters(jobs, DEFAULT_FILTERS).length === 3);

{
	const r = applyJobFilters(jobs, { ...DEFAULT_FILTERS, suit: 75 });
	assert.ok(r.length === 1);
	assert.ok((r[0] as { ID: string }).ID === "1");
}

{
	const r = applyJobFilters(jobs, {
		...DEFAULT_FILTERS,
		src: ["LinkedIn", "Lever"],
	});
	assert.ok(r.length === 2);
}

{
	const r = applyJobFilters(jobs, {
		...DEFAULT_FILTERS,
		work: ["remote", "hybrid"],
	});
	assert.ok(r.length === 2);
}

{
	const r = applyJobFilters(jobs, {
		...DEFAULT_FILTERS,
		sal: true,
		salMin: 70000,
		salMax: 90000,
	});
	assert.ok(r.length === 1);
	assert.ok((r[0] as { ID: string }).ID === "1");
}

{
	const graded = (jobs as object[]).map((j, i) => ({
		...j,
		ID: String(i + 1),
		Grade: i === 0 ? "ok" : "",
	})) as never[];
	const ids = (graded_: unknown[]) =>
		graded_.map((j) => (j as { ID: string }).ID);
	assert.deepEqual(
		ids(applyJobFilters(graded, { ...DEFAULT_FILTERS, graded: "ungraded" })),
		["2", "3"],
	);
	assert.deepEqual(
		ids(applyJobFilters(graded, { ...DEFAULT_FILTERS, graded: "graded" })),
		["1"],
	);
	assert.equal(ids(applyJobFilters(graded, DEFAULT_FILTERS)).length, 3);
}

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

assert.equal(parseSearch({}).scored, true, "Jobs page defaults scored on");
assert.equal(parseSearch({ scored: false }).scored, false);
assert.equal(parseSearch({ scored: "1" }).scored, true);

assert.equal(isDefaultView(DEFAULT_FILTERS, false), true);
assert.equal(isDefaultView(DEFAULT_FILTERS, true), false);
assert.equal(isDefaultView({ ...DEFAULT_FILTERS, q: "x" }, false), false);
assert.equal(isDefaultView({ ...DEFAULT_FILTERS, company: "c" }, false), false);
assert.equal(isDefaultView({ ...DEFAULT_FILTERS, src: ["a"] }, false), false);

console.log("✓ jobFilters checks passed");
