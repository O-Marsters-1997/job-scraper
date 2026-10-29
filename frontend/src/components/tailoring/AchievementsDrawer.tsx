import { createSignal, Show } from "solid-js";
import { Icon } from "@/components/Icon";
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

function ClosestLine(props: { quote: string | undefined }) {
	const [open, setOpen] = createSignal(false);
	return (
		<span class="mt-1 block text-xs text-muted">
			<button
				type="button"
				aria-expanded={open()}
				class="underline decoration-dotted underline-offset-2 hover:text-foreground"
				onClick={(e) => {
					e.stopPropagation();
					setOpen(!open());
				}}
				onKeyDown={(e) => {
					if (e.key === " " || e.key === "Enter") e.stopPropagation();
				}}
			>
				Closest line in the job ad
			</button>
			<Show when={open()}>
				<span class="mt-1 block italic">
					{props.quote
						? `“${props.quote}”`
						: "No close match in the job description"}
				</span>
			</Show>
		</span>
	);
}

export function AchievementsDrawer(
	props: Omit<PickListProps, "label" | "items"> & {
		title: string;
		bank: Suggestion[];
		emptyText: string;
		loading: boolean;
		usedLines: number;
		budgetLines: number;
		roleBudget: RoleBudgetValue | undefined;
		carried: { id: string; text: string }[];
		onEmpty: () => void;
		onReset: () => void;
		onBackToRoles?: (() => void) | undefined;
		onClose: () => void;
	},
) {
	return (
		<Drawer
			label="Choose Achievements"
			title={props.title}
			crumb={
				<Show when={props.onBackToRoles}>
					{(back) => (
						<button
							type="button"
							class="mb-0.5 text-sm text-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-primary"
							onClick={() => back()()}
						>
							Roles ›
						</button>
					)}
				</Show>
			}
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
				<div class="flex items-center gap-1">
					<Button
						variant="ghost"
						size="icon-sm"
						aria-label="Reset to generated"
						title="Reset to generated"
						onClick={() => props.onReset()}
					>
						<Icon name="rotateCcw" size={14} />
					</Button>
					<Button
						variant="ghost"
						size="icon-sm"
						aria-label="Empty this role"
						title="Empty this role"
						onClick={() => props.onEmpty()}
					>
						<Icon name="trash" size={14} />
					</Button>
				</div>
			</div>
			<Show
				when={props.bank.length > 0 || props.carried.length > 0}
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
					items={[
						...props.carried.map((c) => ({
							id: c.id,
							meta: "Kept",
							body: c.text,
						})),
						...props.bank.map((s) => ({
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
									<ClosestLine quote={s.quote} />
								</>
							),
						})),
					]}
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
