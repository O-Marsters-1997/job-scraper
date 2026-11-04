import {
	type ContactEvent,
	channelLabels,
	replyOutcomes,
} from "@/types/contact";
import { outreachKinds } from "@/types/outreach";

export function eventLabel(e: ContactEvent): string {
	const outcome = replyOutcomes.find((o) => o.key === e.outcome)?.label;
	switch (e.kind) {
		case "added":
			return "Added";
		case "sent":
			return e.template_kind && e.template_name
				? [
						e.channel ? channelLabels[e.channel] : "Sent",
						outreachKinds.find((k) => k.key === e.template_kind)?.label,
						e.template_name,
					]
						.filter(Boolean)
						.join(" · ")
				: `Sent${e.channel ? ` by ${channelLabels[e.channel]}` : ""}`;
		case "reply":
			return `Replied${outcome ? `: ${outcome}` : ""}`;
		case "closed":
			return "Closed: no reply";
	}
}
