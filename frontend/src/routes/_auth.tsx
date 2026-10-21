import { createFileRoute, Outlet, redirect } from "@tanstack/solid-router";
import { createSignal, Suspense } from "solid-js";
import { PageSkeleton } from "../components/PageSkeleton";
import { PushBanner } from "../components/PushBanner";
import Sidebar from "../components/Sidebar";
import Topbar from "../components/Topbar";
import TweaksPanel from "../components/TweaksPanel";
import { FullPageSpinner } from "../components/ui/spinner";
import { meQueryOptions } from "../hooks/useAuth";
import {
	createUsageWarning,
	UsageWarningContext,
} from "../hooks/useUsageWarning";
import { queryClient } from "../lib/queryClient";

export const Route = createFileRoute("/_auth")({
	beforeLoad: async () => {
		try {
			await queryClient.ensureQueryData(meQueryOptions);
		} catch {
			throw redirect({ to: "/login" });
		}
	},
	pendingComponent: FullPageSpinner,
	component: AuthLayout,
});

function AuthLayout() {
	const usageWarning = createUsageWarning();
	const [mobileNavOpen, setMobileNavOpen] = createSignal(false);
	let menuButtonEl: HTMLButtonElement | undefined;
	return (
		<UsageWarningContext.Provider value={usageWarning}>
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
							<Suspense fallback={<PageSkeleton />}>
								<Outlet />
							</Suspense>
						</div>
					</main>
				</div>
				<TweaksPanel />
			</div>
		</UsageWarningContext.Provider>
	);
}
