import { For, Show } from "solid-js";
import {
	SortableTableHead,
	type SortDir,
} from "@/components/SortableTableHead";
import { Card } from "@/components/ui/card";
import {
	Table,
	TableBody,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import type { CV } from "@/types/cv";
import { CVRow } from "./CVRow";

export type SortKey = "Title" | "ModifiedAt";

export function CVTemplatesTable(props: {
	rows: CV[];
	searchQuery: string;
	ariaSort: (key: SortKey) => SortDir;
	onSort: (key: SortKey) => void;
	hidePending: boolean;
	showPending: boolean;
	onHide: (cv: CV) => void;
	onRestore: (cv: CV) => void;
}) {
	return (
		<Card class="overflow-hidden">
			<Table>
				<TableHeader>
					<TableRow>
						<SortableTableHead
							sorted={props.ariaSort("Title")}
							onToggle={() => props.onSort("Title")}
						>
							Title
						</SortableTableHead>
						<TableHead>Source doc</TableHead>
						<SortableTableHead
							sorted={props.ariaSort("ModifiedAt")}
							onToggle={() => props.onSort("ModifiedAt")}
						>
							Last modified
						</SortableTableHead>
						<TableHead class="w-10" />
					</TableRow>
				</TableHeader>
				<TableBody>
					<For each={props.rows}>
						{(cv) => (
							<CVRow
								cv={cv}
								hidePending={props.hidePending}
								showPending={props.showPending}
								onHide={() => props.onHide(cv)}
								onRestore={() => props.onRestore(cv)}
							/>
						)}
					</For>
				</TableBody>
			</Table>

			<Show when={props.rows.length === 0 && props.searchQuery}>
				<div class="px-4 py-8 text-center text-sm text-muted">
					No CVs match "{props.searchQuery}"
				</div>
			</Show>
		</Card>
	);
}
