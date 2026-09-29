import { faker } from "@faker-js/faker";
import type {
	Application,
	ApplicationWithDetails,
	CreateApplicationPayload,
	UpdateApplicationPayload,
} from "@/types/application";
import type { ApplicationStatus } from "@/types/applicationStatus";
import type { Company, CompanyBoard, CompanyTracking } from "@/types/company";
import type { CV } from "@/types/cv";
import type {
	Achievement,
	ImportPosition,
	Position,
	PositionInput,
} from "@/types/experience";
import type { Job, ScoreRow } from "@/types/job";
import type { ResolvedBoard, SourceInfo } from "@/types/source";
import type {
	CreateSourceTargetPayload,
	SourceTarget,
	UpdateSourceTargetPayload,
} from "@/types/sourceTarget";
import type {
	CVHeading,
	Draft,
	DraftInput,
	DraftRef,
	HeadingMapping,
	Suggestion,
} from "@/types/tailoring";
import type { AiPrefs } from "../types/aiPrefs";
import type { GoogleStatus } from "../types/google";
import type { Profile } from "../types/profile";
import type { RecomputeResult, ScoringStatus } from "../types/scores";
import type { ScoringConfig } from "../types/scoringConfig";
import type {
	DimensionSpec,
	ScoringOption,
	ScoringOptionsView,
} from "../types/scoringOptions";

faker.seed(1234);

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

let scoringConfig: ScoringConfig = {
	notifyThreshold: 70,
	excludedTitleKeywords: [],
	excludedCompanies: [],
	excludedLocations: [],
	preferences: {
		picks: [
			{
				optionId: "tech:go",
				stance: "nice",
				source: "manual",
				overridden: false,
			},
			{
				optionId: "role:backend",
				stance: "nice",
				source: "manual",
				overridden: false,
			},
			{
				optionId: "domain:fintech",
				stance: "avoid",
				source: "manual",
				overridden: false,
			},
		],
		salaryFloor: null,
		preferenceText: "",
	},
	updatedAt: new Date("2024-01-01").toISOString(),
	backfillQueued: 0,
};

const scoringDimensions: DimensionSpec[] = [
	{ key: "tech", kind: "pair", stances: ["nice", "avoid"] },
	{ key: "role", kind: "pair", stances: ["nice", "avoid"] },
	{ key: "domain", kind: "pair", stances: ["nice", "avoid", "block"] },
	{ key: "seniority", kind: "multi", stances: ["nice"] },
	{ key: "work", kind: "multi", stances: ["nice"] },
	{ key: "stage", kind: "multi", stances: ["nice"] },
];

const scoringOptions: ScoringOption[] = [
	{ id: "tech:go", dimension: "tech", label: "Go" },
	{ id: "tech:python", dimension: "tech", label: "Python" },
	{ id: "tech:typescript", dimension: "tech", label: "TypeScript" },
	{ id: "tech:rust", dimension: "tech", label: "Rust" },
	{ id: "tech:react", dimension: "tech", label: "React" },
	{ id: "tech:solidjs", dimension: "tech", label: "SolidJS" },
	{ id: "tech:postgres", dimension: "tech", label: "Postgres" },
	{ id: "tech:kafka", dimension: "tech", label: "Kafka" },
	{ id: "tech:kubernetes", dimension: "tech", label: "Kubernetes" },
	{ id: "tech:aws", dimension: "tech", label: "AWS" },
	{ id: "tech:terraform", dimension: "tech", label: "Terraform" },
	{ id: "tech:graphql", dimension: "tech", label: "GraphQL" },
	{ id: "role:backend", dimension: "role", label: "Backend" },
	{ id: "role:platform", dimension: "role", label: "Platform" },
	{ id: "role:full-stack", dimension: "role", label: "Full-stack" },
	{ id: "role:frontend", dimension: "role", label: "Frontend" },
	{ id: "role:sre", dimension: "role", label: "SRE" },
	{ id: "role:data-engineering", dimension: "role", label: "Data engineering" },
	{ id: "domain:fintech", dimension: "domain", label: "fintech" },
	{ id: "domain:devtools", dimension: "domain", label: "devtools" },
	{ id: "domain:health", dimension: "domain", label: "health" },
	{ id: "domain:climate", dimension: "domain", label: "climate" },
	{ id: "domain:crypto", dimension: "domain", label: "crypto" },
	{ id: "domain:gambling", dimension: "domain", label: "gambling" },
	{ id: "seniority:mid", dimension: "seniority", label: "Mid" },
	{ id: "seniority:senior", dimension: "seniority", label: "Senior" },
	{ id: "seniority:staff", dimension: "seniority", label: "Staff" },
	{ id: "work:remote", dimension: "work", label: "Remote" },
	{ id: "work:hybrid", dimension: "work", label: "Hybrid" },
	{ id: "work:onsite", dimension: "work", label: "Onsite" },
	{ id: "stage:seed", dimension: "stage", label: "Seed" },
	{ id: "stage:series-a", dimension: "stage", label: "Series A" },
	{ id: "stage:series-b", dimension: "stage", label: "Series B" },
	{ id: "stage:public", dimension: "stage", label: "Public" },
];

