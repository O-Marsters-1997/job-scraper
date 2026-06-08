import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createSignal } from "solid-js";
import { signup } from "../api/auth";
import { Card, CardContent, CardFooter, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";

export const Route = createFileRoute("/signup")({
	component: SignupPage,
});

function SignupPage() {
	const navigate = useNavigate();
	const [username, setUsername] = createSignal("");
	const [password, setPassword] = createSignal("");
	const [error, setError] = createSignal("");
	const [loading, setLoading] = createSignal(false);

	const handleSubmit = async (e: SubmitEvent) => {
		e.preventDefault();
		setError("");
		setLoading(true);
		try {
			const res = await signup(username(), password());
			if (res.ok) {
				navigate({ to: "/jobs", search: { page: 1 } });
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
		<div class="flex min-h-screen items-center justify-center bg-background p-4">
			<Card class="w-full max-w-sm shadow-xl">
				<CardHeader>
					<div class="mb-4 flex flex-col items-center gap-3">
						<div class="flex h-12 w-12 items-center justify-center rounded-xl bg-accent-subtle text-primary">
							<svg
								aria-hidden="true"
								width="22"
								height="22"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
							>
								<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
								<circle cx="9" cy="7" r="4" />
								<line x1="19" y1="8" x2="19" y2="14" />
								<line x1="22" y1="11" x2="16" y2="11" />
							</svg>
						</div>
						<div class="text-center">
							<p class="mb-1 text-xs font-semibold tracking-wider text-faint uppercase">
								Get started
							</p>
							<h1 class="text-xl font-bold text-foreground">Create account</h1>
						</div>
					</div>
				</CardHeader>

				<CardContent>
					<form id="signup-form" onSubmit={handleSubmit} class="flex flex-col gap-4">
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
							/>
						</div>

						{error() && (
							<p class="rounded-lg border border-destructive/30 bg-destructive-subtle px-3 py-2 text-sm text-destructive-strong">
								{error()}
							</p>
						)}
					</form>
				</CardContent>

				<CardFooter class="flex-col gap-3 pt-0">
					<Button
						type="submit"
						form="signup-form"
						disabled={loading()}
						class="mt-2 w-full"
					>
						{loading() ? "Creating account…" : "Create account"}
					</Button>

					<p class="text-center text-xs text-faint">
						Already have an account?{" "}
						<a href="/login" class="font-semibold text-primary hover:underline">
							Sign in
						</a>
					</p>
				</CardFooter>
			</Card>
		</div>
	);
}
