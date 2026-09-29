import { createFileRoute, Link } from "@tanstack/solid-router";
import { createMemo, createSignal, onCleanup, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { parseSearchParams, type SearchParams } from "@/lib/searchTargets";
import { AlreadyRunningError } from "../../../api/sourceTargets";
import { useSources } from "../../../hooks/useSources";
import {
	useRerunSourceTarget,
	useSourceTargets,
	useUpdateSourceTarget,
} from "../../../hooks/useSourceTargets";
import type { SourceTarget } from "../../../types/sourceTarget";
import { BoardSearchesTable } from "./-searches/BoardSearchesTable";
import { SegmentedTabs } from "./-searches/parts";
import { SearchForm } from "./-searches/SearchForm";
import { UndoToasts, useUndoDelete } from "./-searches/useUndoDelete";

export const Route = createFileRoute("/_auth/settings/searches")({
	validateSearch: (raw: Record<string, unknown>): SearchParams =>
		parseSearchParams(raw),
	component: SearchesPage,
});

function SearchesPage() {
	const search = Route.useSearch();
	const navigate = Route.useNavigate();
	const query = useSourceTargets();
	const sourcesQuery = useSources();
	const updateMutation = useUpdateSourceTarget();
	const rerunMutation = useRerunSourceTarget();

	const [showForm, setShowForm] = createSignal(false);
	const [notice, setNotice] = createSignal<string | null>(null);
	const [error, setError] = createSignal<string | null>(null);
	let noticeTimer: ReturnType<typeof setTimeout> | undefined;
	onCleanup(() => clearTimeout(noticeTimer));

	const announce = (message: string) => {
		setNotice(message);
		clearTimeout(noticeTimer);
		noticeTimer = setTimeout(() => setNotice(null), 6000);
	};

	const undo = useUndoDelete(setError);

	const params = createMemo(() => parseSearchParams(search()));
	const tab = () => params().tab ?? "boards";
	const boardSources = () =>
		(sourcesQuery.data ?? []).filter((s) => s.role === "discovery");
	const visibleTargets = (all: SourceTarget[]) =>
		all.filter((t) => !undo.isHidden(t.ID));

	const setParams = (
		patch: Partial<SearchParams>,
		opts?: { replace?: boolean },
	) =>
		navigate({
			search: (prev: SearchParams) => ({ ...prev, ...patch }),
			replace: opts?.replace ?? false,
		});

	const run = async (t: SourceTarget) => {
		setError(null);
		setNotice(null);
		try {
			await rerunMutation.mutateAsync(t.ID);
			announce("Search queued. Matching jobs will appear shortly.");
		} catch (err) {
			setError(
				err instanceof AlreadyRunningError
					? "That search is already running."
					: "Could not start the search. Please try again.",
			);
		}
	};

	const toggle = (t: SourceTarget) => {
		setError(null);
		updateMutation.mutate(
			{ id: t.ID, enabled: !t.Enabled },
			{ onError: () => setError("Could not update the search.") },
		);
	};

	return (
		<>
			<FormFeedback success={notice() ?? false} error={error()} />

			<div class="mb-3 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
				<SegmentedTabs
					value={tab()}
					onChange={(t) =>
						setParams({ tab: t === "ats" ? "ats" : undefined, page: undefined })
					}
					boardCount={query.data?.length}
				/>
				<Show when={tab() === "boards" && !showForm()}>
					<Button
						size="sm"
						onClick={() => {
							setNotice(null);
							setShowForm(true);
						}}
					>
						<Icon name="plus" size={12} strokeWidth={2.5} />
						Build from fields
					</Button>
				</Show>
			</div>

			<Show when={tab() === "ats"}>
				<Card class="px-4 py-6 text-sm text-muted">
					Company boards are managed from the{" "}
					<Link
						to="/companies"
						class="font-medium text-primary hover:underline"
					>
						Companies directory
					</Link>
					.
				</Card>
			</Show>

			<Show when={tab() === "boards"}>
				<QueryBoundary query={query} fallbackRows={4}>
					{(data) => (
						<>
							<Show when={showForm() && sourcesQuery.isSuccess}>
								<SearchForm
									sources={boardSources()}
									onCancel={() => setShowForm(false)}
									onCreated={(created, info) => {
										setShowForm(false);
										if (created.RunStatus === "failed") {
											setError(
												"Search saved, but it could not start. Use Run now to retry.",
											);
										} else {
											announce(
												`Added “${created.Value}” on ${info?.label ?? created.Source}. First run queued.`,
											);
										}
									}}
								/>
							</Show>
							<BoardSearchesTable
								targets={visibleTargets(data())}
								sources={boardSources()}
								params={params()}
								onParams={setParams}
								onToggle={toggle}
								onRun={run}
								onDelete={(t) => undo.remove(t.ID, t.Value)}
								runPending={rerunMutation.isPending}
							/>
						</>
					)}
				</QueryBoundary>
			</Show>

			<UndoToasts items={undo.pending()} onUndo={undo.undo} />
		</>
	);
}
