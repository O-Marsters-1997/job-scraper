import type {
	Application,
	ApplicationWithDetails,
	JobApplicationSummary,
} from "../types/application";
import { apiFetch, apiFetchVoid } from "./client";
import { API_BASE, mockDelay, useMocks } from "./config";

export async function fetchApplications(
	statusId?: string,
): Promise<ApplicationWithDetails[]> {
	if (useMocks()) {
		const { getApplications } = await import("../mocks/db");
		await mockDelay();
		return getApplications(statusId);
	}
	const path = statusId
		? `/applications?status_id=${statusId}`
		: "/applications";
	return apiFetch<ApplicationWithDetails[]>(path);
}

export async function createApplication(data: {
	job_id: string;
	status_id?: string;
	notes?: string;
	applied_at?: string | null;
	salary_info?: string;
}): Promise<Application> {
	if (useMocks()) {
		const { createApplication: mockCreate } = await import("../mocks/db");
		await mockDelay(80);
		return mockCreate(data);
	}
	const res = await fetch(`${API_BASE}/applications`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(data),
	});
	if (res.status === 409) throw new Error("application_exists");
	if (!res.ok) throw new Error(`Failed to create application: ${res.status}`);
	return res.json();
}

export async function updateApplication(
	id: string,
	data: {
		status_id?: string;
		notes?: string;
		applied_at?: string | null;
		salary_info?: string;
	},
): Promise<Application> {
	if (useMocks()) {
		const { updateApplication: mockUpdate } = await import("../mocks/db");
		await mockDelay(80);
		return mockUpdate(id, data);
	}
	return apiFetch<Application>(`/applications/${id}`, {
		method: "PATCH",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(data),
	});
}

export async function deleteApplication(id: string): Promise<void> {
	if (useMocks()) {
		const { deleteApplication: mockDelete } = await import("../mocks/db");
		await mockDelay(80);
		return mockDelete(id);
	}
	return apiFetchVoid(`/applications/${id}`, { method: "DELETE" });
}

export async function fetchApplicationsForJobs(
	jobIds: string[],
): Promise<Record<string, JobApplicationSummary>> {
	if (useMocks()) {
		const { getApplicationsForJobs } = await import("../mocks/db");
		await mockDelay();
		return getApplicationsForJobs(jobIds);
	}
	if (jobIds.length === 0) return {};
	return apiFetch<Record<string, JobApplicationSummary>>(
		`/applications/for-jobs?job_ids=${jobIds.join(",")}`,
	);
}
