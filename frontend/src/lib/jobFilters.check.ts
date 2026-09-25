import {
	activeFilterCount,
	applyJobFilters,
	DEFAULT_FILTERS,
	filterCompanyJobs,
	normalizeArrangement,
	parseSalary,
} from "./jobFilters";

// ponytail: inline assert so tsgo doesn't need @types/node
function ok(cond: boolean, msg?: string): void {
	if (!cond) throw new Error(msg ?? "assertion failed");
}
function eq(a: unknown, b: unknown): void {
	if (JSON.stringify(a) !== JSON.stringify(b))
		throw new Error(
			`expected ${JSON.stringify(a)} to equal ${JSON.stringify(b)}`,
		);
}

eq(parseSalary("£80,000 to £95,000"), { min: 80000, max: 95000 });
eq(parseSalary("£80k - £95k"), { min: 80000, max: 95000 });
eq(parseSalary("Up to £90k"), { min: 90000, max: 90000 });
eq(parseSalary("$120,000"), { min: 120000, max: 120000 });
eq(parseSalary("€50k–€70k"), { min: 50000, max: 70000 });
ok(parseSalary("") === null);
ok(parseSalary(null) === null);
ok(parseSalary("competitive salary") === null);

ok(normalizeArrangement({ WorkArrangement: "remote" } as never) === "remote");
ok(normalizeArrangement({ WorkArrangement: "hybrid" } as never) === "hybrid");
ok(normalizeArrangement({ WorkArrangement: "onsite" } as never) === "onsite");
ok(normalizeArrangement({ DaysInOffice: 0 } as never) === "remote");
ok(normalizeArrangement({ DaysInOffice: 3 } as never) === "hybrid");
ok(normalizeArrangement({ DaysInOffice: 5 } as never) === "onsite");
ok(normalizeArrangement({} as never) === "unknown");

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

ok(applyJobFilters(jobs, DEFAULT_FILTERS).length === 3);

{
	const r = applyJobFilters(jobs, { ...DEFAULT_FILTERS, suit: 75 });
	ok(r.length === 1);
	ok((r[0] as { ID: string }).ID === "1");
}

{
	const r = applyJobFilters(jobs, {
		...DEFAULT_FILTERS,
		src: ["LinkedIn", "Lever"],
	});
	ok(r.length === 2);
}

{
	const r = applyJobFilters(jobs, {
		...DEFAULT_FILTERS,
		work: ["remote", "hybrid"],
	});
	ok(r.length === 2);
}

{
	const r = applyJobFilters(jobs, {
		...DEFAULT_FILTERS,
		sal: true,
		salMin: 70000,
		salMax: 90000,
	});
	ok(r.length === 1);
	ok((r[0] as { ID: string }).ID === "1");
}

ok(activeFilterCount(DEFAULT_FILTERS) === 0);
ok(
	activeFilterCount({
		...DEFAULT_FILTERS,
		suit: 50,
		src: ["LinkedIn"],
		sal: true,
	}) === 3,
);

eq(
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

console.log("✓ jobFilters checks passed");
