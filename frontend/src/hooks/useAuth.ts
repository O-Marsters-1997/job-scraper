import { queryOptions } from "@tanstack/solid-query";
import { fetchMe } from "../api/auth";
import { keys } from "../api/keys";

export const meQueryOptions = queryOptions({
	queryKey: keys.me,
	queryFn: fetchMe,
	staleTime: 5 * 60_000,
	retry: false,
});
