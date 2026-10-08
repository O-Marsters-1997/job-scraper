import { faker } from "@faker-js/faker";
import type {
	Achievement,
	BankSkill,
	BankSkillInput,
	ImportPosition,
	ImportPreview,
	ImportSkill,
	Position,
	PositionInput,
} from "@/types/experience";

let experience: Position[] = [
	{
		id: "position-1",
		employer: "Acme Ltd",
		title: "Senior Backend Engineer",
		startDate: "2021-03-01",
		endDate: null,
		achievements: [
			{
				id: "achievement-1",
				positionId: "position-1",
				text: "Cut p99 API latency by 40% by moving hot paths to a read-through cache",
			},
			{
				id: "achievement-2",
				positionId: "position-1",
				text: "Led the migration of 12 services from EC2 to Kubernetes",
			},
		],
	},
	{
		id: "position-2",
		employer: "Globex",
		title: "Software Engineer",
		startDate: "2018-06-01",
		endDate: "2021-02-28",
		achievements: [
			{
				id: "achievement-3",
				positionId: "position-2",
				text: "Built the billing export pipeline used by finance every month",
			},
		],
	},
];

export function getExperience(): Position[] {
	return experience;
}

export function createPosition(input: PositionInput): Position {
	const position: Position = {
		id: `position-${faker.string.uuid().slice(0, 8)}`,
		...input,
		achievements: [],
	};
	experience = [position, ...experience];
	return position;
}

export function updatePosition(id: string, input: PositionInput): Position {
	const existing = experience.find((p) => p.id === id);
	if (!existing) throw new Error("position not found");
	const updated = { ...existing, ...input };
	experience = experience.map((p) => (p.id === id ? updated : p));
	return updated;
}

export function deletePosition(id: string): void {
	experience = experience.filter((p) => p.id !== id);
}

export function previewExperienceImport(): ImportPreview {
	const positions: ImportPosition[] = [
		{
			employer: "Acme Ltd",
			title: "Senior Backend Engineer",
			startDate: "2021-03-01",
			endDate: null,
			achievements: ["Cut p99 API latency by 40%", "Mentored four engineers"],
			employerExists: experience.some((p) => p.employer === "Acme Ltd"),
		},
		{
			employer: "Initech",
			title: "Developer",
			startDate: null,
			endDate: null,
			achievements: ["Maintained the payroll batch jobs"],
			employerExists: false,
		},
	];
	const skills: ImportSkill[] = [
		{ name: "Go", category: "Languages", exists: false },
		{ name: "Docker", category: "Tools", exists: false },
	];
	return { positions, skills };
}

export function importExperience(
	positions: ImportPosition[],
	skills: ImportSkill[],
): Position[] {
	for (const skill of skills) {
		const taken = bankSkills.some(
			(b) => b.name.toLowerCase() === skill.name.toLowerCase(),
		);
		if (!taken) createBankSkill({ name: skill.name, category: skill.category });
	}
	const created = positions.map((p): Position => {
		const id = `position-${faker.string.uuid().slice(0, 8)}`;
		return {
			id,
			employer: p.employer,
			title: p.title,
			startDate: p.startDate,
			endDate: p.endDate,
			achievements: p.achievements.map((text) => ({
				id: `achievement-${faker.string.uuid().slice(0, 8)}`,
				positionId: id,
				text,
			})),
		};
	});
	experience = [...created, ...experience];
	return created;
}

function inOrder<T extends { id: string }>(items: T[], ids: string[]): T[] {
	return ids.flatMap((id) => items.filter((item) => item.id === id));
}

export function reorderPositions(ids: string[]): void {
	experience = inOrder(experience, ids);
}

export function createAchievement(
	positionId: string,
	text: string,
): Achievement {
	const achievement: Achievement = {
		id: `achievement-${faker.string.uuid().slice(0, 8)}`,
		positionId,
		text,
	};
	experience = experience.map((p) =>
		p.id === positionId
			? { ...p, achievements: [...p.achievements, achievement] }
			: p,
	);
	return achievement;
}

export function updateAchievement(id: string, text: string): Achievement {
	let updated: Achievement | undefined;
	experience = experience.map((p) => ({
		...p,
		achievements: p.achievements.map((a) => {
			if (a.id !== id) return a;
			updated = { ...a, text };
			return updated;
		}),
	}));
	if (!updated) throw new Error("achievement not found");
	return updated;
}

export function deleteAchievement(id: string): void {
	experience = experience.map((p) => ({
		...p,
		achievements: p.achievements.filter((a) => a.id !== id),
	}));
}

export function reorderAchievements(positionId: string, ids: string[]): void {
	experience = experience.map((p) =>
		p.id === positionId
			? { ...p, achievements: inOrder(p.achievements, ids) }
			: p,
	);
}

let bankSkills: BankSkill[] = [
	{ id: "skill-1", name: "Go", category: "Languages", sortOrder: 1 },
	{ id: "skill-2", name: "TypeScript", category: "Languages", sortOrder: 2 },
	{ id: "skill-3", name: "PostgreSQL", category: "Databases", sortOrder: 3 },
];

function renumber(skills: BankSkill[]): BankSkill[] {
	return skills.map((s, i) => ({ ...s, sortOrder: i + 1 }));
}

function assertNameFree(name: string, exceptId?: string): void {
	const taken = bankSkills.some(
		(s) => s.id !== exceptId && s.name.toLowerCase() === name.toLowerCase(),
	);
	if (taken) throw new Error("a bank skill with that name already exists");
}

export function getBankSkills(): BankSkill[] {
	return bankSkills;
}

function trimmed(input: BankSkillInput): BankSkillInput {
	return { name: input.name.trim(), category: input.category.trim() };
}

export function createBankSkill(raw: BankSkillInput): BankSkill {
	const input = trimmed(raw);
	assertNameFree(input.name);
	const skill: BankSkill = {
		id: `skill-${faker.string.uuid().slice(0, 8)}`,
		...input,
		sortOrder: bankSkills.length + 1,
	};
	bankSkills = [...bankSkills, skill];
	return skill;
}

export function updateBankSkill(id: string, raw: BankSkillInput): BankSkill {
	const input = trimmed(raw);
	const existing = bankSkills.find((s) => s.id === id);
	if (!existing) throw new Error("bank skill not found");
	assertNameFree(input.name, id);
	const updated = { ...existing, ...input };
	bankSkills = bankSkills.map((s) => (s.id === id ? updated : s));
	return updated;
}

export function deleteBankSkill(id: string): void {
	bankSkills = renumber(bankSkills.filter((s) => s.id !== id));
}

export function reorderBankSkills(ids: string[]): void {
	bankSkills = renumber(inOrder(bankSkills, ids));
}
