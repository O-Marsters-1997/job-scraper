import { createFileRoute } from "@tanstack/solid-router";
import { AuthForm } from "@/components/auth/AuthForm";

export const Route = createFileRoute("/signup")({
	component: SignupPage,
});

function SignupPage() {
	return (
		<AuthForm
			mode={{
				path: "/auth/signup",
				formId: "signup-form",
				heading: "Create your account",
				subtext: "Start gathering every fresh role in one place.",
				buttonText: "Create account",
				loadingText: "Creating account…",
				passwordAutocomplete: "new-password",
				statusMessages: { 409: "That username is already taken." },
				defaultErrorMessage: "Something went wrong. Please try again.",
				footerText: "Already have an account?",
				footerLinkTo: "/login",
				footerLinkLabel: "Sign in",
			}}
		/>
	);
}
