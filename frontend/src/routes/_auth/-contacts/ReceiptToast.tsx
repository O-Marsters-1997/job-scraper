import { createEffect, onCleanup, Show } from "solid-js";
import { formatChaseDate } from "@/lib/chase";
import type { ContactSent } from "@/types/contact";

export function ReceiptToast(props: {
	sent: ContactSent | undefined;
	onDismiss: () => void;
}) {
	createEffect(() => {
		if (!props.sent) return;
		const timer = setTimeout(props.onDismiss, 8000);
		onCleanup(() => clearTimeout(timer));
	});
	return (
		<Show when={props.sent}>
			{(s) => (
				<output class="fixed bottom-6 right-6 z-50 max-w-sm rounded-lg border border-border bg-surface p-4 text-sm shadow-lg">
					<p class="font-medium text-foreground">
						Sent to {s().contact.name}.
						<Show when={s().contact.follow_up_by}>
							{(day) => <> Next chase {formatChaseDate(day())}</>}
						</Show>
					</p>
					<Show when={s().moved_application_chase}>
						<p class="mt-1 text-xs text-muted">
							The Application chase moved too.
						</p>
					</Show>
				</output>
			)}
		</Show>
	);
}
