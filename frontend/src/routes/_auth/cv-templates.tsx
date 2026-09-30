import { createFileRoute } from "@tanstack/solid-router";
import { createMemo, createSignal, Show } from "solid-js";
import { AddDocDialog } from "@/components/cv/AddDocDialog";
import { Icon } from "@/components/Icon";
import { PageHeading } from "@/components/PageHeading";
import type { SortDir } from "@/components/SortableTableHead";
import { Button } from "@/components/ui/button";
import {
	cvTemplatesQueryOptions,
	useCVTemplates,
	useHideTab,
	useShowTab,
} from "../../hooks/useCVTemplates";
import { useTableSort } from "../../hooks/useTableSort";
import { queryClient } from "../../lib/queryClient";
import type { CV } from "../../types/cv";
import {
	CVTemplatesAllHidden,
	CVTemplatesEmpty,
	CVTemplatesError,
	CVTemplatesLoading,
	CVTemplatesNotConnected,
} from "./-cv-templates/CVTemplatesStates";
import {
	CVTemplatesTable,
	type SortKey,
} from "./-cv-templates/CVTemplatesTable";
import { CVTemplatesToolbar } from "./-cv-templates/CVTemplatesToolbar";

export const Route = createFileRoute("/_auth/cv-templates")({
	loader: () => queryClient.ensureQueryData(cvTemplatesQueryOptions),
	component: CVTemplatesPage,
});

function CVTemplatesPage() {
	const query = useCVTemplates();
	const hideMutation = useHideTab();
	const showMutation = useShowTab();

	const handleHide = (cv: CV) => {
		hideMutation.mutate({ docId: cv.DocID, tabId: cv.TabID });
	};

	const handleRestore = (cv: CV) => {
		showMutation.mutate({ docId: cv.DocID, tabId: cv.TabID });
	};

	const [searchQuery, setSearchQuery] = createSignal("");
	const [showHidden, setShowHidden] = createSignal(false);
	const [dialogOpen, setDialogOpen] = createSignal(false);

	const { sortKey, sortDir, handleSort } = useTableSort<SortKey>("Title");

	const ariaSort = (key: SortKey): SortDir =>
		sortKey() === key ? sortDir() : "none";

	const filteredSorted = createMemo<CV[]>(() => {
		const q = searchQuery().toLowerCase();
		const data = query.data ?? [];
		const visibilityFiltered = showHidden()
			? data
			: data.filter((cv) => cv.Visible);
		const filtered = q
			? visibilityFiltered.filter(
					(cv) =>
						cv.Title.toLowerCase().includes(q) ||
						cv.SourceDoc.toLowerCase().includes(q),
				)
			: visibilityFiltered;

		const key = sortKey();
		const dir = sortDir();
		return [...filtered].sort((a, b) => {
			const av = a[key];
			const bv = b[key];
			const cmp = av < bv ? -1 : av > bv ? 1 : 0;
			return dir === "asc" ? cmp : -cmp;
		});
	});

	const hasHiddenCVs = () => (query.data ?? []).some((cv) => !cv.Visible);

	const isNotConnected = () =>
		!!query.error?.message?.toLowerCase().includes("not connected");

	return (
		<div class="px-7 py-6">
			<PageHeading title="CVs" subtitle="Google Docs tracked as CVs">
				<Button class="shrink-0" onClick={() => setDialogOpen(true)}>
					<Icon name="plus" size={12} strokeWidth={2.5} />
					Add doc
				</Button>
			</PageHeading>

			<Show when={query.isPending}>
				<CVTemplatesLoading />
			</Show>

			<Show when={query.isError && isNotConnected()}>
				<CVTemplatesNotConnected />
			</Show>

			<Show when={query.isError && !isNotConnected()}>
				<CVTemplatesError message={query.error?.message} />
			</Show>

			<Show when={query.isSuccess}>
				<Show when={(query.data?.length ?? 0) === 0}>
					<CVTemplatesEmpty onAdd={() => setDialogOpen(true)} />
				</Show>

				<Show when={(query.data?.length ?? 0) > 0}>
					<CVTemplatesToolbar
						searchQuery={searchQuery()}
						onSearch={setSearchQuery}
						showHidden={showHidden()}
						onShowHidden={setShowHidden}
					/>

					<Show
						when={
							filteredSorted().length === 0 &&
							!searchQuery() &&
							!showHidden() &&
							hasHiddenCVs()
						}
					>
						<CVTemplatesAllHidden onShowHidden={() => setShowHidden(true)} />
					</Show>

					<Show when={filteredSorted().length > 0 || searchQuery()}>
						<CVTemplatesTable
							rows={filteredSorted()}
							searchQuery={searchQuery()}
							ariaSort={ariaSort}
							onSort={handleSort}
							hidePending={hideMutation.isPending}
							showPending={showMutation.isPending}
							onHide={handleHide}
							onRestore={handleRestore}
						/>
					</Show>
				</Show>
			</Show>

			<AddDocDialog open={dialogOpen()} onOpenChange={setDialogOpen} />
		</div>
	);
}
