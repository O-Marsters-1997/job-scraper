import type { OutreachDraft, OutreachDraftQuery } from "@/types/outreach";
import { getContacts } from "./contacts";
import { getExperience } from "./experience";
import { getOutreachTemplates } from "./outreachTemplates";

export function getOutreachDraft(
	contactId: string,
	query: OutreachDraftQuery,
): OutreachDraft {
	const contact = getContacts().find((c) => c.id === contactId);
	if (!contact) throw new Error("contact not found");
	const kind =
		contact.sent_since_reply > 0
			? "follow_up"
			: /recruit|talent|consultant/i.test(contact.role)
				? "recruiter"
				: "hiring_manager";
	const templates = getOutreachTemplates();
	const template = query.templateId
		? templates.find((t) => t.id === query.templateId)
		: templates.find((t) => t.kind === kind && t.isDefault);
	if (!template) throw new Error("no template to draft from");
	const achievements = getExperience().flatMap((p) => p.achievements);
	const achievement = query.achievementId
		? achievements.find((a) => a.id === query.achievementId)
		: achievements.at(-1);
	const body = template.body
		.replaceAll("{first_name}", contact.name.split(/\s+/)[0] ?? "there")
		.replaceAll("{company}", contact.company_name || "[company]")
		.replaceAll("{role}", contact.job_title || contact.role || "[role]")
		.replaceAll("{achievement}", achievement?.text ?? "[achievement]")
		.replaceAll("{hook}", "[hook]");
	return {
		templateId: template.id,
		kind: template.kind,
		body,
		achievementId: achievement?.id ?? "",
		hasHookSlot: template.body.includes("{hook}"),
	};
}
