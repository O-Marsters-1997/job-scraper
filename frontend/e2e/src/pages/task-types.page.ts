import type { Locator, Page } from "@playwright/test";

export class TaskTypesPage {
	private readonly heading: Locator;
	readonly main: Locator;

	constructor(private readonly page: Page) {
		this.heading = page.getByRole("heading", { name: "Task types", level: 1 });
		this.main = page.locator("main");
	}

	async goto() {
		await this.page.goto("/settings/task-types");
		await this.heading.waitFor({ state: "visible" });
	}

	async add(name: string) {
		await this.main.getByRole("button", { name: "Add task type" }).click();
		await this.page.getByLabel("Task type name").fill(name);
		await this.page.getByRole("button", { name: "Add", exact: true }).click();
	}

	async rename(from: string, to: string) {
		await this.row(from).getByRole("button", { name: "Edit" }).click();
		await this.page.getByLabel("Task type name").fill(to);
		await this.page.getByRole("button", { name: "Save" }).click();
	}

	async startDelete(name: string) {
		await this.row(name).getByRole("button", { name: "Delete" }).click();
	}

	row(name: string) {
		return this.main.locator("form", { hasText: name });
	}

	get dialog() {
		return this.page.getByRole("dialog");
	}
}
