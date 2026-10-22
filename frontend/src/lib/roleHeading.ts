import { splitAtTab } from "@/lib/docLayout";
import type { LayoutBlock } from "@/types/tailoring";

export type RoleText = {
	title: string;
	employer: string;
	startDate: string | null;
	endDate: string | null;
};

const MONTH = "(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.? +";
const DATE_RANGE = new RegExp(
	`((?:${MONTH})?\\d{4})( *[–—-] *)((?:${MONTH})?\\d{4}|[A-Za-z]+)`,
);

function dateLike(sample: string, iso: string | null, present: string) {
	if (!iso) return present;
	const d = new Date(iso);
	const month = /^[A-Za-z]+/.exec(sample)?.[0];
	if (!month) return String(d.getFullYear());
	const name = d.toLocaleDateString("en-GB", { month: "long" });
	return `${month.length > 3 ? name : name.slice(0, 3)} ${d.getFullYear()}`;
}

function datesLike(sample: string, to: RoleText) {
	const m = DATE_RANGE.exec(sample);
	if (!m) return undefined;
	const [, start = "", sep = " - ", end = ""] = m;
	const endHasYear = /\d/.test(end);
	const present = endHasYear ? "Present" : end;
	return `${dateLike(start, to.startDate, present)}${sep}${dateLike(endHasYear ? end : start, to.endDate, present)}`;
}

export function cloneRoleHeading(
	template: LayoutBlock,
	from: Pick<RoleText, "title" | "employer">,
	to: RoleText,
): LayoutBlock {
	const whole = template.runs.map((r) => r.text).join("");
	const named = whole.includes(from.title) || whole.includes(from.employer);
	if (!named) {
		const [left, right] = splitAtTab(template.runs);
		const lead = left[0] ?? template.runs[0];
		const tail = right[0];
		const dates = datesLike(right.map((r) => r.text).join(""), to);
		return {
			...template,
			runs: [
				...(lead ? [{ ...lead, text: `${to.title}, ${to.employer}` }] : []),
				...(tail && dates ? [{ ...tail, text: `\t${dates}` }] : []),
			],
		};
	}
	return {
		...template,
		runs: template.runs.map((r) => {
			const text = r.text
				.replaceAll(from.title, to.title)
				.replaceAll(from.employer, to.employer);
			const dates = datesLike(text, to);
			return { ...r, text: dates ? text.replace(DATE_RANGE, dates) : text };
		}),
	};
}
