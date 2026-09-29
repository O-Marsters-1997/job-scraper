import { Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { QueryBoundary } from "@/components/QueryBoundary";
import { useCVTemplates } from "../../../hooks/useCVTemplates";
import type { CVRef } from "../../../hooks/useTailoring";
import type { CV } from "../../../types/cv";
import { Panel } from "./Panel";

export function CvStep(props: { onPick: (ref: CVRef) => void }) {
	const query = useCVTemplates();
	const visible = (cvs: CV[]) => cvs.filter((c) => c.Visible);
	return (
		<QueryBoundary query={query}>
			{(cvs) => (
				<Show
					when={visible(cvs()).length > 0}
					fallback={
						<Panel>
							<p class="text-sm text-muted">
								No CVs tracked yet.{" "}
								<Link to="/cv-templates" class="text-accent-text underline">
									Track a Google Doc
								</Link>{" "}
								first.
							</p>
						</Panel>
					}
				>
					<ul class="space-y-2">
						<For each={visible(cvs())}>
							{(c) => (
								<li>
									<button
										type="button"
										class="w-full rounded-xl border border-border bg-surface p-4 text-left transition-colors hover:border-primary"
										onClick={() =>
											props.onPick({ docId: c.DocID, tabId: c.TabID })
										}
									>
										<span class="block text-sm font-semibold text-foreground">
											{c.Title}
										</span>
										<span class="block text-xs text-muted">{c.SourceDoc}</span>
									</button>
								</li>
							)}
						</For>
					</ul>
				</Show>
			)}
		</QueryBoundary>
	);
}
