import { format } from "date-fns";
import type { ApplicationWithDetails } from "@/types/application";
import type { Contact } from "@/types/contact";
import { chaseDateKey } from "./chase";

export type ChaseRow =
	| { kind: "contact"; contact: Contact; applicationStatus: string | undefined }
	| {
			kind: "application";
			app: ApplicationWithDetails;
			via: Contact | undefined;
	  };

function slugify(name: string): string {
	return name
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, "-")
		.replace(/^-|-$/g, "");
}

export function chaseNowRows(
	contacts: Contact[],
	dueApps: ApplicationWithDetails[],
	now: Date,
	allApps: ApplicationWithDetails[] = dueApps,
): ChaseRow[] {
	const today = format(now, "yyyy-MM-dd");
	const dueContacts = contacts.filter(
		(c) => c.follow_up_by !== null && chaseDateKey(c.follow_up_by) <= today,
	);
	const linked = new Set(
		dueContacts.flatMap((c) => (c.application_id ? [c.application_id] : [])),
	);
	const appById = new Map(allApps.map((a) => [a.ID, a]));
	return [
		...dueContacts.map(
			(contact): ChaseRow => ({
				kind: "contact",
				contact,
				applicationStatus: contact.application_id
					? appById.get(contact.application_id)?.StatusName
					: undefined,
			}),
		),
		...dueApps
			.filter((a) => !linked.has(a.ID))
			.map(
				(app): ChaseRow => ({
					kind: "application",
					app,
					via: contacts.find(
						(c) =>
							c.stage !== "closed" &&
							app.JobCompanySlug !== "" &&
							slugify(c.company_name) === app.JobCompanySlug,
					),
				}),
			),
	];
}
