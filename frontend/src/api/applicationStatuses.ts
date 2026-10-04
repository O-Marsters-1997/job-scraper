import {
	type ApplicationStatus,
	applicationStatusSchema,
} from "../types/applicationStatus";
import { ApiError, apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchApplicationStatuses(): Promise<ApplicationStatus[]> {
	return mocked(
		(db) => db.getStatuses(),
		() => apiFetch("/application-statuses", applicationStatusSchema.array()),
	);
}

export async function createApplicationStatus(
	name: string,
	colour: string,
	replyWindowDays: number | null,
): Promise<ApplicationStatus> {
	return mocked(
		(db) => db.createStatus(name, colour, replyWindowDays),
		() =>
			apiFetch(
				"/application-statuses",
				applicationStatusSchema,
				jsonInit("POST", { name, colour, replyWindowDays }),
			),
	);
}

export async function updateApplicationStatus(
	id: string,
	name: string,
	colour: string,
	replyWindowDays: number | null,
): Promise<ApplicationStatus> {
	return mocked(
		(db) => db.updateStatus(id, name, colour, replyWindowDays),
		() =>
			apiFetch(
				`/application-statuses/${id}`,
				applicationStatusSchema,
				jsonInit("PATCH", { name, colour, replyWindowDays }),
			),
	);
}

export async function deleteApplicationStatus(
	id: string,
): Promise<{ count?: number }> {
	return mocked(
		(db) => db.deleteStatus(id),
		() =>
			apiFetchVoid(`/application-statuses/${id}`, { method: "DELETE" }).then(
				(): { count?: number } => ({}),
				(err) => {
					if (err instanceof ApiError && err.status === 409) {
						return { count: (err.body as { count: number }).count };
					}
					throw err;
				},
			),
	);
}
