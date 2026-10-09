import { faker } from "@faker-js/faker";
import { addDays, format } from "date-fns";
import type { Application } from "@/types/application";
import type { ApplicationStatus } from "@/types/applicationStatus";
import type { Company, CompanyBoard } from "@/types/company";
import type { CV } from "@/types/cv";
import type { Band, Job } from "@/types/job";
import type { SourceTarget } from "@/types/sourceTarget";
import { slugify } from "./helpers";
import {
	APP_DISTRIBUTION,
	ATS_SOURCES,
	COMPANIES,
	COMPANY_SIZES,
	deriveExperienceLevel,
	EMPLOYMENT_TYPES,
	JOB_DESCRIPTIONS,
	JOB_TITLES,
	LOCATIONS,
	mockBreakdown,
	SALARY_RANGES,
	SKILL_SETS,
	SOURCES,
	TEAM_NAMES,
} from "./seedData";

faker.seed(1234);

const STATUS_DEFINITIONS = [
	{ name: "Saved", colour: "#64748b" },
	{ name: "Applied", colour: "#2563eb" },
	{ name: "Phone Screen", colour: "#7c3aed" },
	{ name: "Interview", colour: "#d97706" },
	{ name: "Offer", colour: "#059669" },
	{ name: "Rejected", colour: "#dc2626" },
];

const statuses: ApplicationStatus[] = STATUS_DEFINITIONS.map((s, i) => ({
	ID: `status-${i + 1}`,
	UserID: "user-1",
	Name: s.name,
	Colour: s.colour,
	ReplyWindowDays: null,
	CreatedAt: new Date("2024-01-01").toISOString(),
}));

function mockBand(score: number): Band {
	if (score >= 80) return "great";
	if (score >= 65) return "good";
	return score >= 45 ? "fair" : "poor";
}

const jobs: Job[] = Array.from({ length: 248 }, (_, i) => {
	const company = faker.helpers.arrayElement(COMPANIES);
	const location = faker.helpers.arrayElement(LOCATIONS);
	const scrapedAt = faker.date.recent({ days: 30 }).toISOString();
	const daysInOffice: number | null =
		location === "Remote"
			? 0
			: faker.number.int({ min: 1, max: 10 }) <= 8
				? faker.number.int({ min: 1, max: 5 })
				: null;
	const title = faker.helpers.arrayElement(JOB_TITLES);
	const scored = i % 5 !== 0;
	const score = scored ? faker.number.int({ min: 30, max: 100 }) : null;
	return {
		ID: faker.string.uuid(),
		Title: title,
		Location: location,
		URL: faker.internet.url(),
		CompanySlug: slugify(company),
		Source: faker.helpers.arrayElement(SOURCES),
		UpdatedAt: scrapedAt,
		ScrapedAt: scrapedAt,
		FirstDiscoveredAt: scrapedAt,
		DaysInOffice: daysInOffice,
		SuitabilityScore: score,
		Band: score == null ? "" : mockBand(score),
		Breakdown: scored ? mockBreakdown(i) : null,
		Description: JOB_DESCRIPTIONS[i % JOB_DESCRIPTIONS.length]!,
		Skills: SKILL_SETS[i % SKILL_SETS.length]!,
		EmploymentType: EMPLOYMENT_TYPES[i % EMPLOYMENT_TYPES.length]!,
		ExperienceLevel: deriveExperienceLevel(title),
		TeamName: TEAM_NAMES[i % TEAM_NAMES.length]!,
		CompanySize: COMPANY_SIZES[i % COMPANY_SIZES.length]!,
		SalaryRange: SALARY_RANGES[i % SALARY_RANGES.length]!,
	};
});

for (const [i, job] of jobs.entries()) {
	const primary = {
		source: job.Source.toLowerCase(),
		url: job.URL,
		first_seen_at: job.ScrapedAt,
	};
	if (i % 3 !== 0) {
		job.Listings = [primary];
		continue;
	}
	const secondary =
		primary.source === "linkedin"
			? {
					source: "wis",
					url: `https://workinstartups.com/details/${5900000000 + i}`,
				}
			: {
					source: "linkedin",
					url: `https://www.linkedin.com/jobs/view/${4470000000 + i}`,
				};
	job.Listings = [primary, { ...secondary, first_seen_at: job.ScrapedAt }];
}

jobs.sort(
	(a, b) => new Date(b.ScrapedAt).getTime() - new Date(a.ScrapedAt).getTime(),
);

