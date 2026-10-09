import { createQuery } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchProxyUsage } from "../api/proxyUsage";
import { useIsAdmin } from "./useIsAdmin";

export function useProxyUsage() {
	const isAdmin = useIsAdmin();
	return createQuery(() => ({
		queryKey: keys.proxyUsage,
		queryFn: fetchProxyUsage,
		staleTime: 60 * 1000,
		enabled: isAdmin(),
	}));
}
