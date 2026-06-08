import { Outlet, createFileRoute, redirect } from "@tanstack/solid-router";
import Sidebar from "../components/Sidebar";
import Topbar from "../components/Topbar";
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
	return (
		<div class="flex h-screen overflow-hidden">
			<Sidebar />
			<div class="flex min-w-0 flex-1 flex-col">
				<Topbar />
				<main class="scroll-slim flex-1 overflow-y-auto">
					<Outlet />
				</main>
			</div>
		</div>
	);
}
