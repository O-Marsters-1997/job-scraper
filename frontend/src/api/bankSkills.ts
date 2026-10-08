import {
	type BankSkill,
	type BankSkillInput,
	bankSkillSchema,
} from "../types/experience";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchBankSkills(): Promise<BankSkill[]> {
	return mocked(
		(db) => db.getBankSkills(),
		() => apiFetch("/experience/skills", bankSkillSchema.array()),
	);
}

export async function createBankSkill(
	input: BankSkillInput,
): Promise<BankSkill> {
	return mocked(
		(db) => db.createBankSkill(input),
		() =>
			apiFetch("/experience/skills", bankSkillSchema, jsonInit("POST", input)),
	);
}

export async function updateBankSkill(
	id: string,
	input: BankSkillInput,
): Promise<BankSkill> {
	return mocked(
		(db) => db.updateBankSkill(id, input),
		() =>
			apiFetch(
				`/experience/skills/${id}`,
				bankSkillSchema,
				jsonInit("PATCH", input),
			),
	);
}

export async function deleteBankSkill(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteBankSkill(id),
		() => apiFetchVoid(`/experience/skills/${id}`, { method: "DELETE" }),
	);
}

export async function reorderBankSkills(ids: string[]): Promise<void> {
	return mocked(
		(db) => db.reorderBankSkills(ids),
		() => apiFetchVoid("/experience/skills/order", jsonInit("PUT", { ids })),
	);
}
