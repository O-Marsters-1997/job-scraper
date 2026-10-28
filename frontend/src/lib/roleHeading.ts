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

type Edit = { from: number; to: number; text: string };

function dateEdit(sample: string, to: RoleText): Edit | undefined {
	const m = DATE_RANGE.exec(sample);
	if (!m) return undefined;
	const [whole, start = "", sep = " - ", end = ""] = m;
	const endHasYear = /\d/.test(end);
	const present = endHasYear ? "Present" : end;
	return {
		from: m.index,
		to: m.index + whole.length,
		text: `${dateLike(start, to.startDate, present)}${sep}${dateLike(endHasYear ? end : start, to.endDate, present)}`,
	};
}

const swap = (text: string, from: string, to: string) =>
	from ? text.replaceAll(from, to) : text;

const fieldEnd = (s: string, from: number) => {
	const i = s.slice(from).search(/[|,\t\n]/);
	return i < 0 ? s.length : from + i;
};
const trimmedEnd = (s: string, from: number, to: number) =>
	from + s.slice(from, to).trimEnd().length;

function fieldEdits(whole: string, to: RoleText): Edit[] {
	const sepAt = fieldEnd(whole, 0);
	const titleEnd = trimmedEnd(whole, 0, sepAt);
	let tailFrom = titleEnd;
	let edits: Edit[] = [
		{ from: 0, to: titleEnd, text: `${to.title}, ${to.employer}` },
	];
	if (whole[sepAt] === "|" || whole[sepAt] === ",") {
		let start = sepAt + 1;
		while (whole[start] === " ") start++;
		const end = trimmedEnd(whole, start, fieldEnd(whole, start));
		edits = [
			{ from: 0, to: titleEnd, text: to.title },
			{ from: start, to: end, text: to.employer },
		];
		tailFrom = end;
	}
	const d = dateEdit(whole.slice(tailFrom), to);
	if (d) edits.push({ ...d, from: d.from + tailFrom, to: d.to + tailFrom });
	return edits;
}

function applyEdits(texts: string[], edits: Edit[]): string[] {
	const total = texts.reduce((n, t) => n + t.length, 0);
	let runStart = 0;
	return texts.map((t, i) => {
		const runEnd = runStart + t.length;
		const base = runStart;
		let out = "";
		let cursor = base;
		for (const e of edits) {
			const from = Math.min(Math.max(e.from, base), runEnd);
			const to = Math.min(Math.max(e.to, base), runEnd);
			const owns =
				base <= e.from &&
				(e.from < runEnd || (e.from === total && i === texts.length - 1));
			if (from >= to && !owns) continue;
			out += t.slice(cursor - base, Math.max(cursor, from) - base);
			if (owns) out += e.text;
			cursor = Math.max(cursor, to);
		}
		out += t.slice(cursor - base);
		runStart = runEnd;
		return out;
	});
}

export function cloneRoleHeading(
	template: LayoutBlock,
	from: Pick<RoleText, "title" | "employer">,
	to: RoleText,
): LayoutBlock {
	const whole = template.runs.map((r) => r.text).join("");
	const named =
		(from.title !== "" && whole.includes(from.title)) ||
		(from.employer !== "" && whole.includes(from.employer));
	if (!named) {
		const texts = applyEdits(
			template.runs.map((r) => r.text),
			fieldEdits(whole, to),
		);
		return {
			...template,
			runs: template.runs.map((r, i) => ({ ...r, text: texts[i] ?? r.text })),
		};
	}
	return {
		...template,
		runs: template.runs.map((r) => {
			const text = swap(
				swap(r.text, from.title, to.title),
				from.employer,
				to.employer,
			);
			const d = dateEdit(text, to);
			return {
				...r,
				text: d ? text.slice(0, d.from) + d.text + text.slice(d.to) : text,
			};
		}),
	};
}
