import { faker } from "@faker-js/faker";
import type {
	Application,
	ApplicationWithDetails,
	JobApplicationSummary,
} from "@/types/application";
import type { ApplicationStatus } from "@/types/applicationStatus";
import type { Company, CompanyBoard } from "@/types/company";
import type { Job } from "@/types/job";
import type { SourceTarget } from "@/types/sourceTarget";

faker.seed(1234);

// ─── Statuses ─────────────────────────────────────────────────────────────────

const STATUS_DEFINITIONS = [
	{ name: "Saved", colour: "#64748b" },
	{ name: "Applied", colour: "#2563eb" },
	{ name: "Phone Screen", colour: "#7c3aed" },
	{ name: "Interview", colour: "#d97706" },
	{ name: "Offer", colour: "#059669" },
	{ name: "Rejected", colour: "#dc2626" },
];

let statuses: ApplicationStatus[] = STATUS_DEFINITIONS.map((s, i) => ({
	ID: `status-${i + 1}`,
	UserID: "user-1",
	Name: s.name,
	Colour: s.colour,
	CreatedAt: new Date("2024-01-01").toISOString(),
}));

// ─── Jobs ─────────────────────────────────────────────────────────────────────

const COMPANIES = [
	"Monzo",
	"Figma",
	"Revolut",
	"Stripe",
	"Deliveroo",
	"Wise",
	"Canva",
	"Shopify",
	"Notion",
	"Cloudflare",
	"Linear",
	"Netflix",
	"Vercel",
	"Supabase",
	"GitHub",
	"GitLab",
	"Atlassian",
	"HubSpot",
	"Intercom",
	"Zendesk",
	"Snowflake",
	"Databricks",
	"MongoDB",
	"Elastic",
	"Datadog",
	"HashiCorp",
	"Airbnb",
	"Spotify",
	"Twilio",
	"Slack",
	"PlanetScale",
	"Retool",
	"Contentful",
	"Figma",
];

const LOCATIONS = [
	"Remote",
	"Remote",
	"London, UK",
	"London, UK",
	"Dublin, IE",
	"Berlin, DE",
	"Amsterdam, NL",
	"New York, US",
	"San Francisco, US",
	"Toronto, CA",
];

// Weighted toward LinkedIn (4 out of 9 slots)
const SOURCES = [
	"LinkedIn",
	"LinkedIn",
	"LinkedIn",
	"LinkedIn",
	"Indeed",
	"Indeed",
	"Greenhouse",
	"Greenhouse",
	"Lever",
];

// ─── Optional rich-field pools (index-modular, no faker calls — seed preserved) ─

const JOB_DESCRIPTIONS = [
	`We're building the infrastructure that powers payments for millions of people. As a key member of our engineering team, you'll design, build, and scale distributed systems that handle real-time financial transactions at global scale.\n\nYou'll work closely with product, data, and design teams to ship features end-to-end — from architecture decisions to production monitoring. We operate a 'you build it, you run it' culture, so you'll own your services in production.\n\nWe're looking for engineers who care deeply about reliability, security, and developer experience. You'll be joining a team of 8 engineers embedded in a product squad, shipping roughly every two weeks.`,
	`Our mission is to make design accessible to everyone. You'll join a small, high-ownership team building the tools that millions of designers and developers use every day.\n\nThis role is fully remote-first. We invest heavily in async communication and documentation. You'll be expected to write clearly, work autonomously, and raise the bar for your team through code review, RFCs, and mentorship.\n\nWe ship frequently, measure impact rigorously, and give engineers real ownership over the systems they build.`,
	`We're reimagining how companies manage their finances. You'll work on core product features — from the first login to reconciliation workflows — used by finance teams across thousands of businesses.\n\nEngineering here means owning problems, not tickets. We expect you to talk to customers, influence the roadmap, and contribute to a culture of technical excellence.\n\nYour team is responsible for the full stack: API design, database modelling, and the frontend surfaces your users interact with every day.`,
	`Join a team that's rethinking how the internet is built. We run one of the world's largest networks, and your code will run in data centres across 300+ cities.\n\nYou'll build tooling, services, and systems that our entire engineering organisation depends on. We value people who can operate at multiple levels of abstraction — from TCP to product experience.\n\nThis is not a maintenance role. We're actively rebuilding core infrastructure and expect strong opinions about how things should work.`,
	`We believe great software is built by people who understand the problem deeply. You'll join a product-focused engineering team shipping features that help millions of professionals find the right opportunities and build meaningful careers.\n\nWe move fast but deliberately — we write RFCs for significant changes, run blameless post-mortems, and keep our on-call burden low through good design. You'll have time to think, not just to ship.`,
];

