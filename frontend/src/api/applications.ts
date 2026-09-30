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
import { apiFetch, apiFetchVoid, jsonInit, rethrowStatus } from "./client";
import { mocked } from "./config";

export async function fetchApplications(
	statusId?: string,
): Promise<ApplicationWithDetails[]> {
	return mocked(
		(db) => db.getApplications(statusId),
		() => {
			const params = new URLSearchParams();
			if (statusId) params.set("status_id", statusId);
			const query = params.toString();
			return apiFetch(
				`/applications${query ? `?${query}` : ""}`,
				applicationWithDetailsSchema.array(),
			);
		},
	);
}

export async function createApplication(
	data: CreateApplicationPayload,
): Promise<Application> {
	return mocked(
		(db) => db.createApplication(data),
		() =>
			apiFetch(
				"/applications",
				applicationSchema,
				jsonInit("POST", data),
			).catch(rethrowStatus({ 409: () => new Error("application_exists") })),
	);
}

export async function updateApplication(
	id: string,
	data: UpdateApplicationPayload,
): Promise<Application> {
	return mocked(
		(db) => db.updateApplication(id, data),
		() =>
			apiFetch(
				`/applications/${id}`,
				applicationSchema,
				jsonInit("PATCH", data),
			),
	);
}

export async function deleteApplication(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteApplication(id),
		() => apiFetchVoid(`/applications/${id}`, { method: "DELETE" }),
	);
}
