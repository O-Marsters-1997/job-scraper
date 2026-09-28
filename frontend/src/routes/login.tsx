import { createFileRoute } from "@tanstack/solid-router";
import { AuthForm } from "@/components/auth/AuthForm";

export const Route = createFileRoute("/login")({
	component: LoginPage,
});

function LoginPage() {
	return (
		<AuthForm
			mode={{
				path: "/auth/login",
				formId: "login-form",
				heading: "Back on track",
				subtext: "Sign in to pick up where you left off.",
				buttonText: "Sign in",
				loadingText: "Signing in…",
				passwordAutocomplete: "current-password",
				statusMessages: {},
				defaultErrorMessage: "Invalid username or password.",
				footerText: "Don't have an account?",
				footerLinkTo: "/signup",
				footerLinkLabel: "Sign up",
			}}
		/>
	);
}
