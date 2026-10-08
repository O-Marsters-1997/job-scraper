import { createMemo, createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
	useBankSkills,
	useCreateBankSkill,
	useDeleteBankSkill,
	useReorderBankSkills,
	useUpdateBankSkill,
} from "../../hooks/useBankSkills";
import type { BankSkill } from "../../types/experience";

const iconButton =
	"inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-foreground disabled:opacity-30";
const dangerIconButton =
	"inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-destructive-subtle hover:text-destructive-strong disabled:opacity-50";

const UNCATEGORISED = "Uncategorised";

interface Group {
	category: string;
	skills: BankSkill[];
}

function groupByCategory(skills: BankSkill[]): Group[] {
	const groups = new Map<string, BankSkill[]>();
	for (const skill of skills) {
		const key = skill.category.toLowerCase();
		const members = groups.get(key) ?? [];
		members.push(skill);
		groups.set(key, members);
	}
	return [...groups.values()].map((members) => ({
		category: members[0]?.category || UNCATEGORISED,
		skills: members,
	}));
}

function swapIds(all: BankSkill[], a: BankSkill, b: BankSkill): string[] {
	return all.map((s) => (s.id === a.id ? b.id : s.id === b.id ? a.id : s.id));
}

export function BankSkills() {
	const query = useBankSkills();
	const create = useCreateBankSkill();
	const [name, setName] = createSignal("");
	const [category, setCategory] = createSignal("");

	const add = (e: SubmitEvent) => {
		e.preventDefault();
		if (!name().trim()) return;
		create.mutate(
			{ name: name().trim(), category: category().trim() },
			{
				onSuccess: () => {
					setName("");
				},
			},
		);
	};

	return (
		<section class="mt-8" aria-labelledby="bank-skills-heading">
			<h2
				id="bank-skills-heading"
				class="mb-1 text-sm font-semibold text-foreground"
			>
				Skills
			</h2>
			<p class="mb-3 text-sm text-muted">
				Technologies you can claim, grouped by category
			</p>
			<Card class="p-4" data-testid="bank-skills">
				<form onSubmit={add} class="flex gap-2">
					<Input
						aria-label="New skill name"
						placeholder="Skill, e.g. Go"
						value={name()}
						onInput={(e) => setName(e.currentTarget.value)}
					/>
					<Input
						aria-label="New skill category"
						placeholder="Category, e.g. Languages"
						value={category()}
						onInput={(e) => setCategory(e.currentTarget.value)}
					/>
					<Button
						type="submit"
						variant="secondary"
						disabled={!name().trim() || create.isPending}
					>
						<Icon name="plus" size={12} strokeWidth={2.5} />
						Add
					</Button>
				</form>
				<Show when={create.error}>
					{(err) => (
						<p role="alert" class="mt-2 text-sm text-destructive-strong">
							{err().message}
						</p>
					)}
				</Show>
				<QueryBoundary query={query}>
					{(skills) => <SkillGroups skills={skills()} />}
				</QueryBoundary>
			</Card>
		</section>
	);
}

function SkillGroups(props: { skills: BankSkill[] }) {
	const reorder = useReorderBankSkills();
	const groups = createMemo(() => groupByCategory(props.skills));

	return (
		<Show
			when={props.skills.length > 0}
			fallback={
				<p class="mt-3 text-sm text-muted">
					No skills yet. Add the technologies you want available when tailoring.
				</p>
			}
		>
			<div class="mt-3 space-y-3">
				<For each={groups()}>
					{(group) => (
						<div>
							<h3 class="mb-1 text-xs font-medium uppercase tracking-wide text-faint">
								{group.category}
							</h3>
							<ul class="space-y-1">
								<For each={group.skills}>
									{(skill, i) => (
										<SkillRow
											skill={skill}
											first={i() === 0}
											last={i() === group.skills.length - 1}
											busy={reorder.isPending}
											onMove={(delta) => {
												const neighbour = group.skills[i() + delta];
												if (neighbour)
													reorder.mutate(
														swapIds(props.skills, skill, neighbour),
													);
											}}
										/>
									)}
								</For>
							</ul>
						</div>
					)}
				</For>
			</div>
		</Show>
	);
}

function SkillRow(props: {
	skill: BankSkill;
	first: boolean;
	last: boolean;
	busy: boolean;
	onMove: (delta: number) => void;
}) {
	const [editing, setEditing] = createSignal(false);
	const [name, setName] = createSignal("");
	const [category, setCategory] = createSignal("");
	const update = useUpdateBankSkill();
	const remove = useDeleteBankSkill();

	const save = (e: SubmitEvent) => {
		e.preventDefault();
		if (!name().trim()) return;
		update.mutate(
			{
				id: props.skill.id,
				input: { name: name().trim(), category: category().trim() },
			},
			{ onSuccess: () => setEditing(false) },
		);
	};

	return (
		<li class="flex items-center gap-2" data-testid="bank-skill">
			<Show
				when={editing()}
				fallback={
					<>
						<p class="flex-1 text-sm text-foreground">{props.skill.name}</p>
						<div class="flex shrink-0 items-center">
							<button
								type="button"
								class={iconButton}
								title="Move skill up"
								aria-label={`Move ${props.skill.name} up`}
								disabled={props.first || props.busy}
								onClick={() => props.onMove(-1)}
							>
								<Icon name="chevronUp" size={14} />
							</button>
							<button
								type="button"
								class={iconButton}
								title="Move skill down"
								aria-label={`Move ${props.skill.name} down`}
								disabled={props.last || props.busy}
								onClick={() => props.onMove(1)}
							>
								<Icon name="chevronDown" size={14} />
							</button>
							<button
								type="button"
								class={iconButton}
								title="Edit skill"
								aria-label={`Edit ${props.skill.name}`}
								onClick={() => {
									setName(props.skill.name);
									setCategory(props.skill.category);
									update.reset();
									setEditing(true);
								}}
							>
								<Icon name="pen" size={14} />
							</button>
							<button
								type="button"
								class={dangerIconButton}
								title="Delete skill"
								aria-label={`Delete ${props.skill.name}`}
								disabled={remove.isPending}
								onClick={() => remove.mutate(props.skill.id)}
							>
								<Icon name="trash" size={14} />
							</button>
						</div>
					</>
				}
			>
				<form onSubmit={save} class="flex flex-1 flex-wrap items-center gap-2">
					<Input
						aria-label="Skill name"
						value={name()}
						onInput={(e) => setName(e.currentTarget.value)}
					/>
					<Input
						aria-label="Skill category"
						value={category()}
						onInput={(e) => setCategory(e.currentTarget.value)}
					/>
					<Button type="submit" disabled={!name().trim() || update.isPending}>
						Save
					</Button>
					<Button
						type="button"
						variant="ghost"
						onClick={() => setEditing(false)}
					>
						Cancel
					</Button>
					<Show when={update.error}>
						{(err) => (
							<p role="alert" class="w-full text-sm text-destructive-strong">
								{err().message}
							</p>
						)}
					</Show>
				</form>
			</Show>
		</li>
	);
}
