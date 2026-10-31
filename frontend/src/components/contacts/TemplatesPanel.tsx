import { createSignal, For, Show, untrack } from "solid-js";
import { ApiError } from "@/api/client";
import { FormFeedback } from "@/components/FormFeedback";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
	useCreateOutreachTemplate,
	useDeleteOutreachTemplate,
	useMakeOutreachTemplateDefault,
	useOutreachTemplates,
	useUpdateOutreachTemplate,
} from "@/hooks/useOutreachTemplates";
import {
	mergeFields,
	type OutreachKind,
	type OutreachTemplate,
	outreachKinds,
} from "@/types/outreach";
import { QueryBoundary } from "../QueryBoundary";

function errorMessage(err: unknown): string {
	if (err instanceof ApiError) {
		const body = err.body as { error?: unknown } | undefined;
		if (typeof body?.error === "string") return body.error;
	} else if (err instanceof Error) {
		return err.message;
	}
	return "Something went wrong. Try again.";
}

export function TemplatesPanel() {
	const templates = useOutreachTemplates();
	const create = useCreateOutreachTemplate();
	const [error, setError] = createSignal<string | null>(null);

	const add = (kind: OutreachKind, existing: OutreachTemplate[]) => {
		let n = existing.length + 1;
		while (existing.some((t) => t.name === `New template ${n}`)) n++;
		setError(null);
		create.mutate(
			{ kind, name: `New template ${n}`, body: "Hi {first_name}, " },
			{ onError: (e) => setError(errorMessage(e)) },
		);
	};

	return (
		<div class="flex flex-col gap-6 lg:flex-row lg:items-start">
			<div class="flex min-w-0 flex-1 flex-col gap-6">
				<FormFeedback error={error()} />
				<QueryBoundary query={templates}>
					{(all) => (
						<For each={outreachKinds}>
							{(k) => {
								const ofKind = () => all().filter((t) => t.kind === k.key);
								return (
									<section class="flex flex-col gap-3">
										<div class="flex items-center justify-between">
											<h2 class="text-xs font-semibold uppercase tracking-wide text-faint">
												{k.label}
											</h2>
											<Button
												variant="ghost"
												size="sm"
												onClick={() => add(k.key, ofKind())}
											>
												Add
											</Button>
										</div>
										<For each={ofKind()}>
											{(t) => (
												<TemplateEditor template={t} onError={setError} />
											)}
										</For>
									</section>
								);
							}}
						</For>
					)}
				</QueryBoundary>
			</div>
			<Card class="flex flex-col gap-2 p-4 lg:w-64">
				<h2 class="text-sm font-semibold text-foreground">Merge fields</h2>
				<p class="text-xs text-muted">
					Written into a message when you send it.
				</p>
				<ul class="flex flex-col gap-1">
					<For each={mergeFields}>
						{(f) => (
							<li>
								<code class="font-mono text-xs text-foreground">{f}</code>
							</li>
						)}
					</For>
				</ul>
			</Card>
		</div>
	);
}

function TemplateEditor(props: {
	template: OutreachTemplate;
	onError: (message: string | null) => void;
}) {
	const update = useUpdateOutreachTemplate();
	const makeDefault = useMakeOutreachTemplateDefault();
	const remove = useDeleteOutreachTemplate();
	const [name, setName] = createSignal(untrack(() => props.template.name));
	const [body, setBody] = createSignal(untrack(() => props.template.body));
	const dirty = () =>
		name().trim() !== props.template.name ||
		body().trim() !== props.template.body;
	const onError = (e: unknown) => props.onError(errorMessage(e));

	return (
		<Card class="flex flex-col gap-2 p-3">
			<div class="flex items-center gap-2">
				<Input
					aria-label="Template name"
					value={name()}
					onInput={(e) => setName(e.currentTarget.value)}
				/>
				<Show
					when={!props.template.isDefault}
					fallback={<Badge>Default</Badge>}
				>
					<Button
						variant="outline"
						size="sm"
						onClick={() => {
							props.onError(null);
							makeDefault.mutate(props.template.id, { onError });
						}}
					>
						Make default
					</Button>
				</Show>
			</div>
			<Textarea
				aria-label={`${props.template.name} body`}
				rows={3}
				value={body()}
				onInput={(e) => setBody(e.currentTarget.value)}
			/>
			<div class="flex justify-end gap-2">
				<Button
					variant="ghost"
					size="sm"
					onClick={() => {
						props.onError(null);
						remove.mutate(props.template.id, { onError });
					}}
				>
					Delete
				</Button>
				<Button
					size="sm"
					disabled={!dirty() || update.isPending}
					onClick={() => {
						props.onError(null);
						update.mutate(
							{ id: props.template.id, name: name(), body: body() },
							{ onError },
						);
					}}
				>
					Save
				</Button>
			</div>
		</Card>
	);
}
