import type { ApplicationStatus } from "../types/applicationStatus";
import { API_BASE, mockDelay, USE_MOCKS } from "./config";

export async function fetchApplicationStatuses(): Promise<ApplicationStatus[]> {
	if (USE_MOCKS) {
		const { getStatuses } = await import("../mocks/db");
		await mockDelay();
		return getStatuses();
	}
	const res = await fetch(`${API_BASE}/application-statuses`, {
		credentials: "include",
	});
	if (!res.ok) throw new Error(`Failed to fetch statuses: ${res.status}`);
	return res.json();
}

export async function createApplicationStatus(
	name: string,
	colour: string,
): Promise<ApplicationStatus> {
	if (USE_MOCKS) {
		const { createStatus } = await import("../mocks/db");
		await mockDelay(80);
		return createStatus(name, colour);
	}
	const res = await fetch(`${API_BASE}/application-statuses`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ name, colour }),
	});
	if (!res.ok) throw new Error(`Failed to create status: ${res.status}`);
	return res.json();
}

export async function updateApplicationStatus(
	id: string,
	name: string,
	colour: string,
): Promise<ApplicationStatus> {
	if (USE_MOCKS) {
		const { updateStatus } = await import("../mocks/db");
		await mockDelay(80);
		return updateStatus(id, name, colour);
	}
	const res = await fetch(`${API_BASE}/application-statuses/${id}`, {
		method: "PATCH",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ name, colour }),
	});
	if (!res.ok) throw new Error(`Failed to update status: ${res.status}`);
	return res.json();
}

export async function deleteApplicationStatus(
	id: string,
): Promise<{ count?: number }> {
	if (USE_MOCKS) {
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
