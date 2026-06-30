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
	const [apiKey, setApiKey] = createSignal("");
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const anthropicConfigured = () =>
		props.data.configuredProviders.includes("anthropic");

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

	const handleSaveKey = async () => {
		setSaveError(null);
		try {
			await credsMutation.mutateAsync({
				provider: "anthropic",
				apiKey: apiKey() || null,
			});
			setApiKey("");
			setSaved(true);
			setTimeout(() => setSaved(false), 3000);
		} catch {
			setSaveError("Failed to save API key. Please try again.");
		}
	};

	const handleClearKey = async () => {
		setSaveError(null);
		try {
			await credsMutation.mutateAsync({ provider: "anthropic", apiKey: null });
		} catch {
			setSaveError("Failed to clear API key. Please try again.");
		}
	};

	return (
		<>
			<FormFeedback success={saved()} error={saveError()} />

			<div class="flex flex-col gap-5">
				{/* Scoring model picker */}
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

				{/* Reasoning model picker */}
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

				{/* API key */}
				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">
							Anthropic API key
						</p>
						<p class="mt-0.5 text-xs text-faint">
							Your own key is used for scoring. Leave blank to use the shared
							key.
						</p>
					</div>
					<div class="px-5 py-4">
						<Show
							when={anthropicConfigured()}
							fallback={
								<div class="flex items-center gap-3">
									<Input
										type="password"
										aria-label="Anthropic API key"
										placeholder="sk-ant-…"
										value={apiKey()}
										onInput={(e) => setApiKey(e.currentTarget.value)}
										class="flex-1"
									/>
									<Button
										onClick={handleSaveKey}
										disabled={credsMutation.isPending || !apiKey()}
									>
										{credsMutation.isPending ? "Saving…" : "Save key"}
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
									disabled={credsMutation.isPending}
									class="text-xs"
								>
									{credsMutation.isPending ? "Clearing…" : "Clear"}
								</Button>
							</div>
						</Show>
					</div>
				</Card>

				<div class="flex items-center gap-3">
					<Button onClick={handleSave} disabled={saveMutation.isPending}>
						{saveMutation.isPending ? "Saving…" : "Save"}
					</Button>
				</div>
			</div>
		</>
	);
}
