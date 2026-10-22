import { Show } from "solid-js";
import { Button } from "@/components/ui/button";
import type { Position } from "@/types/experience";
import { Drawer, LineBudget } from "./Drawer";
import { PickList, type PickListProps } from "./PickList";

const HINT =
	"Click a role to add it, or its circle to remove it. Press 1–9 to move the selected role to that place. ↑ ↓ to move, Esc to deselect.";

const shortDate = (iso: string | null) =>
	iso
		? new Date(iso).toLocaleDateString("en-GB", {
				month: "short",
				year: "numeric",
			})
		: "Present";

export function RolesDrawer(
	props: Omit<PickListProps, "label" | "items"> & {
		positions: Position[];
		usedLines: number;
		budgetLines: number;
		onChooseAchievements: (positionId: string) => void;
		onClose: () => void;
	},
) {
	return (
		<Drawer label="Roles" title="Roles" hint={HINT} onClose={props.onClose}>
			<div class="flex items-center justify-between border-b border-border px-5 py-2">
				<LineBudget used={props.usedLines} budget={props.budgetLines} />
				<Show when={props.selected && props.picked.includes(props.selected)}>
					<Button
						variant="ghost"
						size="sm"
						onClick={() => {
							if (props.selected) props.onChooseAchievements(props.selected);
						}}
					>
						Choose its Achievements
					</Button>
				</Show>
			</div>
			<PickList
				label="Roles"
				items={props.positions.map((p) => ({
					id: p.id,
					meta: `${shortDate(p.startDate)} – ${shortDate(p.endDate)}`,
					body: (
						<>
							<span class="block font-medium">
								{p.title}, {p.employer}
							</span>
							<span class="block text-xs text-faint">
								{p.achievements.length} Achievement
								{p.achievements.length === 1 ? "" : "s"}
							</span>
						</>
					),
				}))}
				picked={props.picked}
				selected={props.selected}
				onSelect={props.onSelect}
				onToggle={props.onToggle}
				onMove={props.onMove}
			/>
		</Drawer>
	);
}
