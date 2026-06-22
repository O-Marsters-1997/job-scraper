import { createQuery, createMutation, useQueryClient, queryOptions } from "@tanstack/solid-query";
import { fetchGoogleStatus, disconnectGoogle } from "../api/google";

export const googleStatusQueryOptions = queryOptions({
  queryKey: ["google-status"],
  queryFn: fetchGoogleStatus,
  retry: false,
});

export function useGoogleStatus() {
  return createQuery(() => googleStatusQueryOptions);
}

export function useDisconnectGoogle() {
  const qc = useQueryClient();
  return createMutation(() => ({
    mutationFn: disconnectGoogle,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["google-status"] }),
  }));
}
