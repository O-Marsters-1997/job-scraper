import {
	type PlaygroundQueue,
	playgroundQueueSchema,
} from "../types/scoringPlayground";
import { apiFetch } from "./client";
import { mocked } from "./config";

export async function fetchPlaygroundQueue(): Promise<PlaygroundQueue> {
	return mocked(
		(db) => db.playgroundQueue(),
		() => apiFetch("/scoring-playground/queue", playgroundQueueSchema),
	);
}
