import { createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	useCreateVariant,
	useDeleteVariant,
	useUpdateVariant,
} from "../../hooks/useExperience";
import { statLine } from "../../lib/achievementStats";
import { variantName } from "../../lib/variantName";
import type { Achievement, AchievementVariant } from "../../types/experience";

const iconButton =
	"inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-foreground disabled:opacity-30";
const dangerIconButton =
	"inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-destructive-subtle hover:text-destructive-strong disabled:opacity-50";

export function VariantForm(props: {
	initial?: Pick<AchievementVariant, "text" | "label">;
	pending: boolean;
	error?: string | undefined;
	onSubmit: (input: { text: string; label: string }) => void;
	onCancel: () => void;
}) {
	const [text, setText] = createSignal(props.initial?.text ?? "");
	const [label, setLabel] = createSignal(props.initial?.label ?? "");

	const submit = (e: SubmitEvent) => {
		e.preventDefault();
		if (!text().trim()) return;
		props.onSubmit({ text: text().trim(), label: label().trim() });
	};

	return (
		<form onSubmit={submit} class="space-y-2">
			<Input
				aria-label="Variant text"
				placeholder="Alternative wording"
				value={text()}
				onInput={(e) => setText(e.currentTarget.value)}
			/>
			<div class="flex gap-2">
				<Input
					aria-label="Variant label"
					placeholder="Label (optional)"
					value={label()}
					onInput={(e) => setLabel(e.currentTarget.value)}
				/>
				<Button type="submit" disabled={!text().trim() || props.pending}>
					Save
				</Button>
				<Button type="button" variant="ghost" onClick={props.onCancel}>
					Cancel
				</Button>
			</div>
			<Show when={props.error}>
				<p role="alert" class="text-sm text-destructive-strong">
					{props.error}
				</p>
			</Show>
		</form>
	);
}

function VariantRow(props: { variant: AchievementVariant }) {
	const [editing, setEditing] = createSignal(false);
	const update = useUpdateVariant();
	const remove = useDeleteVariant();

	return (
		<li data-testid="variant">
			<Show
				when={editing()}
				fallback={
					<div class="flex items-center gap-2">
						<span class="flex-1 text-sm text-muted" title={props.variant.text}>
							{variantName(props.variant)}
							<Show when={statLine(props.variant.stats)}>
								<span
									class="ml-2 font-mono text-xs tabular-nums text-faint"
									data-testid="variant-stats"
								>
									{statLine(props.variant.stats)}
								</span>
							</Show>
						</span>
						<button
							type="button"
							class={iconButton}
							title="Edit variant"
							aria-label={`Edit variant ${variantName(props.variant)}`}
							onClick={() => setEditing(true)}
						>
							<Icon name="pen" size={14} />
						</button>
						<button
							type="button"
							class={dangerIconButton}
							title="Delete variant"
							aria-label={`Delete variant ${variantName(props.variant)}`}
							disabled={remove.isPending}
							onClick={() => remove.mutate(props.variant.id)}
						>
							<Icon name="trash" size={14} />
						</button>
					</div>
				}
			>
				<VariantForm
					initial={props.variant}
					pending={update.isPending}
					error={update.error?.message}
					onCancel={() => setEditing(false)}
					onSubmit={(input) =>
						update.mutate(
							{ id: props.variant.id, input },
							{ onSuccess: () => setEditing(false) },
						)
					}
				/>
			</Show>
		</li>
	);
}

export function VariantList(props: { achievement: Achievement }) {
	const [adding, setAdding] = createSignal(false);
	const create = useCreateVariant();

	return (
		<div class="mt-1.5 border-l border-border pl-3">
			<Show when={props.achievement.variants.length > 0}>
				<ul class="space-y-1" aria-label="Variants">
					<For each={props.achievement.variants}>
						{(variant) => <VariantRow variant={variant} />}
					</For>
				</ul>
			</Show>
			<Show
				when={adding()}
				fallback={
					<button
						type="button"
						class="mt-1 text-xs text-faint hover:text-foreground"
						onClick={() => setAdding(true)}
					>
						Add variant
					</button>
				}
			>
				<div class="mt-1">
					<VariantForm
						pending={create.isPending}
						error={create.error?.message}
						onCancel={() => setAdding(false)}
						onSubmit={(input) =>
							create.mutate(
								{ achievementId: props.achievement.id, input },
								{ onSuccess: () => setAdding(false) },
							)
						}
					/>
				</div>
			</Show>
		</div>
	);
}
