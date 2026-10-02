import { z } from "zod";

export const vapidKeySchema = z.object({ key: z.string() });

export type PushSubscriptionPayload = {
	endpoint: string;
	keys: { p256dh: string; auth: string };
};
