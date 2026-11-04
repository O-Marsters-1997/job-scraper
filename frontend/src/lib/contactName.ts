import type { Contact } from "@/types/contact";

export function displayName(c: Pick<Contact, "name" | "company_name">): string {
	return c.name ?? `Someone at ${c.company_name || "this company"}`;
}
