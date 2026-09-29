import { createSignal, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { PositionInput } from "../../types/experience";

export function PositionForm(props: {
	initial?: PositionInput;
	submitLabel: string;
	pending: boolean;
	error?: string | undefined;
	onSubmit: (input: PositionInput) => void;
	onCancel: () => void;
}) {
	const [employer, setEmployer] = createSignal(props.initial?.employer ?? "");
	const [title, setTitle] = createSignal(props.initial?.title ?? "");
	const [startDate, setStartDate] = createSignal(
		props.initial?.startDate ?? "",
	);
	const [endDate, setEndDate] = createSignal(props.initial?.endDate ?? "");

	const valid = () => employer().trim() !== "" && title().trim() !== "";

	const submit = (e: SubmitEvent) => {
		e.preventDefault();
		if (!valid()) return;
		props.onSubmit({
			employer: employer().trim(),
			title: title().trim(),
			startDate: startDate() || null,
			endDate: endDate() || null,
		});
	};

	return (
		<form onSubmit={submit} class="grid gap-3 sm:grid-cols-2">
			<div>
				<Label for="position-employer">Employer</Label>
				<Input
					id="position-employer"
					value={employer()}
					onInput={(e) => setEmployer(e.currentTarget.value)}
					required
				/>
			</div>
			<div>
				<Label for="position-title">Title</Label>
				<Input
					id="position-title"
					value={title()}
					onInput={(e) => setTitle(e.currentTarget.value)}
					required
				/>
			</div>
			<div>
				<Label for="position-start">Start date</Label>
				<Input
					id="position-start"
					type="date"
					value={startDate()}
					onInput={(e) => setStartDate(e.currentTarget.value)}
				/>
			</div>
			<div>
				<Label for="position-end">End date (blank if current)</Label>
				<Input
					id="position-end"
					type="date"
					value={endDate()}
					onInput={(e) => setEndDate(e.currentTarget.value)}
				/>
			</div>
			<Show when={props.error}>
				<p role="alert" class="text-sm text-destructive-strong sm:col-span-2">
					{props.error}
				</p>
			</Show>
			<div class="flex gap-2 sm:col-span-2">
				<Button type="submit" disabled={!valid() || props.pending}>
					{props.submitLabel}
				</Button>
				<Button type="button" variant="ghost" onClick={props.onCancel}>
					Cancel
				</Button>
			</div>
		</form>
	);
}
