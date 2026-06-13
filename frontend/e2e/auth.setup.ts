import * as fs from "node:fs";
import * as path from "node:path";
import { test as setup } from "@playwright/test";
import { LoginPage } from "./src/pages/login.page";

const authFile = ".playwright/.auth/user.json";

setup("authenticate", async ({ page }) => {
	const username = process.env.E2E_USERNAME ?? "demo";
	const password = process.env.E2E_PASSWORD ?? "demo";

	const loginPage = new LoginPage(page);
	await loginPage.goto();
	await loginPage.login(username, password);

	await page.waitForURL(/\/jobs/);

	fs.mkdirSync(path.dirname(authFile), { recursive: true });
	await page.context().storageState({ path: authFile });

	if (!fs.existsSync(authFile)) {
		throw new Error(`Auth state was not saved to ${authFile}`);
	}
});
