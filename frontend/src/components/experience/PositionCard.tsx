import { createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
	useCreateAchievement,
	useDeleteAchievement,
	useDeletePosition,
	useReorderAchievements,
	useReorderPositions,
	useUpdateAchievement,
	useUpdatePosition,
} from "../../hooks/useExperience";
import type { Achievement, Position } from "../../types/experience";
import { PositionForm } from "./PositionForm";

const iconButton =
	"inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-foreground disabled:opacity-30";
const dangerIconButton =
	"inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-destructive-subtle hover:text-destructive-strong disabled:opacity-50";

function formatMonth(iso: string | null): string {
	if (!iso) return "";
	const d = new Date(`${iso}T00:00:00`);
	return d.toLocaleDateString("en-GB", { month: "short", year: "numeric" });
}

function dateRange(p: Position): string {
	const start = formatMonth(p.startDate);
	const end = p.endDate ? formatMonth(p.endDate) : "Present";
	return start ? `${start} – ${end}` : "";
}

function move<T>(items: T[], from: number, to: number): T[] {
	const next = [...items];
	next.splice(to, 0, ...next.splice(from, 1));
	return next;
}

export function PositionCard(props: {
	position: Position;
	index: number;
	positionIds: string[];
}) {
	const [editing, setEditing] = createSignal(false);
	const update = useUpdatePosition();
	const remove = useDeletePosition();
	const reorder = useReorderPositions();

	const shiftPosition = (delta: number) =>
		reorder.mutate(move(props.positionIds, props.index, props.index + delta));

	return (
		<Card class="p-4" data-testid="position">
			<Show
				when={!editing()}
				fallback={
					<PositionForm
						initial={props.position}
						submitLabel="Save"
						pending={update.isPending}
						error={update.error?.message}
						onCancel={() => setEditing(false)}
						onSubmit={(input) =>
							update.mutate(
								{ id: props.position.id, input },
								{ onSuccess: () => setEditing(false) },
							)
						}
					/>
				}
			>
				<div class="flex items-start justify-between gap-3">
					<div>
						<h2 class="text-sm font-semibold text-foreground">
							{props.position.title}
						</h2>
						<p class="text-sm text-muted">{props.position.employer}</p>
						<Show when={dateRange(props.position)}>
							<p class="mt-0.5 font-mono text-xs tabular-nums text-faint">
								{dateRange(props.position)}
							</p>
						</Show>
					</div>
					<div class="flex shrink-0 items-center">
						<button
							type="button"
							class={iconButton}
							title="Move position up"
							aria-label={`Move ${props.position.title} at ${props.position.employer} up`}
							disabled={props.index === 0 || reorder.isPending}
							onClick={() => shiftPosition(-1)}
						>
							<Icon name="chevronUp" size={14} />
						</button>
						<button
							type="button"
							class={iconButton}
							title="Move position down"
							aria-label={`Move ${props.position.title} at ${props.position.employer} down`}
							disabled={
								props.index === props.positionIds.length - 1 ||
								reorder.isPending
							}
							onClick={() => shiftPosition(1)}
						>
							<Icon name="chevronDown" size={14} />
						</button>
						<button
							type="button"
							class={iconButton}
							title="Edit position"
							aria-label={`Edit ${props.position.title} at ${props.position.employer}`}
							onClick={() => setEditing(true)}
						>
							<Icon name="pen" size={14} />
						</button>
						<button
							type="button"
							class={dangerIconButton}
							title="Delete position"
							aria-label={`Delete ${props.position.title} at ${props.position.employer}`}
							disabled={remove.isPending}
							onClick={() => {
								if (
									window.confirm(
										`Delete ${props.position.title} at ${props.position.employer} and its achievements?`,
									)
								)
									remove.mutate(props.position.id);
							}}
						>
							<Icon name="trash" size={14} />
						</button>
					</div>
				</div>
			</Show>

			<AchievementList position={props.position} />
		</Card>
	);
}

