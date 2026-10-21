import { type PushSubscriptionPayload, vapidKeySchema } from "../types/push";
import { ApiError, apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchVapidKey(): Promise<string | null> {
	return mocked(
		(db) => db.getVapidKey(),
		() =>
			apiFetch("/push/vapid-public-key", vapidKeySchema).then(
				(v) => v.key,
				(err) => {
					if (err instanceof ApiError && err.status === 404) return null;
					throw err;
				},
			),
	);
}

export async function subscribePush(
	sub: PushSubscriptionPayload,
): Promise<void> {
	return mocked(
		(db) => db.addPushSubscription(sub),
		() => apiFetchVoid("/push/subscriptions", jsonInit("POST", sub)),
	);
}
