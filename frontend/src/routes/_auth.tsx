import { Outlet, createFileRoute, redirect } from "@tanstack/solid-router";
import { createSignal } from "solid-js";
import Sidebar from "../components/Sidebar";
import Topbar from "../components/Topbar";
import TweaksPanel from "../components/TweaksPanel";
import { getMe } from "../api/auth";

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
				<main class="scroll-slim flex-1 overflow-y-auto">
					<Outlet />
				</main>
			</div>
			<TweaksPanel />
		</div>
	);
}
