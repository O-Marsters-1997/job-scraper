import { Show } from "solid-js";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SkeletonList } from "@/components/ui/skeleton";
import type { RoleBudgetValue } from "@/lib/tailoring";
import type { Suggestion } from "@/types/tailoring";
import { Drawer, LineBudget, RoleBudget } from "./Drawer";
import { fitPercent } from "./Lean";
import { PickList, type PickListProps } from "./PickList";

const HINT =
	"Click a row to add it, or its circle to remove it. Press 1–9 to move the selected row to that place. ↑ ↓ to move, Esc to deselect.";

export function AchievementsDrawer(
	props: Omit<PickListProps, "label" | "items"> & {
		title: string;
		bank: Suggestion[];
		emptyText: string;
		loading: boolean;
		usedLines: number;
		budgetLines: number;
		roleBudget: RoleBudgetValue | undefined;
		onClear: () => void;
		onClose: () => void;
	},
) {
	return (
		<Drawer
			label="Choose Achievements"
			title={props.title}
			hint={HINT}
			onClose={props.onClose}
		>
			<div class="flex items-center justify-between border-b border-border px-5 py-2">
				<div class="flex items-center gap-4">
					<LineBudget used={props.usedLines} budget={props.budgetLines} />
					<Show when={props.roleBudget}>
						{(b) => <RoleBudget budget={b()} />}
					</Show>
				</div>
				<Button variant="ghost" size="sm" onClick={() => props.onClear()}>
					Clear and re-pick
				</Button>
			</div>
			<Show
				when={props.bank.length > 0}
				fallback={
					<Show
						when={props.loading}
						fallback={
							<p class="px-5 py-4 text-sm text-muted">{props.emptyText}</p>
						}
					>
						<div class="px-5 py-4">
							<SkeletonList rows={3} />
						</div>
					</Show>
				}
			>
				<PickList
					label="Achievements"
					items={props.bank.map((s) => ({
						id: s.achievementId,
						meta: fitPercent(s.score),
						body: (
							<>
								{s.text}
								<Show when={s.state !== "fit"}>
									<Badge variant="outline" class="ml-1.5 align-middle">
										{s.state === "low" ? "Low fit" : "Unclear"}
									</Badge>
								</Show>
							</>
						),
					}))}
					picked={props.picked}
					selected={props.selected}
					onSelect={props.onSelect}
					onToggle={props.onToggle}
					onMove={props.onMove}
				/>
			</Show>
		</Drawer>
	);
}
