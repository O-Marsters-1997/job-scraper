import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchProfile, updateProfile } from "../api/profile";

export const profileQueryOptions = queryOptions({
	queryKey: keys.profile,
	queryFn: fetchProfile,
});

export function useProfile() {
	return createQuery(() => profileQueryOptions);
}

export function useUpdateProfile() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: { email: string }) => updateProfile(payload),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: keys.profile }),
	}));
}
