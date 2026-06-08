import { faker } from "@faker-js/faker";
import type {
	Application,
	ApplicationWithDetails,
	JobApplicationSummary,
} from "@/types/application";
import type { ApplicationStatus } from "@/types/applicationStatus";
import type { Job } from "@/types/job";

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

const jobs: Job[] = Array.from({ length: 248 }, () => {
	const company = faker.helpers.arrayElement(COMPANIES);
	const location = faker.helpers.arrayElement(LOCATIONS);
	const scrapedAt = faker.date.recent({ days: 30 }).toISOString();
	const daysInOffice: number | null =
		location === "Remote"
			? 0
			: faker.number.int({ min: 1, max: 10 }) <= 8
				? faker.number.int({ min: 1, max: 5 })
				: null;
	return {
		ID: faker.string.uuid(),
		Title: faker.helpers.arrayElement(JOB_TITLES),
		Location: location,
		URL: faker.internet.url(),
		CompanySlug: slugify(company),
		Source: faker.helpers.arrayElement(SOURCES),
		UpdatedAt: scrapedAt,
		ScrapedAt: scrapedAt,
		DaysInOffice: daysInOffice,
	};
});

// Sort descending so most-recent jobs appear first
jobs.sort(
	(a, b) => new Date(b.ScrapedAt).getTime() - new Date(a.ScrapedAt).getTime(),
);

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
	const status = statuses[statusIndex];
	for (let c = 0; c < count; c++) {
		const job = jobs[appJobCursor++];
		const isSaved = statusIndex === 0;
		const appliedAt = isSaved
			? null
			: faker.date.recent({ days: 20 }).toISOString().split("T")[0];
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
	status_id?: string;
	notes?: string;
	applied_at?: string | null;
	salary_info?: string;
}): Application {
	const status = data.status_id
		? (statuses.find((s) => s.ID === data.status_id) ?? statuses[0])
		: statuses[0];
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
		status_id?: string;
		notes?: string;
		applied_at?: string | null;
		salary_info?: string;
	},
): Application {
	const idx = applications.findIndex((a) => a.ID === id);
	if (idx === -1) throw new Error("Application not found");
	const prev = applications[idx];
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
	const updated = { ...statuses[idx], Name: name, Colour: colour };
	statuses = [...statuses.slice(0, idx), updated, ...statuses.slice(idx + 1)];
	return updated;
}

export function deleteStatus(id: string): { count?: number } {
	const count = applications.filter((a) => a.StatusID === id).length;
	if (count > 0) return { count };
	statuses = statuses.filter((s) => s.ID !== id);
	return {};
}

export const mockUser = { id: "user-1", username: "demo" };
