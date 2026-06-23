import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createSignal, onMount } from "solid-js";
import { AuthShell } from "@/components/auth/auth-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { login } from "../api/auth";

export const Route = createFileRoute("/login")({
	component: LoginPage,
});

function LoginPage() {
	const navigate = useNavigate();
	const [username, setUsername] = createSignal("");
	const [password, setPassword] = createSignal("");
	const [error, setError] = createSignal("");
	const [loading, setLoading] = createSignal(false);
	let formRef: HTMLFormElement | undefined;

	onMount(() => formRef?.querySelector("input")?.focus());

	const handleSubmit = async (e: SubmitEvent) => {
		e.preventDefault();
		setError("");
		setLoading(true);
		try {
			const res = await login(username(), password());
			if (res.ok) {
				navigate({ to: "/jobs", search: { page: 1 } });
			} else {
				setError("Invalid username or password.");
			}
		} catch {
			setError("Could not reach the server. Please try again.");
		} finally {
			setLoading(false);
		}
	};

	return (
		<AuthShell>
			<h1 class="text-[1.85rem] font-bold tracking-[-0.025em] text-foreground">
				Back on track
			</h1>
			<p class="mt-1 text-[0.95rem] text-muted">
				Sign in to pick up where you left off.
			</p>

			<form
				ref={formRef}
				id="login-form"
				onSubmit={handleSubmit}
				class="mt-6 flex flex-col gap-4"
			>
				<div class="flex flex-col gap-1.5">
					<Label for="username" variant="uppercase">
						Username
					</Label>
					<Input
						id="username"
						type="text"
						autocomplete="username"
						required
						value={username()}
						onInput={(e) => setUsername(e.currentTarget.value)}
						placeholder="alice"
						class="h-11 rounded-xl"
					/>
				</div>

				<div class="flex flex-col gap-1.5">
					<Label for="password" variant="uppercase">
						Password
					</Label>
					<Input
						id="password"
						type="password"
						autocomplete="current-password"
						required
						value={password()}
						onInput={(e) => setPassword(e.currentTarget.value)}
						placeholder="••••••••"
						class="h-11 rounded-xl"
					/>
				</div>

				{error() && (
					<p
						role="alert"
						class="rounded-lg border border-destructive/30 bg-destructive-subtle px-3 py-2 text-sm text-destructive-strong"
					>
						{error()}
					</p>
				)}

				<Button
					type="submit"
					disabled={loading()}
					aria-busy={loading()}
					class="mt-2 h-12 w-full rounded-xl text-[0.97rem]"
				>
					{loading() ? (
						"Signing in…"
					) : (
						<>
							Sign in
							<svg
								aria-hidden="true"
								width="17"
								height="17"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2.2"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<path d="M5 12h14M13 6l6 6-6 6" />
							</svg>
						</>
					)}
				</Button>
			</form>

			<p class="mt-6 text-center text-sm text-faint">
				Don't have an account?{" "}
				<a href="/signup" class="font-semibold text-primary hover:underline">
					Sign up
				</a>
			</p>
		</AuthShell>
	);
}
