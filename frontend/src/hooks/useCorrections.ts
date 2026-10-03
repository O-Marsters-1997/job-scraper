import { revertCorrection, setCorrection } from "../api/corrections";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

const stale = [keys.jobs.all, keys.scores.all] as const;

export function useSetCorrection() {
	return useInvalidatingMutation(setCorrection, stale);
}

export function useRevertCorrection() {
	return useInvalidatingMutation(revertCorrection, stale);
}
