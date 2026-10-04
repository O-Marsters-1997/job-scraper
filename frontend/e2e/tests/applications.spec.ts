import type { Page } from "@playwright/test";
import { expect, test } from "../src/fixtures";

async function openFirstRowApplicationDialog(page: Page) {
	const rows = page
		.getByRole("row")
		.filter({ hasNot: page.getByRole("columnheader") });
	await rows.first().getByRole("button", { name: "Job actions" }).click();
	await page.getByRole("menuitem").nth(1).click();
	return page.getByRole("dialog");
}

test.describe("Applications", () => {
	test("editing a status from /jobs keeps notes, applied date and salary", async ({
		page,
		jobsPage,
	}) => {
		await jobsPage.goto();

		let dialog = await openFirstRowApplicationDialog(page);
		await dialog.getByLabel("Notes").fill("Spoke to the recruiter");
		await dialog.getByLabel("Applied date").fill("2026-03-04");
		await dialog.getByLabel("Salary / comp").fill("£90,000");
		await dialog.getByLabel("Status").selectOption({ label: "Applied" });
		await dialog.getByRole("button", { name: /^(Save|Update)$/ }).click();
		await expect(dialog).toBeHidden();

		dialog = await openFirstRowApplicationDialog(page);
		await dialog.getByLabel("Status").selectOption({ index: 2 });
		await dialog.getByRole("button", { name: "Update" }).click();
		await expect(dialog).toBeHidden();

		dialog = await openFirstRowApplicationDialog(page);
		await expect(dialog.getByLabel("Notes")).toHaveValue(
			"Spoke to the recruiter",
		);
		await expect(dialog.getByLabel("Applied date")).toHaveValue("2026-03-04");
		await expect(dialog.getByLabel("Salary / comp")).toHaveValue("£90,000");
	});

	test("a failing save shows an inline error and keeps the dialog open", async ({
		page,
		jobsPage,
	}) => {
		const unhandled: string[] = [];
		page.on("pageerror", (err) => unhandled.push(err.message));
		await jobsPage.goto();
		await page.evaluate(() =>
			localStorage.setItem("mock-fail", "updateApplication,createApplication"),
		);

		const dialog = await openFirstRowApplicationDialog(page);
		await dialog.getByLabel("Status").selectOption({ index: 1 });
		await dialog.getByRole("button", { name: /^(Save|Update)$/ }).click();

		await expect(
			dialog.getByText("Something went wrong. Please try again."),
		).toBeVisible();
		await expect(dialog).toBeVisible();
		expect(unhandled).toEqual([]);
	});
});
