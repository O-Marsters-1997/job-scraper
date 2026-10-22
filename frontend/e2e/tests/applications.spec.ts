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

async function openFirstApplication(page: Page) {
	await page.goto("/applications");
	await page.locator('a[href^="/applications/"]').first().click();
	await page.getByRole("heading", { level: 1 }).waitFor();
}

test.describe("Application page", () => {
	test("tabs show only their own content", async ({ page }) => {
		await openFirstApplication(page);

		await expect(page.getByRole("tab", { name: "Checklist" })).toHaveAttribute(
			"aria-selected",
			"true",
		);
		await expect(page.getByLabel("Note", { exact: true })).toBeHidden();

		await page.getByRole("tab", { name: "Notes" }).click();
		await expect(page.getByLabel("Note", { exact: true })).toBeVisible();

		await page.getByRole("tab", { name: "Description" }).click();
		await expect(page.getByLabel("Note", { exact: true })).toBeHidden();
		await expect(
			page.getByRole("heading", { name: "Description" }),
		).toBeVisible();
	});

	test("confirming the application moves it to Applied and ticks Apply", async ({
		page,
	}) => {
		await openFirstApplication(page);
		const status = page.getByRole("button", { name: "Status" });
		await status.click();
		await page.getByRole("option", { name: "Draft" }).click();
		await expect(status).toContainText("Draft");

		const trigger = page.getByRole("button", { name: /Apply on/ });
		const region = page.getByRole("region", { name: /Apply on/ });
		await expect(trigger.getByRole("img", { name: "To do" })).toBeVisible();
		await trigger.click();
		const popup = page.waitForEvent("popup");
		await region.getByRole("link", { name: /Open application/ }).click();
		await (await popup).close();
		await expect(region.getByText("Did you send it?")).toBeVisible();
		await region.getByRole("button", { name: "Yes, I applied" }).click();

		await expect(status).toContainText("Applied");
		await expect(trigger.getByRole("img", { name: "Done" })).toBeVisible();
	});

	test("checklist rows start collapsed and open one at a time", async ({
		page,
	}) => {
		await openFirstApplication(page);
		const cv = page.getByRole("button", { name: /^(To do|Done) CV/ });
		const apply = page.getByRole("button", { name: /Apply on/ });
		const cvRegion = page.getByRole("region", { name: /^(To do|Done) CV/ });
		await expect(cv).toHaveAttribute("aria-expanded", "false");
		await expect(apply).toHaveAttribute("aria-expanded", "false");

		await cv.click();
		await expect(cvRegion).toBeVisible();
		await apply.click();
		await expect(page.getByRole("region", { name: /Apply on/ })).toBeVisible();
		await expect(cvRegion).toBeHidden();
	});

	test("the Did you send it prompt survives collapsing Apply", async ({
		page,
	}) => {
		await openFirstApplication(page);
		const status = page.getByRole("button", { name: "Status" });
		await status.click();
		await page.getByRole("option", { name: "Draft" }).click();
		await expect(status).toContainText("Draft");

		const trigger = page.getByRole("button", { name: /Apply on/ });
		await trigger.click();
		const popup = page.waitForEvent("popup");
		await page
			.getByRole("region", { name: /Apply on/ })
			.getByRole("link", { name: /Open application/ })
			.click();
		await (await popup).close();
		await trigger.click();
		await expect(trigger).toHaveAttribute("aria-expanded", "false");

		await page.evaluate(() => window.dispatchEvent(new Event("focus")));
		await expect(page.getByText("Did you send it?")).toBeVisible();
	});

	test("adding a task lists it and offers no Apply type", async ({ page }) => {
		await openFirstApplication(page);
		await page.getByRole("button", { name: "Add task" }).click();
		const dialog = page.getByRole("dialog");
		await expect(
			dialog.getByRole("option", { name: "Apply", exact: true }),
		).toHaveCount(0);
		await dialog.getByLabel("Notes").fill("Email the recruiter");
		await dialog.getByRole("button", { name: "Save" }).click();
		await expect(dialog).toBeHidden();

		const row = page.getByRole("button", { name: /Task$/ });
		await expect(row).toBeVisible();
		await row.click();
		const region = page.getByRole("region", { name: /Task$/ });
		await expect(region).toContainText("Email the recruiter");
		await region.getByRole("button", { name: "Done" }).click();
		await expect(row).toBeHidden();
	});

	test("the status dropdown saves and the list reflects it", async ({
		page,
	}) => {
		await openFirstApplication(page);
		const path = new URL(page.url()).pathname;

		const status = page.getByRole("button", { name: "Status" });
		await status.click();
		await page.getByRole("option", { name: "Rejected" }).click();
		await expect(status).toContainText("Rejected");

		await page
			.getByRole("navigation", { name: "Breadcrumb" })
			.getByRole("link", { name: "Applications" })
			.click();
		await expect(
			page
				.locator("div.items-center")
				.filter({ has: page.locator(`a[href="${path}"]`) }),
		).toContainText("Rejected");
	});

	test("the breadcrumb shows the job title and the listing link opens the job URL", async ({
		page,
	}) => {
		await openFirstApplication(page);
		const title = await page.getByRole("heading", { level: 1 }).innerText();

		await expect(
			page.getByRole("navigation", { name: "Breadcrumb" }),
		).toContainText(title);
		const listing = page.getByRole("link", { name: /Original listing/ });
		await expect(listing).toHaveAttribute("target", "_blank");
		await expect(listing).toHaveAttribute("href", /^https?:\/\//);
	});
});

test.describe("Applications", () => {
	test("editing a status from /jobs keeps applied date and salary", async ({
		page,
		jobsPage,
	}) => {
		await jobsPage.goto();

		let dialog = await openFirstRowApplicationDialog(page);
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
