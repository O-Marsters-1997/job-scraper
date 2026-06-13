import { test as base } from "@playwright/test";
import { AppShellComponent } from "../components/app-shell.component";
import { JobsPage } from "../pages/jobs.page";
import { LoginPage } from "../pages/login.page";

type Fixtures = {
	loginPage: LoginPage;
	jobsPage: JobsPage;
	appShell: AppShellComponent;
};

export const test = base.extend<Fixtures>({
	loginPage: async ({ page }, use) => {
		await use(new LoginPage(page));
	},
	jobsPage: async ({ page }, use) => {
		await use(new JobsPage(page));
	},
	appShell: async ({ page }, use) => {
		await use(new AppShellComponent(page, page.locator("aside")));
	},
});

export { expect } from "@playwright/test";
