import type { Locator, Page } from "@playwright/test";

export class JobsPage {
	private readonly searchInput: Locator;
	private readonly heading: Locator;
	readonly rows: Locator;

	constructor(readonly page: Page) {
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

	async pressPageKey(key: "[" | "]") {
		await this.page.keyboard.press(key);
	}

	async search(query: string) {
		await this.searchInput.fill(query);
	}

	async focusSearchWithKey() {
		await this.page.keyboard.press("/");
	}

	get searchBox() {
		return this.searchInput;
	}

	async openFirstJob() {
		await this.rows.first().getByRole("link").first().click();
	}

	firstJobTitleLink() {
		return this.rows.first().getByRole("link").first();
	}

	async sortByCompany() {
		await this.page.getByRole("columnheader", { name: "Company" }).click();
	}

	firstRowCompanyText() {
		return this.rows.first().locator("td").nth(1);
	}

	async trackFirstJob(statusName: string) {
		await this.rows
			.first()
			.getByRole("button", { name: "Job actions" })
			.click();
		await this.page
			.getByRole("menuitem", { name: "Track application" })
			.click();
		const dialog = this.page.getByRole("dialog");
		await dialog.getByLabel("Status").selectOption({ label: statusName });
		await dialog.getByRole("button", { name: "Save" }).click();
	}

	get gradeDialog() {
		return this.page.getByRole("dialog");
	}

	async openRowMenu(row: Locator) {
		await row.getByRole("button", { name: "Job actions" }).click();
	}

	async selectRows(count: number) {
		for (let i = 0; i < count; i++) {
			await this.rows.nth(i).getByRole("checkbox").check();
		}
	}

	async openBulkGrade() {
		await this.page.getByRole("button", { name: "Grade", exact: true }).click();
	}

	async undoBulkGrade() {
		await this.page
			.getByRole("status")
			.getByRole("button", { name: "Undo" })
			.dispatchEvent("click");
	}
}