const BREAKDOWN_PICKS: {
	key: string;
	label: string;
	stance: "nice" | "avoid";
}[] = [
	{ key: "tech:go", label: "Go", stance: "nice" },
	{ key: "tech:python", label: "Python", stance: "nice" },
	{ key: "role:backend", label: "Backend", stance: "nice" },
	{ key: "domain:fintech", label: "fintech", stance: "avoid" },
];

function mockBreakdown(i: number): ScoreRow[] {
	return BREAKDOWN_PICKS.map((p, j) => {
		const roll = (i + j) % 3;
		if (roll === 0) {
			return {
				key: p.key,
				label: p.label,
				stance: p.stance,
				resolved: "unknown",
				effect: "unknown",
				overridden: false,
			} satisfies ScoreRow;
		}
		if (p.stance === "nice") {
			const matched = roll === 1;
			return {
				key: p.key,
				label: p.label,
				stance: p.stance,
				resolved: matched ? "yes" : "no",
				effect: matched ? "meets" : "misses",
				overridden: false,
			} satisfies ScoreRow;
		}
		const hit = roll === 1;
		return {
			key: p.key,
			label: p.label,
			stance: p.stance,
			resolved: hit ? "yes" : "no",
			effect: hit ? "misses" : "neutral",
			overridden: false,
		} satisfies ScoreRow;
	});
}

function slugify(name: string): string {
	return name
		.toLowerCase()
		.replace(/\s+/g, "-")
		.replace(/[^a-z0-9-]/g, "");
}

