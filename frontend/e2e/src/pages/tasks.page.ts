import type { Locator, Page } from "@playwright/test";

export class TasksPage {
	private readonly heading: Locator;
	readonly rows: Locator;
	readonly navBadge: Locator;

	constructor(private readonly page: Page) {
		this.heading = page.getByRole("heading", { name: "Tasks", level: 1 });
		this.rows = page.locator("main").getByRole("listitem");
		this.navBadge = page.getByRole("link", { name: /^Tasks/ }).first();
	}

	async goto() {
		await this.page.goto("/tasks");
		await this.heading.waitFor({ state: "visible" });
	}

	async filterByType(name: string) {
		await this.page.getByRole("button", { name: "Filter by type" }).click();
		await this.page.getByRole("option", { name }).click();
	}

	async sortByJobPosted() {
		await this.page.getByRole("button", { name: "Sort tasks" }).click();
		await this.page.getByRole("option", { name: "Job posted" }).click();
	}

	async toggleShowDone() {
		await this.page.getByRole("button", { name: "Show done" }).click();
	}

	async markFirstDone() {
		await this.rows.first().getByRole("checkbox").check();
	}
}
