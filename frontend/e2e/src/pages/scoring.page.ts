import type { Locator, Page } from "@playwright/test";

export class ScoringPage {
	private readonly heading: Locator;
	readonly excludedCompaniesCombobox: Locator;

	constructor(private readonly page: Page) {
		this.heading = page.getByRole("heading", { name: "Scoring", level: 1 });
		this.excludedCompaniesCombobox = page.getByRole("combobox", {
			name: "Excluded companies",
		});
	}

	async gotoFilters() {
		await this.page.goto("/settings/scoring/filters");
		await this.heading.waitFor({ state: "visible" });
	}
}
