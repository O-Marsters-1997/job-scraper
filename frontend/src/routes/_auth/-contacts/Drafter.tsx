import { createEffect, createSignal, For, on, Show, untrack } from "solid-js";
import { ApiError } from "@/api/client";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useMarkContactSent } from "@/hooks/useContacts";
import { useExperience } from "@/hooks/useExperience";
import {
	useOutreachDraft,
	useWriteOutreachHook,
} from "@/hooks/useOutreachDraft";
import { useOutreachTemplates } from "@/hooks/useOutreachTemplates";
import { cn } from "@/lib/utils";
import {
	type Contact,
	type ContactChannel,
	type ContactSent,
	channelLabels,
} from "@/types/contact";
import { findingLabels } from "@/types/coverLetter";
import { draftWordLimit, wordCount } from "@/types/outreach";

const channels: ContactChannel[] = ["linkedin", "email", "call"];

function hookErrorMessage(err: unknown): string {
	if (err instanceof ApiError) {
		const msg = (err.body as { error?: string } | undefined)?.error;
		if (msg) return msg;
	}
	return "Could not write the hook. Please try again.";
}

export function Drafter(props: {
	contact: Contact;
	onSent: (sent: ContactSent) => void;
}) {
	const [templateId, setTemplateId] = createSignal<string | undefined>();
	const [achievementId, setAchievementId] = createSignal<string | undefined>();
	const [channel, setChannel] = createSignal<ContactChannel>(
		untrack(() => (props.contact.email ? "email" : "linkedin")),
	);
	const [text, setText] = createSignal("");
	const [error, setError] = createSignal("");
	const draft = useOutreachDraft(
		() => props.contact.id,
		() => ({ templateId: templateId(), achievementId: achievementId() }),
	);
	const templates = useOutreachTemplates();
	const experience = useExperience();
	const mark = useMarkContactSent();
	const writeHook = useWriteOutreachHook(() => props.contact.id);

	createEffect(
		on(
			() => draft.data,
			(d) => d && setText(d.body),
		),
	);

	const sameKind = () =>
		(templates.data ?? []).filter((t) => t.kind === draft.data?.kind);
	const chosen = () => sameKind().find((t) => t.id === draft.data?.templateId);
	const achievements = () =>
		(experience.data ?? []).flatMap((p) =>
			p.achievements.map((a) => ({ ...a, employer: p.employer })),
		);
	const words = () => wordCount(text());

	const copyAndMark = async () => {
		const template = chosen();
		if (!template) return;
		setError("");
		try {
			await navigator.clipboard.writeText(text());
			props.onSent(
				await mark.mutateAsync({
					id: props.contact.id,
					channel: channel(),
					template_kind: template.kind,
					template_name: template.name,
				}),
			);
		} catch (e) {
			setError(e instanceof Error ? e.message : "Could not copy and mark sent");
		}
	};

	return (
		<div class="flex flex-col gap-3 border-t border-border pt-3">
			<FormFeedback
				error={
					error() ||
					(draft.error?.message ?? "") ||
					(writeHook.error ? hookErrorMessage(writeHook.error) : "")
				}
			/>
			<Show when={draft.data}>
				{(d) => (
					<>
						<div class="flex flex-wrap gap-1.5">
							<For each={sameKind()}>
								{(t) => (
									<Button
										size="sm"
										variant={t.id === d().templateId ? "default" : "outline"}
										onClick={() => setTemplateId(t.id)}
									>
										{t.name}
									</Button>
								)}
							</For>
						</div>
						<Textarea
							rows={5}
							aria-label="Message"
							value={text()}
							onInput={(e) => setText(e.currentTarget.value)}
						/>
						<Show when={d().hasHookSlot}>
							<div class="flex flex-wrap items-center gap-3">
								<Button
									size="sm"
									variant="outline"
									loading={writeHook.isPending}
									onClick={() =>
										writeHook.mutate({ achievementId: d().achievementId })
									}
								>
									{d().hook ? "Rewrite hook" : "Write hook"}
								</Button>
								<Show when={!d().hook}>
									<p class="text-xs text-faint">
										Replace [hook] with a line about them before sending.
									</p>
								</Show>
							</div>
							<Show when={d().hookFindings.length > 0}>
								<ul class="list-disc pl-5 text-xs text-muted">
									<For each={d().hookFindings}>
										{(f) => (
											<li>
												<span class="font-medium">{findingLabels[f.kind]}</span>
												{": "}
												{f.text}
											</li>
										)}
									</For>
								</ul>
							</Show>
						</Show>
						<div class="flex flex-wrap items-center gap-3 text-xs">
							<label class="flex items-center gap-1.5 text-muted">
								Achievement
								<select
									class="field"
									value={d().achievementId}
									onChange={(e) => setAchievementId(e.currentTarget.value)}
								>
									<For each={achievements()}>
										{(a) => (
											<option value={a.id}>
												{a.text.slice(0, 60)} ({a.employer})
											</option>
										)}
									</For>
								</select>
							</label>
							<label class="flex items-center gap-1.5 text-muted">
								Sent by
								<select
									class="field"
									value={channel()}
									onChange={(e) =>
										setChannel(e.currentTarget.value as ContactChannel)
									}
								>
									<For each={channels}>
										{(c) => <option value={c}>{channelLabels[c]}</option>}
									</For>
								</select>
							</label>
							<span
								class={cn(
									"ml-auto tabular-nums",
									words() > draftWordLimit
										? "font-semibold text-destructive"
										: "text-faint",
								)}
							>
								{words()} words
								<Show when={words() > draftWordLimit}>
									{" "}
									(aim for {draftWordLimit} or fewer)
								</Show>
								<Show when={d().cost > 0}> · AI ${d().cost.toFixed(4)}</Show>
							</span>
						</div>
						<div>
							<Button
								size="sm"
								loading={mark.isPending}
								onClick={() => void copyAndMark()}
							>
								Copy &amp; mark sent
							</Button>
						</div>
					</>
				)}
			</Show>
		</div>
	);
}
