import { Show } from "solid-js";
import { API_BASE } from "../api/config";
import { useGoogleStatus } from "../hooks/useGoogle";

type GoogleWriteConsentProps = {
	returnTo: string;
};

/** Renders only when the user's Google link lacks write access. */
export function GoogleWriteConsent(props: GoogleWriteConsentProps) {
	const status = useGoogleStatus();
	const href = () =>
		`${API_BASE}/google/oauth/start?write=1&return=${encodeURIComponent(props.returnTo)}`;

	return (
		<Show when={status.data && !status.data.canWrite}>
			<div class="rounded-md border border-border bg-surface-muted p-4">
				<p class="text-sm font-medium text-foreground">
					Allow FastTrack to create Google Docs
				</p>
				<p class="mt-1 text-xs text-faint">
					Tailoring copies your CV into a new Doc. FastTrack can only touch
					files it creates, never the rest of your Drive.
				</p>
				<a
					href={href()}
					class="mt-3 inline-flex h-8 items-center rounded-md bg-primary px-3 text-xs font-medium text-primary-foreground transition hover:bg-primary-hover"
				>
					Grant access
				</a>
			</div>
		</Show>
	);
}
