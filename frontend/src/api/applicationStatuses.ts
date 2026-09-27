import {
	type ApplicationStatus,
	applicationStatusSchema,
} from "../types/applicationStatus";
import { apiFetch } from "./client";
import { API_BASE, mockDelay, useMocks } from "./config";

export async function fetchApplicationStatuses(): Promise<ApplicationStatus[]> {
	if (useMocks()) {
		const { getStatuses } = await import("../mocks/db");
		await mockDelay();
		return getStatuses();
	}
	return apiFetch(
		"/application-statuses",
		undefined,
		applicationStatusSchema.array(),
	);
}

export async function createApplicationStatus(
	name: string,
	colour: string,
): Promise<ApplicationStatus> {
	if (useMocks()) {
		const { createStatus } = await import("../mocks/db");
		await mockDelay(80);
		return createStatus(name, colour);
	}
	return apiFetch(
		"/application-statuses",
		{
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ name, colour }),
		},
		applicationStatusSchema,
	);
}

export async function updateApplicationStatus(
	id: string,
	name: string,
	colour: string,
): Promise<ApplicationStatus> {
	if (useMocks()) {
		const { updateStatus } = await import("../mocks/db");
		await mockDelay(80);
		return updateStatus(id, name, colour);
	}
	return apiFetch(
		`/application-statuses/${id}`,
		{
			method: "PATCH",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ name, colour }),
		},
		applicationStatusSchema,
	);
}

export async function deleteApplicationStatus(
	id: string,
): Promise<{ count?: number }> {
	if (useMocks()) {
		const { deleteStatus } = await import("../mocks/db");
		await mockDelay(80);
		return deleteStatus(id);
	}
	const response = await fetch(`${API_BASE}/application-statuses/${id}`, {
		method: "DELETE",
		credentials: "include",
	});
	if (response.status === 409) {
		const body = await response.json();
		return { count: body.count };
	}
	if (!response.ok)
		throw new Error(`Failed to delete status: ${response.status}`);
	return {};
}
