import {
	type Achievement,
	achievementSchema,
	type Position,
	type PositionInput,
	positionSchema,
} from "../types/experience";
import { apiFetch, apiFetchVoid } from "./client";
import { mockDelay, mocked } from "./config";

const jsonInit = (method: string, body: unknown): RequestInit => ({
	method,
	headers: { "Content-Type": "application/json" },
	body: JSON.stringify(body),
});

export async function fetchExperience(): Promise<Position[]> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getExperience();
		},
		() => apiFetch("/experience", undefined, positionSchema.array()),
	);
}

export async function createPosition(input: PositionInput): Promise<Position> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.createPosition(input);
		},
		() =>
			apiFetch(
				"/experience/positions",
				jsonInit("POST", input),
				positionSchema,
			),
	);
}

export async function updatePosition(
	id: string,
	input: PositionInput,
): Promise<Position> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.updatePosition(id, input);
		},
		() =>
			apiFetch(
				`/experience/positions/${id}`,
				jsonInit("PATCH", input),
				positionSchema,
			),
	);
}

export async function deletePosition(id: string): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.deletePosition(id);
		},
		() => apiFetchVoid(`/experience/positions/${id}`, { method: "DELETE" }),
	);
}

export async function reorderPositions(ids: string[]): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.reorderPositions(ids);
		},
		() => apiFetchVoid("/experience/positions/order", jsonInit("PUT", { ids })),
	);
}

export async function createAchievement(
	positionId: string,
	text: string,
): Promise<Achievement> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.createAchievement(positionId, text);
		},
		() =>
			apiFetch(
				`/experience/positions/${positionId}/achievements`,
				jsonInit("POST", { text }),
				achievementSchema,
			),
	);
}

export async function updateAchievement(
	id: string,
	text: string,
): Promise<Achievement> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.updateAchievement(id, text);
		},
		() =>
			apiFetch(
				`/experience/achievements/${id}`,
				jsonInit("PATCH", { text }),
				achievementSchema,
			),
	);
}

export async function deleteAchievement(id: string): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.deleteAchievement(id);
		},
		() => apiFetchVoid(`/experience/achievements/${id}`, { method: "DELETE" }),
	);
}

export async function reorderAchievements(
	positionId: string,
	ids: string[],
): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.reorderAchievements(positionId, ids);
		},
		() =>
			apiFetchVoid(
				`/experience/positions/${positionId}/achievements/order`,
				jsonInit("PUT", { ids }),
			),
	);
}
