import { For, Show } from "solid-js";
import { reviewFindings, skillGaps } from "@/lib/tailoring";
import type { DraftFinding, DraftProvenance } from "@/types/tailoring";

const SEVERITY_LABEL: Record<DraftFinding["severity"], string> = {
	block: "Blocking",
	warn: "Warning",
	info: "Note",
};

function Bullet(props: {
	bullet: DraftProvenance["positions"][number]["bullets"][number];
}) {
	return (
		<li class="rounded-md border border-border bg-surface px-3 py-2.5">
			<p class="text-sm text-foreground">
				<For each={props.bullet.segments}>
					{(seg) => (
						<Show when={seg.novel} fallback={seg.text}>
							<mark class="rounded-sm bg-destructive-subtle px-0.5 text-destructive-strong underline decoration-dotted underline-offset-2">
								{seg.text}
							</mark>
						</Show>
					)}
				</For>
			</p>
			<p class="mt-2 text-xs text-faint">Drawn from</p>
			<ul class="mt-1 space-y-1">
				<For
					each={props.bullet.achievements}
					fallback={
						<li class="text-xs text-faint">
							— the cited Achievements no longer exist
						</li>
					}
				>
					{(a) => (
						<li class="border-l-2 border-border pl-2 text-xs text-muted">
							{a.text}
						</li>
					)}
				</For>
			</ul>
		</li>
	);
}

export function ProvenanceDiff(props: {
	provenance: DraftProvenance | null;
	findings: DraftFinding[];
}) {
	const flagged = () => reviewFindings(props.findings);
	const gaps = () => skillGaps(props.findings);
	return (
		<div class="space-y-6">
			<section aria-labelledby="draft-findings">
				<h2 id="draft-findings" class="mb-2 text-sm font-semibold">
					Findings
				</h2>
				<Show
					when={flagged().length > 0}
					fallback={<p class="text-sm text-faint">No findings recorded.</p>}
				>
					<ul class="space-y-1.5">
						<For each={flagged()}>
							{(f) => (
								<li class="rounded-md border border-border bg-surface px-3 py-2 text-sm">
									<span
										class="mr-2 text-xs font-medium"
										classList={{
											"text-destructive-strong": f.severity === "block",
											"text-muted": f.severity !== "block",
										}}
									>
										{SEVERITY_LABEL[f.severity]}
									</span>
									<span class="text-foreground">{f.message}</span>
									<Show when={f.slotId}>
										<span class="ml-2 font-mono text-xs text-faint">
											{f.slotId}
										</span>
									</Show>
								</li>
							)}
						</For>
					</ul>
				</Show>
			</section>

			<section aria-labelledby="draft-skill-gaps">
				<h2 id="draft-skill-gaps" class="mb-2 text-sm font-semibold">
					Skill gaps
				</h2>
				<Show
					when={gaps().length > 0}
					fallback={<p class="text-sm text-faint">—</p>}
				>
					<ul class="list-inside list-disc space-y-1 text-sm text-muted">
						<For each={gaps()}>{(g) => <li>{g}</li>}</For>
					</ul>
				</Show>
			</section>

			<section aria-labelledby="draft-bullets">
				<h2 id="draft-bullets" class="mb-1 text-sm font-semibold">
					Rewritten bullets
				</h2>
				<p class="mb-3 text-xs text-faint">
					Highlighted words do not appear in the Achievements the bullet cites.
					Check each one against your own experience.
				</p>
				<For each={props.provenance?.positions ?? []}>
					{(p) => (
						<div class="mb-4">
							<h3 class="mb-2 text-xs font-medium text-muted">
								{p.title}, {p.employer}
							</h3>
							<ul class="space-y-2">
								<For each={p.bullets}>{(b) => <Bullet bullet={b} />}</For>
							</ul>
						</div>
					)}
				</For>
			</section>
		</div>
	);
}
