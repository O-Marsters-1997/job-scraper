import { test as base } from "@playwright/test";
import { AppShellComponent } from "../components/app-shell.component";
import { MobileNavComponent } from "../components/mobile-nav.component";
import { TweaksPanelComponent } from "../components/tweaks-panel.component";
import { CompaniesPage } from "../pages/companies.page";
import { DraftPage, PhoneDraftPage } from "../pages/draft.page";
import { ExperiencePage } from "../pages/experience.page";
import { InsightsPage } from "../pages/insights.page";
import { JobsPage } from "../pages/jobs.page";
import { LoginPage } from "../pages/login.page";
import { OverviewPage } from "../pages/overview.page";
import { ScoringPage } from "../pages/scoring.page";
import { StatusesPage } from "../pages/statuses.page";

type Fixtures = {
	loginPage: LoginPage;
	jobsPage: JobsPage;
	companiesPage: CompaniesPage;
	draftPage: DraftPage;
	phoneDraftPage: PhoneDraftPage;
	overviewPage: OverviewPage;
	experiencePage: ExperiencePage;
	insightsPage: InsightsPage;
	statusesPage: StatusesPage;
	appShell: AppShellComponent;
	tweaksPanel: TweaksPanelComponent;
	mobileNav: MobileNavComponent;
	scoringPage: ScoringPage;
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
	draftPage: async ({ page }, use) => {
		await use(new DraftPage(page));
	},
	phoneDraftPage: async ({ page }, use) => {
		await use(new PhoneDraftPage(page));
	},
	overviewPage: async ({ page }, use) => {
		await use(new OverviewPage(page));
	},
	experiencePage: async ({ page }, use) => {
		await use(new ExperiencePage(page));
	},
	insightsPage: async ({ page }, use) => {
		await use(new InsightsPage(page));
	},
	statusesPage: async ({ page }, use) => {
		await use(new StatusesPage(page));
	},
	appShell: async ({ page }, use) => {
		await use(new AppShellComponent(page, page.locator("aside")));
	},
	tweaksPanel: async ({ page }, use) => {
		await use(new TweaksPanelComponent(page));
	},
	mobileNav: async ({ page }, use) => {
		await use(new MobileNavComponent(page));
	},
	scoringPage: async ({ page }, use) => {
		await use(new ScoringPage(page));
	},
});

export { expect } from "@playwright/test";
