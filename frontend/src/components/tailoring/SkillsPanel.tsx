import { createMemo, createSignal, For, Index, Show } from "solid-js";
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
import {
	useSaveDraftSkills,
	useSkillSuggestions,
} from "../../hooks/useTailoring";
import { fitPercent } from "./Lean";

const normalise = (s: string) => s.toLowerCase();

const lineName = (g: SkillGroup) => g.label || "Skills";

function SkillsForm(props: {
	draft: Draft;
	groups: SkillGroup[];
	flush: () => Promise<boolean>;
	onSaved: () => void;
}) {
	const bank = useBankSkills();
	const save = useSaveDraftSkills(() => props.draft.id);
	const suggestions = useSkillSuggestions(
		() => props.draft.jobId,
		() => ({ docId: props.draft.baseDocId, tabId: props.draft.baseTabId }),
	);
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
	const candidatesFor = (line: number) => {
		const onCv = new Set(groups().flatMap((g) => g.items.map(normalise)));
		return (
			suggestions.data?.lines[line]?.candidates.filter(
				(c) => !onCv.has(normalise(c.name)),
			) ?? []
		);
	};
	const fitByName = createMemo(() => {
		const scores = new Map<string, number>();
		for (const l of suggestions.data?.lines ?? []) {
			for (const b of l.base) scores.set(normalise(b.text), b.score);
			for (const c of l.candidates) scores.set(normalise(c.name), c.score);
		}
		return scores;
	});
	const chipMeta = (name: string) => {
		const score = fitByName().get(normalise(name));
		return score === undefined ? undefined : fitPercent(score);
	};
	const valid = () => groups().every((g) => g.items.length > 0);
	const submit = async () => {
		if (!(await props.flush())) return;
		save.mutate(groups(), { onSuccess: props.onSaved });
	};
	return (
		<div class="flex flex-1 flex-col gap-5 overflow-y-auto px-6 py-5">
			<Index each={groups()}>
				{(group, i) => (
					<section class="space-y-2">
						<MultiCombobox
							label={lineName(group())}
							hint="Your CV's skills and your Bank Skills."
							options={optionsFor(i)}
							value={group().items}
							onChange={(ids) => setItems(i, lineWithIds(group().items, ids))}
							onReorder={(from, to) =>
								setItems(i, moveItem(group().items, from, to))
							}
							chipMeta={chipMeta}
							placeholder="Add a skill…"
							chipClass="border-border bg-surface-muted text-foreground"
						/>
						<Show when={candidatesFor(i).length > 0}>
							<div class="space-y-1">
								<p class="text-xs font-medium text-faint">From your bank</p>
								<ul class="flex flex-wrap gap-1.5">
									<For each={candidatesFor(i)}>
										{(c) => (
											<li>
												<button
													type="button"
													aria-label={`Add ${c.name}, ${fitPercent(c.score)} fit`}
													onClick={() =>
														setItems(i, [...group().items, c.name])
													}
													class="inline-flex h-6 items-center gap-1.5 rounded-full border border-dashed border-border px-2.5 text-xs text-foreground hover:bg-accent-subtle focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
												>
													{c.name}
													<span class="text-faint">{fitPercent(c.score)}</span>
												</button>
											</li>
										)}
									</For>
								</ul>
							</div>
						</Show>
					</section>
				)}
			</Index>
			<Show when={save.isError}>
				<p role="alert" class="text-sm text-destructive-strong">
					{save.error?.message ?? "Saving the skills failed."}
				</p>
			</Show>
			<Button disabled={!valid()} loading={save.isPending} onClick={submit}>
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
