import { createResource, createSignal, Show } from "solid-js";
import { fetchVapidKey, subscribePush } from "../api/push";
import { pushSupported, urlBase64ToBytes } from "../lib/push";
import { Button } from "./ui/button";

async function subscribableKey(): Promise<string | null> {
	if (!pushSupported() || Notification.permission !== "default") return null;
	const key = await fetchVapidKey().catch(() => null);
	if (!key) return null;
	const reg = await navigator.serviceWorker.ready;
	return (await reg.pushManager.getSubscription()) ? null : key;
}

export function PushBanner() {
	const [key] = createResource(subscribableKey);
	const [hidden, setHidden] = createSignal(false);
	const [failed, setFailed] = createSignal(false);

	async function allow(applicationServerKey: string) {
		setFailed(false);
		try {
			if ((await Notification.requestPermission()) !== "granted") {
				setHidden(true);
				return;
			}
			const reg = await navigator.serviceWorker.ready;
			const sub = await reg.pushManager.subscribe({
				userVisibleOnly: true,
				applicationServerKey: urlBase64ToBytes(applicationServerKey),
			});
			const { endpoint, keys } = sub.toJSON();
			try {
				await subscribePush({
					endpoint: endpoint ?? sub.endpoint,
					keys: { p256dh: keys?.p256dh ?? "", auth: keys?.auth ?? "" },
				});
			} catch (err) {
				await sub.unsubscribe();
				throw err;
			}
			setHidden(true);
		} catch {
			setFailed(true);
		}
	}

	return (
		<Show when={!hidden() && key()}>
			{(applicationServerKey) => (
				<section
					aria-label="Alerts"
					class="flex flex-wrap items-center gap-3 border-b border-border bg-surface-muted px-4 py-3"
				>
					<p class="min-w-0 flex-1 text-sm text-foreground">
						Get alerts for new matches
						<Show when={failed()}>
							<span class="ml-2 text-xs text-destructive">
								Could not enable alerts. Try again.
							</span>
						</Show>
					</p>
					<Button size="sm" onClick={() => allow(applicationServerKey())}>
						Allow
					</Button>
					<Button size="sm" variant="ghost" onClick={() => setHidden(true)}>
						Not now
					</Button>
				</section>
			)}
		</Show>
	);
}
