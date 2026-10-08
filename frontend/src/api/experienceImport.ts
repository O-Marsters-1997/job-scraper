import {
	type ImportPosition,
	type ImportPreview,
	type ImportSkill,
	importPreviewSchema,
	type Position,
	positionSchema,
} from "../types/experience";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function previewExperienceImport(
	docId: string,
	tabId: string,
): Promise<ImportPreview> {
	return mocked(
		(db) => db.previewExperienceImport(),
		() =>
			apiFetch(
				"/experience/import/preview",
				importPreviewSchema,
				jsonInit("POST", { docId, tabId }),
			),
	);
}

export async function importExperience(
	positions: ImportPosition[],
	skills: ImportSkill[],
): Promise<Position[]> {
	return mocked(
		(db) => db.importExperience(positions, skills),
		() =>
			apiFetch(
				"/experience/import",
				positionSchema.array(),
				jsonInit("POST", { positions, skills }),
			),
	);
}
