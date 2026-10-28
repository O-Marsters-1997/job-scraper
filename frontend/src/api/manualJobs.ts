import {
	type AddManualJobInput,
	type ManualJobResult,
	manualJobResultSchema,
} from "../types/manualJob";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function addManualJob(
	input: AddManualJobInput,
): Promise<ManualJobResult> {
	return mocked(
		(db) => db.addManualJob(input),
		() =>
			apiFetch("/jobs/manual", manualJobResultSchema, jsonInit("POST", input)),
	);
}
