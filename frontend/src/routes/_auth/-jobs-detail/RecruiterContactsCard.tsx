import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	useLogRecruiterContact,
	useRecruiterContacts,
} from "@/hooks/useRecruiterContacts";
import { formatDate } from "@/lib/datetime";
import { loadedData } from "@/lib/queryData";
import {
	type ContactChannel,
	type ContactOutcome,
	channelLabels,
	outcomeLabels,
} from "@/types/recruiterContact";

const today = () => {
	const now = new Date();
	const pad = (n: number) => String(n).padStart(2, "0");
	return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
};

export function RecruiterContactsCard(props: { jobId: string }) {
	const contacts = useRecruiterContacts(() => props.jobId);
	const log = useLogRecruiterContact(() => props.jobId);
	const [open, setOpen] = createSignal(false);
	const [date, setDate] = createSignal(today());
	const [channel, setChannel] = createSignal<ContactChannel>("email");
	const [outcome, setOutcome] = createSignal<ContactOutcome>("no_reply");

	const openForm = () => {
		log.reset();
		setDate(today());
		setChannel("email");
		setOutcome("no_reply");
		setOpen(true);
	};

	const submit = (e: SubmitEvent) => {
		e.preventDefault();
		log.mutate(
			{ contacted_on: date(), channel: channel(), outcome: outcome() },
			{ onSuccess: () => setOpen(false) },
		);
	};

	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Recruiter contact</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<Show
					when={open()}
					fallback={
						<Button
							variant="outline"
							size="sm"
							class="w-full"
							onClick={openForm}
						>
							Log recruiter contact
						</Button>
					}
				>
					<form onSubmit={submit} class="flex flex-col gap-3">
						<FormFeedback
							error={log.isError ? "Could not log the contact." : null}
						/>
						<div>
							<Label for="contact-date">Date</Label>
							<Input
								id="contact-date"
								type="date"
								required
								value={date()}
								onInput={(e) => setDate(e.currentTarget.value)}
							/>
						</div>
						<div>
							<Label for="contact-channel">Channel</Label>
							<select
								id="contact-channel"
								class="field"
								value={channel()}
								onChange={(e) =>
									setChannel(e.currentTarget.value as ContactChannel)
								}
							>
								<For each={Object.entries(channelLabels)}>
									{([value, label]) => <option value={value}>{label}</option>}
								</For>
							</select>
						</div>
						<div>
							<Label for="contact-outcome">Outcome</Label>
							<select
								id="contact-outcome"
								class="field"
								value={outcome()}
								onChange={(e) =>
									setOutcome(e.currentTarget.value as ContactOutcome)
								}
							>
								<For each={Object.entries(outcomeLabels)}>
									{([value, label]) => <option value={value}>{label}</option>}
								</For>
							</select>
						</div>
						<div class="flex gap-2">
							<Button type="submit" size="sm" disabled={log.isPending}>
								Save
							</Button>
							<Button
								type="button"
								variant="outline"
								size="sm"
								onClick={() => setOpen(false)}
							>
								Cancel
							</Button>
						</div>
					</form>
				</Show>
				<Show when={(loadedData(contacts)?.length ?? 0) > 0}>
					<ul class="flex flex-col gap-1.5">
						<For each={loadedData(contacts)}>
							{(c) => (
								<li class="flex items-baseline justify-between gap-2 text-xs">
									<span class="font-mono tabular-nums text-foreground">
										{formatDate(`${c.contacted_on}T00:00:00`)}
									</span>
									<span class="text-muted">
										{channelLabels[c.channel]} · {outcomeLabels[c.outcome]}
									</span>
								</li>
							)}
						</For>
					</ul>
				</Show>
			</CardContent>
		</Card>
	);
}
