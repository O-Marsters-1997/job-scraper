import { createFileRoute, Outlet, redirect } from "@tanstack/solid-router";
import { createSignal } from "solid-js";
import { PushBanner } from "../components/PushBanner";
import Sidebar from "../components/Sidebar";
import Topbar from "../components/Topbar";
import TweaksPanel from "../components/TweaksPanel";
import { meQueryOptions } from "../hooks/useAuth";
import { queryClient } from "../lib/queryClient";

export const Route = createFileRoute("/_auth")({
	beforeLoad: async () => {
		try {
			await queryClient.ensureQueryData(meQueryOptions);
		} catch {
			throw redirect({ to: "/login" });
		}
	},
	component: AuthLayout,
});

function AuthLayout() {
	const [mobileNavOpen, setMobileNavOpen] = createSignal(false);
	let menuButtonEl: HTMLButtonElement | undefined;
	return (
		<div class="flex h-screen overflow-hidden">
			<Sidebar
				mobileOpen={mobileNavOpen()}
				onMobileClose={() => setMobileNavOpen(false)}
				restoreFocusTo={() => menuButtonEl}
			/>
			<div class="flex min-w-0 flex-1 flex-col">
				<Topbar
					onMenuClick={() => setMobileNavOpen(true)}
					menuButtonRef={(el) => {
						menuButtonEl = el;
					}}
				/>
				<main class="scroll-slim relative flex-1 overflow-y-auto">
					<div class="brand-aurora-shell" aria-hidden="true">
						<span class="brand-ribbon brand-ribbon-top" />
					</div>
					<div class="relative z-10">
						<PushBanner />
						<Outlet />
					</div>
				</main>
			</div>
			<TweaksPanel />
		</div>
	);
}
