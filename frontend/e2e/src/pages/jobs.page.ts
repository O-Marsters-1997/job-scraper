import type { Locator, Page } from "@playwright/test";

export class JobsPage {
	private readonly searchInput: Locator;
	private readonly heading: Locator;
	readonly rows: Locator;

	constructor(private readonly page: Page) {
		this.searchInput = page.getByPlaceholder("Search by role or company…");
		this.rows = page.getByRole("row").filter({
			hasNot: page.getByRole("columnheader"),
		});
		this.heading = page.getByRole("heading", { name: "Jobs" });
	}

	async goto() {
		await this.page.goto("/jobs?page=1");
		await this.heading.waitFor({ state: "visible" });
	}

	async search(query: string) {
		await this.searchInput.fill(query);
	}

	async openFirstJob() {
		await this.rows.first().getByRole("link").first().click();
	}

	firstJobTitleLink() {
		return this.rows.first().getByRole("link").first();
	}
}
