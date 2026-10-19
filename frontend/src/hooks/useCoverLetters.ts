import { createQuery } from "@tanstack/solid-query";
import type { Accessor } from "solid-js";
import { createCoverLetter, fetchCoverLetters } from "../api/coverLetters";
import { keys } from "../api/keys";
import { fetchSuggestions } from "../api/tailoring";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useCoverLetters(jobId: Accessor<string>) {
	return createQuery(() => ({
		queryKey: keys.coverLetters(jobId()),
		queryFn: () => fetchCoverLetters(jobId()),
		retry: false,
	}));
}

export function useCoverLetterAchievements(jobId: Accessor<string>) {
	return createQuery(() => ({
		queryKey: keys.tailoring.suggestions(jobId(), "", ""),
		queryFn: () => fetchSuggestions(jobId(), "", ""),
		retry: false,
		staleTime: 5 * 60 * 1000,
	}));
}

export function useCreateCoverLetter() {
	return useInvalidatingMutation(createCoverLetter, (letter) => [
		keys.coverLetters(letter.jobId),
	]);
}
