import type { OutreachTemplate, OutreachTemplateInput } from "@/types/outreach";

let templates: OutreachTemplate[] = [
	{
		id: "template-1",
		kind: "recruiter",
		name: "Agency intro",
		body: "Hi {first_name}, I'm a backend engineer looking at {company}'s {role} roles. Recently I {achievement}. Open to a quick chat about what you're hiring for?",
		isDefault: true,
	},
	{
		id: "template-2",
		kind: "hiring_manager",
		name: "Short and direct",
		body: "Hi {first_name}, {hook} I {achievement}, and I'd love to bring that to {company}. Worth 15 minutes?",
		isDefault: true,
	},
	{
		id: "template-3",
		kind: "referral",
		name: "Referral ask",
		body: "Hi {first_name}, I'm applying to {company} and saw you're on the team. {hook} Would you be open to referring me, or telling me what the team's like?",
		isDefault: true,
	},
	{
		id: "template-4",
		kind: "follow_up",
		name: "Gentle bump",
		body: "Hi {first_name}, just bumping this in case it got buried. Still keen on {company}, happy to work around you.",
		isDefault: true,
	},
];
let nextId = 5;

function find(id: string): OutreachTemplate {
	const found = templates.find((t) => t.id === id);
	if (!found) throw new Error("template not found");
	return found;
}

function assertNameFree(kind: string, name: string, exceptId: string) {
	if (
		templates.some(
			(t) => t.kind === kind && t.name === name && t.id !== exceptId,
		)
	) {
		throw new Error("a template of that kind already has that name");
	}
}

export function getOutreachTemplates(): OutreachTemplate[] {
	return [...templates];
}

export function createOutreachTemplate(
	input: OutreachTemplateInput,
): OutreachTemplate {
	assertNameFree(input.kind, input.name, "");
	const created: OutreachTemplate = {
		id: `template-${nextId++}`,
		...input,
		isDefault: !templates.some((t) => t.kind === input.kind),
	};
	templates = [...templates, created];
	return created;
}

export function updateOutreachTemplate(
	id: string,
	input: Pick<OutreachTemplateInput, "name" | "body">,
): OutreachTemplate {
	const current = find(id);
	assertNameFree(current.kind, input.name, id);
	const updated = { ...current, ...input };
	templates = templates.map((t) => (t.id === id ? updated : t));
	return updated;
}

export function makeOutreachTemplateDefault(id: string): OutreachTemplate {
	const current = find(id);
	templates = templates.map((t) =>
		t.kind === current.kind ? { ...t, isDefault: t.id === id } : t,
	);
	return find(id);
}

export function deleteOutreachTemplate(id: string): void {
	const current = find(id);
	const others = templates.filter(
		(t) => t.kind === current.kind && t.id !== id,
	);
	if (current.isDefault && others.length > 0) {
		throw new Error(
			"make another template the default before deleting this one",
		);
	}
	templates = templates.filter((t) => t.id !== id);
}
