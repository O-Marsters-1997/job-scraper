import {
	createSignal,
	Match,
	onCleanup,
	onMount,
	Show,
	Switch,
} from "solid-js";
import { Icon } from "@/components/Icon";
import { DEFAULTS, loadTweaks, saveTweaks, type Tweaks } from "@/lib/tweaks";
import { applyTweaks } from "@/lib/tweaks.apply";
import { cn } from "@/lib/utils";
import { CustomThemeView, TweaksMainView } from "./tweaks/views";

export default function TweaksPanel() {
	let panelRef: HTMLDivElement | undefined;

	const [open, setOpen] = createSignal(false);
	const [view, setView] = createSignal<"main" | "custom">("main");
	const [tweaks, setTweaks] = createSignal<Tweaks>(DEFAULTS);

	onMount(() => {
		const loaded = loadTweaks();
		setTweaks(loaded);
		applyTweaks(loaded);

		function handlePointerDown(e: PointerEvent) {
			if (!open()) return;
			const target = e.target as Element | null;
			if (!target) return;
			// Kobalte Popover portals render in <body>; let clicks inside them pass
			if (target.closest("[data-tweaks-portal]")) return;
			if (panelRef && !panelRef.contains(target)) close();
		}

		function handleKeyDown(e: KeyboardEvent) {
			if (e.key !== "Escape" || !open()) return;
			const target = e.target as Element | null;
			// Let Kobalte's own popover (e.g. the colour picker) handle its own Escape
			if (target?.closest("[data-tweaks-portal]")) return;
			close();
		}

		document.addEventListener("pointerdown", handlePointerDown);
		document.addEventListener("keydown", handleKeyDown);
		onCleanup(() => {
			document.removeEventListener("pointerdown", handlePointerDown);
			document.removeEventListener("keydown", handleKeyDown);
		});
	});

	function close() {
		setView("main");
		setOpen(false);
	}

	function commit(next: Tweaks) {
		setTweaks(next);
		saveTweaks(next);
		applyTweaks(next);
	}

	return (
		<div
			ref={panelRef}
			class="fixed bottom-4 right-4 z-50 flex flex-col items-end gap-2"
		>
			<Show when={open()}>
				<div
					class="flex w-72 flex-col overflow-hidden rounded-xl border border-border bg-surface shadow-xl"
					style={{ "max-height": "86vh" }}
				>
					<Switch>
						<Match when={view() === "main"}>
							<TweaksMainView
								tweaks={tweaks}
								onChange={commit}
								onOpenCustom={() => setView("custom")}
								onClose={close}
							/>
						</Match>
						<Match when={view() === "custom"}>
							<CustomThemeView
								tweaks={tweaks}
								onSave={(next) => {
									commit(next);
									setView("main");
								}}
								onCancel={() => setView("main")}
								onClose={close}
							/>
						</Match>
					</Switch>
				</div>
			</Show>

			<button
				type="button"
				onClick={() => (open() ? close() : setOpen(true))}
				aria-expanded={open()}
				class={cn(
					"flex size-9 items-center justify-center rounded-full border shadow-lg transition-colors",
					open()
						? "border-primary bg-primary text-primary-foreground"
						: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
				)}
				title="Tweaks"
				aria-label="Open tweaks panel"
			>
				<Icon name="sliders" />
			</button>
		</div>
	);
}
