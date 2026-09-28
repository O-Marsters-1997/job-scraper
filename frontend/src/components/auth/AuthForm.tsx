import { Link, useNavigate } from "@tanstack/solid-router";
import { createSignal, onMount } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { queryClient } from "@/lib/queryClient";
import { auth } from "../../api/auth";
import { AuthShell } from "./auth-shell";

export interface AuthFormMode {
	path: "/auth/login" | "/auth/signup";
	formId: string;
	heading: string;
	subtext: string;
	buttonText: string;
	loadingText: string;
	passwordAutocomplete: "current-password" | "new-password";
	statusMessages: Record<number, string>;
	defaultErrorMessage: string;
	footerText: string;
	footerLinkTo: "/login" | "/signup";
	footerLinkLabel: string;
}

export function AuthForm(props: { mode: AuthFormMode }) {
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
			const res = await auth(props.mode.path, username(), password());
			if (res.ok) {
				queryClient.clear();
				navigate({ to: "/jobs" });
			} else {
				setError(
					props.mode.statusMessages[res.status] ??
						props.mode.defaultErrorMessage,
				);
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
				{props.mode.heading}
			</h1>
			<p class="mt-1 text-auth-subtext text-muted">{props.mode.subtext}</p>

			<form
				ref={formRef}
				id={props.mode.formId}
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
						autocomplete={props.mode.passwordAutocomplete}
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
						props.mode.loadingText
					) : (
						<>
							{props.mode.buttonText}
							<Icon name="arrowRight" size={17} strokeWidth={2.2} />
						</>
					)}
				</Button>
			</form>

			<p class="mt-6 text-center text-sm text-faint">
				{props.mode.footerText}{" "}
				<Link
					to={props.mode.footerLinkTo}
					class="font-semibold text-primary hover:underline"
				>
					{props.mode.footerLinkLabel}
				</Link>
			</p>
		</AuthShell>
	);
}
