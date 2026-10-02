import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "../src/fixtures";

const SERIOUS_OR_WORSE = ["serious", "critical"];

const ROUTES = [
	"/overview",
	"/jobs",
	"/companies",
	"/applications",
	"/cv-templates",
	"/settings/profile",
	"/settings/scoring/role",
	"/settings/scoring/stack",
	"/settings/scoring/filters",
	"/settings/scoring/other",
	"/settings/ai",
	"/settings/statuses",
	"/settings/searches",
	"/settings/integrations",
	"/tailoring/drafts/draft-ready",
];

test.describe("Accessibility", () => {
	for (const route of ROUTES) {
		test(`${route} has no serious or critical violations`, async ({ page }) => {
			await page.goto(route);
			await page
				.getByRole("heading", { level: 1 })
				.waitFor({ state: "visible" });

			const results = await new AxeBuilder({ page })
				.withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
				.analyze();

			const serious = results.violations.filter((v) =>
				SERIOUS_OR_WORSE.includes(v.impact ?? ""),
			);
			expect(
				serious,
				serious
					.map(
						(v) =>
							`${v.id}: ${v.help}\n${v.nodes.map((n) => n.html).join("\n")}`,
					)
					.join("\n\n"),
			).toEqual([]);
		});
	}

	test("the Draft's wand menu has no serious or critical violations", async ({
		page,
		draftPage,
	}) => {
		await draftPage.goto();
		await draftPage.line("Senior Backend Engineer bullet 1").click();
		await draftPage.wand.click();
		await page.getByRole("menu").waitFor({ state: "visible" });

		const results = await new AxeBuilder({ page })
			.withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
			.analyze();
		expect(
			results.violations.filter((v) =>
				SERIOUS_OR_WORSE.includes(v.impact ?? ""),
			),
		).toEqual([]);
	});
});
