import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { ApiError } from "../../../../api/client";
import {
	useCreateCustomOption,
	useDeleteCustomOption,
	useDraftCustomOption,
	useUpdateCustomOption,
} from "../../../../hooks/useCustomOptions";
import type {
	CustomOption,
	CustomOptionDraft,
	CustomOptionDraftInput,
} from "../../../../types/customOption";
import type { Stance } from "../../../../types/scoringConfig";
import type { ScoringOptionsView } from "../../../../types/scoringOptions";
import { StanceRow } from "./StanceRow";

const ANSWER_TONE: Record<
	CustomOptionDraft["answers"][number]["resolved"],
	string
> = {
	yes: "text-accent-text",
	no: "text-muted",
	unknown: "text-faint",
};

function errorMessage(err: unknown): string {
	if (err instanceof ApiError) {
		const body = err.body as { error?: unknown } | undefined;
		if (typeof body?.error === "string") return body.error;
	}
	return "Something went wrong. Please try again.";
}

export function CustomOptions(props: {
	options: ScoringOptionsView;
	stanceOf: (id: string) => Stance | undefined;
	onRate: (id: string, stance: Stance | undefined) => void;
}) {
	const draftMutation = useDraftCustomOption();
	const createMutation = useCreateCustomOption();
	const updateMutation = useUpdateCustomOption();
	const deleteMutation = useDeleteCustomOption();

	const [phrase, setPhrase] = createSignal("");
	const [draft, setDraft] = createSignal<CustomOptionDraft | null>(null);
	const [editing, setEditing] = createSignal<string | null>(null);
	const [error, setError] = createSignal<string | null>(null);

	const atCap = () => props.options.custom.length >= props.options.customCap;
	const busy = () => createMutation.isPending || updateMutation.isPending;
	const allowed = (o: CustomOption) =>
		props.options.dimensions.find((d) => d.key === o.dimension)?.stances ?? [];

	let latestDraft = 0;
	const run = async (input: CustomOptionDraftInput) => {
		setError(null);
		const mine = ++latestDraft;
		try {
			const result = await draftMutation.mutateAsync(input);
			if (mine === latestDraft) setDraft(result);
		} catch (err) {
			if (mine === latestDraft) setError(errorMessage(err));
		}
	};

	const reset = () => {
		latestDraft++;
		setPhrase("");
		setDraft(null);
		setEditing(null);
		setError(null);
	};

	const confirm = async (d: CustomOptionDraft) => {
		setError(null);
		const input = {
			label: phrase().trim(),
			question: d.question.trim(),
			dimension: d.dimension,
		};
		try {
			const id = editing();
			if (id) await updateMutation.mutateAsync({ id, input });
			else await createMutation.mutateAsync(input);
			reset();
		} catch (err) {
			setError(errorMessage(err));
		}
	};

	const remove = async (id: string) => {
		setError(null);
		try {
			await deleteMutation.mutateAsync(id);
			props.onRate(id, undefined);
			if (editing() === id) reset();
		} catch (err) {
			setError(errorMessage(err));
		}
	};

	const edit = (o: CustomOption) => {
		setPhrase(o.label);
		setEditing(o.id);
		setDraft(null);
		void run({
			phrase: o.label,
			question: o.question,
			dimension: o.dimension,
		});
	};

	return (
		<div class="flex flex-col gap-6">
			<FormFeedback error={error()} />
			<div class="flex flex-col">
				<For each={props.options.custom}>
					{(o) => (
						<div class="border-b border-border last:border-b-0">
							<StanceRow
								option={o}
								stance={props.stanceOf(o.id)}
								allowed={allowed(o)}
								onChange={(s) => props.onRate(o.id, s)}
							/>
							<div class="-mt-1 flex items-center gap-2 pb-3">
								<Badge variant="outline">{o.dimension}</Badge>
								<p class="min-w-0 flex-1 truncate text-xs text-faint">
									{o.question}
								</p>
								<Button
									size="sm"
									variant="ghost"
									disabled={draftMutation.isPending || deleteMutation.isPending}
									onClick={() => edit(o)}
								>
									Edit
								</Button>
								<Button
									size="sm"
									variant="ghost"
									loading={
										deleteMutation.isPending &&
										deleteMutation.variables === o.id
									}
									onClick={() => remove(o.id)}
								>
									Remove
								</Button>
							</div>
						</div>
					)}
				</For>
			</div>

			<form
				class="flex flex-col gap-3 rounded-md border border-border bg-surface-muted p-4"
				onSubmit={(e) => {
					e.preventDefault();
					setEditing(null);
					void run({ phrase: phrase().trim() });
				}}
			>
				<div class="flex items-baseline justify-between gap-3">
					<label
						for="custom-phrase"
						class="text-sm font-medium text-foreground"
					>
						{editing() ? "Edit option" : "Add your own"}
					</label>
					<span class="font-mono text-xs tabular-nums text-faint">
						{props.options.custom.length} of {props.options.customCap}
					</span>
				</div>
				<p class="-mt-2 text-xs text-faint">
					Describe it in a few words. We'll write the question we ask about each
					job and show how it answers on your recent jobs.
				</p>
				<div class="flex flex-col gap-2 sm:flex-row">
					<Input
						id="custom-phrase"
						value={phrase()}
						onInput={(e) => {
							setPhrase(e.currentTarget.value);
							setDraft(null);
						}}
						placeholder="e.g. Building AI agents"
						disabled={atCap() || editing() !== null}
					/>
					<Button
						type="submit"
						size="sm"
						variant="outline"
						class="sm:h-9"
						loading={draftMutation.isPending}
						disabled={atCap() || editing() !== null || phrase().trim() === ""}
					>
						Draft question
					</Button>
				</div>
				<Show when={atCap() && !editing()}>
					<p class="text-xs text-faint">
						You're at the limit. Remove one to add another.
					</p>
				</Show>

				<Show when={draft()}>
					{(d) => (
						<div class="flex flex-col gap-3 border-t border-border pt-3">
							<div class="flex flex-col gap-1">
								<label
									for="custom-question"
									class="text-xs font-semibold uppercase tracking-wide text-faint"
								>
									Question
								</label>
								<Textarea
									id="custom-question"
									value={d().question}
									onInput={(e) =>
										setDraft({ ...d(), question: e.currentTarget.value })
									}
									class="min-h-16"
								/>
								<p class="text-xs text-faint">
									Counts towards <span class="text-muted">{d().dimension}</span>
								</p>
							</div>
							<div>
								<p class="text-xs font-semibold uppercase tracking-wide text-faint">
									On your recent jobs
								</p>
								<Show
									when={d().answers.length > 0}
									fallback={
										<p class="mt-1 text-xs text-faint">No scored jobs yet.</p>
									}
								>
									<ul class="mt-1 flex flex-col">
										<For each={d().answers}>
											{(a) => (
												<li class="flex items-center justify-between gap-3 border-b border-border py-1.5 text-sm last:border-b-0">
													<span class="min-w-0 truncate text-foreground">
														{a.title}
														<span class="text-faint"> · {a.company}</span>
													</span>
													<span
														class={cn(
															"shrink-0 text-xs",
															ANSWER_TONE[a.resolved],
														)}
													>
														{a.resolved}
													</span>
												</li>
											)}
										</For>
									</ul>
								</Show>
							</div>
							<div class="flex gap-2">
								<Button
									type="button"
									size="sm"
									variant="outline"
									loading={draftMutation.isPending}
									disabled={d().question.trim() === ""}
									onClick={() =>
										run({
											phrase: phrase().trim(),
											question: d().question.trim(),
											dimension: d().dimension,
										})
									}
								>
									Preview answers
								</Button>
								<Button
									type="button"
									size="sm"
									loading={busy()}
									disabled={d().question.trim() === ""}
									onClick={() => confirm(d())}
								>
									{editing() ? "Save changes" : "Add option"}
								</Button>
								<Button type="button" size="sm" variant="ghost" onClick={reset}>
									Discard
								</Button>
							</div>
						</div>
					)}
				</Show>
			</form>
		</div>
	);
}
