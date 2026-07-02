import type { Locator, Page } from "@playwright/test";

export class CompaniesPage {
	private readonly searchInput: Locator;
	private readonly heading: Locator;
	readonly rows: Locator;

	constructor(private readonly page: Page) {
		this.searchInput = page.getByPlaceholder("Search companies…");
		this.rows = page.getByRole("row").filter({
			hasNot: page.getByRole("columnheader"),
		});
		this.heading = page.getByRole("heading", { name: "Companies" });
	}

	async goto() {
		await this.page.goto("/companies");
		await this.heading.waitFor({ state: "visible" });
	}

	async search(query: string) {
		await this.searchInput.fill(query);
	}

	async openFirstCompany() {
		await this.rows.first().getByRole("link").first().click();
	}

	firstCompanyNameLink() {
		return this.rows.first().getByRole("link").first();
	}
}
