import { For, Show } from "solid-js";
import { Icon, type IconName } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import { relativeTime, type SearchTab } from "@/lib/searchTargets";
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
							<span class="font-mono tabular-nums opacity-70">
								{count(t.key)}
							</span>
						</Show>
					</button>
				)}
			</For>
		</div>
	);
}

export function SearchField(props: {
	id: string;
	label: string;
	placeholder: string;
	value: string;
	onInput: (value: string) => void;
	class?: string;
}) {
	return (
		<div class={cn("relative w-full sm:max-w-64", props.class)}>
			<Icon
				name="search"
				size={14}
				class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-faint"
			/>
			<Label for={props.id} class="sr-only">
				{props.label}
			</Label>
			<Input
				id={props.id}
				type="search"
				placeholder={props.placeholder}
				value={props.value}
				onInput={(e) => props.onInput(e.currentTarget.value)}
				class="h-8 pr-3 pl-9 text-sm"
			/>
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
					<button
						type="button"
						aria-pressed={props.value === o.value}
						onClick={() => props.onChange(o.value)}
						class={cn(
							"inline-flex h-7 items-center gap-1.5 rounded-full border px-2.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
							props.value === o.value
								? "border-accent-border bg-accent-subtle text-accent-text"
								: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
						)}
					>
						{o.label}
						<Show when={o.count !== undefined}>
							<span class="font-mono tabular-nums opacity-70">{o.count}</span>
						</Show>
					</button>
				)}
			</For>
		</fieldset>
	);
}

export function Pager(props: {
	from: number;
	to: number;
	total: number;
	page: number;
	pageCount: number;
	noun: string;
	onPage: (page: number) => void;
}) {
	return (
		<div class="flex items-center justify-end gap-2 border-t border-border px-4 py-2.5">
			<span class="mr-auto text-xs text-faint" aria-live="polite">
				<Show when={props.total > 0} fallback={`No ${props.noun}`}>
					<span class="font-mono tabular-nums">
						{props.from}–{props.to}
					</span>{" "}
					of <span class="font-mono tabular-nums">{props.total}</span>{" "}
					{props.noun}
				</Show>
			</span>
			<Show when={props.pageCount > 1}>
				<span class="hidden text-xs text-faint sm:inline">
					Page {props.page} of {props.pageCount}
				</span>
				<Button
					variant="outline"
					size="sm"
					disabled={props.page <= 1}
					onClick={() => props.onPage(props.page - 1)}
				>
					Previous
				</Button>
				<Button
					variant="outline"
					size="sm"
					disabled={props.page >= props.pageCount}
					onClick={() => props.onPage(props.page + 1)}
				>
					Next
				</Button>
			</Show>
		</div>
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
