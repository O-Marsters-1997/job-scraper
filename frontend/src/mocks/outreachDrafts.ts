import type {
	OutreachDraft,
	OutreachDraftQuery,
	OutreachHookInput,
} from "@/types/outreach";
import { getContacts } from "./contacts";
import { getExperience } from "./experience";
import { getOutreachTemplates } from "./outreachTemplates";

const hooks = new Map<string, { text: string; cost: number }>();
const MOCK_HOOK_COST = 0.0009;

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
	const hook = hooks.get(contactId);
	const body = template.body
		.replaceAll("{first_name}", contact.name.split(/\s+/)[0] ?? "there")
		.replaceAll("{company}", contact.company_name || "[company]")
		.replaceAll("{role}", contact.job_title || contact.role || "[role]")
		.replaceAll("{achievement}", achievement?.text ?? "[achievement]")
		.replaceAll("{hook}", hook?.text || "[hook]");
	return {
		templateId: template.id,
		kind: template.kind,
		body,
		achievementId: achievement?.id ?? "",
		hasHookSlot: template.body.includes("{hook}"),
		hook: hook?.text ?? "",
		hookFindings: [],
		cost: hook?.cost ?? 0,
	};
}

export function writeOutreachHook(
	contactId: string,
	input: OutreachHookInput,
): OutreachDraft {
	const contact = getContacts().find((c) => c.id === contactId);
	if (!contact) throw new Error("contact not found");
	hooks.set(contactId, {
		text: `${contact.company_name || "Your team"}'s work on this role is the kind of problem I like, and I would enjoy doing more of it with a small, focused group of engineers.`,
		cost: (hooks.get(contactId)?.cost ?? 0) + MOCK_HOOK_COST,
	});
	return getOutreachDraft(contactId, { achievementId: input.achievementId });
}
