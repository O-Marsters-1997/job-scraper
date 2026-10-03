import { For } from "solid-js";
import { cn, titleCase } from "@/lib/utils";
import { GRADE_REASONS, type GradeReason } from "@/types/grade";

export function GradeChips(props: {
	selected: GradeReason[];
	onChange: (reasons: GradeReason[]) => void;
}) {
	const isSelected = (reason: GradeReason) => props.selected.includes(reason);
	const toggle = (reason: GradeReason) =>
		props.onChange(
			isSelected(reason)
				? props.selected.filter((r) => r !== reason)
				: [...props.selected, reason],
		);
	return (
		<div class="flex flex-wrap gap-1.5">
			<For each={GRADE_REASONS}>
				{(reason) => (
					<button
						type="button"
						aria-pressed={isSelected(reason)}
						onClick={() => toggle(reason)}
						class={cn(
							"rounded-full border px-2.5 py-0.5 text-xs transition-colors",
							isSelected(reason)
								? "border-accent-border bg-accent-subtle text-accent-text"
								: "border-border bg-surface text-muted hover:text-foreground",
						)}
					>
						{titleCase(reason)}
					</button>
				)}
			</For>
		</div>
	);
}
