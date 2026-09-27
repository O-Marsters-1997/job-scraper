import type { Locator, Page } from "@playwright/test";

export class MobileNavComponent {
	readonly toggle: Locator;
	readonly dialog: Locator;

	constructor(page: Page) {
		this.toggle = page.getByRole("button", { name: "Open menu" });
		this.dialog = page.getByRole("dialog");
	}

	async open() {
		await this.toggle.click();
		await this.dialog.waitFor({ state: "visible" });
	}

	link(name: string) {
		return this.dialog.getByRole("link", { name, exact: true });
	}
}
