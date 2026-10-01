import { Combobox, useComboboxContext } from "@kobalte/core/combobox";
import { createMemo, createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { cn } from "@/lib/utils";

function SearchInput() {
	const ctx = useComboboxContext();
	const highlightBestMatch = () =>
		queueMicrotask(() => {
			const q = ctx.inputValue()?.trim().toLowerCase();
			if (!q) return;
			const list = ctx.listState();
			const nodes = [...list.collection()];
			const rank = (node: (typeof nodes)[number]) => {
				const text = node.textValue.toLowerCase();
				if (list.selectionManager().isSelected(node.key)) return 4;
				if ((node.rawValue as ComboOption).created) return 2;
				return text === q ? 0 : text.startsWith(q) ? 1 : 3;
			};
			const best = nodes.sort((a, b) => rank(a) - rank(b))[0];
			if (best) list.selectionManager().setFocusedKey(best.key);
		});
	return (
		<Combobox.Input
			onInput={highlightBestMatch}
			class="h-7 min-w-24 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-faint"
		/>
	);
}

export type ComboOption = { id: string; label: string; created?: boolean };

const MAX_RENDERED = 200;

export function MultiCombobox(props: {
	label: string;
	options: ComboOption[];
	value: string[];
	onChange: (ids: string[]) => void;
	hint?: string;
	placeholder?: string;
	chipClass?: string;
	creatable?: boolean;
	onSearch?: (query: string) => void;
}) {
	const [query, setQuery] = createSignal("");
	const known = createMemo(
		() => new Map(props.options.map((o) => [o.id, o] as const)),
	);
	const selected = createMemo(() =>
		props.value.map((id) => known().get(id) ?? { id, label: id }),
	);
	const shown = createMemo(() => {
		const q = query().trim();
		const lower = q.toLowerCase();
		const picked = new Set(props.value);
		const exists = [...selected(), ...props.options].some(
			(o) => o.label.toLowerCase() === lower,
		);
		const created =
			props.creatable && q && !exists
				? [{ id: q, label: q, created: true }]
				: [];
		const matches = props.options
			.filter(
				(o) =>
					!picked.has(o.id) &&
					(props.onSearch || o.label.toLowerCase().includes(lower)),
			)
			.slice(0, MAX_RENDERED);
		return [...created, ...selected(), ...matches];
	});
	return (
		<Combobox<ComboOption>
			multiple
			options={shown()}
			value={selected()}
			onInputChange={(value) => {
				setQuery(value);
				props.onSearch?.(value);
			}}
			onChange={(opts) => props.onChange(opts.map((o) => o.id))}
			optionValue="id"
			optionTextValue="label"
			optionLabel="label"
			triggerMode="focus"
			closeOnSelection={false}
			defaultFilter={props.onSearch ? () => true : "contains"}
			placeholder={props.placeholder ?? "Search…"}
			gutter={4}
			sameWidth
			itemComponent={(item) => (
				<Combobox.Item
					item={item.item}
					class="flex cursor-pointer items-center justify-between gap-3 rounded-md px-2.5 py-1.5 text-sm text-foreground outline-none select-none data-[highlighted]:bg-accent-subtle"
				>
					<Combobox.ItemLabel>
						<Show
							when={item.item.rawValue.created}
							fallback={item.item.rawValue.label}
						>
							Add “{item.item.rawValue.label}”
						</Show>
					</Combobox.ItemLabel>
					<Combobox.ItemIndicator class="text-accent-text">
						<Icon name="check" size={14} strokeWidth={2.5} />
					</Combobox.ItemIndicator>
				</Combobox.Item>
			)}
		>
			<Combobox.Label class="block text-sm font-medium text-foreground">
				{props.label}
			</Combobox.Label>
			<Show when={props.hint}>
				<Combobox.Description class="mt-0.5 text-xs text-faint">
					{props.hint}
				</Combobox.Description>
			</Show>
			<Combobox.Control<ComboOption> class="field mt-2 flex min-h-9 flex-wrap items-center gap-1.5 py-1 pr-1 pl-2.5 focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/10">
				{(state) => (
					<>
						<Icon name="search" size={14} class="shrink-0 text-faint" />
						<For each={state.selectedOptions()}>
							{(o) => (
								<span
									class={cn(
										"inline-flex h-6 items-center gap-0.5 rounded-full border pr-0.5 pl-2.5 text-xs font-medium",
										props.chipClass,
									)}
								>
									{o.label}
									<button
										type="button"
										aria-label={`Remove ${o.label}`}
										onPointerDown={(e) => e.preventDefault()}
										onClick={() => state.remove(o)}
										class="grid size-5 place-items-center rounded-full opacity-60 transition-opacity hover:opacity-100 focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
									>
										<Icon name="x" size={10} strokeWidth={2.5} />
									</button>
								</span>
							)}
						</For>
						<SearchInput />
						<Combobox.Trigger
							aria-label="Show all options"
							class="grid size-7 shrink-0 place-items-center rounded-md text-faint transition-colors hover:bg-surface-muted hover:text-foreground"
						>
							<Combobox.Icon>
								<Icon name="chevronDown" size={12} strokeWidth={2.5} />
							</Combobox.Icon>
						</Combobox.Trigger>
					</>
				)}
			</Combobox.Control>
			<Combobox.Portal>
				<Combobox.Content
					onCloseAutoFocus={(e) => e.preventDefault()}
					class="z-50 overflow-hidden rounded-lg border border-border bg-surface shadow-xl data-[expanded]:animate-in data-[expanded]:fade-in-0 data-[closed]:animate-out data-[closed]:fade-out-0"
				>
					<Combobox.Listbox class="scroll-slim max-h-64 overflow-y-auto p-1" />
				</Combobox.Content>
			</Combobox.Portal>
		</Combobox>
	);
}
