import { createFileRoute } from "@tanstack/solid-router";
import { createSignal } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { Profile } from "../../../api/profile";
import { useProfile, useUpdateProfile } from "../../../hooks/useProfile";

export const Route = createFileRoute("/_auth/settings/profile")({
	component: ProfilePage,
});

function ProfilePage() {
	const query = useProfile();
	return (
		<div class="max-w-2xl px-7 py-6">
			<PageHeading title="Profile" subtitle="Manage your account details." />
			<QueryBoundary query={query} fallbackRows={3}>
				{(data) => <ProfileForm data={data} />}
			</QueryBoundary>
		</div>
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
			<FormFeedback success={saved()} error={saveError()} />

			<div class="flex flex-col gap-5">
				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">
							Account details
						</p>
					</div>
					<div class="divide-y divide-border">
						<div class="px-5 py-4">
							<label for="username" class="text-xs font-medium text-foreground">
								Username
							</label>
							<p class="mt-2 text-sm text-muted">{props.data.username}</p>
						</div>
						<div class="px-5 py-4">
							<label for="email" class="text-xs font-medium text-foreground">
								Email
							</label>
							<Input
								id="email"
								type="email"
								value={email()}
								onInput={(e) => setEmail(e.currentTarget.value)}
								class="mt-2"
							/>
						</div>
					</div>
				</Card>

				<div class="flex items-center gap-3">
					<Button onClick={handleSave} disabled={mutation.isPending}>
						{mutation.isPending ? "Saving…" : "Save"}
					</Button>
				</div>
			</div>
		</>
	);
}
