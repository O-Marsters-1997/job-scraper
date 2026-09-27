import { createFileRoute } from "@tanstack/solid-router";
import { createSignal } from "solid-js";
import { Field } from "@/components/Field";
import { FormFeedback } from "@/components/FormFeedback";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SettingsActions } from "@/components/SettingsLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { Profile } from "../../../api/profile";
import { useProfile, useUpdateProfile } from "../../../hooks/useProfile";

export const Route = createFileRoute("/_auth/settings/profile")({
	component: ProfilePage,
});

function ProfilePage() {
	const query = useProfile();
	return (
		<QueryBoundary query={query} fallbackRows={3}>
			{(data) => <ProfileForm data={data} />}
		</QueryBoundary>
	);
}

// Extracted so createSignal initializes from resolved data once at mount —
// background refetches never clobber in-progress edits.
function ProfileForm(props: { data: Profile }) {
	const mutation = useUpdateProfile();
	const [email, setEmail] = createSignal(props.data.email);
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await mutation.mutateAsync({ email: email() });
			setSaved(true);
			setTimeout(() => setSaved(false), 3000);
		} catch {
			setSaveError("Failed to save. Please try again.");
		}
	};

	return (
		<>
			<SettingsActions>
				<Button size="sm" onClick={handleSave} disabled={mutation.isPending}>
					{mutation.isPending ? "Saving…" : "Save"}
				</Button>
			</SettingsActions>
			<FormFeedback success={saved()} error={saveError()} />
			<div>
				<p class="text-sm font-medium text-foreground">Username</p>
				<p class="mt-2 text-sm text-muted">{props.data.username}</p>
			</div>
			<Field label="Email" for="email">
				<Input
					id="email"
					type="email"
					value={email()}
					onInput={(e) => setEmail(e.currentTarget.value)}
					class="max-w-sm"
				/>
			</Field>
		</>
	);
}
