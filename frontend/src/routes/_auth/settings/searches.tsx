import { createFileRoute } from "@tanstack/solid-router";
import { createMemo, createSignal, onCleanup, Show } from "solid-js";
import { untrackCompany } from "@/api/companies";
import { keys } from "@/api/keys";
import { deleteSourceTarget } from "@/api/sourceTargets";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { parseSearchParams, type SearchParams } from "@/lib/searchTargets";
import { AlreadyRunningError } from "../../../api/sourceTargets";
import {
	useAddCompanyBoard,
	useSetCompanyTracking,
	useTrackedCompanies,
} from "../../../hooks/useCompanies";
import { useSources } from "../../../hooks/useSources";
import {
	useRerunSourceTarget,
	useSourceTargets,
	useUpdateSourceTarget,
} from "../../../hooks/useSourceTargets";
import type { TrackedBoard, TrackedCompany } from "../../../types/company";
import type { SourceInfo } from "../../../types/source";
import type { SourceTarget } from "../../../types/sourceTarget";
import { BoardSearchesTable } from "./-searches/BoardSearchesTable";
import { CompanyBoardsTable } from "./-searches/CompanyBoardsTable";
import { PasteBox } from "./-searches/PasteBox";
import { SegmentedTabs } from "./-searches/parts";
import { SearchForm, type SearchPrefill } from "./-searches/SearchForm";
import { TrackCompanyBox } from "./-searches/TrackCompanyBox";
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
	const [confirming, setConfirming] = createSignal<string[]>([]);
	const trackedQuery = useTrackedCompanies(() => confirming().length > 0);
	const trackingMutation = useSetCompanyTracking();
	const confirmMutation = useAddCompanyBoard();

	const [showForm, setShowForm] = createSignal(false);
	const [prefill, setPrefill] = createSignal<SearchPrefill>();
	const [notice, setNotice] = createSignal<string | null>(null);
	const [error, setError] = createSignal<string | null>(null);
	let noticeTimer: ReturnType<typeof setTimeout> | undefined;
	onCleanup(() => clearTimeout(noticeTimer));

	const announce = (message: string) => {
		setNotice(message);
		clearTimeout(noticeTimer);
		noticeTimer = setTimeout(() => setNotice(null), 6000);
	};

	const undo = useUndoDelete({
		commit: (id, keepalive) => deleteSourceTarget(id, { keepalive }),
		queryKey: keys.sourceTargets,
		verb: "Deleted",
		errorMessage: "Could not delete the search. Please try again.",
		onError: setError,
	});
	const untrack = useUndoDelete({
		commit: (id, keepalive) => untrackCompany(id, { keepalive }),
		queryKey: keys.companies.tracked,
		verb: "Untracked",
		errorMessage: "Could not untrack the company. Please try again.",
		onError: setError,
	});

	const params = createMemo(() => parseSearchParams(search()));
	const tab = () => params().tab ?? "boards";
	const boardSources = () =>
		(sourcesQuery.data ?? []).filter((s) => s.role === "discovery");
	const visibleTargets = (all: SourceTarget[]) =>
		all.filter((t) => !undo.isHidden(t.ID));

	const visibleCompanies = (all: TrackedCompany[]) =>
		all.filter((c) => !untrack.isHidden(c.id));

	const setParams = (
		patch: Partial<SearchParams>,
		opts?: { replace?: boolean },
	) =>
		navigate({
			search: (prev: SearchParams) => ({ ...prev, ...patch }),
			replace: opts?.replace ?? false,
		});

	const closeForm = () => {
		setShowForm(false);
		setPrefill(undefined);
	};

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

	const toggleCompany = (c: TrackedCompany) => {
		setError(null);
		trackingMutation.mutate(
			{
				id: c.id,
				enabled: !c.enabled,
				checkIntervalMinutes: c.check_interval_minutes,
			},
			{ onError: () => setError("Could not update the company.") },
		);
	};

	const confirmBoard = (c: TrackedCompany, board: TrackedBoard) => {
		setError(null);
		setConfirming((ids) => [...ids, board.id]);
		confirmMutation.mutate(
			{ id: c.id, url: board.url, confirm: true },
			{
				onSuccess: () => announce(`Verification queued for ${c.name}.`),
				onError: () => setError("Could not confirm the board."),
				onSettled: () =>
					setTimeout(
						() => setConfirming((ids) => ids.filter((x) => x !== board.id)),
						30000,
					),
			},
		);
	};

	const onCreated = (created: SourceTarget, info: SourceInfo | undefined) => {
		closeForm();
		if (created.RunStatus === "failed") {
			setError("Search saved, but it could not start. Use Run now to retry.");
		} else {
			announce(
				`Added “${created.Value}” on ${info?.label ?? created.Source}. First run queued.`,
			);
		}
	};

	return (
		<>
			<FormFeedback success={notice() ?? false} error={error()} />

			<PasteBox
				targets={query.data}
				onSearch={(r) => {
					setNotice(null);
					setPrefill({
						source: r.source,
						value: r.value,
						filters: r.filters,
						dropped: r.dropped,
					});
					setShowForm(true);
					setParams({ tab: undefined, page: undefined });
				}}
			/>

			<div class="mb-3 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
				<SegmentedTabs
					value={tab()}
					onChange={(t) =>
						setParams({
							tab: t === "ats" ? "ats" : undefined,
							q: undefined,
							src: undefined,
							status: undefined,
							sort: undefined,
							dir: undefined,
							page: undefined,
						})
					}
					boardCount={query.data?.length}
					companyCount={trackedQuery.data?.length}
				/>
				<Show when={tab() === "boards" && !showForm()}>
					<Button
						size="sm"
						onClick={() => {
							setNotice(null);
							setPrefill(undefined);
							setShowForm(true);
						}}
					>
						<Icon name="plus" size={12} strokeWidth={2.5} />
						Build from fields
					</Button>
				</Show>
			</div>

			<Show when={tab() === "ats"}>
				<div class="flex flex-col gap-3">
					<TrackCompanyBox
						sources={sourcesQuery.data ?? []}
						onTracked={(name) => {
							setError(null);
							announce(`Tracking ${name}. Confirm the board to start polling.`);
						}}
						onError={setError}
					/>
					<QueryBoundary query={trackedQuery} fallbackRows={4}>
						{(data) => (
							<CompanyBoardsTable
								companies={visibleCompanies(data())}
								sources={sourcesQuery.data ?? []}
								params={params()}
								onParams={setParams}
								onToggle={toggleCompany}
								onUntrack={(c) => untrack.remove(c.id, c.name)}
								onConfirm={confirmBoard}
								confirming={(id) => confirming().includes(id)}
							/>
						)}
					</QueryBoundary>
				</div>
			</Show>

			<Show when={tab() === "boards"}>
				<QueryBoundary query={query} fallbackRows={4}>
					{(data) => (
						<>
							<Show when={showForm() && sourcesQuery.isSuccess}>
								<Show
									when={prefill()}
									keyed
									fallback={
										<SearchForm
											sources={boardSources()}
											onCancel={closeForm}
											onCreated={onCreated}
										/>
									}
								>
									{(initial) => (
										<SearchForm
											sources={boardSources()}
											initial={initial}
											onCancel={closeForm}
											onCreated={onCreated}
										/>
									)}
								</Show>
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

			<UndoToasts
				items={[...undo.pending(), ...untrack.pending()]}
				onUndo={(id) => {
					undo.undo(id);
					untrack.undo(id);
				}}
			/>
		</>
	);
}
