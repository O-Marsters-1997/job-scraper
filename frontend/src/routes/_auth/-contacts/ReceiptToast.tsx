import { createEffect, onCleanup, Show } from "solid-js";
import { firstName } from "@/lib/contactCompanies";
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
						Sent to {firstName(s().contact.name ?? "")}.
					</p>
				</output>
			)}
		</Show>
	);
}
