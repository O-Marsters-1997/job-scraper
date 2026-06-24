import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createSignal, onMount } from "solid-js";
import { AuthShell } from "@/components/auth/auth-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { signup } from "../api/auth";

export const Route = createFileRoute("/signup")({
	component: SignupPage,
});

function SignupPage() {
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
			const res = await signup(username(), password());
			if (res.ok) {
				navigate({ to: "/jobs" });
			} else if (res.status === 409) {
				setError("That username is already taken.");
			} else {
				setError("Something went wrong. Please try again.");
			}
		} catch {
			setError("Could not reach the server. Please try again.");
		} finally {
			setLoading(false);
		}
	};

	return (
		<AuthShell>
			<h1 class="text-auth-heading font-bold tracking-[-0.025em] text-foreground">
				Create your account
			</h1>
			<p class="mt-1 text-auth-subtext text-muted">
				Start gathering every fresh role in one place.
			</p>

			<form
				ref={formRef}
				id="signup-form"
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
						autocomplete="new-password"
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
					class="mt-2 h-12 w-full rounded-xl text-auth-action"
				>
					{loading() ? (
						"Creating account…"
					) : (
						<>
							Create account
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
				Already have an account?{" "}
				<a href="/login" class="font-semibold text-primary hover:underline">
					Sign in
				</a>
			</p>
		</AuthShell>
	);
}
