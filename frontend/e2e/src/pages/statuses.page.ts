import type { Locator, Page } from "@playwright/test";

export class StatusesPage {
	private readonly heading: Locator;
	readonly rows: Locator;

	constructor(private readonly page: Page) {
		this.heading = page.getByRole("heading", { name: "Statuses", level: 1 });
		this.rows = page.locator("main").getByRole("button", { name: "Edit" });
	}

	async goto() {
		await this.page.goto("/settings/statuses");
		await this.heading.waitFor({ state: "visible" });
	}

	async renameFirstStatus(newName: string) {
		await this.rows.first().click();
		const nameInput = this.page.getByLabel("Status name");
		await nameInput.fill(newName);
		await this.page.getByRole("button", { name: "Save" }).click();
	}

	statusText(name: string) {
		return this.page.locator("main").getByText(name, { exact: true });
	}
}
