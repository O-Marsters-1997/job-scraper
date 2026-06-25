import { createFileRoute } from "@tanstack/solid-router";
import { createEffect, createSignal, For, Show } from "solid-js";
import { SkeletonList } from "@/components/ui/skeleton";
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
	const saveMutation = useUpdateAiPrefs();
	const credsMutation = useUpdateAiCredentials();

	const [selectedModel, setSelectedModel] = createSignal("");
	const [apiKey, setApiKey] = createSignal("");
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const anthropicConfigured = () =>
		query.data?.configuredProviders.includes("anthropic") ?? false;

	createEffect(() => {
		const data = query.data;
		if (!data) return;
		setSelectedModel(data.suitabilityModel ?? "");
	});

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await saveMutation.mutateAsync({ suitabilityModel: selectedModel() });
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
		<div class="max-w-2xl px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					AI settings
				</h1>
				<p class="mt-0.5 text-xs text-faint">
					Choose which Claude model is used to score job suitability.
				</p>
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

			<Show when={query.isPending}>
				<SkeletonList rows={4} />
			</Show>

			<Show when={query.isSuccess}>
				<div class="flex flex-col gap-5">
					{/* Model picker */}
					<div class="overflow-hidden rounded-xl border border-border bg-surface">
						<div class="border-b border-border px-5 py-4">
							<p class="text-base font-semibold text-foreground">
								Suitability model
							</p>
							<p class="mt-0.5 text-xs text-faint">
								The model used when Claude scores each job against your rubric.
							</p>
						</div>
						<div class="px-5 py-4">
							<fieldset class="flex flex-col gap-3">
								<legend class="sr-only">Select suitability model</legend>
								<For each={query.data?.availableModels ?? []}>
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

							<p class="mt-4 text-xs text-faint">
								The default model (Haiku) works out of the box. Only change this
								if you want higher-quality scoring at higher cost.
							</p>
						</div>
					</div>

					{/* API key */}
					<div class="overflow-hidden rounded-xl border border-border bg-surface">
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
										<input
											type="password"
											placeholder="sk-ant-…"
											value={apiKey()}
											onInput={(e) => setApiKey(e.currentTarget.value)}
											class="flex-1 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/10"
										/>
										<button
											type="button"
											onClick={handleSaveKey}
											disabled={credsMutation.isPending || !apiKey()}
											class="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition hover:bg-primary-hover disabled:opacity-50"
										>
											{credsMutation.isPending ? "Saving…" : "Save key"}
										</button>
									</div>
								}
							>
								<div class="flex items-center gap-3">
									<span class="text-sm text-foreground">
										configured{" "}
										<span class="text-primary font-medium">✓</span>
									</span>
									<button
										type="button"
										onClick={handleClearKey}
										disabled={credsMutation.isPending}
										class="inline-flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1.5 text-xs font-medium text-muted transition hover:border-border-strong hover:text-foreground disabled:opacity-50"
									>
										{credsMutation.isPending ? "Clearing…" : "Clear"}
									</button>
								</div>
							</Show>
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
			</Show>
		</div>
	);
}
