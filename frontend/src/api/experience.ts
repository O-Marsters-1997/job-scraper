import {
	type Achievement,
	type AchievementVariant,
	type AchievementVariantInput,
	achievementSchema,
	achievementVariantSchema,
	type Position,
	type PositionInput,
	positionSchema,
} from "../types/experience";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchExperience(): Promise<Position[]> {
	return mocked(
		(db) => db.getExperience(),
		() => apiFetch("/experience", positionSchema.array()),
	);
}

export async function createPosition(input: PositionInput): Promise<Position> {
	return mocked(
		(db) => db.createPosition(input),
		() =>
			apiFetch(
				"/experience/positions",
				positionSchema,
				jsonInit("POST", input),
			),
	);
}

export async function updatePosition(
	id: string,
	input: PositionInput,
): Promise<Position> {
	return mocked(
		(db) => db.updatePosition(id, input),
		() =>
			apiFetch(
				`/experience/positions/${id}`,
				positionSchema,
				jsonInit("PATCH", input),
			),
	);
}

export async function deletePosition(id: string): Promise<void> {
	return mocked(
		(db) => db.deletePosition(id),
		() => apiFetchVoid(`/experience/positions/${id}`, { method: "DELETE" }),
	);
}

export async function reorderPositions(ids: string[]): Promise<void> {
	return mocked(
		(db) => db.reorderPositions(ids),
		() => apiFetchVoid("/experience/positions/order", jsonInit("PUT", { ids })),
	);
}

export async function createAchievement(
	positionId: string,
	text: string,
): Promise<Achievement> {
	return mocked(
		(db) => db.createAchievement(positionId, text),
		() =>
			apiFetch(
				`/experience/positions/${positionId}/achievements`,
				achievementSchema,
				jsonInit("POST", { text }),
			),
	);
}

export async function updateAchievement(
	id: string,
	text: string,
): Promise<Achievement> {
	return mocked(
		(db) => db.updateAchievement(id, text),
		() =>
			apiFetch(
				`/experience/achievements/${id}`,
				achievementSchema,
				jsonInit("PATCH", { text }),
			),
	);
}

export async function deleteAchievement(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteAchievement(id),
		() => apiFetchVoid(`/experience/achievements/${id}`, { method: "DELETE" }),
	);
}

export async function createVariant(
	achievementId: string,
	input: AchievementVariantInput,
): Promise<AchievementVariant> {
	return mocked(
		(db) => db.createVariant(achievementId, input),
		() =>
			apiFetch(
				`/experience/achievements/${achievementId}/variants`,
				achievementVariantSchema,
				jsonInit("POST", input),
			),
	);
}

export async function updateVariant(
	id: string,
	input: AchievementVariantInput,
): Promise<AchievementVariant> {
	return mocked(
		(db) => db.updateVariant(id, input),
		() =>
			apiFetch(
				`/experience/variants/${id}`,
				achievementVariantSchema,
				jsonInit("PATCH", input),
			),
	);
}

export async function deleteVariant(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteVariant(id),
		() => apiFetchVoid(`/experience/variants/${id}`, { method: "DELETE" }),
	);
}

export async function reorderAchievements(
	positionId: string,
	ids: string[],
): Promise<void> {
	return mocked(
		(db) => db.reorderAchievements(positionId, ids),
		() =>
			apiFetchVoid(
				`/experience/positions/${positionId}/achievements/order`,
				jsonInit("PUT", { ids }),
			),
	);
}
