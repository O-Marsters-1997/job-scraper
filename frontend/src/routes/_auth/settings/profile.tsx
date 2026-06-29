import { createFileRoute } from "@tanstack/solid-router";
import { createEffect, createSignal, Show } from "solid-js";
import { QueryBoundary } from "@/components/QueryBoundary";
import { useProfile, useUpdateProfile } from "../../../hooks/useProfile";

export const Route = createFileRoute("/_auth/settings/profile")({
	component: ProfilePage,
});

function ProfilePage() {
	const query = useProfile();
	const saveMutation = useUpdateProfile();

	const [email, setEmail] = createSignal("");
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	createEffect(() => {
		const data = query.data;
		if (!data) return;
		setEmail(data.email ?? "");
	});

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await saveMutation.mutateAsync({ email: email() });
			setSaved(true);
			setTimeout(() => setSaved(false), 3000);
		} catch {
			setSaveError("Failed to save. Please try again.");
		}
	};

	return (
		<div class="max-w-2xl px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Profile
				</h1>
				<p class="mt-0.5 text-xs text-faint">Manage your account details.</p>
			</div>

			<Show when={saved()}>
				<div class="mb-4 rounded-lg border border-primary/30 bg-accent-subtle px-4 py-3 text-sm text-primary">
					Saved.
				</div>
			</Show>

			<Show when={saveError()}>
				<div class="mb-4 rounded-lg border border-destructive/30 bg-destructive-subtle px-4 py-3 text-sm text-destructive-strong">
					{saveError()}
				</div>
			</Show>

			<QueryBoundary query={query} fallbackRows={3}>
				{(data) => (
					<div class="flex flex-col gap-5">
						<div class="overflow-hidden rounded-xl border border-border bg-surface">
							<div class="border-b border-border px-5 py-4">
								<p class="text-base font-semibold text-foreground">
									Account details
								</p>
							</div>
							<div class="divide-y divide-border">
								<div class="px-5 py-4">
									<label
										for="username"
										class="text-xs font-medium text-foreground"
									>
										Username
									</label>
									<p class="mt-2 text-sm text-muted">{data.username}</p>
								</div>
								<div class="px-5 py-4">
									<label
										for="email"
										class="text-xs font-medium text-foreground"
									>
										Email
									</label>
									<input
										id="email"
										type="email"
										value={email()}
										onInput={(e) => setEmail(e.currentTarget.value)}
										class="mt-2 w-full rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/10"
									/>
								</div>
							</div>
						</div>

						<div class="flex items-center gap-3">
							<button
								type="button"
								onClick={handleSave}
								disabled={saveMutation.isPending}
								class="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition hover:bg-primary-hover disabled:opacity-50"
							>
								{saveMutation.isPending ? "Saving…" : "Save"}
							</button>
						</div>
					</div>
				)}
			</QueryBoundary>
		</div>
	);
}
