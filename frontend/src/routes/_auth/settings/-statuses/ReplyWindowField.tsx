import { Show } from "solid-js";
import { Input } from "@/components/ui/input";
import { parseReplyWindow } from "@/lib/replyWindow";

export function ReplyWindowField(props: {
	value: string;
	onChange: (value: string) => void;
}) {
	const error = () => parseReplyWindow(props.value).error;
	return (
		<div class="flex flex-col gap-1">
			<Input
				class="w-28"
				inputMode="numeric"
				aria-label="Reply window (working days)"
				aria-invalid={error() ? true : undefined}
				placeholder="Reply days"
				value={props.value}
				onInput={(e) => props.onChange(e.currentTarget.value)}
			/>
			<Show when={error()}>
				{(message) => (
					<p role="alert" class="text-xs text-destructive-strong">
						{message()}
					</p>
				)}
			</Show>
		</div>
	);
}
