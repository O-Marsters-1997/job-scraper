import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { fetchProfile, updateProfile } from "../api/profile";

export const profileQueryOptions = queryOptions({
	queryKey: ["profile"],
	queryFn: fetchProfile,
});

export function useProfile() {
	return createQuery(() => profileQueryOptions);
}

export function useUpdateProfile() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: { email: string }) => updateProfile(payload),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ["profile"] }),
	}));
}