const SKILL_SETS = [
	["TypeScript", "React", "CSS", "Node.js", "PostgreSQL"],
	["Go", "PostgreSQL", "gRPC", "Docker", "Kubernetes"],
	["Python", "SQL", "Spark", "Airflow", "dbt"],
	["TypeScript", "Go", "PostgreSQL", "Docker", "CI/CD"],
	["Kotlin", "Java", "Spring Boot", "AWS", "Kafka"],
	["Swift", "SwiftUI", "iOS SDK", "Xcode", "Objective-C"],
	["Python", "PyTorch", "TensorFlow", "SQL", "MLflow"],
	["Terraform", "Kubernetes", "Prometheus", "AWS", "Grafana"],
];

const EMPLOYMENT_TYPES = ["Full-time", "Full-time", "Full-time", "Contract"];
const TEAM_NAMES = [
	"Platform",
	"Growth",
	"Infrastructure",
	"Product Engineering",
	"Data & Analytics",
	"Security",
	"Developer Experience",
	"Mobile",
	"Core Services",
];
const COMPANY_SIZES = [
	"50–200",
	"200–500",
	"500–2,000",
	"2,000–10,000",
	"10,000+",
];
const SALARY_RANGES = [
	"£60,000–£80,000",
	"£80,000–£110,000",
	"£110,000–£150,000",
	"$130,000–$170,000",
	"£70,000–£90,000",
	"Competitive + equity",
];

function deriveExperienceLevel(title: string): string {
	const t = title.toLowerCase();
	if (t.includes("principal") || t.includes("staff"))
		return "Principal / Staff";
	if (t.includes("vp") || t.includes("manager") || t.includes("lead"))
		return "Leadership";
	if (t.includes("senior")) return "Senior (5+ years)";
	return "Mid-level (2–5 years)";
}

const JOB_TITLES = [
	"Software Engineer",
	"Senior Software Engineer",
	"Staff Engineer",
	"Principal Engineer",
	"Frontend Engineer",
	"Senior Frontend Engineer",
	"Full Stack Engineer",
	"Senior Full Stack Engineer",
	"Backend Engineer",
	"Senior Backend Engineer",
	"Platform Engineer",
	"DevOps Engineer",
	"Site Reliability Engineer",
	"Infrastructure Engineer",
	"Data Engineer",
	"Senior Data Engineer",
	"Machine Learning Engineer",
	"Product Designer",
	"Senior Product Designer",
	"UX Engineer",
	"Engineering Manager",
	"Senior Engineering Manager",
	"VP of Engineering",
	"Product Manager",
	"Senior Product Manager",
	"Technical Program Manager",
	"iOS Engineer",
	"Android Engineer",
	"Mobile Engineer",
	"Security Engineer",
	"Staff Security Engineer",
	"QA Engineer",
	"Solutions Architect",
	"Data Scientist",
	"Senior Data Scientist",
	"Analytics Engineer",
	"Business Intelligence Engineer",
	"Software Development Engineer in Test",
];

