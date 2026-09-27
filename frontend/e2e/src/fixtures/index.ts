import { test as base } from "@playwright/test";
import { AppShellComponent } from "../components/app-shell.component";
import { TweaksPanelComponent } from "../components/tweaks-panel.component";
import { CompaniesPage } from "../pages/companies.page";
import { InsightsPage } from "../pages/insights.page";
import { JobsPage } from "../pages/jobs.page";
import { LoginPage } from "../pages/login.page";
import { OverviewPage } from "../pages/overview.page";

type Fixtures = {
	loginPage: LoginPage;
	jobsPage: JobsPage;
	companiesPage: CompaniesPage;
	overviewPage: OverviewPage;
	insightsPage: InsightsPage;
	appShell: AppShellComponent;
	tweaksPanel: TweaksPanelComponent;
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
	overviewPage: async ({ page }, use) => {
		await use(new OverviewPage(page));
	},
	insightsPage: async ({ page }, use) => {
		await use(new InsightsPage(page));
	},
	appShell: async ({ page }, use) => {
		await use(new AppShellComponent(page, page.locator("aside")));
	},
	tweaksPanel: async ({ page }, use) => {
		await use(new TweaksPanelComponent(page));
	},
});

export { expect } from "@playwright/test";
