import { expect, test } from "../src/fixtures";

test("shows an updated score on the job detail page after recompute, without reloading", async ({
	page,
	appShell,
}) => {
	await page.goto("/jobs?page=1");
	const rows = page
		.getByRole("row")
		.filter({ hasNot: page.getByRole("columnheader") });
	await expect(rows.first()).toBeVisible();
	const rowCount = await rows.count();

	let scoredRow = null;
	for (let i = 0; i < rowCount; i++) {
		const cell = rows.nth(i).locator("td").nth(4);
		const text = (await cell.textContent())?.trim();
		if (text && text !== "—") {
			scoredRow = rows.nth(i);
			break;
		}
	}
	if (!scoredRow) throw new Error("No scored job found on the first page");

	await scoredRow.getByRole("link").first().click();
	await expect(page).toHaveURL(/\/jobs\/.+/);
	const jobUrl = page.url();

	const suitabilityCard = page.locator("div.rounded-xl", {
		has: page.getByRole("heading", { name: "Suitability", level: 3 }),
	});
	const scoreValue = suitabilityCard.locator("span.font-mono").first();
	const before = Number(await scoreValue.textContent());

	await appShell.navigateTo(appShell.settingsLink);
	await page
		.getByRole("navigation", { name: "Settings sections" })
		.getByRole("link", { name: /^Scoring/ })
		.click();
	await page.getByRole("button", { name: "Recompute scores" }).click();
	await expect(page.getByText(/jobs re-ranked\./)).toBeVisible();

	for (let i = 0; i < 5 && page.url() !== jobUrl; i++) {
		await page.goBack();
	}
	await expect(page).toHaveURL(jobUrl);

	await expect(scoreValue).toHaveText(String(Math.min(100, before + 1)));
});
