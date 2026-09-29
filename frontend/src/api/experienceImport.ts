import {
	type ImportPosition,
	importPreviewSchema,
	type Position,
	positionSchema,
} from "../types/experience";
import { apiFetch } from "./client";
import { mockDelay, mocked } from "./config";

const post = (body: unknown): RequestInit => ({
	method: "POST",
	headers: { "Content-Type": "application/json" },
	body: JSON.stringify(body),
});

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
				post({ docId, tabId }),
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
				post({ positions }),
				positionSchema.array(),
			),
	);
}
