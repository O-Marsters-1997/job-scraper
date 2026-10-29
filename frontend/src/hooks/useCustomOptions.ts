import { createMutation } from "@tanstack/solid-query";
import {
	createCustomOption,
	deleteCustomOption,
	draftCustomOption,
	updateCustomOption,
} from "../api/customOptions";
import { keys } from "../api/keys";
import type { CustomOptionInput } from "../types/customOption";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

const invalidates = [keys.scoringOptions, keys.scores.status()] as const;

export function useDraftCustomOption() {
	return createMutation(() => ({ mutationFn: draftCustomOption }));
}

export function useCreateCustomOption() {
	return useInvalidatingMutation(createCustomOption, invalidates);
}

export function useUpdateCustomOption() {
	return useInvalidatingMutation(
		(v: { id: string; input: CustomOptionInput }) =>
			updateCustomOption(v.id, v.input),
		invalidates,
	);
}

export function useDeleteCustomOption() {
	return useInvalidatingMutation(deleteCustomOption, invalidates);
}
