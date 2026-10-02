export function urlBase64ToBytes(value: string): Uint8Array<ArrayBuffer> {
	const padded = value
		.replace(/-/g, "+")
		.replace(/_/g, "/")
		.padEnd(Math.ceil(value.length / 4) * 4, "=");
	return Uint8Array.from(atob(padded), (c) => c.charCodeAt(0));
}

export const pushSupported = () =>
	"serviceWorker" in navigator &&
	"PushManager" in window &&
	"Notification" in window;
