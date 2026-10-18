import { createMemo, createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import {
	useBankSkills,
	useCreateBankSkill,
	useDeleteBankSkill,
	useDeleteBankSkills,
	useReorderBankSkills,
	useSetBankSkillsCategory,
	useUpdateBankSkill,
} from "../../hooks/useBankSkills";
import type { BankSkill } from "../../types/experience";

const iconButton =
	"inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-foreground disabled:opacity-30";
const dangerIconButton =
	"inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-destructive-subtle hover:text-destructive-strong disabled:opacity-50";
const checkbox = "size-4 cursor-pointer align-middle accent-primary";

const CATEGORY_LIST_ID = "bank-skill-categories";

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
			{ onSuccess: () => setName("") },
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
				Technologies you can claim. Select several to set their category or
				delete them together.
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
						aria-label="New skill category (optional)"
						placeholder="Category (optional)"
						list={CATEGORY_LIST_ID}
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
					{(skills) => <SkillTable skills={skills()} />}
				</QueryBoundary>
			</Card>
		</section>
	);
}

function SkillTable(props: { skills: BankSkill[] }) {
	const reorder = useReorderBankSkills();
	const setCategory = useSetBankSkillsCategory();
	const removeMany = useDeleteBankSkills();
	const [picked, setPicked] = createSignal(new Set<string>());
	const [bulkCategory, setBulkCategory] = createSignal("");

	const selected = createMemo(() =>
		props.skills.filter((s) => picked().has(s.id)),
	);
	const categories = createMemo(() => [
		...new Set(props.skills.map((s) => s.category).filter(Boolean)),
	]);
	const allSelected = () =>
		props.skills.length > 0 && selected().length === props.skills.length;

	const toggle = (id: string) =>
		setPicked((prev) => {
			const next = new Set(prev);
			if (!next.delete(id)) next.add(id);
			return next;
		});
	const toggleAll = () =>
		setPicked(new Set(allSelected() ? [] : props.skills.map((s) => s.id)));
	const ids = () => selected().map((s) => s.id);

	const applyCategory = (e: SubmitEvent) => {
		e.preventDefault();
		setCategory.mutate(
			{ ids: ids(), category: bulkCategory().trim() },
			{
				onSuccess: () => {
					setBulkCategory("");
					setPicked(new Set<string>());
				},
			},
		);
	};

	return (
		<Show
			when={props.skills.length > 0}
			fallback={
				<p class="mt-3 text-sm text-muted">
					No skills yet. Add the technologies you want available when tailoring.
				</p>
			}
		>
			<datalist id={CATEGORY_LIST_ID}>
				<For each={categories()}>{(c) => <option value={c} />}</For>
			</datalist>
			<Show when={selected().length > 0}>
				<form
					onSubmit={applyCategory}
					class="mt-3 flex flex-wrap items-center gap-2"
					aria-label="Bulk actions"
				>
					<span class="text-sm text-muted">{selected().length} selected</span>
					<Input
						class="w-48"
						aria-label="Category for selected skills"
						placeholder="Category (blank clears)"
						list={CATEGORY_LIST_ID}
						value={bulkCategory()}
						onInput={(e) => setBulkCategory(e.currentTarget.value)}
					/>
					<Button
						type="submit"
						variant="secondary"
						disabled={setCategory.isPending}
					>
						Set category
					</Button>
					<Button
						type="button"
						variant="ghost"
						disabled={removeMany.isPending}
						onClick={() =>
							removeMany.mutate(ids(), {
								onSuccess: () => setPicked(new Set()),
							})
						}
					>
						Delete selected
					</Button>
					<Show when={setCategory.error ?? removeMany.error}>
						{(err) => (
							<p role="alert" class="w-full text-sm text-destructive-strong">
								{err().message}
							</p>
						)}
					</Show>
				</form>
			</Show>
			<Table class="mt-3">
				<TableHeader>
					<TableRow>
						<TableHead class="w-10">
							<input
								type="checkbox"
								class={checkbox}
								aria-label="Select all skills"
								checked={allSelected()}
								onChange={toggleAll}
							/>
						</TableHead>
						<TableHead>Skill</TableHead>
						<TableHead>Category</TableHead>
						<TableHead class="w-36" />
					</TableRow>
				</TableHeader>
				<TableBody>
					<For each={props.skills}>
						{(skill, i) => (
							<SkillRow
								skill={skill}
								checked={picked().has(skill.id)}
								onToggle={() => toggle(skill.id)}
								first={i() === 0}
								last={i() === props.skills.length - 1}
								busy={reorder.isPending}
								onMove={(delta) => {
									const neighbour = props.skills[i() + delta];
									if (neighbour)
										reorder.mutate(swapIds(props.skills, skill, neighbour));
								}}
							/>
						)}
					</For>
				</TableBody>
			</Table>
		</Show>
	);
}

function SkillRow(props: {
	skill: BankSkill;
	checked: boolean;
	onToggle: () => void;
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
		<TableRow data-testid="bank-skill">
			<TableCell class="w-10 py-2">
				<input
					type="checkbox"
					class={checkbox}
					aria-label={`Select ${props.skill.name}`}
					checked={props.checked}
					onChange={() => props.onToggle()}
				/>
			</TableCell>
			<Show
				when={editing()}
				fallback={
					<>
						<TableCell class="py-2">{props.skill.name}</TableCell>
						<TableCell class="py-2 text-muted">
							{props.skill.category || "Uncategorised"}
						</TableCell>
						<TableCell class="py-2">
							<div class="flex items-center justify-end">
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
						</TableCell>
					</>
				}
			>
				<TableCell colSpan={3} class="py-2">
					<form onSubmit={save} class="flex flex-wrap items-center gap-2">
						<Input
							aria-label="Skill name"
							value={name()}
							onInput={(e) => setName(e.currentTarget.value)}
						/>
						<Input
							aria-label="Skill category"
							list={CATEGORY_LIST_ID}
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
				</TableCell>
			</Show>
		</TableRow>
	);
}
