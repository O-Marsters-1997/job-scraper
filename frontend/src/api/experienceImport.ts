import {
	type ImportPosition,
	importPreviewSchema,
	type Position,
	positionSchema,
} from "../types/experience";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function previewExperienceImport(
	docId: string,
	tabId: string,
): Promise<ImportPosition[]> {
	return mocked(
		(db) => db.previewExperienceImport(),
		async () => {
			const preview = await apiFetch(
				"/experience/import/preview",
				importPreviewSchema,
				jsonInit("POST", { docId, tabId }),
			);
			return preview.positions;
		},
	);
}

export async function importExperience(
	positions: ImportPosition[],
): Promise<Position[]> {
	return mocked(
		(db) => db.importExperience(positions),
		() =>
			apiFetch(
				"/experience/import",
				positionSchema.array(),
				jsonInit("POST", { positions }),
			),
	);
}
