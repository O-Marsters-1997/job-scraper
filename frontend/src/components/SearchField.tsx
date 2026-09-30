import { Icon } from "@/components/Icon";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

export function SearchField(props: {
	id: string;
	label: string;
	placeholder: string;
	value: string;
	onInput: (value: string) => void;
	class?: string;
}) {
	return (
		<div class={cn("relative w-full sm:max-w-64", props.class)}>
			<Icon
				name="search"
				size={14}
				class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-faint"
			/>
			<Label for={props.id} class="sr-only">
				{props.label}
			</Label>
			<Input
				id={props.id}
				type="search"
				placeholder={props.placeholder}
				value={props.value}
				onInput={(e) => props.onInput(e.currentTarget.value)}
				class="h-8 pr-3 pl-9 text-sm"
			/>
		</div>
	);
}
