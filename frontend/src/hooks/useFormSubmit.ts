import { createSignal } from "solid-js";

const DEFAULT_MESSAGE = "Something went wrong. Please try again.";

export function useFormSubmit(
	run: () => Promise<void>,
	errorMessage: (err: unknown) => string = () => DEFAULT_MESSAGE,
) {
	const [error, setError] = createSignal<string | null>(null);
	const [pending, setPending] = createSignal(false);

	const submit = async (event?: Event) => {
		event?.preventDefault();
		if (pending()) return;
		setError(null);
		setPending(true);
		try {
			await run();
		} catch (err) {
			setError(errorMessage(err));
		} finally {
			setPending(false);
		}
	};

	return { submit, error, setError, pending };
}
