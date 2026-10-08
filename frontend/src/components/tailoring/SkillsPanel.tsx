import { createMemo, createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { MultiCombobox } from "@/components/MultiCombobox";
import { Button } from "@/components/ui/button";
import {
	Sheet,
	SheetContent,
	SheetHeader,
	SheetTitle,
} from "@/components/ui/sheet";
import { lineWithIds, moveItem } from "@/lib/tailoring";
import type { Draft, SkillGroup } from "@/types/tailoring";
import { useBankSkills } from "../../hooks/useBankSkills";
import { useSaveDraftSkills } from "../../hooks/useTailoring";

const lineName = (g: SkillGroup) => g.label || "Skills";

function SkillsForm(props: {
	draft: Draft;
	groups: SkillGroup[];
	flush: () => Promise<boolean>;
	onSaved: () => void;
}) {
	const bank = useBankSkills();
	const save = useSaveDraftSkills(() => props.draft.id);
	// eslint-disable-next-line solid/reactivity -- pedantic: the panel is seeded once on open
	const [groups, setGroups] = createSignal(props.groups);
	const sourced = createMemo(() => {
		const names = [
			...(props.draft.base?.skillGroups.flatMap((g) => g.items) ?? []),
			...(bank.data?.map((b) => b.name) ?? []),
		];
		return [...new Set(names)];
	});
	const setItems = (line: number, items: string[]) =>
		setGroups((gs) => gs.map((g, i) => (i === line ? { ...g, items } : g)));
	const optionsFor = (line: number) => {
		const elsewhere = new Set(
			groups().flatMap((g, i) => (i === line ? [] : g.items)),
		);
		return sourced()
			.filter((name) => !elsewhere.has(name))
			.map((name) => ({ id: name, label: name }));
	};
	const valid = () => groups().every((g) => g.items.length > 0);
	const submit = async () => {
		if (!(await props.flush())) return;
		save.mutate(groups(), { onSuccess: props.onSaved });
	};
	return (
		<div class="flex flex-1 flex-col gap-5 overflow-y-auto px-6 py-5">
			<For each={groups()}>
				{(group, i) => (
					<section class="space-y-2">
						<MultiCombobox
							label={lineName(group)}
							hint="Your CV's skills and your Bank Skills."
							options={optionsFor(i())}
							value={group.items}
							onChange={(ids) => setItems(i(), lineWithIds(group.items, ids))}
							placeholder="Add a skill…"
							chipClass="border-border bg-surface-muted text-foreground"
						/>
						<ol class="flex flex-col gap-1">
							<For each={group.items}>
								{(item, j) => (
									<li class="flex items-center gap-1 text-sm text-foreground">
										<span class="flex-1">{item}</span>
										<Button
											variant="ghost"
											size="sm"
											aria-label={`Move ${item} up`}
											disabled={j() === 0}
											onClick={() =>
												setItems(i(), moveItem(group.items, j(), j() - 1))
											}
										>
											<Icon name="chevronUp" size={14} />
										</Button>
										<Button
											variant="ghost"
											size="sm"
											aria-label={`Move ${item} down`}
											disabled={j() === group.items.length - 1}
											onClick={() =>
												setItems(i(), moveItem(group.items, j(), j() + 1))
											}
										>
											<Icon name="chevronDown" size={14} />
										</Button>
									</li>
								)}
							</For>
						</ol>
					</section>
				)}
			</For>
			<Show when={save.isError}>
				<p role="alert" class="text-sm text-destructive-strong">
					{save.error?.message ?? "Saving the skills failed."}
				</p>
			</Show>
			<Button disabled={!valid() || save.isPending} onClick={submit}>
				Save skills
			</Button>
		</div>
	);
}

export function SkillsPanel(props: {
	draft: Draft;
	open: boolean;
	onClose: () => void;
	flush: () => Promise<boolean>;
}) {
	return (
		<Sheet open={props.open} onOpenChange={(o) => !o && props.onClose()}>
			<SheetContent class="w-96">
				<SheetHeader>
					<SheetTitle>Edit skills</SheetTitle>
				</SheetHeader>
				<Show when={props.draft.content?.skillGroups}>
					{(groups) => (
						<SkillsForm
							draft={props.draft}
							groups={groups()}
							flush={props.flush}
							onSaved={props.onClose}
						/>
					)}
				</Show>
			</SheetContent>
		</Sheet>
	);
}
