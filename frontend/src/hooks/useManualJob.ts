import { keys } from "../api/keys";
import { addManualJob } from "../api/manualJobs";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useAddManualJob() {
	return useInvalidatingMutation(addManualJob, [keys.jobs.all]);
}
