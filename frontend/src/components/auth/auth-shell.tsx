import { createSignal, type JSX, onCleanup, Show } from "solid-js";
import { RoleCloud } from "@/components/auth/role-cloud";
import { FastTrackMark } from "@/components/brand-mark";
import { oklchLightness } from "@/lib/color";

const Wordmark = (props: { class?: string; tone?: "dark" | "light" }) => (
	<span
		class={`inline-flex items-center gap-2.5 font-bold tracking-[-0.02em] ${props.class ?? ""}`}
	>
		<span
			class={
				props.tone === "light"
					? "grid h-9 w-9 place-items-center rounded-[10px] bg-primary/10 text-primary"
					: "grid h-[34px] w-[34px] place-items-center rounded-[10px] border border-overlay-20 bg-overlay-10 text-[var(--bright)]"
			}
		>
			<FastTrackMark size={19} />
		</span>
		<span>
			Fast
			<span
				class={props.tone === "light" ? "text-primary" : "text-[var(--bright)]"}
			>
				Track
			</span>
		</span>
	</span>
);

function createDesktop() {
	const query = window.matchMedia("(min-width: 1024px)");
	const [desktop, setDesktop] = createSignal(query.matches);
	const onChange = (e: MediaQueryListEvent) => setDesktop(e.matches);
	query.addEventListener("change", onChange);
	onCleanup(() => query.removeEventListener("change", onChange));
	return desktop;
}

export function AuthShell(props: { children: JSX.Element }) {
	const desktop = createDesktop();
	const sidebarL = oklchLightness(
		getComputedStyle(document.documentElement)
			.getPropertyValue("--color-sidebar")
			.trim(),
	);
	const authTone = sidebarL !== null && sidebarL > 0.5 ? "light" : "dark";

	return (
		<div class="flex min-h-screen flex-col lg:grid lg:grid-cols-[1.05fr_0.95fr]">
			<div
				class="auth-brand relative flex h-[254px] shrink-0 flex-col px-5 pt-[max(1.5rem,env(safe-area-inset-top))] lg:hidden"
				data-auth-tone={authTone}
			>
				<div class="auth-aurora">
					<span class="auth-ribbon auth-ribbon-1" />
					<span class="auth-ribbon auth-ribbon-2" />
					<span class="auth-ribbon auth-ribbon-3" />
				</div>
				<div class="auth-grain" />

				<div
					class="absolute inset-x-0 bottom-0 top-[72px] z-[1]"
					aria-hidden="true"
				>
					<Show when={!desktop()}>
						<RoleCloud variant="compact" />
					</Show>
				</div>

				<Wordmark class="relative z-[2]" />
			</div>

			<section
				class="auth-brand hidden flex-col px-14 py-12 lg:flex"
				data-auth-tone={authTone}
			>
				<div class="auth-aurora">
					<span class="auth-ribbon auth-ribbon-1" />
					<span class="auth-ribbon auth-ribbon-2" />
					<span class="auth-ribbon auth-ribbon-3" />
				</div>
				<div class="auth-grain" />

				<Wordmark class="relative z-[2]" />

				<div class="relative z-[2] flex flex-1 flex-col justify-center gap-[clamp(2.5rem,6vh,4rem)]">
					<div>
						<h2 class="max-w-[16ch] text-balance text-display font-bold tracking-[-0.025em]">
							Your whole job search, on one track.
						</h2>
						<p
							class="mt-[1.3rem] max-w-[38ch] text-pretty text-display-prose"
							style={{ color: "var(--auth-prose)" }}
						>
							New roles arrive on their own and stay current. You just decide
							what to chase.
						</p>
					</div>

					<div class="auth-field" aria-hidden="true">
						<Show when={desktop()}>
							<RoleCloud />
						</Show>
					</div>
				</div>
			</section>

			<section class="relative z-[1] flex flex-1 flex-col bg-surface lg:items-center lg:justify-center lg:p-8">
				<div class="auth-rise -mt-5 w-full flex-1 rounded-t-[22px] bg-surface px-[22px] pt-7 pb-[max(2.75rem,env(safe-area-inset-bottom))] lg:mt-0 lg:max-w-[380px] lg:flex-none lg:rounded-none lg:bg-transparent lg:px-0 lg:pt-0 lg:pb-0">
					{props.children}
				</div>
			</section>
		</div>
	);
}
