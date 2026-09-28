import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchProfile, updateProfile } from "../api/profile";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const profileQueryOptions = queryOptions({
	queryKey: keys.profile,
	queryFn: fetchProfile,
});

export function useProfile() {
	return createQuery(() => profileQueryOptions);
}

export function useUpdateProfile() {
	return useInvalidatingMutation(
		(payload: { email: string }) => updateProfile(payload),
		[keys.profile],
	);
}
