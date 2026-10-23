import { onCleanup, onMount, Show } from "solid-js";
import { Icon } from "@/components/Icon";

export function PrototypeSwitcher(props: {
	variants: { key: string; name: string }[];
	current: string;
	onChange: (key: string) => void;
}) {
	const index = () =>
		Math.max(
			0,
			props.variants.findIndex((v) => v.key === props.current),
		);
	const step = (by: number) => {
		const n = props.variants.length;
		props.onChange(props.variants[(index() + by + n) % n]?.key ?? "A");
	};
	const onKey = (e: KeyboardEvent) => {
		const el = e.target as HTMLElement;
		if (el.closest("input, textarea, [contenteditable]")) return;
		if (e.key === "ArrowLeft") step(-1);
		if (e.key === "ArrowRight") step(1);
	};
	onMount(() => window.addEventListener("keydown", onKey));
	onCleanup(() => window.removeEventListener("keydown", onKey));

	return (
		<Show when={import.meta.env.DEV}>
			<div class="fixed bottom-5 left-1/2 z-50 flex -translate-x-1/2 items-center gap-1 rounded-full bg-sidebar px-2 py-1.5 text-sm text-sidebar-foreground-strong shadow-lg">
				<button
					type="button"
					aria-label="Previous variant"
					class="rounded-full p-1.5 hover:bg-sidebar-hover"
					onClick={() => step(-1)}
				>
					<Icon name="chevronLeft" />
				</button>
				<span class="min-w-56 text-center font-medium">
					{props.variants[index()]?.key} — {props.variants[index()]?.name}
				</span>
				<button
					type="button"
					aria-label="Next variant"
					class="rounded-full p-1.5 hover:bg-sidebar-hover"
					onClick={() => step(1)}
				>
					<Icon name="chevronRight" />
				</button>
			</div>
		</Show>
	);
}
