import type { JSX } from "solid-js";
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
					: "grid h-[34px] w-[34px] place-items-center rounded-[10px] border border-white/20 bg-white/10 text-[var(--bright)]"
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

export function AuthShell(props: { children: JSX.Element }) {
	// Compute the panel tone synchronously so data-auth-tone is set as part of
	// the initial DOM creation — before RoleCloud's onMount reads --auth-* vars.
	// applyAll(loadTweaks()) has already run in main.tsx, so --color-sidebar is
	// current on :root before any component renders.
	const sidebarL = oklchLightness(
		getComputedStyle(document.documentElement)
			.getPropertyValue("--color-sidebar")
			.trim(),
	);
	const authTone = sidebarL !== null && sidebarL > 0.5 ? "light" : "dark";

	return (
		<div class="grid min-h-screen lg:grid-cols-[1.05fr_0.95fr]">
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
						<RoleCloud />
					</div>
				</div>
			</section>

			<section class="flex items-center justify-center bg-surface p-6 sm:p-8">
				<div class="auth-rise w-full max-w-[380px]">
					<Wordmark tone="light" class="mb-8 flex lg:hidden" />
					{props.children}
				</div>
			</section>
		</div>
	);
}
