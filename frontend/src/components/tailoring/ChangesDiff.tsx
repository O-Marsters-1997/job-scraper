import { For, Match, Show, Switch } from "solid-js";
import { Button } from "@/components/ui/button";
import { PROFILE_SLOT } from "@/hooks/useDraftEditor";
import { type BulletRow, bulletDiff } from "@/lib/bulletDiff";
import { listDiff, type WordOp, wordDiff } from "@/lib/wordDiff";
import type { DraftContent, DraftProvenance } from "@/types/tailoring";

const SKILL_SR_PREFIX = { same: "", add: "Added ", del: "Removed " };

const BULLET_LABEL: Record<BulletRow["kind"], string> = {
	same: "Unchanged",
	moved: "Moved",
	rewritten: "Rewritten",
	added: "Added",
	removed: "Removed",
};

function Marked(props: { ops: WordOp[] }) {
	return (
		<For each={props.ops}>
			{(d) => (
				<Switch fallback={d.text}>
					<Match when={d.op === "add"}>
						<ins class="rounded-sm bg-surface-muted font-medium text-foreground underline decoration-primary underline-offset-2">
							{d.text}
						</ins>
					</Match>
					<Match when={d.op === "del"}>
						<del class="text-destructive-strong decoration-destructive-strong">
							{d.text}
						</del>
					</Match>
				</Switch>
			)}
		</For>
	);
}

function Row(props: { row: BulletRow; onUndo?: (() => void) | undefined }) {
	return (
		<li class="flex items-start gap-2 rounded-md border border-border bg-surface px-3 py-2.5">
			<div class="min-w-0 flex-1">
				<span class="mr-2 text-xs font-medium text-muted">
					{BULLET_LABEL[props.row.kind]}
				</span>
				<span
					class="text-sm"
					classList={{
						"text-faint": props.row.kind === "same",
						"text-destructive-strong line-through":
							props.row.kind === "removed",
						"text-foreground":
							props.row.kind !== "same" && props.row.kind !== "removed",
					}}
				>
					<Show
						when={props.row.kind === "rewritten" ? props.row : undefined}
						fallback={props.row.text}
					>
						{(row) => <Marked ops={wordDiff(row().from, row().text)} />}
					</Show>
				</span>
			</div>
			<Show when={props.onUndo}>
				{(undo) => (
					<Button variant="ghost" size="sm" onClick={() => undo()()}>
						Undo
					</Button>
				)}
			</Show>
		</li>
	);
}

export function ChangesDiff(props: {
	base: DraftContent;
	content: DraftContent;
	provenance: DraftProvenance | null;
	onUndo?: ((slotId: string, text: string) => void) | undefined;
}) {
	const label = (id: string) => {
		const p = props.provenance?.positions.find((x) => x.positionId === id);
		return p ? `${p.title}, ${p.employer}` : "Position";
	};
	const baseBullets = (id: string) =>
		props.base.positions
			.find((p) => p.positionId === id)
			?.bullets.map((b) => b.text) ?? [];
	const slotIdOf = (positionId: string, index: number) =>
		props.provenance?.positions.find((x) => x.positionId === positionId)
			?.bullets[index]?.slotId;
	const skills = () => listDiff(props.base.skills, props.content.skills);
	const profile = () =>
		props.base.profile !== null && props.content.profile !== null
			? wordDiff(props.base.profile, props.content.profile)
			: null;

	return (
		<div class="flex flex-col gap-6">
			<Show when={profile()}>
				{(ops) => (
					<section aria-labelledby="diff-profile">
						<h3 id="diff-profile" class="mb-2 text-sm font-semibold">
							Profile
						</h3>
						<div class="flex items-start gap-2 rounded-md border border-border bg-surface px-3 py-2.5">
							<p class="min-w-0 flex-1 text-sm text-foreground">
								<Marked ops={ops()} />
							</p>
							<Show when={props.onUndo && props.base.profile}>
								{(from) => (
									<Button
										variant="ghost"
										size="sm"
										onClick={() => props.onUndo?.(PROFILE_SLOT, from())}
									>
										Undo
									</Button>
								)}
							</Show>
						</div>
					</section>
				)}
			</Show>
			<Show when={props.content.skills.length > 0}>
				<section aria-labelledby="diff-skills">
					<h3 id="diff-skills" class="mb-2 text-sm font-semibold">
						Skills
					</h3>
					<ul class="flex flex-wrap gap-1.5 text-sm">
						<For each={skills()}>
							{(s) => (
								<li
									class="rounded-md border border-border bg-surface px-2 py-0.5"
									classList={{
										"text-faint": s.op === "same",
										"font-medium text-foreground": s.op === "add",
										"text-destructive-strong line-through": s.op === "del",
									}}
								>
									<span class="sr-only">{SKILL_SR_PREFIX[s.op]}</span>
									{s.text}
								</li>
							)}
						</For>
					</ul>
				</section>
			</Show>
			<For each={props.content.positions}>
				{(p) => (
					<section>
						<h3 class="mb-2 text-xs font-medium text-muted">
							{label(p.positionId)}
						</h3>
						<ul class="flex flex-col gap-2">
							<For
								each={bulletDiff(
									baseBullets(p.positionId),
									p.bullets.map((b) => b.text),
								)}
							>
								{(row) => {
									const index = () =>
										p.bullets.findIndex((b) => b.text === row.text);
									const slotId = () => slotIdOf(p.positionId, index());
									return (
										<Row
											row={row}
											onUndo={
												row.kind === "rewritten" && props.onUndo && slotId()
													? () => {
															const id = slotId();
															if (id) props.onUndo?.(id, row.from);
														}
													: undefined
											}
										/>
									);
								}}
							</For>
						</ul>
					</section>
				)}
			</For>
		</div>
	);
}
