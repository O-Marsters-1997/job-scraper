import type {
	Application,
	ApplicationWithDetails,
	CreateApplicationPayload,
	UpdateApplicationPayload,
} from "../types/application";
import {
	applicationSchema,
	applicationWithDetailsSchema,
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
	const params = new URLSearchParams();
	if (statusId) params.set("status_id", statusId);
	const query = params.toString();
	return apiFetch(
		`/applications${query ? `?${query}` : ""}`,
		undefined,
		applicationWithDetailsSchema.array(),
	);
}

export async function createApplication(
	data: CreateApplicationPayload,
): Promise<Application> {
	if (useMocks()) {
		const { createApplication: mockCreate } = await import("../mocks/db");
		await mockDelay(80);
		return mockCreate(data);
	}
	const response = await fetch(`${API_BASE}/applications`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(data),
	});
	if (response.status === 409) throw new Error("application_exists");
	if (!response.ok)
		throw new Error(`Failed to create application: ${response.status}`);
	return applicationSchema.parse(await response.json());
}

export async function updateApplication(
	id: string,
	data: UpdateApplicationPayload,
): Promise<Application> {
	if (useMocks()) {
		const { updateApplication: mockUpdate } = await import("../mocks/db");
		await mockDelay(80);
		return mockUpdate(id, data);
	}
	return apiFetch(
		`/applications/${id}`,
		{
			method: "PATCH",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(data),
		},
		applicationSchema,
	);
}

export async function deleteApplication(id: string): Promise<void> {
	if (useMocks()) {
		const { deleteApplication: mockDelete } = await import("../mocks/db");
		await mockDelay(80);
		return mockDelete(id);
	}
	return apiFetchVoid(`/applications/${id}`, { method: "DELETE" });
}