function slugify(name: string): string {
	return name
		.toLowerCase()
		.replace(/\s+/g, "-")
		.replace(/[^a-z0-9-]/g, "");
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
	return {
		ID: faker.string.uuid(),
		Title: title,
		Location: location,
		URL: faker.internet.url(),
		CompanySlug: slugify(company),
		Source: faker.helpers.arrayElement(SOURCES),
		UpdatedAt: scrapedAt,
		ScrapedAt: scrapedAt,
		DaysInOffice: daysInOffice,
		RelevanceScore:
			i % 4 === 0 ? null : faker.number.int({ min: 40, max: 100 }),
		SuitabilityScore:
			i % 5 === 0 ? null : faker.number.int({ min: 30, max: 100 }),
		// Optional rich fields — index-modular, no faker (seed preserved)
		Description: JOB_DESCRIPTIONS[i % JOB_DESCRIPTIONS.length]!,
		Skills: SKILL_SETS[i % SKILL_SETS.length]!,
		EmploymentType: EMPLOYMENT_TYPES[i % EMPLOYMENT_TYPES.length]!,
		ExperienceLevel: deriveExperienceLevel(title),
		TeamName: TEAM_NAMES[i % TEAM_NAMES.length]!,
		CompanySize: COMPANY_SIZES[i % COMPANY_SIZES.length]!,
		SalaryRange: SALARY_RANGES[i % SALARY_RANGES.length]!,
	};
});

// Sort descending so most-recent jobs appear first
jobs.sort(
	(a, b) => new Date(b.ScrapedAt).getTime() - new Date(a.ScrapedAt).getTime(),
);

// ─── Companies ────────────────────────────────────────────────────────────────

const ATS_SOURCES = [
	"greenhouse",
	"lever",
	"ashby",
	"workable",
	"recruitee",
	"personio",
];

// ponytail: even index → has a known ATS board, every fourth of those is tracked
let companies: Company[] = Array.from(new Set(COMPANIES)).map((name, i) => {
	const slug = slugify(name);
	const hasBoard = i % 2 === 0;
	const tracked = hasBoard && i % 4 === 0;
	return {
		ID: `company-${i + 1}`,
		Slug: slug,
		Name: name,
		ATSSource: hasBoard ? ATS_SOURCES[i % ATS_SOURCES.length]! : "",
		ATSToken: hasBoard ? slug : "",
		FirstSeenAt: faker.date.past({ years: 1 }).toISOString(),
		JobCount: jobs.filter((j) => j.CompanySlug === slug).length,
		Tracked: tracked,
		TargetID: tracked ? `target-${i + 1}` : "",
		CheckIntervalMinutes: 360,
		LastCheckedAt: tracked
			? faker.date.recent({ days: 2 }).toISOString()
			: null,
	};
});

let companyBoards: CompanyBoard[] = companies
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
		CreatedAt: company.FirstSeenAt,
	}));

const ATS_HOSTS: Record<string, string> = {
	"greenhouse.io": "greenhouse",
	"lever.co": "lever",
	"ashbyhq.com": "ashby",
	"workable.com": "workable",
	"recruitee.com": "recruitee",
	"personio.de": "personio",
};

function humanizeSlug(slug: string): string {
	return slug
		.split(/[-_]/)
		.filter(Boolean)
		.map((w) => w[0]!.toUpperCase() + w.slice(1))
		.join(" ");
}

// ─── Applications ─────────────────────────────────────────────────────────────
//
// Distribution across 12 jobs gives 42 % response rate (5 of 12 heard back)
// and 7 awaiting response (Saved + Applied).

const APP_DISTRIBUTION: Array<{ statusIndex: number; count: number }> = [
	{ statusIndex: 0, count: 2 }, // Saved
	{ statusIndex: 1, count: 5 }, // Applied
	{ statusIndex: 2, count: 2 }, // Phone Screen
	{ statusIndex: 3, count: 1 }, // Interview
	{ statusIndex: 4, count: 1 }, // Offer
	{ statusIndex: 5, count: 1 }, // Rejected
];

let applications: Application[] = [];
let appJobCursor = 0;

for (const { statusIndex, count } of APP_DISTRIBUTION) {
	const status = statuses[statusIndex]!;
	for (let c = 0; c < count; c++) {
		const job = jobs[appJobCursor++]!;
		const isSaved = statusIndex === 0;
		const appliedAt = isSaved
			? null
			: (faker.date.recent({ days: 20 }).toISOString().split("T")[0] ?? null);
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
		});
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function buildWithDetails(app: Application): ApplicationWithDetails {
	const job = jobs.find((j) => j.ID === app.JobID);
	const status = statuses.find((s) => s.ID === app.StatusID);
	if (!job || !status) {
		throw new Error(
			`Mock db inconsistency: job=${app.JobID} status=${app.StatusID}`,
		);
	}
	return {
		...app,
		JobTitle: job.Title,
		JobCompanySlug: job.CompanySlug,
		JobLocation: job.Location,
		JobURL: job.URL,
		StatusName: status.Name,
		StatusColour: status.Colour,
	};
}

