import { type ProxyUsage, proxyUsageSchema } from "../types/quota";
import { apiFetch } from "./client";
import { mocked } from "./config";

export async function fetchProxyUsage(): Promise<ProxyUsage> {
	return mocked(
		(db) => db.getProxyUsage(),
		() => apiFetch("/usage/proxies", proxyUsageSchema),
	);
}
