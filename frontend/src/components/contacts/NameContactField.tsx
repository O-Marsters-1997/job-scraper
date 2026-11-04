import { createSignal } from "solid-js";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useNameContact } from "@/hooks/useContacts";
import type { Contact } from "@/types/contact";

export function NameContactField(props: { contact: Contact }) {
	const nameContact = useNameContact();
	const [name, setName] = createSignal("");

	const save = (e: Event) => {
		e.preventDefault();
		if (name().trim() === "" || nameContact.isPending) return;
		nameContact.mutate({ id: props.contact.id, name: name().trim() });
	};

	return (
		<form onSubmit={save} class="flex shrink-0 items-center gap-2">
			<Input
				class="w-36"
				aria-label="Name"
				placeholder="Name not added yet"
				maxLength={200}
				value={name()}
				onInput={(e) => setName(e.currentTarget.value)}
			/>
			<Button
				type="submit"
				size="sm"
				loading={nameContact.isPending}
				disabled={name().trim() === ""}
			>
				Save
			</Button>
		</form>
	);
}
