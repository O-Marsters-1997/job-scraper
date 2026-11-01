import { keys } from "../api/keys";
import { submitJobUrl } from "../api/submitJobUrl";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useSubmitJobUrl() {
	return useInvalidatingMutation(submitJobUrl, [keys.jobs.all]);
}
