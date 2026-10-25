import { createMemo, createSignal, Show } from "solid-js";
import { AddDocDialog } from "@/components/cv/AddDocDialog";
import { Icon } from "@/components/Icon";
import { PageHeading } from "@/components/PageHeading";
import type { SortDir } from "@/components/SortableTableHead";
import { Button } from "@/components/ui/button";
import { SkeletonList } from "@/components/ui/skeleton";
import {
	useCVTemplates,
	useHideTab,
	useSetDefaultTab,
	useShowTab,
} from "../../../hooks/useCVTemplates";
import { useTableSort } from "../../../hooks/useTableSort";
import type { CV, TrackedDocKind } from "../../../types/cv";
import {
	CVTemplatesAllHidden,
	CVTemplatesEmpty,
	CVTemplatesError,
	CVTemplatesNotConnected,
} from "./CVTemplatesStates";
import { CVTemplatesTable, type SortKey } from "./CVTemplatesTable";
import { CVTemplatesToolbar } from "./CVTemplatesToolbar";

export function TrackedDocsSection(props: {
	kind: TrackedDocKind;
	title: string;
	subtitle: string;
}) {
	const query = useCVTemplates(() => props.kind);
	const hideMutation = useHideTab();
	const showMutation = useShowTab();
	const defaultMutation = useSetDefaultTab();

	const handleHide = (cv: CV) => {
		hideMutation.mutate({ docId: cv.DocID, tabId: cv.TabID });
	};

	const handleRestore = (cv: CV) => {
		showMutation.mutate({ docId: cv.DocID, tabId: cv.TabID });
	};

	const cvs = (): CV[] => (query.isSuccess ? (query.data ?? []) : []);
	const [searchQuery, setSearchQuery] = createSignal("");
	const [showHidden, setShowHidden] = createSignal(false);
	const [dialogOpen, setDialogOpen] = createSignal(false);

	const { sortKey, sortDir, handleSort } = useTableSort<SortKey>("Title");

	const ariaSort = (key: SortKey): SortDir =>
		sortKey() === key ? sortDir() : "none";

	const filteredSorted = createMemo<CV[]>(() => {
		const q = searchQuery().toLowerCase();
		const data = cvs();
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

	const hasHiddenCVs = () => cvs().some((cv) => !cv.Visible);

	const isNotConnected = () =>
		!!query.error?.message?.toLowerCase().includes("not connected");

	return (
		<section class="mb-10">
			<PageHeading
				title={props.title}
				subtitle={props.subtitle}
				level={props.kind === "cv" ? "h1" : "h2"}
			>
				<Button class="shrink-0" onClick={() => setDialogOpen(true)}>
					<Icon name="plus" size={12} strokeWidth={2.5} />
					Add doc
				</Button>
			</PageHeading>

			<Show when={query.isPending}>
				<SkeletonList rows={3} />
			</Show>

			<Show when={query.isError && isNotConnected()}>
				<CVTemplatesNotConnected />
			</Show>

			<Show when={query.isError && !isNotConnected()}>
				<CVTemplatesError message={query.error?.message} />
			</Show>

			<Show when={query.isSuccess}>
				<Show when={cvs().length === 0}>
					<CVTemplatesEmpty onAdd={() => setDialogOpen(true)} />
				</Show>

				<Show when={cvs().length > 0}>
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
							onSetDefault={
								props.kind === "cv"
									? (cv) =>
											defaultMutation.mutate({
												docId: cv.DocID,
												tabId: cv.TabID,
											})
									: undefined
							}
						/>
					</Show>
				</Show>
			</Show>

			<AddDocDialog
				open={dialogOpen()}
				onOpenChange={setDialogOpen}
				kind={props.kind}
			/>
		</section>
	);
}
