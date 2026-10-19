import { createMemo, createSignal, For, Show } from "solid-js";
import { ApiError } from "@/api/client";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import {
	useCheckCoverLetter,
	useCoverLetterAchievements,
	useCoverLetters,
	useCreateCoverLetter,
	useMarkCoverLetterDone,
} from "@/hooks/useCoverLetters";
import { useCVTemplates } from "@/hooks/useCVTemplates";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { formatDate } from "@/lib/datetime";
import { loadedData } from "@/lib/queryData";
import type { CoverLetter, LetterFinding } from "@/types/coverLetter";

const TOP_ACHIEVEMENTS = 8;
const checkbox = "size-4 cursor-pointer align-middle accent-primary";

const findingLabels: Record<LetterFinding["kind"], string> = {
	leftover_bullet: "Leftover bullet",
	leftover_gap: "Leftover gap",
	length: "Length",
	ai_tell: "AI phrase",
	restates_achievement: "Restates an achievement",
	missing_company: "Company not named",
	missing_role: "Role not named",
};

function errorMessage(err: unknown): string {
	if (err instanceof ApiError) {
		const msg = (err.body as { error?: string } | undefined)?.error;
		if (msg) return msg;
	}
	return "Could not write the cover letter. Please try again.";
}

function LetterRow(props: { letter: CoverLetter }) {
	const check = useCheckCoverLetter();
	const done = useMarkCoverLetterDone();

	return (
		<div class="grid gap-1.5 rounded-md px-2 py-1.5 text-xs">
			<div class="flex items-center justify-between gap-3">
				<span class="font-mono tabular-nums text-faint">
					{formatDate(props.letter.createdAt)}
				</span>
				<a
					href={props.letter.tabUrl}
					target="_blank"
					rel="noreferrer"
					class="text-foreground hover:underline"
				>
					Open tab
				</a>
				<Button
					variant="outline"
					size="sm"
					onClick={() => check.mutate(props.letter.id)}
					disabled={check.isPending}
				>
					{check.isPending ? "Checking…" : "Check letter"}
				</Button>
				<Show
					when={props.letter.doneAt}
					fallback={
						<Button
							variant="outline"
							size="sm"
							onClick={() => done.mutate(props.letter.id)}
							disabled={done.isPending}
						>
							Mark done
						</Button>
					}
				>
					{(at) => <span class="text-muted">Done {formatDate(at())}</span>}
				</Show>
			</div>
			<Show when={check.isError || done.isError}>
				<p class="text-destructive">
					{errorMessage(check.isError ? check.error : done.error)}
				</p>
			</Show>
			<Show when={check.data}>
				{(result) => (
					<div class="grid gap-1">
						<p class="text-muted">{result().words} words</p>
						<Show
							when={result().findings.length > 0}
							fallback={<p class="text-foreground">No problems found.</p>}
						>
							<ul class="grid gap-1">
								<For each={result().findings}>
									{(f) => (
										<li class="text-foreground">
											<span class="font-medium">{findingLabels[f.kind]}</span>
											{": "}
											{f.text}
										</li>
									)}
								</For>
							</ul>
						</Show>
					</div>
				)}
			</Show>
		</div>
	);
}

export function CoverLetterCard(props: { jobId: string }) {
	const letters = useCoverLetters(() => props.jobId);
	const sources = useCVTemplates(() => "letter");
	const suggestions = useCoverLetterAchievements(() => props.jobId);
	const create = useCreateCoverLetter();

	const [source, setSource] = createSignal("");
	const [picked, setPicked] = createSignal<string[] | null>(null);

	const ranked = createMemo(() =>
		[...(loadedData(suggestions) ?? [])].sort((a, b) => b.score - a.score),
	);
	const chosen = () =>
		picked() ??
		ranked()
			.slice(0, TOP_ACHIEVEMENTS)
			.map((s) => s.achievementId);
	const toggle = (id: string) =>
		setPicked(
			chosen().includes(id)
				? chosen().filter((c) => c !== id)
				: [...chosen(), id],
		);

	const visibleSources = () =>
		loadedData(sources)?.filter((s) => s.Visible) ?? [];
	const generate = useFormSubmit(async () => {
		const [docId = "", tabId = ""] = source().split("/");
		await create.mutateAsync({
			jobId: props.jobId,
			docId,
			tabId,
			achievementIds: chosen(),
		});
	}, errorMessage);

	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Cover letters</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<For each={loadedData(letters)}>{(l) => <LetterRow letter={l} />}</For>
				<Show
					when={visibleSources().length > 0}
					fallback={
						<p class="text-xs text-muted">
							Track a past cover letter in CV templates to write one.
						</p>
					}
				>
					<FormFeedback error={generate.error()} />
					<div>
						<Label for="cover-letter-source">Letter source</Label>
						<select
							id="cover-letter-source"
							class="h-9 w-full rounded-md border border-border bg-surface px-3 text-sm text-foreground focus:border-primary focus:outline-none"
							value={source()}
							onChange={(e) => setSource(e.currentTarget.value)}
						>
							<option value="">Choose a letter tab</option>
							<For each={visibleSources()}>
								{(s) => (
									<option value={`${s.DocID}/${s.TabID}`}>
										{s.SourceDoc} / {s.Title}
									</option>
								)}
							</For>
						</select>
					</div>
					<fieldset class="grid gap-1.5">
						<legend class="mb-1 text-xs text-muted">Achievements</legend>
						<For each={ranked()}>
							{(s) => (
								<label class="flex items-start gap-2 text-xs text-foreground">
									<input
										type="checkbox"
										class={checkbox}
										checked={chosen().includes(s.achievementId)}
										onChange={() => toggle(s.achievementId)}
									/>
									<span>{s.text}</span>
								</label>
							)}
						</For>
					</fieldset>
					<Button
						onClick={() => void generate.submit()}
						disabled={generate.pending() || !source() || chosen().length === 0}
					>
						{generate.pending() ? "Writing…" : "Generate scaffold"}
					</Button>
				</Show>
			</CardContent>
		</Card>
	);
}
