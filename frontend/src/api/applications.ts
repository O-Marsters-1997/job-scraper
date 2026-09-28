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
import { API_BASE, mockDelay, mocked } from "./config";

export async function fetchApplications(
	statusId?: string,
): Promise<ApplicationWithDetails[]> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getApplications(statusId);
		},
		() => {
			const params = new URLSearchParams();
			if (statusId) params.set("status_id", statusId);
			const query = params.toString();
			return apiFetch(
				`/applications${query ? `?${query}` : ""}`,
				undefined,
				applicationWithDetailsSchema.array(),
			);
		},
	);
}

export async function createApplication(
	data: CreateApplicationPayload,
): Promise<Application> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.createApplication(data);
		},
		async () => {
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
		},
	);
}

export async function updateApplication(
	id: string,
	data: UpdateApplicationPayload,
): Promise<Application> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.updateApplication(id, data);
		},
		() =>
			apiFetch(
				`/applications/${id}`,
				{
					method: "PATCH",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify(data),
				},
				applicationSchema,
			),
	);
}

export async function deleteApplication(id: string): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.deleteApplication(id);
		},
		() => apiFetchVoid(`/applications/${id}`, { method: "DELETE" }),
	);
}
