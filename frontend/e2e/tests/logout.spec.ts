import { expect, test } from "../src/fixtures";

test("should redirect to /login after signing out", async ({
	page,
	appShell,
}) => {
	await page.goto("/overview");
	await appShell.signOut();

	await expect(page).toHaveURL(/\/login/);
});
