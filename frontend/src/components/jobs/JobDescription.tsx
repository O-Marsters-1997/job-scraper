import DOMPurify from "dompurify";
import { For, Show } from "solid-js";

interface Props {
	html: string;
}

/** Decode HTML entities (handles Greenhouse's entity-encoded content). */
function decodeEntities(s: string): string {
	const el = document.createElement("textarea");
	el.innerHTML = s;
	return el.value;
}

function looksHtml(s: string): boolean {
	return /<[a-z][\s\S]*?>/i.test(s);
}

export default function JobDescription(props: Props) {
	const decoded = () => decodeEntities(props.html);

	return (
		<Show
			when={looksHtml(decoded())}
			fallback={
				<div class="flex flex-col gap-3 text-sm leading-relaxed text-foreground">
					<For each={decoded().split("\n\n")}>{(para) => <p>{para}</p>}</For>
				</div>
			}
		>
			<div class="job-description" innerHTML={DOMPurify.sanitize(decoded())} />
		</Show>
	);
}