let jobs: Job[] = Array.from({ length: 248 }, (_, i) => {
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
		SuitabilityScore: scored ? faker.number.int({ min: 30, max: 100 }) : null,
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

jobs.sort(
	(a, b) => new Date(b.ScrapedAt).getTime() - new Date(a.ScrapedAt).getTime(),
);

const ATS_SOURCES = [
	"greenhouse",
	"lever",
	"ashby",
	"workable",
	"recruitee",
	"personio",
];

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
		LastCompletedAt: company.LastCheckedAt,
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

function hostMatches(hostname: string, host: string): boolean {
	return hostname === host || hostname.endsWith(`.${host}`);
}

function humanizeSlug(slug: string): string {
	return slug
		.split(/[-_]/)
		.filter(Boolean)
		.map((w) => w[0]!.toUpperCase() + w.slice(1))
		.join(" ");
}

const APP_DISTRIBUTION: Array<{ statusIndex: number; count: number }> = [
	{ statusIndex: 0, count: 2 },
	{ statusIndex: 1, count: 5 },
	{ statusIndex: 2, count: 2 },
	{ statusIndex: 3, count: 1 },
	{ statusIndex: 4, count: 1 },
	{ statusIndex: 5, count: 1 },
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

const SOURCE_INFOS: SourceInfo[] = [
	{
		name: "greenhouse",
		label: "Greenhouse",
		kind: "board",
		role: "ats",
		url_prefix: "https://boards.greenhouse.io",
		filters: [],
	},
	{
		name: "lever",
		label: "Lever",
		kind: "board",
		role: "ats",
		url_prefix: "https://jobs.lever.co",
		filters: [],
	},
	{
		name: "ashby",
		label: "Ashby",
		kind: "board",
		role: "ats",
		url_prefix: "https://jobs.ashbyhq.com",
		filters: [],
	},
	{
		name: "workable",
		label: "Workable",
		kind: "board",
		role: "ats",
		url_prefix: "https://apply.workable.com",
		filters: [],
	},
	{
		name: "recruitee",
		label: "Recruitee",
		kind: "board",
		role: "ats",
		url_prefix: "https://recruitee.com",
		filters: [],
	},
	{
		name: "personio",
		label: "Personio",
		kind: "board",
		role: "ats",
		url_prefix: "https://personio.de",
		filters: [],
	},
	{
		name: "wis",
		label: "Work in Startups",
		kind: "filter",
		role: "discovery",
		url_prefix: "https://workinstartups.com",
		filters: [{ name: "region", label: "Region", required: false }],
	},
	{
		name: "linkedin",
		label: "LinkedIn",
		kind: "filter",
		role: "discovery",
		url_prefix: "https://www.linkedin.com/jobs",
		filters: [{ name: "location", label: "Location", required: false }],
	},
	{
		name: "indeed",
		label: "Indeed",
		kind: "url",
		role: "discovery",
		url_prefix: "https://www.indeed.com",
		filters: [],
	},
	{
		name: "remoteok",
		label: "RemoteOK",
		kind: "filter",
		role: "discovery",
		url_prefix: "https://remoteok.com",
		filters: [],
	},
	{
		name: "remotive",
		label: "Remotive",
		kind: "filter",
		role: "discovery",
		url_prefix: "https://remotive.com",
		filters: [],
	},
];

export function getSources(): SourceInfo[] {
	return SOURCE_INFOS;
}

export function resolveBoard(url: string): ResolvedBoard | null {
	let hostname: string;
	try {
		hostname = new URL(url).hostname;
	} catch {
		return null;
	}
	const match = SOURCE_INFOS.find(
		(s) =>
			s.kind === "board" &&
			hostMatches(hostname, new URL(s.url_prefix).hostname),
	);
	if (!match) return null;
	const value = url.replace(/\/$/, "").split("/").pop() ?? hostname;
	return { source: match.name, value };
}

let discoverySourceTargets: SourceTarget[] = [
	{
		ID: "target-wis-1",
		UserID: "user-1",
		Source: "wis",
		Value: "engineer",
		Enabled: true,
		Filters: { region: "uk" },
		RunStatus: "succeeded",
		LastRunAt: faker.date.recent({ days: 1 }).toISOString(),
		LastRunError: "",
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
	},
];

function companyToSourceTarget(company: Company): SourceTarget {
	return {
		ID: company.TargetID,
		UserID: "user-1",
		Source: company.ATSSource,
		Value: company.ATSToken,
		Enabled: company.Tracked,
		Filters: {},
		RunStatus: "idle",
		LastRunAt: company.LastCheckedAt,
		LastRunError: "",
	};
}

export function getSourceTargets(): SourceTarget[] {
	const atsTargets = companies
		.filter((c) => c.TargetID)
		.map(companyToSourceTarget);
	return [...atsTargets, ...discoverySourceTargets];
}

export function createSourceTarget(
	payload: CreateSourceTargetPayload,
): SourceTarget | null {
	const exists = getSourceTargets().some(
		(t) => t.Source === payload.source && t.Value === payload.value,
	);
	if (exists) return null;
	const target: SourceTarget = {
		ID: faker.string.uuid(),
		UserID: "user-1",
		Source: payload.source,
		Value: payload.value,
		Enabled: payload.enabled ?? true,
		Filters: payload.filters ?? {},
		RunStatus: "succeeded",
		LastRunAt: new Date().toISOString(),
		LastRunError: "",
	};
	discoverySourceTargets = [...discoverySourceTargets, target];
	return target;
}

export function updateSourceTarget(
	id: string,
	patch: UpdateSourceTargetPayload,
): SourceTarget {
	const companyIdx = companies.findIndex((c) => c.TargetID === id);
	if (companyIdx !== -1) {
		const company = companies[companyIdx]!;
		const updated: Company = {
			...company,
			Tracked: patch.enabled ?? company.Tracked,
			CheckIntervalMinutes:
				patch.check_interval_minutes ?? company.CheckIntervalMinutes,
		};
		companies = [
			...companies.slice(0, companyIdx),
			updated,
			...companies.slice(companyIdx + 1),
		];
		return companyToSourceTarget(updated);
	}
	const idx = discoverySourceTargets.findIndex((t) => t.ID === id);
	if (idx === -1) throw new Error("Source target not found");
	const updated: SourceTarget = {
		...discoverySourceTargets[idx]!,
		Enabled: patch.enabled ?? discoverySourceTargets[idx]!.Enabled,
	};
	discoverySourceTargets = [
		...discoverySourceTargets.slice(0, idx),
		updated,
		...discoverySourceTargets.slice(idx + 1),
	];
	return updated;
}

export function deleteSourceTarget(id: string): void {
	discoverySourceTargets = discoverySourceTargets.filter((t) => t.ID !== id);
}

export function rerunSourceTarget(id: string): SourceTarget {
	const idx = discoverySourceTargets.findIndex((t) => t.ID === id);
	if (idx === -1) {
		const existing = getSourceTargets().find((t) => t.ID === id);
		if (!existing) throw new Error("Source target not found");
		return {
			...existing,
			RunStatus: "succeeded",
			LastRunAt: new Date().toISOString(),
		};
	}
	const updated: SourceTarget = {
		...discoverySourceTargets[idx]!,
		RunStatus: "succeeded",
		LastRunAt: new Date().toISOString(),
	};
	discoverySourceTargets = [
		...discoverySourceTargets.slice(0, idx),
		updated,
		...discoverySourceTargets.slice(idx + 1),
	];
	return updated;
}

let cvs: CV[] = [
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

export function getCVs(): CV[] {
	return cvs;
}

export function addTrackedDoc(url: string): void {
	const docId = `doc-${faker.string.uuid().slice(0, 8)}`;
	cvs = [
		...cvs,
		{
			DocID: docId,
			TabID: "t.0",
			Title: "New Tracked Doc",
			SourceDoc: "New Tracked Doc",
			ModifiedAt: new Date().toISOString(),
			DocURL: url,
			Visible: true,
		},
	];
}

export function removeTrackedDoc(docId: string): void {
	cvs = cvs.filter((cv) => cv.DocID !== docId);
}

export function setTabVisibility(
	docId: string,
	tabId: string,
	visible: boolean,
): void {
	cvs = cvs.map((cv) =>
		cv.DocID === docId && cv.TabID === tabId ? { ...cv, Visible: visible } : cv,
	);
}

let experience: Position[] = [
	{
		id: "position-1",
		employer: "Acme Ltd",
		title: "Senior Backend Engineer",
		startDate: "2021-03-01",
		endDate: null,
		achievements: [
			{
				id: "achievement-1",
				positionId: "position-1",
				text: "Cut p99 API latency by 40% by moving hot paths to a read-through cache",
			},
			{
				id: "achievement-2",
				positionId: "position-1",
				text: "Led the migration of 12 services from EC2 to Kubernetes",
			},
		],
	},
	{
		id: "position-2",
		employer: "Globex",
		title: "Software Engineer",
		startDate: "2018-06-01",
		endDate: "2021-02-28",
		achievements: [
			{
				id: "achievement-3",
				positionId: "position-2",
				text: "Built the billing export pipeline used by finance every month",
			},
		],
	},
];

export function getExperience(): Position[] {
	return experience;
}

export function createPosition(input: PositionInput): Position {
	const position: Position = {
		id: `position-${faker.string.uuid().slice(0, 8)}`,
		...input,
		achievements: [],
	};
	experience = [position, ...experience];
	return position;
}

export function updatePosition(id: string, input: PositionInput): Position {
	const existing = experience.find((p) => p.id === id);
	if (!existing) throw new Error("position not found");
	const updated = { ...existing, ...input };
	experience = experience.map((p) => (p.id === id ? updated : p));
	return updated;
}

export function deletePosition(id: string): void {
	experience = experience.filter((p) => p.id !== id);
}

export function previewExperienceImport(): ImportPosition[] {
	return [
		{
			employer: "Acme Ltd",
			title: "Senior Backend Engineer",
			startDate: "2021-03-01",
			endDate: null,
			achievements: ["Cut p99 API latency by 40%", "Mentored four engineers"],
			employerExists: experience.some((p) => p.employer === "Acme Ltd"),
		},
		{
			employer: "Initech",
			title: "Developer",
			startDate: null,
			endDate: null,
			achievements: ["Maintained the payroll batch jobs"],
			employerExists: false,
		},
	];
}

export function importExperience(positions: ImportPosition[]): Position[] {
	const created = positions.map((p): Position => {
		const id = `position-${faker.string.uuid().slice(0, 8)}`;
		return {
			id,
			employer: p.employer,
			title: p.title,
			startDate: p.startDate,
			endDate: p.endDate,
			achievements: p.achievements.map((text) => ({
				id: `achievement-${faker.string.uuid().slice(0, 8)}`,
				positionId: id,
				text,
			})),
		};
	});
	experience = [...created, ...experience];
	return created;
}

function inOrder<T extends { id: string }>(items: T[], ids: string[]): T[] {
	return ids.flatMap((id) => items.filter((item) => item.id === id));
}

export function reorderPositions(ids: string[]): void {
	experience = inOrder(experience, ids);
}

export function createAchievement(
	positionId: string,
	text: string,
): Achievement {
	const achievement: Achievement = {
		id: `achievement-${faker.string.uuid().slice(0, 8)}`,
		positionId,
		text,
	};
	experience = experience.map((p) =>
		p.id === positionId
			? { ...p, achievements: [...p.achievements, achievement] }
			: p,
	);
	return achievement;
}

export function updateAchievement(id: string, text: string): Achievement {
	let updated: Achievement | undefined;
	experience = experience.map((p) => ({
		...p,
		achievements: p.achievements.map((a) => {
			if (a.id !== id) return a;
			updated = { ...a, text };
			return updated;
		}),
	}));
	if (!updated) throw new Error("achievement not found");
	return updated;
}

export function deleteAchievement(id: string): void {
	experience = experience.map((p) => ({
		...p,
		achievements: p.achievements.filter((a) => a.id !== id),
	}));
}

export function reorderAchievements(positionId: string, ids: string[]): void {
	experience = experience.map((p) =>
		p.id === positionId
			? { ...p, achievements: inOrder(p.achievements, ids) }
			: p,
	);
}

const MOCK_PDF = `%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 58 >>
stream
BT /F1 24 Tf 72 700 Td (Demo CV preview) Tj ET
endstream
endobj
5 0 obj
<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>
endobj
trailer
<< /Size 6 /Root 1 0 R >>
%%EOF
`;

export function getMockPdfBytes(): ArrayBuffer {
	return new TextEncoder().encode(MOCK_PDF).buffer;
}

let aiPrefs: AiPrefs = {
	configuredProviders: ["openrouter"],
	scoringEnabled: true,
};

export function getAiPrefs(): AiPrefs {
	return structuredClone(aiPrefs);
}

export function setAiCredential(provider: string, apiKey: string | null): void {
	const providers = new Set(aiPrefs.configuredProviders);
	if (apiKey) providers.add(provider);
	else providers.delete(provider);
	aiPrefs = { ...aiPrefs, configuredProviders: [...providers] };
}

let googleStatus: GoogleStatus = { connected: false, canWrite: false };

export function getGoogleStatus(): GoogleStatus {
	return structuredClone(googleStatus);
}

export function disconnectGoogle(): void {
	googleStatus = { connected: false, canWrite: false };
}

let profile: Profile = { username: "demo", email: "demo@example.com" };

export function getProfile(): Profile {
	return structuredClone(profile);
}

export function updateProfile(payload: { email: string }): void {
	profile = { ...profile, email: payload.email };
}

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

export function getScoringConfig(): ScoringConfig {
	return structuredClone(scoringConfig);
}

export function getScoringOptions(): ScoringOptionsView {
	return { dimensions: scoringDimensions, options: scoringOptions };
}

export function getScoringStatus(): ScoringStatus {
	return { pending: 0 };
}

export function recomputeScores(): RecomputeResult {
	let recomputed = 0;
	jobs = jobs.map((job) => {
		if (job.SuitabilityScore == null) return job;
		recomputed++;
		return {
			...job,
			SuitabilityScore: Math.min(100, job.SuitabilityScore + 1),
		};
	});
	return { recomputed };
}

export function createApplication(data: CreateApplicationPayload): Application {
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
	data: UpdateApplicationPayload,
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
		hostMatches(hostname, host),
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
	addCompanyBoard(company.ID, url, false);
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
		hostMatches(hostname, host),
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
		LastCompletedAt: null,
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
): CompanyTracking {
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

export function updateScoringConfig(payload: ScoringConfig): ScoringConfig {
	scoringConfig = payload;
	return scoringConfig;
}

export const mockUser = { id: "user-1", username: "demo" };

export function switchMockUser(username: string): void {
	if (username === mockUser.username) return;
	mockUser.id = `user-${username}`;
	mockUser.username = username;
	applications = [];
}

const savedHeadings = new Map<string, HeadingMapping[]>();

export function getHeadings(docId: string, tabId: string): CVHeading[] {
	const saved = savedHeadings.get(`${docId}/${tabId}`);
	const roles: { text: string; match: string | null; slots: number }[] = [
		{
			text: "Senior Backend Engineer, Acme Ltd",
			match: "position-1",
			slots: 2,
		},
		{ text: "Software Engineer, Globex", match: "position-2", slots: 1 },
	];
	return roles.map((r) => {
		const s = saved?.find((m) => m.headingText === r.text);
		return {
			text: r.text,
			positionId: s ? s.positionId : r.match,
			confirmed: s !== undefined,
			slotCount: r.slots,
		};
	});
}

export function saveHeadings(
	docId: string,
	tabId: string,
	mappings: HeadingMapping[],
): HeadingMapping[] {
	savedHeadings.set(`${docId}/${tabId}`, mappings);
	return mappings;
}

export function getSuggestions(): Suggestion[] {
	const scores = [0.81, 0.64, 0.42, 0.17];
	return experience.flatMap((p) => {
		const slots = p.id === "position-1" ? 2 : 3;
		return p.achievements.map((a, i) => ({
			achievementId: a.id,
			positionId: p.id,
			text: a.text,
			score: scores[i] ?? 0.1,
			preselected: i < slots,
		}));
	});
}

const mockDrafts = new Map<string, { draft: Draft; polls: number }>();

export function createDraft(input: DraftInput): DraftRef {
	const id = `draft-${mockDrafts.size + 1}`;
	mockDrafts.set(id, {
		draft: {
			id,
			jobId: input.jobId,
			status: "pending",
			draftDocUrl: null,
			lastError: "",
		},
		polls: 0,
	});
	return { id };
}

export function getDraft(id: string): Draft {
	const entry = mockDrafts.get(id);
	if (!entry) throw new Error(`no mock draft ${id}`);
	entry.polls += 1;
	if (entry.polls >= 3) {
		entry.draft = {
			...entry.draft,
			status: "ready",
			draftDocUrl: "https://docs.google.com/document/d/mock-draft/edit",
		};
	} else if (entry.polls === 2) {
		entry.draft = { ...entry.draft, status: "running" };
	}
	return entry.draft;
}