// ─── Read helpers ─────────────────────────────────────────────────────────────

export function getJobs(): Job[] {
	return jobs;
}

export function getStatuses(): ApplicationStatus[] {
	return statuses;
}

export function getCompanies(): Company[] {
	return companies;
}

export function getApplications(statusId?: string): ApplicationWithDetails[] {
	const list = statusId
		? applications.filter((a) => a.StatusID === statusId)
		: applications;
	return list.map(buildWithDetails);
}

export function getApplicationsForJobs(
	jobIds: string[],
): Record<string, JobApplicationSummary> {
	const result: Record<string, JobApplicationSummary> = {};
	for (const app of applications) {
		if (jobIds.includes(app.JobID)) {
			const status = statuses.find((s) => s.ID === app.StatusID);
			if (!status) continue;
			result[app.JobID] = {
				ApplicationID: app.ID,
				StatusID: app.StatusID,
				StatusName: status.Name,
				StatusColour: status.Colour,
			};
		}
	}
	return result;
}

// ─── Mutation helpers ─────────────────────────────────────────────────────────

export function createApplication(data: {
	job_id: string;
	status_id?: string | undefined;
	notes?: string | undefined;
	applied_at?: string | null | undefined;
	salary_info?: string | undefined;
}): Application {
	const status = data.status_id
		? (statuses.find((s) => s.ID === data.status_id) ?? statuses[0]!)
		: statuses[0]!;
	const app: Application = {
		ID: faker.string.uuid(),
		UserID: "user-1",
		JobID: data.job_id,
		StatusID: status.ID,
		Notes: data.notes ?? "",
		AppliedAt: data.applied_at ?? null,
		SalaryInfo: data.salary_info ?? "",
		CreatedAt: new Date().toISOString(),
		UpdatedAt: new Date().toISOString(),
	};
	applications = [...applications, app];
	return app;
}

export function updateApplication(
	id: string,
	data: {
		status_id?: string | undefined;
		notes?: string | undefined;
		applied_at?: string | null | undefined;
		salary_info?: string | undefined;
	},
): Application {
	const idx = applications.findIndex((a) => a.ID === id);
	if (idx === -1) throw new Error("Application not found");
	const prev = applications[idx]!;
	const updated: Application = {
		...prev,
		StatusID: data.status_id ?? prev.StatusID,
		Notes: data.notes ?? prev.Notes,
		AppliedAt: data.applied_at !== undefined ? data.applied_at : prev.AppliedAt,
		SalaryInfo: data.salary_info ?? prev.SalaryInfo,
		UpdatedAt: new Date().toISOString(),
	};
	applications = [
		...applications.slice(0, idx),
		updated,
		...applications.slice(idx + 1),
	];
	return updated;
}

export function deleteApplication(id: string): void {
	applications = applications.filter((a) => a.ID !== id);
}

export function createStatus(name: string, colour: string): ApplicationStatus {
	const status: ApplicationStatus = {
		ID: faker.string.uuid(),
		UserID: "user-1",
		Name: name,
		Colour: colour,
		CreatedAt: new Date().toISOString(),
	};
	statuses = [...statuses, status];
	return status;
}

export function updateStatus(
	id: string,
	name: string,
	colour: string,
): ApplicationStatus {
	const idx = statuses.findIndex((s) => s.ID === id);
	if (idx === -1) throw new Error("Status not found");
	const updated = { ...statuses[idx]!, Name: name, Colour: colour };
	statuses = [...statuses.slice(0, idx), updated, ...statuses.slice(idx + 1)];
	return updated;
}

export function deleteStatus(id: string): { count?: number } {
	const count = applications.filter((a) => a.StatusID === id).length;
	if (count > 0) return { count };
	statuses = statuses.filter((s) => s.ID !== id);
	return {};
}

