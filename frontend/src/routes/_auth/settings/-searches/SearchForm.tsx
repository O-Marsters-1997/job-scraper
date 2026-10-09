import { createSignal, For, Show } from "solid-js";
import { ToggleChip } from "@/components/ToggleChip";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import {
	DEFAULT_RUN_WINDOW,
	draftError,
	draftFrom,
	effectiveDraft,
	toRunWindow,
} from "@/lib/runWindow";
import { runWindowErrorMessage } from "@/lib/runWindowError";
import { ConflictError } from "../../../../api/sourceTargets";
import { useCreateSourceTarget } from "../../../../hooks/useSourceTargets";
import type { SourceFilterField, SourceInfo } from "../../../../types/source";
import type { SourceTarget } from "../../../../types/sourceTarget";
import { RunWindowFields } from "./RunWindowFields";

type Option = { value: string; label: string };
const ANY = "__any__";

const defaultFilters = (source: string | undefined): Record<string, string> =>
	source === "linkedin" ? { recency: "r604800" } : {};

function FilterSelect(props: {
	field: SourceFilterField;
	value: string;
	onChange: (value: string) => void;
}) {
	const options = (): Option[] => [
		{ value: ANY, label: "Any" },
		...(props.field.options ?? []),
	];
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
				{props.field.label}
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

export interface SearchPrefill {
	source: string;
	value: string;
	filters: Record<string, string>;
	dropped: string[];
}

export function SearchForm(props: {
	sources: SourceInfo[];
	initial?: SearchPrefill;
	onCreated: (target: SourceTarget, source: SourceInfo | undefined) => void;
	onCancel: () => void;
}) {
	const createMutation = useCreateSourceTarget();
	const [picked, setSourceName] = createSignal<string | undefined>(
		props.initial?.source,
	);
	const sourceName = () => picked() ?? props.sources[0]?.name;
	const [value, setValue] = createSignal(props.initial?.value ?? "");
	const [filters, setFilters] = createSignal<Record<string, string>>(
		props.initial?.filters ?? defaultFilters(sourceName()),
	);

	const [schedule, setSchedule] = createSignal(
		draftFrom(DEFAULT_RUN_WINDOW, true),
	);
	const scheduleError = () =>
		draftError(effectiveDraft(schedule(), source()?.incremental ?? false));

	const source = () => props.sources.find((s) => s.name === sourceName());

	const pickSource = (name: string) => {
		setSourceName(name);
		setValue("");
		setFilters(defaultFilters(name));
	};

	const form = useFormSubmit(
		async () => {
			const info = source();
			const trimmed = value().trim();
			if (!info || !trimmed) return;
			if (info.kind === "url" && !trimmed.startsWith(info.url_prefix)) {
				throw new PrefixError(info.url_prefix);
			}
			const payload =
				info.kind === "filter"
					? Object.fromEntries(Object.entries(filters()).filter(([, v]) => v))
					: {};
			const created = await createMutation.mutateAsync({
				source: info.name,
				value: trimmed,
				filters: payload,
				run_window: toRunWindow({
					...schedule(),
					automatic: schedule().automatic && info.incremental,
				}),
			});
			props.onCreated(created, info);
		},
		(err) => {
			if (err instanceof PrefixError)
				return `URL must start with ${err.prefix}`;
			if (err instanceof ConflictError)
				return "A search with these settings already exists.";
			const capped = runWindowErrorMessage(err);
			if (capped) return capped;
			return "Failed to add search. Please try again.";
		},
	);

	return (
		<form
			onSubmit={form.submit}
			class="mb-4 flex flex-col gap-4 rounded-lg border border-border bg-surface-muted px-4 py-4"
		>
			<Show when={props.initial?.dropped.length}>
				<p class="text-xs text-muted">
					Won't be used:{" "}
					<span class="font-mono">{props.initial?.dropped.join(", ")}</span>
				</p>
			</Show>

			<fieldset
				class="flex flex-wrap items-center gap-2"
				disabled={Boolean(props.initial)}
			>
				<legend class="sr-only">Job board</legend>
				<span class="text-xs font-medium text-foreground">Job board</span>
				<For
					each={
						props.initial
							? props.sources.filter((s) => s.name === sourceName())
							: props.sources
					}
				>
					{(s) => (
						<ToggleChip
							active={sourceName() === s.name}
							onClick={() => pickSource(s.name)}
						>
							{s.label}
						</ToggleChip>
					)}
				</For>
			</fieldset>

			<Show
				when={source()?.kind === "url"}
				fallback={
					<div class="flex flex-col gap-1.5">
						<label
							for="search-keywords"
							class="text-xs font-medium text-foreground"
						>
							Keywords
						</label>
						<Input
							id="search-keywords"
							placeholder="e.g. product engineer"
							value={value()}
							onInput={(e) => setValue(e.currentTarget.value)}
						/>
					</div>
				}
			>
				<div class="flex flex-col gap-1.5">
					<label for="search-url" class="text-xs font-medium text-foreground">
						Search URL
					</label>
					<Input
						id="search-url"
						type="url"
						placeholder={source()?.url_prefix}
						value={value()}
						onInput={(e) => setValue(e.currentTarget.value)}
					/>
					<p class="text-xs text-faint">
						Must start with{" "}
						<span class="font-mono">{source()?.url_prefix}</span>
					</p>
				</div>
			</Show>

			<Show when={source()?.kind === "filter"}>
				<div class="grid gap-3 sm:grid-cols-2">
					<For each={source()?.filters ?? []}>
						{(field) => (
							<Show
								when={field.options?.length}
								fallback={
									<div class="flex flex-col gap-1.5">
										<label
											for={`filter-${field.name}`}
											class="text-xs font-medium text-foreground"
										>
											{field.label}
										</label>
										<Input
											id={`filter-${field.name}`}
											placeholder="Any"
											value={filters()[field.name] ?? ""}
											onInput={(e) =>
												setFilters((prev) => ({
													...prev,
													[field.name]: e.currentTarget.value,
												}))
											}
										/>
									</div>
								}
							>
								<FilterSelect
									field={field}
									value={filters()[field.name] ?? ""}
									onChange={(v) =>
										setFilters((prev) => ({ ...prev, [field.name]: v }))
									}
								/>
							</Show>
						)}
					</For>
				</div>
			</Show>

			<RunWindowFields
				idPrefix="new-search"
				value={schedule()}
				onChange={setSchedule}
				incremental={source()?.incremental ?? false}
				error={null}
			/>

			<Show when={scheduleError() ?? form.error()}>
				<p role="alert" class="text-xs text-destructive-strong">
					{scheduleError() ?? form.error()}
				</p>
			</Show>

			<div class="flex items-center justify-end gap-2 border-t border-border pt-3">
				<Button
					type="button"
					variant="ghost"
					size="sm"
					onClick={props.onCancel}
				>
					Cancel
				</Button>
				<Button
					type="submit"
					size="sm"
					disabled={form.pending() || !value().trim() || !!scheduleError()}
				>
					{form.pending() ? "Adding…" : "Add search"}
				</Button>
			</div>
		</form>
	);
}

class PrefixError extends Error {
	constructor(readonly prefix: string) {
		super("url prefix mismatch");
	}
}
