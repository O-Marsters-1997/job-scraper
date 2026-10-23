import { API_BASE } from "../api/config";

export const googleWriteHref = (returnTo: string) =>
	`${API_BASE}/google/oauth/start?write=1&return=${encodeURIComponent(returnTo)}`;
