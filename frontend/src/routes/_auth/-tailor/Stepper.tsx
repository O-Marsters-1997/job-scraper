import { For, Show } from "solid-js";

export type Step = "cv" | "headings" | "achievements" | "skills" | "generate";

const STEP_LABELS: { step: Step; label: string }[] = [
	{ step: "cv", label: "Base CV" },
	{ step: "headings", label: "Headings" },
	{ step: "achievements", label: "Achievements" },
	{ step: "skills", label: "Skills" },
	{ step: "generate", label: "Generate" },
];

export function Stepper(props: { current: Step; skippedHeadings: boolean }) {
	const steps = () =>
		STEP_LABELS.filter(
			(s) => !(s.step === "headings" && props.skippedHeadings),
		);
	const currentIndex = () => steps().findIndex((s) => s.step === props.current);
	return (
		<ol aria-label="Tailor CV progress" class="mb-6 flex gap-2 text-sm">
			<For each={steps()}>
				{(s, i) => (
					<li
						class="min-w-0 flex-1"
						aria-current={i() === currentIndex() ? "step" : undefined}
					>
						<div
							class="h-1.5 rounded-full"
							classList={{
								"bg-primary": i() <= currentIndex(),
								"bg-border": i() > currentIndex(),
							}}
						/>
						<span
							class="mt-1.5 block break-words"
							classList={{
								"font-semibold text-foreground": i() === currentIndex(),
								"text-foreground": i() < currentIndex(),
								"text-faint": i() > currentIndex(),
							}}
						>
							{s.label}
							<Show when={i() < currentIndex()}>
								<span class="sr-only"> (completed)</span>
							</Show>
						</span>
					</li>
				)}
			</For>
		</ol>
	);
}
