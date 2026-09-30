import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	createSourceTarget,
	fetchSourceTargets,
	rerunSourceTarget,
	updateSourceTarget,
} from "../api/sourceTargets";
import type { UpdateSourceTargetPayload } from "../types/sourceTarget";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

const sourceTargetsQueryOptions = queryOptions({
	queryKey: keys.sourceTargets,
	queryFn: fetchSourceTargets,
	refetchInterval: 5000,
});

export function useSourceTargets() {
	return createQuery(() => sourceTargetsQueryOptions);
}

export function useRerunSourceTarget() {
	return useInvalidatingMutation(rerunSourceTarget, [keys.sourceTargets], {
		onSettled: true,
	});
}

export function useCreateSourceTarget() {
	return useInvalidatingMutation(createSourceTarget, [
		keys.sourceTargets,
		keys.companies.all,
	]);
}

export function useUpdateSourceTarget() {
	return useInvalidatingMutation(
		({ id, ...patch }: { id: string } & UpdateSourceTargetPayload) =>
			updateSourceTarget(id, patch),
		[keys.sourceTargets, keys.companies.all],
	);
}
