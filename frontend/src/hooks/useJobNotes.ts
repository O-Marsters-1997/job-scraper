import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { fetchJobNote, saveJobNote } from "../api/jobNotes";
import { keys } from "../api/keys";

export const jobNoteQueryOptions = (jobId: string) =>
	queryOptions({
		queryKey: keys.jobNote(jobId),
		queryFn: () => fetchJobNote(jobId),
	});

export function useJobNote(jobId: () => string) {
	return createQuery(() => jobNoteQueryOptions(jobId()));
}

export function useSaveJobNote(jobId: () => string) {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (body: string) => saveJobNote(jobId(), body),
		onSuccess: (note) =>
			queryClient.setQueryData(keys.jobNote(note.JobID), note),
	}));
}
