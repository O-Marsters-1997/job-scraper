import { createMutation } from "@tanstack/solid-query";
import {
	applyContext,
	fetchGrillPrompt,
	previewContext,
} from "../api/experienceContext";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useGrillPrompt() {
	return createMutation(() => ({ mutationFn: fetchGrillPrompt }));
}

export function usePreviewContext() {
	return createMutation(() => ({ mutationFn: previewContext }));
}

export function useApplyContext() {
	return useInvalidatingMutation(applyContext, [keys.experience]);
}
