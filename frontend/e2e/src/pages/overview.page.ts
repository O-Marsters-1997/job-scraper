import type { Locator, Page } from "@playwright/test";

export class OverviewPage {
	private readonly heading: Locator;
	readonly totalJobsValue: Locator;

	constructor(private readonly page: Page) {
		this.heading = page.getByRole("heading", { name: "Overview" });
		this.totalJobsValue = page
			.locator("div.flex.items-center.justify-between.gap-3", {
				hasText: "Total jobs",
			})
			.locator("p.font-mono");
	}

	async goto() {
		await this.page.goto("/overview");
		await this.heading.waitFor({ state: "visible" });
	}
}
