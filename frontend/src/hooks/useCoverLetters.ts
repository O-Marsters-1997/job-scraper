import { createQuery } from "@tanstack/solid-query";
import type { Accessor } from "solid-js";
import { createCoverLetter, fetchCoverLetters } from "../api/coverLetters";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useCoverLetters(jobId: Accessor<string>) {
	return createQuery(() => ({
		queryKey: keys.coverLetters(jobId()),
		queryFn: () => fetchCoverLetters(jobId()),
		retry: false,
	}));
}

export function useCreateCoverLetter() {
	return useInvalidatingMutation(createCoverLetter, (letter) => [
		keys.coverLetters(letter.jobId),
	]);
}
