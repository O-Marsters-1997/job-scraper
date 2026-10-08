import { For } from "solid-js";

export type Step = "cv" | "headings" | "achievements" | "skills" | "generate";

const STEP_LABELS: { step: Step; label: string }[] = [
	{ step: "cv", label: "Base CV" },
	{ step: "headings", label: "Headings" },
	{ step: "achievements", label: "Achievements" },
	{ step: "skills", label: "Skills" },
	{ step: "generate", label: "Generate" },
];

export function Stepper(props: { current: Step; skippedHeadings: boolean }) {
	return (
		<ol class="mb-6 flex flex-wrap gap-4 text-sm">
			<For each={STEP_LABELS}>
				{(s, i) => (
					<li
						classList={{
							"font-semibold text-foreground": props.current === s.step,
							"text-faint": props.current !== s.step,
							"line-through": s.step === "headings" && props.skippedHeadings,
						}}
					>
						{i() + 1}. {s.label}
					</li>
				)}
			</For>
		</ol>
	);
}
