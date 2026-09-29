import { createMutation } from "@tanstack/solid-query";
import {
	importExperience,
	previewExperienceImport,
} from "../api/experienceImport";
import { keys } from "../api/keys";
import type { ImportPosition } from "../types/experience";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function usePreviewExperienceImport() {
	return createMutation(() => ({
		mutationFn: ({ docId, tabId }: { docId: string; tabId: string }) =>
			previewExperienceImport(docId, tabId),
	}));
}

export function useImportExperience() {
	return useInvalidatingMutation(
		(positions: ImportPosition[]) => importExperience(positions),
		[keys.experience],
	);
}
