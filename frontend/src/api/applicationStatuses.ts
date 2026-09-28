import {
	type ApplicationStatus,
	applicationStatusSchema,
} from "../types/applicationStatus";
import { apiFetch } from "./client";
import { API_BASE, mockDelay, mocked } from "./config";

export async function fetchApplicationStatuses(): Promise<ApplicationStatus[]> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getStatuses();
		},
		() =>
			apiFetch(
				"/application-statuses",
				undefined,
				applicationStatusSchema.array(),
			),
	);
}

export async function createApplicationStatus(
	name: string,
	colour: string,
): Promise<ApplicationStatus> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.createStatus(name, colour);
		},
		() =>
			apiFetch(
				"/application-statuses",
				{
					method: "POST",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ name, colour }),
				},
				applicationStatusSchema,
			),
	);
}

export async function updateApplicationStatus(
	id: string,
	name: string,
	colour: string,
): Promise<ApplicationStatus> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.updateStatus(id, name, colour);
		},
		() =>
			apiFetch(
				`/application-statuses/${id}`,
				{
					method: "PATCH",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ name, colour }),
				},
				applicationStatusSchema,
			),
	);
}

export async function deleteApplicationStatus(
	id: string,
): Promise<{ count?: number }> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.deleteStatus(id);
		},
		async () => {
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
		},
	);
}
