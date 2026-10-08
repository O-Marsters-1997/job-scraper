import { faker } from "@faker-js/faker";
import type {
	Application,
	ApplicationWithDetails,
	CreateApplicationPayload,
	UpdateApplicationPayload,
} from "@/types/application";
import type { ApplicationStatus } from "@/types/applicationStatus";
import { failIfRequested } from "./helpers";
import { getJobs } from "./jobs";
import { seed } from "./seed";

let statuses: ApplicationStatus[] = seed.statuses;
let applications: Application[] = seed.applications;

export function clearApplications(): void {
	applications = [];
}

function buildWithDetails(app: Application): ApplicationWithDetails {
	const job = getJobs().find((j) => j.ID === app.JobID);
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
		JobClosedAt: seed.closedJobIDs.has(job.ID) ? job.UpdatedAt : null,
		StatusName: status.Name,
		StatusColour: status.Colour,
	};
}

export function getStatuses(): ApplicationStatus[] {
	return statuses;
}

export function getApplications(
	statusId?: string,
	chase = false,
): ApplicationWithDetails[] {
	const byStatus = statusId
		? applications.filter((a) => a.StatusID === statusId)
		: applications;
	if (!chase) return byStatus.map(buildWithDetails);
	return byStatus
		.filter((a) => a.ChaseBy !== null)
		.sort((a, b) => a.ChaseBy!.localeCompare(b.ChaseBy!))
		.map(buildWithDetails);
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
		ChaseBy: null,
	};
	applications = [...applications, app];
	return app;
}

export function updateApplication(
	id: string,
	data: UpdateApplicationPayload,
): Application {
	failIfRequested("updateApplication");
	const idx = applications.findIndex((a) => a.ID === id);
	if (idx === -1) throw new Error("Application not found");
	const prev = applications[idx]!;
	const updated: Application = {
		...prev,
		StatusID: data.status_id ?? prev.StatusID,
		ChaseBy:
			data.status_id === undefined || data.status_id === prev.StatusID
				? prev.ChaseBy
				: null,
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

export function setChase(id: string, chaseBy: string): Application {
	const prev = applications.find((a) => a.ID === id);
	if (!prev) throw new Error("Application not found");
	const updated: Application = { ...prev, ChaseBy: `${chaseBy}T00:00:00Z` };
	applications = applications.map((a) => (a.ID === id ? updated : a));
	return updated;
}

export function clearChase(id: string): void {
	applications = applications.map((a) =>
		a.ID === id ? { ...a, ChaseBy: null } : a,
	);
}

export function deleteApplication(id: string): void {
	applications = applications.filter((a) => a.ID !== id);
}

export function createStatus(
	name: string,
	colour: string,
	replyWindowDays: number | null,
): ApplicationStatus {
	failIfRequested("createStatus");
	const status: ApplicationStatus = {
		ID: faker.string.uuid(),
		UserID: "user-1",
		Name: name,
		Colour: colour,
		ReplyWindowDays: replyWindowDays,
		CreatedAt: new Date().toISOString(),
	};
	statuses = [...statuses, status];
	return status;
}

export function updateStatus(
	id: string,
	name: string,
	colour: string,
	replyWindowDays: number | null,
): ApplicationStatus {
	const idx = statuses.findIndex((s) => s.ID === id);
	if (idx === -1) throw new Error("Status not found");
	const updated = {
		...statuses[idx]!,
		Name: name,
		Colour: colour,
		ReplyWindowDays: replyWindowDays,
	};
	statuses = [...statuses.slice(0, idx), updated, ...statuses.slice(idx + 1)];
	return updated;
}

export function deleteStatus(id: string): { count?: number } {
	const count = applications.filter((a) => a.StatusID === id).length;
	if (count > 0) return { count };
	statuses = statuses.filter((s) => s.ID !== id);
	return {};
}
