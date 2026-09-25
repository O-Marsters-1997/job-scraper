import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { AiPrefs } from "../../../api/aiPrefs";
import {
	useAiPrefs,
	useUpdateAiCredentials,
	useUpdateAiPrefs,
} from "../../../hooks/useAiPrefs";

export const Route = createFileRoute("/_auth/settings/ai")({
	component: AiPage,
});

const MODEL_LABELS: Record<string, string> = {
	"claude-haiku-4-5-20251001": "Haiku 4.5 — Fast & cheap (default)",
	"claude-sonnet-4-6": "Sonnet 4.6 — Balanced",
	"claude-opus-4-8": "Opus 4.8 — Most capable",
};

function modelLabel(id: string): string {
	return MODEL_LABELS[id] ?? id;
}

function AiPage() {
	const query = useAiPrefs();
	return (
		<div class="max-w-2xl px-7 py-6">
			<PageHeading
				title="AI settings"
				subtitle="Choose which Claude models are used to score and explain job suitability."
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
	const saveMutation = useUpdateAiPrefs();
	const credsMutation = useUpdateAiCredentials();

	const [selectedModel, setSelectedModel] = createSignal(
		props.data.suitabilityModel,
	);
	const [selectedReasoningModel, setSelectedReasoningModel] = createSignal(
		props.data.reasoningModel,
	);
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await saveMutation.mutateAsync({
				suitabilityModel: selectedModel(),
				reasoningModel: selectedReasoningModel(),
			});
			setSaved(true);
			setTimeout(() => setSaved(false), 3000);
		} catch {
			setSaveError("Failed to save. Please try again.");
		}
	};

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
				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">Scoring model</p>
						<p class="mt-0.5 text-xs text-faint">
							The model used at ingest time to score each job (score only, no
							reasoning). Haiku is fast and cheap.
						</p>
					</div>
					<div class="px-5 py-4">
						<fieldset class="flex flex-col gap-3">
							<legend class="sr-only">Select scoring model</legend>
							<For each={props.data.availableModels}>
								{(id) => (
									<label class="flex cursor-pointer items-start gap-3">
										<input
											type="radio"
											name="suitability-model"
											value={id}
											checked={selectedModel() === id}
											onChange={() => setSelectedModel(id)}
											class="mt-0.5 accent-primary"
										/>
										<span class="flex flex-col">
											<span class="text-sm font-medium text-foreground">
												{modelLabel(id)}
											</span>
											<span class="font-mono text-xs text-faint">{id}</span>
										</span>
									</label>
								)}
							</For>
						</fieldset>
					</div>
				</Card>

				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">
							Reasoning model
						</p>
						<p class="mt-0.5 text-xs text-faint">
							The model used when you request an explanation for a job. It
							re-scores the job and writes matched/missing criteria and a
							rationale. Sonnet gives better explanations than Haiku.
						</p>
					</div>
					<div class="px-5 py-4">
						<fieldset class="flex flex-col gap-3">
							<legend class="sr-only">Select reasoning model</legend>
							<For each={props.data.availableModels}>
								{(id) => (
									<label class="flex cursor-pointer items-start gap-3">
										<input
											type="radio"
											name="reasoning-model"
											value={id}
											checked={selectedReasoningModel() === id}
											onChange={() => setSelectedReasoningModel(id)}
											class="mt-0.5 accent-primary"
										/>
										<span class="flex flex-col">
											<span class="text-sm font-medium text-foreground">
												{modelLabel(id)}
											</span>
											<span class="font-mono text-xs text-faint">{id}</span>
										</span>
									</label>
								)}
							</For>
						</fieldset>
					</div>
				</Card>

				<CredentialCard
					provider="anthropic"
					title="Anthropic API key"
					description="Your own key is used for scoring. Leave blank to use the shared key."
					placeholder="sk-ant-…"
					configured={props.data.configuredProviders.includes("anthropic")}
					mutation={credsMutation}
					onSaved={onKeySaved}
					onError={onKeyError}
				/>

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

				<div class="flex items-center gap-3">
					<Button onClick={handleSave} disabled={saveMutation.isPending}>
						{saveMutation.isPending ? "Saving…" : "Save"}
					</Button>
				</div>
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
