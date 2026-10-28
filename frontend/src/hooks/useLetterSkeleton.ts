import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchLetterSkeleton, saveLetterSkeleton } from "../api/letterSkeleton";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

const letterSkeletonQueryOptions = queryOptions({
	queryKey: keys.letterSkeleton,
	queryFn: fetchLetterSkeleton,
});

export function useLetterSkeleton() {
	return createQuery(() => letterSkeletonQueryOptions);
}

export function useSaveLetterSkeleton() {
	return useInvalidatingMutation(saveLetterSkeleton, [keys.letterSkeleton]);
}
