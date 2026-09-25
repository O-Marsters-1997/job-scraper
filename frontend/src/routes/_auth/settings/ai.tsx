import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { AiPrefs } from "../../../api/aiPrefs";
import { useAiPrefs, useUpdateAiCredentials } from "../../../hooks/useAiPrefs";

export const Route = createFileRoute("/_auth/settings/ai")({
	component: AiPage,
});

function AiPage() {
	const query = useAiPrefs();
	return (
		<div class="max-w-2xl px-7 py-6">
			<PageHeading
				title="AI settings"
				subtitle="Connect the OpenRouter key used to score job suitability."
			/>
			<QueryBoundary query={query} fallbackRows={4}>
				{(data) => <AiForm data={data} />}
			</QueryBoundary>
		</div>
	);
}

// Extracted so createSignal initializes from resolved data once at mount —
// background refetches never clobber in-progress edits.
function AiForm(props: { data: AiPrefs }) {
	const credsMutation = useUpdateAiCredentials();

	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const onKeySaved = () => {
		setSaveError(null);
		setSaved(true);
		setTimeout(() => setSaved(false), 3000);
	};
	const onKeyError = (message: string) => setSaveError(message);

	return (
		<>
			<FormFeedback success={saved()} error={saveError()} />

			<div class="flex flex-col gap-5">
				<CredentialCard
					provider="openrouter"
					title="OpenRouter API key"
					description="Used for Jev suitability scoring via OpenRouter."
					placeholder="sk-or-v1-…"
					configured={props.data.configuredProviders.includes("openrouter")}
					mutation={credsMutation}
					onSaved={onKeySaved}
					onError={onKeyError}
				/>
			</div>
		</>
	);
}

function CredentialCard(props: {
	provider: string;
	title: string;
	description: string;
	placeholder: string;
	configured: boolean;
	mutation: ReturnType<typeof useUpdateAiCredentials>;
	onSaved: () => void;
	onError: (message: string) => void;
}) {
	const [apiKey, setApiKey] = createSignal("");

	const handleSaveKey = async () => {
		try {
			await props.mutation.mutateAsync({
				provider: props.provider,
				apiKey: apiKey() || null,
			});
			setApiKey("");
			props.onSaved();
		} catch {
			props.onError("Failed to save API key. Please try again.");
		}
	};

	const handleClearKey = async () => {
		try {
			await props.mutation.mutateAsync({
				provider: props.provider,
				apiKey: null,
			});
		} catch {
			props.onError("Failed to clear API key. Please try again.");
		}
	};

	return (
		<Card class="overflow-hidden">
			<div class="border-b border-border px-5 py-4">
				<p class="text-base font-semibold text-foreground">{props.title}</p>
				<p class="mt-0.5 text-xs text-faint">{props.description}</p>
			</div>
			<div class="px-5 py-4">
				<Show
					when={props.configured}
					fallback={
						<div class="flex items-center gap-3">
							<Input
								type="password"
								aria-label={props.title}
								placeholder={props.placeholder}
								value={apiKey()}
								onInput={(e) => setApiKey(e.currentTarget.value)}
								class="flex-1"
							/>
							<Button
								onClick={handleSaveKey}
								disabled={props.mutation.isPending || !apiKey()}
							>
								{props.mutation.isPending ? "Saving…" : "Save key"}
							</Button>
						</div>
					}
				>
					<div class="flex items-center gap-3">
						<span class="text-sm text-foreground">
							configured <span class="text-primary font-medium">✓</span>
						</span>
						<Button
							variant="outline"
							size="sm"
							onClick={handleClearKey}
							disabled={props.mutation.isPending}
							class="text-xs"
						>
							{props.mutation.isPending ? "Clearing…" : "Clear"}
						</Button>
					</div>
				</Show>
			</div>
		</Card>
	);
}
