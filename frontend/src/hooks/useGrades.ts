import { createQuery } from "@tanstack/solid-query";
import { clearGrade, fetchGrade, setGrade } from "../api/grades";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useGrade(jobId: () => string) {
	return createQuery(() => ({
		queryKey: keys.grade(jobId()),
		queryFn: () => fetchGrade(jobId()),
	}));
}

export function useSetGrade() {
	return useInvalidatingMutation(setGrade, (grade) => [
		keys.grade(grade.jobId),
		keys.jobs.all,
	]);
}

export function useClearGrade() {
	return useInvalidatingMutation(clearGrade, [["grade"], keys.jobs.all]);
}
