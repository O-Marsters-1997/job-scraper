import { createFileRoute, Outlet, redirect } from "@tanstack/solid-router";
import { createSignal } from "solid-js";
import { getMe } from "../api/auth";
import Sidebar from "../components/Sidebar";
import Topbar from "../components/Topbar";
import TweaksPanel from "../components/TweaksPanel";

export const Route = createFileRoute("/_auth")({
	beforeLoad: async () => {
		const res = await getMe();
		if (!res.ok) {
			throw redirect({ to: "/login" });
		}
	},
	component: AuthLayout,
});

function AuthLayout() {
	const [mobileNavOpen, setMobileNavOpen] = createSignal(false);
	return (
		<div class="flex h-screen overflow-hidden">
			<Sidebar
				mobileOpen={mobileNavOpen()}
				onMobileClose={() => setMobileNavOpen(false)}
			/>
			<div class="flex min-w-0 flex-1 flex-col">
				<Topbar onMenuClick={() => setMobileNavOpen(true)} />
				<main class="scroll-slim relative flex-1 overflow-y-auto">
					{/* Faint violet aurora wash behind the topbar/header zone only —
					    pointer-events-none, never behind table rows */}
					<div class="brand-aurora-shell" aria-hidden="true">
						<span class="brand-ribbon brand-ribbon-top" />
					</div>
					<div class="relative z-10">
						<Outlet />
					</div>
				</main>
			</div>
			<TweaksPanel />
		</div>
	);
}