function AchievementList(props: { position: Position }) {
	const reorder = useReorderAchievements();
	const create = useCreateAchievement();
	const [draft, setDraft] = createSignal("");
	const ids = () => props.position.achievements.map((a) => a.id);

	const add = (e: SubmitEvent) => {
		e.preventDefault();
		const text = draft().trim();
		if (!text) return;
		create.mutate(
			{ positionId: props.position.id, text },
			{ onSuccess: () => setDraft("") },
		);
	};

	return (
		<div class="mt-3 border-t border-border pt-3">
			<ul class="space-y-1.5">
				<For each={props.position.achievements}>
					{(achievement, i) => (
						<AchievementRow
							achievement={achievement}
							first={i() === 0}
							last={i() === props.position.achievements.length - 1}
							busy={reorder.isPending}
							onMove={(delta) =>
								reorder.mutate({
									positionId: props.position.id,
									ids: move(ids(), i(), i() + delta),
								})
							}
						/>
					)}
				</For>
			</ul>
			<form onSubmit={add} class="mt-2 flex gap-2">
				<Input
					aria-label={`New achievement for ${props.position.employer}`}
					placeholder="Add an achievement"
					value={draft()}
					onInput={(e) => setDraft(e.currentTarget.value)}
				/>
				<Button
					type="submit"
					variant="secondary"
					disabled={!draft().trim() || create.isPending}
				>
					<Icon name="plus" size={12} strokeWidth={2.5} />
					Add
				</Button>
			</form>
		</div>
	);
}

function AchievementRow(props: {
	achievement: Achievement;
	first: boolean;
	last: boolean;
	busy: boolean;
	onMove: (delta: number) => void;
}) {
	const [editing, setEditing] = createSignal(false);
	const [text, setText] = createSignal("");
	const update = useUpdateAchievement();
	const remove = useDeleteAchievement();

	const save = (e: SubmitEvent) => {
		e.preventDefault();
		const next = text().trim();
		if (!next) return;
		update.mutate(
			{ id: props.achievement.id, text: next },
			{ onSuccess: () => setEditing(false) },
		);
	};

	return (
		<li class="flex items-start gap-2" data-testid="achievement">
			<Show
				when={editing()}
				fallback={
					<>
						<span
							aria-hidden="true"
							class="mt-2 size-1 shrink-0 rounded-full bg-faint"
						/>
						<p class="flex-1 text-sm text-foreground">
							{props.achievement.text}
						</p>
						<div class="flex shrink-0 items-center">
							<button
								type="button"
								class={iconButton}
								title="Move achievement up"
								aria-label="Move achievement up"
								disabled={props.first || props.busy}
								onClick={() => props.onMove(-1)}
							>
								<Icon name="chevronUp" size={14} />
							</button>
							<button
								type="button"
								class={iconButton}
								title="Move achievement down"
								aria-label="Move achievement down"
								disabled={props.last || props.busy}
								onClick={() => props.onMove(1)}
							>
								<Icon name="chevronDown" size={14} />
							</button>
							<button
								type="button"
								class={iconButton}
								title="Edit achievement"
								aria-label="Edit achievement"
								onClick={() => {
									setText(props.achievement.text);
									setEditing(true);
								}}
							>
								<Icon name="pen" size={14} />
							</button>
							<button
								type="button"
								class={dangerIconButton}
								title="Delete achievement"
								aria-label="Delete achievement"
								disabled={remove.isPending}
								onClick={() => remove.mutate(props.achievement.id)}
							>
								<Icon name="trash" size={14} />
							</button>
						</div>
					</>
				}
			>
				<form onSubmit={save} class="flex flex-1 gap-2">
					<Input
						aria-label="Achievement text"
						value={text()}
						onInput={(e) => setText(e.currentTarget.value)}
					/>
					<Button type="submit" disabled={!text().trim() || update.isPending}>
						Save
					</Button>
					<Button
						type="button"
						variant="ghost"
						onClick={() => setEditing(false)}
					>
						Cancel
					</Button>
				</form>
			</Show>
		</li>
	);
}
