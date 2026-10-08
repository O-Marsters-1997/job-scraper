import type { Locator, Page } from "@playwright/test";

export class ExperiencePage {
	private readonly heading: Locator;
	readonly positions: Locator;
	readonly bankSkills: Locator;

	constructor(private readonly page: Page) {
		this.heading = page.getByRole("heading", { name: "Experience", level: 1 });
		this.positions = page.getByTestId("position");
		this.bankSkills = page.getByTestId("bank-skill");
	}

	async goto() {
		await this.page.goto("/experience");
		await this.heading.waitFor({ state: "visible" });
	}

	async addPosition(employer: string, title: string) {
		await this.page
			.getByRole("button", { name: "Add position" })
			.first()
			.click();
		await this.page.getByLabel("Employer").fill(employer);
		await this.page.getByLabel("Title").fill(title);
		await this.page
			.getByRole("button", { name: "Add position" })
			.last()
			.click();
	}

	async addAchievement(employer: string, text: string) {
		const card = this.positions.filter({ hasText: employer });
		await card.getByLabel(`New achievement for ${employer}`).fill(text);
		await card.getByRole("button", { name: "Add", exact: true }).click();
	}

	async deletePosition(employer: string) {
		this.page.once("dialog", (dialog) => dialog.accept());
		await this.positions
			.filter({ hasText: employer })
			.getByRole("button", { name: /^Delete .* at /u })
			.click();
	}

	async addBankSkill(name: string, category: string) {
		await this.page.getByLabel("New skill name").fill(name);
		await this.page.getByLabel("New skill category").fill(category);
		await this.page
			.getByTestId("bank-skills")
			.getByRole("button", { name: "Add", exact: true })
			.click();
	}

	async deleteBankSkill(name: string) {
		await this.page.getByRole("button", { name: `Delete ${name}` }).click();
	}
}
