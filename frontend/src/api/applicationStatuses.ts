import type { ApplicationStatus } from "../types/applicationStatus";
import { apiFetch } from "./client";
import { API_BASE, mockDelay, useMocks } from "./config";

export async function fetchApplicationStatuses(): Promise<ApplicationStatus[]> {
	if (useMocks()) {
		const { getStatuses } = await import("../mocks/db");
		await mockDelay();
		return getStatuses();
	}
	return apiFetch<ApplicationStatus[]>("/application-statuses");
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
	return apiFetch<ApplicationStatus>("/application-statuses", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ name, colour }),
	});
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
	return apiFetch<ApplicationStatus>(`/application-statuses/${id}`, {
		method: "PATCH",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ name, colour }),
	});
}

export async function deleteApplicationStatus(
	id: string,
): Promise<{ count?: number }> {
	if (useMocks()) {
		const { deleteStatus } = await import("../mocks/db");
		await mockDelay(80);
		return deleteStatus(id);
	}
	const res = await fetch(`${API_BASE}/application-statuses/${id}`, {
		method: "DELETE",
		credentials: "include",
	});
	if (res.status === 409) {
		const body = await res.json();
		return { count: body.count };
	}
	if (!res.ok) throw new Error(`Failed to delete status: ${res.status}`);
	return {};
}
