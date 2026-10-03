import { For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { useClearGrade, useGrade, useSetGrade } from "@/hooks/useGrades";
import { GRADES, type GradeValue } from "@/types/grade";
import { GradeChips } from "./GradeChips";

const LABELS: Record<GradeValue, string> = {
	great: "Great",
	ok: "OK",
	no: "No",
};

export function GradeControl(props: { jobId: string }) {
	const grade = useGrade(() => props.jobId);
	const set = useSetGrade();
	const clear = useClearGrade();
	const current = () => grade.data ?? undefined;
	const isCurrent = (value: GradeValue) => current()?.grade === value;

	return (
		<div class="flex flex-col gap-2">
			<p class="text-xs font-medium text-muted">Your grade</p>
			<div class="flex gap-2">
				<For each={GRADES}>
					{(value) => (
						<Button
							size="sm"
							variant={isCurrent(value) ? "default" : "outline"}
							aria-pressed={isCurrent(value)}
							disabled={set.isPending}
							onClick={() =>
								isCurrent(value)
									? clear.mutate(props.jobId)
									: set.mutate({
											jobId: props.jobId,
											grade: value,
											reasons: current()?.reasons ?? [],
										})
							}
						>
							{LABELS[value]}
						</Button>
					)}
				</For>
			</div>
			<Show when={current()}>
				{(g) => (
					<GradeChips
						selected={g().reasons}
						onChange={(reasons) =>
							set.mutate({ jobId: props.jobId, grade: g().grade, reasons })
						}
					/>
				)}
			</Show>
		</div>
	);
}
