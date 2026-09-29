import { createMemo, createSignal, For, type JSX, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import type { SourceTarget } from "@/types/sourceTarget";
import {
	BOARD_SPECS,
	buildBoardUrl,
	draftFromParse,
	type ParamSpec,
	type ParseOk,
	parseBoardUrl,
	type SearchDraft,
	specFor,
} from "./boardUrl";
import { relativeTime, type SourceTab } from "./useProtoSearches";

const TABS: { key: SourceTab; label: string; icon: "building" | "search" }[] = [
	{ key: "boards", label: "Job board searches", icon: "search" },
	{ key: "ats", label: "Company boards", icon: "building" },
];

export function SegmentedTabs(props: {
	value: SourceTab;
	onChange: (tab: SourceTab) => void;
	counts: Record<SourceTab, number>;
}) {
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
						<span
							class={cn(
								"font-mono tabular-nums",
								props.value === t.key ? "text-primary/70" : "text-faint",
							)}
						>
							{props.counts[t.key]}
						</span>
					</button>
				)}
			</For>
		</div>
	);
}

export function EnabledSwitch(props: {
	target: SourceTarget;
	onToggle: () => void;
	name: string;
}) {
	return (
		<Switch checked={props.target.Enabled} onChange={props.onToggle}>
			<SwitchLabel class="sr-only">
				{props.target.Enabled ? "Pause" : "Resume"} {props.name}
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

export function useBoardDraft() {
	const empty = (): SearchDraft => ({
		source: "linkedin",
		keywords: "",
		filters: {},
	});
	const [urlText, setUrlText] = createSignal("");
	const [draft, setDraft] = createSignal<SearchDraft>(empty());
	const [parsed, setParsed] = createSignal<ParseOk | null>(null);
	const [error, setError] = createSignal<string | null>(null);

	const pasteUrl = (text: string) => {
		setUrlText(text);
		setError(null);
		setParsed(null);
		if (!text.trim()) return;
		const r = parseBoardUrl(text);
		if (!r.ok) {
			setError(r.reason);
			return;
		}
		setParsed(r);
		setDraft(draftFromParse(r));
	};

	const edit = (next: SearchDraft) => {
		setDraft(next);
		setUrlText(buildBoardUrl(next));
	};

	const baseUrl = createMemo(() =>
		draft().keywords.trim() || Object.values(draft().filters).some(Boolean)
			? buildBoardUrl(draft())
			: "",
	);

	return {
		urlText,
		pasteUrl,
		draft,
		setSource: (source: string) =>
			edit({ source, keywords: draft().keywords, filters: {} }),
		setKeywords: (keywords: string) => edit({ ...draft(), keywords }),
		setFilter: (key: string, value: string) =>
			edit({ ...draft(), filters: { ...draft().filters, [key]: value } }),
		parsed,
		error,
		baseUrl,
		reset: () => {
			setUrlText("");
			setParsed(null);
			setError(null);
			setDraft(empty());
		},
	};
}

export type BoardDraft = ReturnType<typeof useBoardDraft>;

export function FieldLabel(props: { for: string; children: JSX.Element }) {
	return (
		<label for={props.for} class="text-xs font-medium text-foreground">
			{props.children}
		</label>
	);
}

type Option = { value: string; label: string };
const ANY = "__any__";

function OptionSelect(props: {
	param: ParamSpec;
	options: Record<string, string>;
	value: string;
	onChange: (value: string) => void;
}) {
	const options = (): Option[] => {
		const list = [
			{ value: ANY, label: "Any" },
			...Object.entries(props.options).map(([value, label]) => ({
				value,
				label,
			})),
		];
		if (props.value && !props.options[props.value]) {
			list.push({ value: props.value, label: `${props.value} (from URL)` });
		}
		return list;
	};
	return (
		<Select<Option>
			class="flex flex-col gap-1.5"
			options={options()}
			optionValue="value"
			optionTextValue="label"
			value={options().find((o) => o.value === (props.value || ANY)) ?? null}
			onChange={(o) => props.onChange(!o || o.value === ANY ? "" : o.value)}
			itemComponent={(p) => (
				<SelectItem item={p.item}>
					<SelectItemLabel>{p.item.rawValue.label}</SelectItemLabel>
				</SelectItem>
			)}
		>
			<Select.Label class="text-xs font-medium text-foreground">
				{props.param.label}
			</Select.Label>
			<SelectTrigger>
				<Select.Value<Option>>
					{(state) => state.selectedOption()?.label ?? "Any"}
				</Select.Value>
			</SelectTrigger>
			<SelectContent />
		</Select>
	);
}

const PRIMARY_FILTERS = 2;

export function DraftFields(props: { board: BoardDraft }) {
	const spec = () => specFor(props.board.draft().source);
	const value = (key: string) => props.board.draft().filters[key] ?? "";
	const [showAll, setShowAll] = createSignal(false);
	const visible = () =>
		(spec()?.params ?? []).filter(
			(p, i) => showAll() || i < PRIMARY_FILTERS || value(p.key),
		);
	const hiddenCount = () => (spec()?.params.length ?? 0) - visible().length;
	const unknownKeys = () =>
		Object.keys(props.board.draft().filters).filter(
			(k) => !spec()?.params.some((p) => p.key === k),
		);

	return (
		<div class="flex flex-col gap-3">
			<div class="flex flex-col gap-1.5">
				<FieldLabel for="draft-keywords">Keywords</FieldLabel>
				<Input
					id="draft-keywords"
					placeholder="e.g. product engineer"
					value={props.board.draft().keywords}
					onInput={(e) => props.board.setKeywords(e.currentTarget.value)}
				/>
			</div>
			<div class="grid gap-3 sm:grid-cols-2">
				<For each={visible()}>
					{(p) => (
						<Show
							when={p.options}
							fallback={
								<div class="flex flex-col gap-1.5">
									<FieldLabel for={`draft-${p.key}`}>{p.label}</FieldLabel>
									<Input
										id={`draft-${p.key}`}
										placeholder={p.placeholder ?? "Any"}
										value={value(p.key)}
										onInput={(e) =>
											props.board.setFilter(p.key, e.currentTarget.value)
										}
									/>
								</div>
							}
						>
							{(options) => (
								<OptionSelect
									param={p}
									options={options()}
									value={value(p.key)}
									onChange={(v) => props.board.setFilter(p.key, v)}
								/>
							)}
						</Show>
					)}
				</For>
				<For each={unknownKeys()}>
					{(k) => (
						<div class="flex flex-col gap-1.5">
							<FieldLabel for={`draft-${k}`}>
								<span class="font-mono">{k}</span>{" "}
								<span class="font-normal text-faint">kept from URL</span>
							</FieldLabel>
							<Input
								id={`draft-${k}`}
								value={value(k)}
								onInput={(e) => props.board.setFilter(k, e.currentTarget.value)}
							/>
						</div>
					)}
				</For>
			</div>
			<Show when={hiddenCount() > 0 || showAll()}>
				<button
					type="button"
					onClick={() => setShowAll(!showAll())}
					class="self-start rounded text-xs font-medium text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
				>
					{showAll() ? "Fewer filters" : `More filters (${hiddenCount()})`}
				</button>
			</Show>
		</div>
	);
}

export function SourcePicker(props: { board: BoardDraft }) {
	return (
		<div class="flex flex-wrap gap-1.5">
			<For each={BOARD_SPECS}>
				{(s) => (
					<button
						type="button"
						aria-pressed={props.board.draft().source === s.source}
						onClick={() => props.board.setSource(s.source)}
						class={cn(
							"rounded-full border px-3 py-1 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
							props.board.draft().source === s.source
								? "border-accent-border bg-accent-subtle text-accent-text"
								: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
						)}
					>
						{s.label}
					</button>
				)}
			</For>
		</div>
	);
}

export function ParseNotes(props: { board: BoardDraft }) {
	return (
		<Show when={props.board.parsed()}>
			{(r) => (
				<>
					<Show when={r().dropped.length > 0}>
						<p class="text-xs text-faint">
							Ignored {r().dropped.length} tracking parameter
							{r().dropped.length === 1 ? "" : "s"}:{" "}
							<span class="font-mono">
								{r()
									.dropped.map((d) => d.key)
									.join(", ")}
							</span>
						</p>
					</Show>
					<Show when={r().pastedPage}>
						<p class="text-xs text-accent-text">
							This was page {r().pastedPage}. Runs always start from page 1.
						</p>
					</Show>
				</>
			)}
		</Show>
	);
}
