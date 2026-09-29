import {
	type ImportPosition,
	importPreviewSchema,
	type Position,
	positionSchema,
} from "../types/experience";
import { apiFetch } from "./client";
import { mockDelay, mocked } from "./config";
import { jsonInit } from "./experience";

export async function previewExperienceImport(
	docId: string,
	tabId: string,
): Promise<ImportPosition[]> {
	return mocked(
		async (db) => {
			await mockDelay(200);
			return db.previewExperienceImport();
		},
		async () => {
			const preview = await apiFetch(
				"/experience/import/preview",
				jsonInit("POST", { docId, tabId }),
				importPreviewSchema,
			);
			return preview.positions;
		},
	);
}

export async function importExperience(
	positions: ImportPosition[],
): Promise<Position[]> {
	return mocked(
		async (db) => {
			await mockDelay(120);
			return db.importExperience(positions);
		},
		() =>
			apiFetch(
				"/experience/import",
				jsonInit("POST", { positions }),
				positionSchema.array(),
			),
	);
}
