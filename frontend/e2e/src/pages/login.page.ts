import type { Locator, Page } from "@playwright/test";

export class LoginPage {
	private readonly username: Locator;
	private readonly password: Locator;
	private readonly submitButton: Locator;

	constructor(private readonly page: Page) {
		this.username = page.locator("#username");
		this.password = page.locator("#password");
		this.submitButton = page.getByRole("button", { name: /sign in/i });
	}

	async goto() {
		await this.page.goto("/login");
	}

	async login(username: string, password: string) {
		await this.username.fill(username);
		await this.password.fill(password);
		await this.submitButton.click();
	}
}