export function addCompany(url: string, track: boolean): Company | null {
	let hostname: string;
	try {
		hostname = new URL(url).hostname;
	} catch {
		return null;
	}
	const atsSource = Object.entries(ATS_HOSTS).find(([host]) =>
		hostname.endsWith(host),
	)?.[1];
	if (!atsSource) return null;

	const token = url.replace(/\/$/, "").split("/").pop() ?? hostname;
	const slug = slugify(token);
	const company: Company = {
		ID: faker.string.uuid(),
		Slug: slug,
		Name: humanizeSlug(token),
		ATSSource: "",
		ATSToken: "",
		FirstSeenAt: new Date().toISOString(),
		JobCount: 0,
		Tracked: track,
		TargetID: track ? faker.string.uuid() : "",
		CheckIntervalMinutes: 360,
		LastCheckedAt: null,
	};
	companies = [company, ...companies];
	companyBoards = [
		{
			ID: faker.string.uuid(),
			CompanyID: company.ID,
			Source: atsSource,
			BoardToken: token,
			Status: "candidate",
			VerificationMethod: "",
			VerifiedAt: null,
			LastLinkedAt: null,
			RetiredAt: null,
			CreatedAt: new Date().toISOString(),
		},
		...companyBoards,
	];
	return company;
}

export function getCompanyBoards(companyID: string): CompanyBoard[] {
	return companyBoards.filter((board) => board.CompanyID === companyID);
}

export function addCompanyBoard(
	companyID: string,
	url: string,
	confirm: boolean,
): CompanyBoard | null {
	let hostname: string;
	try {
		hostname = new URL(url).hostname;
	} catch {
		return null;
	}
	const source = Object.entries(ATS_HOSTS).find(([host]) =>
		hostname.endsWith(host),
	)?.[1];
	if (!source) return null;
	const token = url.replace(/\/$/, "").split("/").pop() ?? hostname;
	const existing = companyBoards.find(
		(board) => board.Source === source && board.BoardToken === token,
	);
	if (existing && existing.CompanyID !== companyID)
		throw new Error("This board belongs to another company.");
	const board: CompanyBoard = existing ?? {
		ID: faker.string.uuid(),
		CompanyID: companyID,
		Source: source,
		BoardToken: token,
		Status: "candidate",
		VerificationMethod: "",
		VerifiedAt: null,
		LastLinkedAt: null,
		RetiredAt: null,
		CreatedAt: new Date().toISOString(),
	};
	if (confirm && board.Status === "candidate") {
		board.Status = "verified";
		board.VerificationMethod = "user_confirmed";
		board.VerifiedAt = new Date().toISOString();
	}
	if (!existing) companyBoards = [...companyBoards, board];
	return board;
}

export function setCompanyTracking(
	id: string,
	enabled: boolean,
	checkIntervalMinutes?: number,
): import("../types/company").CompanyTracking {
	const idx = companies.findIndex((c) => c.ID === id);
	if (idx === -1) throw new Error("Company not found");
	const company = companies[idx]!;
	const updated: Company = {
		...company,
		Tracked: enabled,
		CheckIntervalMinutes:
			checkIntervalMinutes ?? (company.CheckIntervalMinutes || 360),
	};
	companies = [
		...companies.slice(0, idx),
		updated,
		...companies.slice(idx + 1),
	];
	return {
		CompanyID: id,
		UserID: mockUser.id,
		Enabled: enabled,
		CheckIntervalMinutes: updated.CheckIntervalMinutes,
	};
}

export function updateSourceTarget(
	id: string,
	patch: { enabled?: boolean; check_interval_minutes?: number },
): SourceTarget {
	const idx = companies.findIndex((c) => c.TargetID === id);
	if (idx === -1) throw new Error("Source target not found");
	const company = companies[idx]!;
	const updated: Company = {
		...company,
		Tracked: patch.enabled ?? company.Tracked,
		CheckIntervalMinutes:
			patch.check_interval_minutes ?? company.CheckIntervalMinutes,
	};
	companies = [
		...companies.slice(0, idx),
		updated,
		...companies.slice(idx + 1),
	];
	return {
		ID: id,
		UserID: mockUser.id,
		Source: company.ATSSource,
		Value: company.ATSToken,
		Enabled: updated.Tracked,
		Filters: {},
	};
}

export const mockUser = { id: "user-1", username: "demo" };
