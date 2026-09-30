import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createSignal, Show } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import { Button } from "@/components/ui/button";
import type { CVRef } from "../../hooks/useTailoring";
import { AchievementsStep } from "./-tailor/AchievementsStep";
import { CvStep } from "./-tailor/CvStep";
import { GenerateStep } from "./-tailor/GenerateStep";
import { HeadingsStep } from "./-tailor/HeadingsStep";
import { type Step, Stepper } from "./-tailor/Stepper";

export const Route = createFileRoute("/_auth/jobs_/$id_/tailor")({
	component: TailorPage,
});

function TailorPage() {
	const params = Route.useParams();
	const navigate = useNavigate();
	const [cv, setCv] = createSignal<CVRef>();
	const [step, setStep] = createSignal<Step>("cv");
	const [skipped, setSkipped] = createSignal(false);
	const [achievementIds, setAchievementIds] = createSignal<string[]>([]);

	const pickCv = (ref: CVRef) => {
		setCv(ref);
		setSkipped(false);
		setStep("headings");
	};

	return (
		<div class="px-7 py-6 pb-16">
			<PageHeading
				title="Tailor CV"
				subtitle="Pick a base CV, confirm its roles, then choose the Achievements to feature"
			>
				<Button
					variant="outline"
					size="sm"
					onClick={() =>
						navigate({ to: "/jobs/$id", params: { id: params().id } })
					}
				>
					Back to job
				</Button>
			</PageHeading>
			<Stepper current={step()} skippedHeadings={skipped()} />

			<Show when={step() === "cv"}>
				<CvStep onPick={pickCv} />
			</Show>
			<Show when={step() === "headings" && cv()}>
				<HeadingsStep
					cv={cv}
					onDone={(wasSkipped) => {
						setSkipped(wasSkipped);
						setStep("achievements");
					}}
					onBack={() => setStep("cv")}
				/>
			</Show>
			<Show when={step() === "achievements" && cv()}>
				<AchievementsStep
					jobId={() => params().id}
					cv={cv}
					onBack={() => setStep(skipped() ? "cv" : "headings")}
					onContinue={(ids) => {
						setAchievementIds(ids);
						setStep("generate");
					}}
				/>
			</Show>
			<Show when={step() === "generate" && cv()}>
				<GenerateStep
					jobId={() => params().id}
					cv={cv}
					achievementIds={achievementIds}
					onBack={() => setStep("achievements")}
				/>
			</Show>
		</div>
	);
}
