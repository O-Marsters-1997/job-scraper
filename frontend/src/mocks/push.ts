import type { PushSubscriptionPayload } from "../types/push";

const subscriptions = new Map<string, PushSubscriptionPayload>();

export function getVapidKey(): string | null {
	return globalThis.localStorage?.getItem("mock-push") === "1"
		? "mock-vapid-key"
		: null;
}

export function addPushSubscription(sub: PushSubscriptionPayload): void {
	subscriptions.set(sub.endpoint, sub);
}

export function removePushSubscription(endpoint: string): void {
	subscriptions.delete(endpoint);
}
