import { createQuery, queryOptions } from "@tanstack/solid-query";
import {
	createBankSkill,
	deleteBankSkill,
	fetchBankSkills,
	reorderBankSkills,
	updateBankSkill,
} from "../api/bankSkills";
import { keys } from "../api/keys";
import type { BankSkillInput } from "../types/experience";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const bankSkillsQueryOptions = queryOptions({
	queryKey: keys.bankSkills,
	queryFn: fetchBankSkills,
});

export function useBankSkills() {
	return createQuery(() => bankSkillsQueryOptions);
}

const invalidate = [keys.bankSkills] as const;

export function useCreateBankSkill() {
	return useInvalidatingMutation(createBankSkill, invalidate);
}

export function useUpdateBankSkill() {
	return useInvalidatingMutation(
		({ id, input }: { id: string; input: BankSkillInput }) =>
			updateBankSkill(id, input),
		invalidate,
	);
}

export function useDeleteBankSkill() {
	return useInvalidatingMutation(deleteBankSkill, invalidate);
}

export function useReorderBankSkills() {
	return useInvalidatingMutation(reorderBankSkills, invalidate);
}
