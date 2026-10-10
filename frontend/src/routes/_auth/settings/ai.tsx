import { createFileRoute } from "@tanstack/solid-router";
import type { Accessor } from "solid-js";
import { createSignal, Show } from "solid-js";
import { Field } from "@/components/Field";
import { FormFeedback } from "@/components/FormFeedback";
import { QueryBoundary } from "@/components/QueryBoundary";
import { QuotaBar } from "@/components/QuotaBar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAiPrefs, useUpdateAiCredentials } from "../../../hooks/useAiPrefs";
import { useAiUsage } from "../../../hooks/useAiUsage";
import type { AiPrefs } from "../../../types/aiPrefs";

export const Route = createFileRoute("/_auth/settings/ai")({
	component: AiPage,
});

function AiPage() {
	const query = useAiPrefs();
	return (
		<QueryBoundary query={query} fallbackRows={2}>
			{(data) => <AiForm data={data} />}
		</QueryBoundary>
	);
}

function AiForm(props: { data: Accessor<AiPrefs> }) {
	const mutation = useUpdateAiCredentials();
	const usage = useAiUsage();
	const [apiKey, setApiKey] = createSignal("");
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const save = async (key: string | null, failure: string) => {
		setSaved(false);
		setSaveError(null);
		try {
			await mutation.mutateAsync({ provider: "openrouter", apiKey: key });
			setApiKey("");
			if (key) {
				setSaved(true);
				setTimeout(() => setSaved(false), 3000);
			}
		} catch {
			setSaveError(failure);
		}
	};

	return (
		<>
			<FormFeedback success={saved()} error={saveError()} />
			<Field
				label="OpenRouter API key"
				for="openrouter-key"
				hint="Used to score how well each job fits you."
			>
				<Show
					when={props.data().configuredProviders.includes("openrouter")}
					fallback={
						<form
							class="flex max-w-lg items-center gap-2"
							onSubmit={(e) => {
								e.preventDefault();
								save(apiKey(), "Failed to save API key. Please try again.");
							}}
						>
							<Input
								id="openrouter-key"
								type="password"
								placeholder="sk-or-v1-…"
								value={apiKey()}
								onInput={(e) => setApiKey(e.currentTarget.value)}
								class="flex-1"
							/>
							<Button type="submit" disabled={mutation.isPending || !apiKey()}>
								{mutation.isPending ? "Saving…" : "Save key"}
							</Button>
						</form>
					}
				>
					<div class="flex items-center gap-3">
						<span id="openrouter-key" class="text-sm text-foreground">
							Key saved
						</span>
						<Button
							variant="outline"
							size="sm"
							disabled={mutation.isPending}
							onClick={() =>
								save(null, "Failed to clear API key. Please try again.")
							}
						>
							{mutation.isPending ? "Clearing…" : "Clear"}
						</Button>
					</div>
				</Show>
			</Field>
			<Show when={usage.data?.status === "ok" ? usage.data : undefined}>
				{(quota) => (
					<div class="mt-4">
						<QuotaBar quota={quota()} />
					</div>
				)}
			</Show>
			<Show when={usage.data?.status === "error" ? usage.data : undefined}>
				{(quota) => <p class="mt-4 text-xs text-faint">{quota().error}</p>}
			</Show>
		</>
	);
}
