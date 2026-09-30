import { createQuery, queryOptions } from "@tanstack/solid-query";
import {
	createAchievement,
	createPosition,
	deleteAchievement,
	deletePosition,
	fetchExperience,
	reorderAchievements,
	reorderPositions,
	updateAchievement,
	updatePosition,
} from "../api/experience";
import { keys } from "../api/keys";
import type { PositionInput } from "../types/experience";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const experienceQueryOptions = queryOptions({
	queryKey: keys.experience,
	queryFn: fetchExperience,
});

export function useExperience() {
	return createQuery(() => experienceQueryOptions);
}

const invalidate = [keys.experience] as const;

export function useCreatePosition() {
	return useInvalidatingMutation(createPosition, invalidate);
}

export function useUpdatePosition() {
	return useInvalidatingMutation(
		({ id, input }: { id: string; input: PositionInput }) =>
			updatePosition(id, input),
		invalidate,
	);
}

export function useDeletePosition() {
	return useInvalidatingMutation(deletePosition, invalidate);
}

export function useReorderPositions() {
	return useInvalidatingMutation(reorderPositions, invalidate);
}

export function useCreateAchievement() {
	return useInvalidatingMutation(
		({ positionId, text }: { positionId: string; text: string }) =>
			createAchievement(positionId, text),
		invalidate,
	);
}

export function useUpdateAchievement() {
	return useInvalidatingMutation(
		({ id, text }: { id: string; text: string }) => updateAchievement(id, text),
		invalidate,
	);
}

export function useDeleteAchievement() {
	return useInvalidatingMutation(deleteAchievement, invalidate);
}

export function useReorderAchievements() {
	return useInvalidatingMutation(
		({ positionId, ids }: { positionId: string; ids: string[] }) =>
			reorderAchievements(positionId, ids),
		invalidate,
	);
}
