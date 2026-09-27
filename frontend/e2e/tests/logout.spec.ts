import { expect, test } from "../src/fixtures";

test("should redirect to /login after signing out", async ({
	page,
	appShell,
}) => {
	await page.goto("/overview");
	await appShell.signOut();

	await expect(page).toHaveURL(/\/login/);
});

test("clears cached data between users", async ({
	page,
	appShell,
	loginPage,
}) => {
	await page.goto("/overview");
	await appShell.navigateTo(appShell.applicationsLink);
	await expect(
		page.getByText("No applications yet. Track a job from the Jobs page."),
	).not.toBeVisible();

	await appShell.signOut();
	await loginPage.login("second-user", "second-password");
	await page.waitForURL(/\/jobs/);
	await appShell.navigateTo(appShell.applicationsLink);

	await expect(
		page.getByText("No applications yet. Track a job from the Jobs page."),
	).toBeVisible();
});
