import { For, type JSX, Show } from "solid-js";
import { Icon, type IconName } from "@/components/Icon";
import {
	SortableTableHead,
	type SortDir,
} from "@/components/SortableTableHead";
import { ToggleChip } from "@/components/ToggleChip";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import { TableCell, TableRow } from "@/components/ui/table";
import {
	relativeTime,
	type SearchParams,
	type SearchSortKey,
	type SearchTab,
} from "@/lib/searchTargets";
import { cn } from "@/lib/utils";
import type { SourceTarget } from "@/types/sourceTarget";

const TABS: { key: SearchTab; label: string; icon: IconName }[] = [
	{ key: "boards", label: "Job board searches", icon: "search" },
	{ key: "ats", label: "Company boards", icon: "building" },
];

export function SegmentedTabs(props: {
	value: SearchTab;
	onChange: (tab: SearchTab) => void;
	boardCount: number | undefined;
	companyCount: number | undefined;
}) {
	const count = (key: SearchTab) =>
		key === "boards" ? props.boardCount : props.companyCount;
	return (
		<div role="tablist" class="inline-flex rounded-md border border-border">
			<For each={TABS}>
				{(t) => (
					<button
						type="button"
						role="tab"
						aria-selected={props.value === t.key}
						onClick={() => props.onChange(t.key)}
						class={cn(
							"inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium transition-colors first:rounded-l-[5px] last:rounded-r-[5px] not-last:border-r not-last:border-border focus-visible:relative focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
							props.value === t.key
								? "bg-primary/10 text-primary"
								: "bg-surface text-muted hover:bg-surface-muted hover:text-foreground",
						)}
					>
						<Icon name={t.icon} size={13} />
						{t.label}
						<Show when={count(t.key) !== undefined}>
							<span class="font-mono tabular-nums">{count(t.key)}</span>
						</Show>
					</button>
				)}
			</For>
		</div>
	);
}

export function FilterChips(props: {
	label: string;
	options: { value: string; label: string; count?: number }[];
	value: string;
	onChange: (value: string) => void;
}) {
	return (
		<fieldset class="flex flex-wrap items-center gap-1">
			<legend class="sr-only">{props.label}</legend>
			<For each={props.options}>
				{(o) => (
					<ToggleChip
						active={props.value === o.value}
						onClick={() => props.onChange(o.value)}
						class="h-7 px-2.5"
					>
						{o.label}
						<Show when={o.count !== undefined}>
							<span class="font-mono tabular-nums">{o.count}</span>
						</Show>
					</ToggleChip>
				)}
			</For>
		</fieldset>
	);
}

export function EnabledSwitch(props: {
	enabled: boolean;
	name: string;
	onToggle: () => void;
}) {
	return (
		<Switch checked={props.enabled} onChange={props.onToggle}>
			<SwitchLabel class="sr-only">
				{props.enabled ? "Pause" : "Resume"} {props.name}
			</SwitchLabel>
			<SwitchControl>
				<SwitchThumb />
			</SwitchControl>
		</Switch>
	);
}

const RUN_STATUS: Record<
	SourceTarget["RunStatus"],
	{ label: string; dot: string }
> = {
	idle: { label: "Not run yet", dot: "bg-border-strong" },
	queued: { label: "Queued", dot: "bg-faint motion-safe:animate-pulse" },
	running: { label: "Running", dot: "bg-primary motion-safe:animate-pulse" },
	succeeded: { label: "Done", dot: "bg-status-offer" },
	failed: { label: "Failed", dot: "bg-destructive" },
};

export function RunStatus(props: { target: SourceTarget }) {
	const s = () => RUN_STATUS[props.target.RunStatus];
	const showTime = () =>
		props.target.LastRunAt &&
		(props.target.RunStatus === "succeeded" ||
			props.target.RunStatus === "failed");
	return (
		<span
			class="inline-flex items-center gap-1.5 whitespace-nowrap text-xs text-muted"
			title={props.target.LastRunError || undefined}
		>
			<span class={cn("size-1.5 shrink-0 rounded-full", s().dot)} />
			{s().label}
			<Show when={showTime()}>
				<span class="text-faint">· {relativeTime(props.target.LastRunAt)}</span>
			</Show>
		</span>
	);
}

export function SortHead(props: {
	sortKey: SearchSortKey;
	firstDir: "asc" | "desc";
	params: SearchParams;
	onParams: (patch: Partial<SearchParams>) => void;
	children: JSX.Element;
}) {
	const sorted = (): SortDir =>
		props.params.sort !== props.sortKey
			? "none"
			: props.params.dir === "desc"
				? "desc"
				: "asc";
	return (
		<SortableTableHead
			sorted={sorted()}
			onToggle={() =>
				props.onParams({
					sort: props.sortKey,
					dir:
						props.params.sort === props.sortKey
							? props.params.dir === "desc"
								? "asc"
								: "desc"
							: props.firstDir,
					page: undefined,
				})
			}
		>
			{props.children}
		</SortableTableHead>
	);
}

export function EmptyRow(props: { colSpan: number; children: JSX.Element }) {
	return (
		<TableRow class="hover:bg-transparent">
			<TableCell
				colSpan={props.colSpan}
				class="py-10 text-center text-sm text-muted"
			>
				{props.children}
			</TableCell>
		</TableRow>
	);
}

export function DeleteIconButton(props: {
	label: string;
	title: string;
	onClick: () => void;
}) {
	return (
		<button
			type="button"
			onClick={() => props.onClick()}
			aria-label={props.label}
			title={props.title}
			class="grid size-7 place-items-center rounded text-faint transition hover:bg-destructive-subtle hover:text-destructive-strong focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-destructive"
		>
			<Icon name="trash" size={13} />
		</button>
	);
}