const companies: Company[] = Array.from(new Set(COMPANIES)).map((name, i) => {
	const slug = slugify(name);
	const hasBoard = i % 2 === 0;
	const isNew = hasBoard && i % 8 === 2;
	const tracked = (hasBoard && i % 4 === 0) || isNew;
	return {
		ID: `company-${i + 1}`,
		Slug: slug,
		Name: name,
		ATSSource: hasBoard ? ATS_SOURCES[i % ATS_SOURCES.length]! : "",
		ATSToken: hasBoard ? slug : "",
		FirstSeenAt: faker.date.past({ years: 1 }).toISOString(),
		JobCount: jobs.filter((j) => j.CompanySlug === slug).length,
		Tracked: tracked,
		ReviewState: isNew ? "new" : tracked ? "kept" : "",
		TargetID: tracked ? `target-${i + 1}` : "",
		CheckIntervalMinutes: 360,
		LastCheckedAt: tracked
			? faker.date.recent({ days: 2 }).toISOString()
			: null,
	};
});

const jobsWithCompanyIDs: Job[] = jobs.map((j) => ({
	...j,
	CompanyID: companies.find((c) => c.Slug === j.CompanySlug)?.ID ?? j.CompanyID,
}));

const companyBoards: CompanyBoard[] = companies
	.filter((company) => company.ATSSource)
	.map((company) => ({
		ID: faker.string.uuid(),
		CompanyID: company.ID,
		Source: company.ATSSource,
		BoardToken: company.ATSToken,
		Status: "verified" as const,
		VerificationMethod: "legacy_import",
		VerifiedAt: company.FirstSeenAt,
		LastLinkedAt: null,
		RetiredAt: null,
		LastCompletedAt: company.LastCheckedAt,
		CreatedAt: company.FirstSeenAt,
	}));

const applications: Application[] = [];
const closedJobIDs = new Set<string>();
let appJobCursor = 0;
let appCursor = 0;

for (const { statusIndex, count } of APP_DISTRIBUTION) {
	const status = statuses[statusIndex]!;
	for (let c = 0; c < count; c++) {
		const job = jobs[appJobCursor++]!;
		const isSaved = statusIndex === 0;
		const appliedAt = isSaved
			? null
			: (faker.date.recent({ days: 20 }).toISOString().split("T")[0] ?? null);
		const chaseOffsetDays = isSaved ? null : (appCursor % 4) - 2;
		if (appCursor % 5 === 0) closedJobIDs.add(job.ID);
		appCursor++;
		applications.push({
			ID: faker.string.uuid(),
			UserID: "user-1",
			JobID: job.ID,
			StatusID: status.ID,
			Notes: "",
			AppliedAt: appliedAt,
			SalaryInfo: "",
			CreatedAt: faker.date.recent({ days: 25 }).toISOString(),
			UpdatedAt: faker.date.recent({ days: 10 }).toISOString(),
			ChaseBy:
				chaseOffsetDays === null
					? null
					: `${format(addDays(new Date(), chaseOffsetDays), "yyyy-MM-dd")}T00:00:00Z`,
		});
	}
}

const discoverySourceTargets: SourceTarget[] = [
	{
		ID: "target-wis-1",
		UserID: "user-1",
		Source: "wis",
		Value: "engineer",
		Enabled: true,
		Filters: { loc: "86383" },
		RunStatus: "succeeded",
		LastRunAt: faker.date.recent({ days: 1 }).toISOString(),
		LastRunError: "",
		DisabledReason: "",
		URL: "",
	},
	{
		ID: "target-linkedin-1",
		UserID: "user-1",
		Source: "linkedin",
		Value: "software engineer",
		Enabled: true,
		Filters: { location: "London" },
		RunStatus: "idle",
		LastRunAt: null,
		LastRunError: "",
		DisabledReason: "",
		URL: "",
	},
];

const cvs: CV[] = [
	{
		DocID: "doc-1",
		TabID: "t.0",
		Title: "Senior Backend Engineer CV",
		SourceDoc: "Master CV",
		ModifiedAt: faker.date.recent({ days: 5 }).toISOString(),
		DocURL: "https://docs.google.com/document/d/doc-1/edit",
		Visible: true,
	},
	{
		DocID: "doc-1",
		TabID: "t.1",
		Title: "Platform Engineer CV",
		SourceDoc: "Master CV",
		ModifiedAt: faker.date.recent({ days: 10 }).toISOString(),
		DocURL: "https://docs.google.com/document/d/doc-1/edit",
		Visible: true,
	},
	{
		DocID: "doc-2",
		TabID: "t.0",
		Title: "Cover Letter Template",
		SourceDoc: "Cover Letters",
		ModifiedAt: faker.date.recent({ days: 20 }).toISOString(),
		DocURL: "https://docs.google.com/document/d/doc-2/edit",
		Visible: false,
	},
];

export const seed = {
	statuses,
	jobs: jobsWithCompanyIDs,
	companies,
	companyBoards,
	applications,
	closedJobIDs,
	discoverySourceTargets,
	cvs,
};
