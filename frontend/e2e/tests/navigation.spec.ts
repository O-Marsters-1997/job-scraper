import { expect, test } from "../src/fixtures";

test.describe("Sidebar navigation", () => {
	test.beforeEach(async ({ page }) => {
		await page.goto("/overview");
		await page.locator("aside").waitFor({ timeout: 15_000 });
	});

	test("should load the overview page", async ({ page }) => {
		await expect(page).toHaveURL(/\/overview/);
		await expect(
			page.getByRole("heading", { name: "Overview" }),
		).toBeVisible();
	});

	test("should navigate to jobs via sidebar link", async ({
		page,
		appShell,
	}) => {
		await appShell.navigateTo(appShell.jobsLink);

		await expect(page).toHaveURL(/\/jobs/);
		await expect(page.getByRole("heading", { name: "Jobs" })).toBeVisible();
	});

	test("should navigate to applications via sidebar link", async ({
		page,
		appShell,
	}) => {
		await appShell.navigateTo(appShell.applicationsLink);

		await expect(page).toHaveURL(/\/applications/);
		await expect(
			page.getByRole("heading", { name: "Applications" }),
		).toBeVisible();
	});

	test("should navigate to statuses via sidebar link", async ({
		page,
		appShell,
	}) => {
		await appShell.navigateTo(appShell.statusesLink);

		await expect(page).toHaveURL(/\/settings\/statuses/);
		await expect(
			page.getByRole("heading", { name: "Application statuses" }),
		).toBeVisible();
	});
});
