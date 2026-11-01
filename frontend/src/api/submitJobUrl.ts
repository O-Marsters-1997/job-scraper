import {
	type SubmitJobUrlResult,
	submitJobUrlResultSchema,
} from "../types/submitJobUrl";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function submitJobUrl(url: string): Promise<SubmitJobUrlResult> {
	return mocked(
		(db) => db.submitJobUrl(url),
		() =>
			apiFetch(
				"/jobs/submit",
				submitJobUrlResultSchema,
				jsonInit("POST", { url }),
			),
	);
}
