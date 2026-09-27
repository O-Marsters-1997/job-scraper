import { test as base } from "@playwright/test";
import { AppShellComponent } from "../components/app-shell.component";
import { CompaniesPage } from "../pages/companies.page";
import { JobsPage } from "../pages/jobs.page";
import { LoginPage } from "../pages/login.page";
import { StatusesPage } from "../pages/statuses.page";

type Fixtures = {
	loginPage: LoginPage;
	jobsPage: JobsPage;
	companiesPage: CompaniesPage;
	statusesPage: StatusesPage;
	appShell: AppShellComponent;
};

export const test = base.extend<Fixtures>({
	loginPage: async ({ page }, use) => {
		await use(new LoginPage(page));
	},
	jobsPage: async ({ page }, use) => {
		await use(new JobsPage(page));
	},
	companiesPage: async ({ page }, use) => {
		await use(new CompaniesPage(page));
	},
	statusesPage: async ({ page }, use) => {
		await use(new StatusesPage(page));
	},
	appShell: async ({ page }, use) => {
		await use(new AppShellComponent(page, page.locator("aside")));
	},
});

export { expect } from "@playwright/test";
