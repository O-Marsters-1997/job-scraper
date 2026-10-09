import { createQuery } from "@tanstack/solid-query";
import { meQueryOptions } from "./useAuth";

export function useIsAdmin(): () => boolean {
	const me = createQuery(() => meQueryOptions);
	return () => me.data?.isAdmin === true;
}
