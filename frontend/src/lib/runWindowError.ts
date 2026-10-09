import { ApiError } from "@/api/client";

const PREFIX = "run_window: ";

export function runWindowErrorMessage(err: unknown): string | null {
	if (!(err instanceof ApiError) || err.status !== 400) return null;
	const body = err.body as { error?: unknown } | undefined;
	const msg = typeof body?.error === "string" ? body.error : "";
	if (!msg.startsWith(PREFIX)) return null;
	const text = msg.slice(PREFIX.length);
	return `${text.charAt(0).toUpperCase()}${text.slice(1)}.`;
}
