import { Link, Outlet, useLocation } from "@tanstack/solid-router";
import {
	createContext,
	createSignal,
	For,
	type JSX,
	Show,
	useContext,
} from "solid-js";
import { Portal } from "solid-js/web";
import { useSettingsGroups } from "@/hooks/useSettingsGroups";
import { useShortcuts } from "@/hooks/useShortcuts";
import {
	findSettingsSection,
	type SettingsSection,
} from "@/lib/settingsSections";

const ActionsSlot = createContext<() => HTMLElement | undefined>(
	() => undefined,
);

export function SettingsActions(props: { children: JSX.Element }) {
	const slot = useContext(ActionsSlot);
	return (
		<Show when={slot()}>
			{(el) => <Portal mount={el()}>{props.children}</Portal>}
		</Show>
	);
}

const NAV_ITEM =
	"block shrink-0 rounded-lg border border-transparent px-3 transition-colors hover:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary";

export function SettingsLayout() {
	const location = useLocation();
	const groups = useSettingsGroups();
	const [slot, setSlot] = createSignal<HTMLElement>();
	const links: Record<string, HTMLAnchorElement> = {};

	const isOpen = (section: SettingsSection) =>
		location().pathname.startsWith(section.to);

	const cycle = (step: number) => {
		const order = groups()
			.flatMap((group) => group.sections)
			.flatMap((section) => [
				section.to,
				...(isOpen(section)
					? (section.children?.map((child) => child.to) ?? [])
					: []),
			]);
		const focused = order.findIndex(
			(to) => links[to] === document.activeElement,
		);
		const i = focused >= 0 ? focused : order.indexOf(location().pathname);
		const next = order[(i + step + order.length) % order.length];
		if (next) links[next]?.focus();
	};

	useShortcuts({ "]": () => cycle(1), "[": () => cycle(-1) });

	return (
		<ActionsSlot.Provider value={slot}>
			<header class="sticky top-0 z-20 flex min-h-14 items-center justify-between gap-4 border-b border-border bg-background px-7 py-2.5">
				<h1 class="truncate text-lg font-bold tracking-tight text-foreground">
					{findSettingsSection(location().pathname)?.label ?? "Settings"}
				</h1>
				<div
					ref={setSlot}
					class="flex shrink-0 items-center gap-2 *:flex *:items-center *:gap-2"
				/>
			</header>
			<div class="flex flex-col gap-6 px-7 py-6 md:flex-row md:items-start">
				<nav
					aria-label="Settings sections"
					class="scroll-slim flex gap-1 overflow-x-auto md:sticky md:top-20 md:w-60 md:shrink-0 md:flex-col md:gap-5 md:overflow-visible"
				>
					<For each={groups()}>
						{(group) => (
							<div class="contents md:flex md:flex-col md:gap-1">
								<p class="hidden px-3 text-2xs font-semibold uppercase tracking-wider text-faint md:block">
									{group.heading}
								</p>
								<For each={group.sections}>
									{(section) => (
										<>
											<Link
												to={section.to}
												activeOptions={{ exact: !section.children }}
												ref={(el: HTMLAnchorElement) => {
													links[section.to] = el;
												}}
												class={`${NAV_ITEM} min-w-40 py-2.5 data-[status=active]:border-border data-[status=active]:bg-surface md:min-w-0`}
											>
												<span class="block text-sm font-medium text-foreground">
													{section.label}
												</span>
												<span class="mt-0.5 block truncate text-xs text-faint">
													{section.description}
												</span>
											</Link>
											<Show when={isOpen(section) && section.children}>
												{(children) => (
													<div class="contents md:ml-3 md:flex md:flex-col md:gap-0.5 md:border-l md:border-border md:pl-2">
														<For each={children()}>
															{(child) => (
																<Link
																	to={child.to}
																	activeOptions={{ exact: true }}
																	ref={(el: HTMLAnchorElement) => {
																		links[child.to] = el;
																	}}
																	class={`${NAV_ITEM} py-1.5 text-sm text-muted data-[status=active]:border-border data-[status=active]:bg-surface data-[status=active]:font-medium data-[status=active]:text-foreground`}
																>
																	{child.label}
																</Link>
															)}
														</For>
													</div>
												)}
											</Show>
										</>
									)}
								</For>
							</div>
						)}
					</For>
					<p class="hidden px-3 text-xs text-faint md:block">
						Press <kbd class="font-mono">[</kbd> or{" "}
						<kbd class="font-mono">]</kbd> to move,{" "}
						<kbd class="font-mono">Enter</kbd> to open
					</p>
				</nav>

				<div class="flex min-w-0 flex-1 flex-col gap-6 self-stretch rounded-xl border border-border bg-surface p-6 md:min-h-[calc(100dvh-10rem-1px)]">
					<Outlet />
				</div>
			</div>
		</ActionsSlot.Provider>
	);
}
